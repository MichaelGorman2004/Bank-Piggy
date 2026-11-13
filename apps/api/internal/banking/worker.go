package banking

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
)

type work struct {
	ID, HouseholdID, Kind, State, Cipher, Owner string
	Generation                                  int64
	Failures                                    int
}

// RunWorker processes one durable leased job at a time and stops with the process.
// Expired leases recover after restart; no in-memory queue owns required work.
func (s *Service) RunWorker(ctx context.Context) {
	if !s.config.Enabled {
		return
	}
	ticker := time.NewTicker(workerPollInterval)
	defer ticker.Stop()
	for {
		err := s.runDueWork(ctx)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) && !errors.Is(err, ErrConflict) && !errors.Is(err, context.Canceled) {
			slog.Error("bank background work failed", "errorType", "bank_sync")
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) claimWork(ctx context.Context, id string) (work, error) {
	owner, err := opaqueID()
	if err != nil {
		return work{}, err
	}
	var job work
	// Reserve the next read before provider work so a crashed worker cannot read
	// again after only the shorter ownership lease expires.
	err = s.db.Pool.QueryRow(ctx, `WITH candidate AS (
 SELECT id FROM bank_connections WHERE ($1='' OR id=$1) AND token_cipher IS NOT NULL AND environment=$5
 AND state IN ('connected','delayed','disconnecting') AND failures<$2 AND next_sync_at<=now()
 AND (lease_until IS NULL OR lease_until<=now()) ORDER BY next_sync_at,id FOR UPDATE SKIP LOCKED LIMIT 1)
 UPDATE bank_connections c SET lease_owner=$3,lease_until=now()+$4::interval,
 next_sync_at=CASE WHEN c.state='disconnecting' THEN c.next_sync_at ELSE now()+$6::interval END FROM candidate
 WHERE c.id=candidate.id RETURNING c.id,c.household_id,c.kind,c.state,c.token_cipher,c.generation,c.failures`, id, MaxFailures, owner, LeaseDuration.String(), s.config.Environment, SyncInterval.String()).Scan(&job.ID, &job.HouseholdID, &job.Kind, &job.State, &job.Cipher, &job.Generation, &job.Failures)
	job.Owner = owner
	return job, err
}

func (s *Service) processOne(ctx context.Context, id string) error {
	job, err := s.claimWork(ctx, id)
	if err != nil {
		return err
	}
	token, err := s.vault.Open(job.Cipher, credentialContext(job.HouseholdID, job.ID, "access"))
	if err != nil {
		return s.failJob(ctx, job, err)
	}
	if job.State == "disconnecting" {
		return s.revokeJob(ctx, job, token)
	}
	snapshot, err := s.provider.Accounts(ctx, token, job.Kind)
	if err != nil {
		return s.failJob(ctx, job, err)
	}
	err = s.saveSnapshot(ctx, job, snapshot)
	if err == nil || errors.Is(err, ErrConflict) {
		return err
	}
	return errors.Join(err, s.failJob(ctx, job, err))
}

func (s *Service) failJob(ctx context.Context, job work, original error) error {
	code, state := "provider_unavailable", "delayed"
	retry := SyncInterval
	var provider *ProviderError
	if errors.As(original, &provider) {
		code = provider.Code
		if code == "reconnect_required" || code == "item_removed" {
			state = "reconnect"
		}
	}
	if job.State == "disconnecting" {
		state = "disconnecting"
		retry = time.Minute * time.Duration(1<<min(job.Failures, 4))
	}
	_, err := s.db.Pool.Exec(ctx, `UPDATE bank_connections SET state=$1,error_code=CASE WHEN $2='provider_unavailable' AND error_code IN ('new_accounts_available','pending_disconnect','pending_expiration') THEN error_code ELSE $2 END,failures=failures+1,next_sync_at=now()+$3::interval,lease_owner=NULL,lease_until=NULL WHERE id=$4 AND generation=$5 AND lease_owner=$6 AND lease_until>now() AND state=$7`, state, code, retry.String(), job.ID, job.Generation, job.Owner, job.State)
	return err
}

func (s *Service) runDueWork(ctx context.Context) error {
	if err := s.acquire(ctx); err != nil {
		return err
	}
	defer s.release()
	jobCtx, cancel := context.WithTimeout(ctx, workerJobTimeout)
	defer cancel()
	if err := s.retireExpiredAttempts(jobCtx); err != nil {
		return err
	}
	return s.processOne(jobCtx, "")
}
