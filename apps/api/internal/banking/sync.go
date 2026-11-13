package banking

import (
	"context"
	"encoding/json"

	"bankpiggy/api/internal/budget"
	"github.com/jackc/pgx/v5"
)

func (s *Service) saveSnapshot(ctx context.Context, job work, snapshot Snapshot) error {
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(tx)
	tag, err := tx.Exec(ctx, `UPDATE bank_connections SET state='connected',institution_name=$1,last_synced_at=now(),next_sync_at=now()+$2::interval,failures=0,error_code=CASE WHEN error_code IN ('new_accounts_available','pending_disconnect','pending_expiration') THEN error_code ELSE NULL END,lease_owner=NULL,lease_until=NULL WHERE id=$3 AND generation=$4 AND lease_owner=$5 AND lease_until>now() AND state=$6`, snapshot.InstitutionName, SyncInterval.String(), job.ID, job.Generation, job.Owner, job.State)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrConflict
	}
	if _, err := tx.Exec(ctx, `UPDATE bank_accounts SET active=false WHERE connection_id=$1`, job.ID); err != nil {
		return err
	}
	for _, account := range snapshot.Accounts {
		if err := upsertAccount(ctx, tx, job, account); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func upsertAccount(ctx context.Context, tx pgx.Tx, job work, account ProviderAccount) error {
	id, err := opaqueID()
	if err != nil {
		return err
	}
	account.Account.ID = id
	data, err := json.Marshal(account.Account)
	if err != nil {
		return err
	}
	mapping, err := json.Marshal(budget.AccountMapping{Enabled: false, Ownership: "member-1", Classification: "unclassified"})
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO bank_accounts(id,household_id,connection_id,provider_id,snapshot,mapping,environment) SELECT $1,$2,$3,$4,$5,$6,c.environment FROM bank_connections c WHERE c.id=$3
 ON CONFLICT(connection_id,provider_id) DO UPDATE SET snapshot=EXCLUDED.snapshot,active=true`, id, job.HouseholdID, job.ID, account.ProviderID, data, mapping)
	return err
}
