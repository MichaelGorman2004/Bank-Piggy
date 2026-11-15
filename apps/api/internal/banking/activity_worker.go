package banking

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

const activityJobTimeout = 90 * time.Second

type activityWork struct {
	ID, HouseholdID, Kind, Cipher, Owner, Cursor string
	Generation, ConnectionGeneration             int64
}

type activityBatch struct {
	Records    []ProviderActivityRecord
	Removed    []string
	Cursor     string
	Start, End string
	Complete   bool
}

func (s *Service) claimActivityWork(ctx context.Context, id string) (activityWork, error) {
	owner, err := opaqueID()
	if err != nil {
		return activityWork{}, err
	}
	var job activityWork
	err = s.db.Pool.QueryRow(ctx, `WITH candidate AS (
 SELECT j.connection_id FROM bank_activity_jobs j JOIN bank_connections c ON c.id=j.connection_id
 WHERE ($1='' OR c.id=$1) AND c.environment=$2 AND c.token_cipher IS NOT NULL
 AND c.state IN ('connected','delayed') AND j.enabled AND j.authorized_at IS NOT NULL
 AND EXISTS(SELECT 1 FROM jsonb_array_elements(j.accounts) a WHERE a->>'enabled'='true')
 AND ((c.kind='investment' AND $3) OR (c.kind='depository' AND $4))
 AND j.failures<$5 AND j.next_sync_at<=now() AND (j.lease_until IS NULL OR j.lease_until<=now())
 ORDER BY j.next_sync_at,j.connection_id FOR UPDATE OF j SKIP LOCKED LIMIT 1)
 UPDATE bank_activity_jobs j SET lease_owner=$6,lease_until=now()+$7::interval,next_sync_at=now()+$8::interval
 FROM candidate,bank_connections c WHERE j.connection_id=candidate.connection_id AND c.id=j.connection_id
 RETURNING c.id,c.household_id,c.kind,c.token_cipher,c.generation,j.generation,j.cursor`, id, s.config.Environment, s.activityAllowed("investment"), s.activityAllowed("depository"), MaxFailures, owner, LeaseDuration.String(), SyncInterval.String()).Scan(&job.ID, &job.HouseholdID, &job.Kind, &job.Cipher, &job.ConnectionGeneration, &job.Generation, &job.Cursor)
	job.Owner = owner
	return job, err
}

func (s *Service) processActivity(ctx context.Context, id string) error {
	job, err := s.claimActivityWork(ctx, id)
	if err != nil {
		return err
	}
	token, err := s.vault.Open(job.Cipher, credentialContext(job.HouseholdID, job.ID, "access"))
	if err != nil {
		return s.failActivity(ctx, job, err)
	}
	batch, err := s.collectActivity(ctx, job, token)
	if err != nil {
		return s.failActivity(ctx, job, err)
	}
	err = s.saveActivity(ctx, job, batch)
	if err == nil || errors.Is(err, ErrConflict) {
		return err
	}
	return errors.Join(err, s.failActivity(ctx, job, err))
}

func (s *Service) failActivity(ctx context.Context, job activityWork, original error) error {
	code := "provider_unavailable"
	var provider *ProviderError
	if errors.As(original, &provider) {
		code = provider.Code
	}
	_, err := s.db.Pool.Exec(ctx, `UPDATE bank_activity_jobs SET failures=failures+1,error_code=$1,
 lease_owner=NULL,lease_until=NULL WHERE connection_id=$2 AND generation=$3 AND enabled
 AND lease_owner=$4 AND lease_until>now()`, code, job.ID, job.Generation, job.Owner)
	return errors.Join(original, err)
}

func (s *Service) runDueActivity(ctx context.Context) error {
	if err := s.acquire(ctx); err != nil {
		return err
	}
	defer s.release()
	jobCtx, cancel := context.WithTimeout(ctx, activityJobTimeout)
	defer cancel()
	return s.processActivity(jobCtx, "")
}

func ignorableWorkError(err error) bool {
	return err == nil || errors.Is(err, pgx.ErrNoRows) || errors.Is(err, ErrConflict) || errors.Is(err, context.Canceled)
}
