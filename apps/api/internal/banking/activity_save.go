package banking

import (
	"context"
	"encoding/json"
	"errors"

	"bankpiggy/api/internal/budget"
	"github.com/jackc/pgx/v5"
)

func (s *Service) saveActivity(ctx context.Context, job activityWork, batch activityBatch) error {
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(tx)
	if err := lockBankHousehold(ctx, tx, job.HouseholdID); err != nil {
		return err
	}
	if err := fenceActivitySave(ctx, tx, job, batch); err != nil {
		return err
	}
	if err := persistActivityBatch(ctx, tx, job, batch); err != nil {
		return err
	}
	if err := pruneActivityHistory(ctx, tx, job.ID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func persistActivityBatch(ctx context.Context, tx pgx.Tx, job activityWork, batch activityBatch) error {
	accounts, err := activityProviderAccounts(ctx, tx, job.ID)
	if err != nil {
		return err
	}
	if err := saveActivityCancellations(ctx, tx, job, accounts, batch.Records); err != nil {
		return err
	}
	for _, record := range batch.Records {
		account, exists := accounts[record.ProviderAccountID]
		if !exists {
			return errors.New("activity references an account missing from the saved snapshot")
		}
		if !activityAccountAvailable(account, job.Kind) {
			continue
		}
		if err := upsertActivityRecord(ctx, tx, job, account.ID, record); err != nil {
			return err
		}
	}
	return cancelActivityRecords(ctx, tx, job, batch)
}

func fenceActivitySave(ctx context.Context, tx pgx.Tx, job activityWork, batch activityBatch) error {
	var generation int64
	err := tx.QueryRow(ctx, `SELECT generation FROM bank_connections WHERE id=$1 AND household_id=$2
 AND state IN ('connected','delayed') AND token_cipher IS NOT NULL FOR UPDATE`, job.ID, job.HouseholdID).Scan(&generation)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && generation != job.ConnectionGeneration) {
		return ErrConflict
	}
	if err != nil {
		return err
	}
	// Investment windows cannot certify a gap; bank cursors carry intervening changes.
	tag, err := tx.Exec(ctx, `UPDATE bank_activity_jobs SET cursor=$1,
 history_start=CASE WHEN $2 OR history_complete THEN GREATEST(
 CASE WHEN $9='investment' AND history_end<$3::date-1 THEN $3::date ELSE COALESCE(history_start,$3::date) END,
 CURRENT_DATE-$8::integer) ELSE history_start END,
 history_end=CASE WHEN $2 OR history_complete THEN $4::date ELSE history_end END,
 history_complete=$2,last_synced_at=now(),failures=0,error_code=NULL,
 lease_owner=NULL,lease_until=NULL
 WHERE connection_id=$5 AND generation=$6 AND enabled AND authorized_at IS NOT NULL
 AND lease_owner=$7 AND lease_until>now()`, batch.Cursor, batch.Complete, batch.Start, batch.End, job.ID, job.Generation, job.Owner, activityRetentionDays, job.Kind)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrConflict
	}
	return nil
}

func activityProviderAccounts(ctx context.Context, tx pgx.Tx, id string) (map[string]budget.ConnectedAccount, error) {
	rows, err := tx.Query(ctx, `SELECT id,provider_id,snapshot,active FROM bank_accounts WHERE connection_id=$1`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]budget.ConnectedAccount{}
	for rows.Next() {
		var id, providerID string
		var data []byte
		var active bool
		if err := rows.Scan(&id, &providerID, &data, &active); err != nil {
			return nil, err
		}
		var account budget.ConnectedAccount
		if err := json.Unmarshal(data, &account); err != nil {
			return nil, err
		}
		account.ID, account.Active = id, active
		result[providerID] = account
	}
	return result, rows.Err()
}

func upsertActivityRecord(ctx context.Context, tx pgx.Tx, job activityWork, accountID string, value ProviderActivityRecord) error {
	id, err := opaqueID()
	if err != nil {
		return err
	}
	data, err := json.Marshal(value.Record)
	if err != nil {
		return err
	}
	var savedID string
	var meaningChanged bool
	err = tx.QueryRow(ctx, `WITH previous AS MATERIALIZED (
 SELECT account_id,kind,posted_on,data,cancelled,cancelled_provider_id,transfer_peer_id,expense_id,goal_id
 FROM bank_activity_records WHERE connection_id=$3 AND provider_id=$5)
 INSERT INTO bank_activity_records(id,household_id,connection_id,account_id,
 provider_id,cancelled_provider_id,kind,posted_on,data,cancelled)
	 VALUES($1,$2,$3,$4,$5,NULLIF($6,''),$7,$8,$9,$10 OR EXISTS(
 SELECT 1 FROM bank_activity_cancellations c WHERE c.connection_id=$3 AND c.provider_id=$5))
 ON CONFLICT(connection_id,provider_id) DO UPDATE SET account_id=EXCLUDED.account_id,kind=EXCLUDED.kind,
 cancelled_provider_id=EXCLUDED.cancelled_provider_id,posted_on=EXCLUDED.posted_on,
 data=EXCLUDED.data,cancelled=EXCLUDED.cancelled,revision=bank_activity_records.revision+1
 WHERE (bank_activity_records.account_id,bank_activity_records.kind,bank_activity_records.data,bank_activity_records.cancelled,
 bank_activity_records.cancelled_provider_id) IS DISTINCT FROM
 (EXCLUDED.account_id,EXCLUDED.kind,EXCLUDED.data,EXCLUDED.cancelled,EXCLUDED.cancelled_provider_id)
 RETURNING id,COALESCE((SELECT (transfer_peer_id IS NOT NULL OR expense_id IS NOT NULL OR goal_id IS NOT NULL) AND
 (account_id,kind,posted_on,data->'amountCents',data->'currency',data->'pending',data->'activityType',data->'activitySubtype',
 data->'primaryCategory',data->'detailedCategory',data->'categoryConfidence',cancelled,cancelled_provider_id) IS DISTINCT FROM
 ($4,$7,$8::date,$9::jsonb->'amountCents',$9::jsonb->'currency',$9::jsonb->'pending',$9::jsonb->'activityType',$9::jsonb->'activitySubtype',
 $9::jsonb->'primaryCategory',$9::jsonb->'detailedCategory',$9::jsonb->'categoryConfidence',$10,NULLIF($6,'')) FROM previous),false)`, id, job.HouseholdID, job.ID, accountID, value.ProviderID, value.CancelledProviderID, value.Record.Kind, string(value.Record.Date), data, value.Record.Cancelled).Scan(&savedID, &meaningChanged)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if meaningChanged {
		return invalidateActivityMeaning(ctx, tx, savedID)
	}
	return nil
}

func invalidateActivityMeaning(ctx context.Context, tx pgx.Tx, id string) error {
	if err := invalidateActivityPair(ctx, tx, id); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `UPDATE bank_activity_records SET review_category=NULL,expense_id=NULL,
 goal_id=NULL,allocation=NULL,revision=revision+1 WHERE id=$1 AND (expense_id IS NOT NULL OR goal_id IS NOT NULL OR allocation IS NOT NULL)`, id)
	return err
}

func invalidateActivityPair(ctx context.Context, tx pgx.Tx, id string) error {
	_, err := tx.Exec(ctx, `UPDATE bank_activity_records SET transfer_peer_id=NULL,review_category=NULL,expense_id=NULL,goal_id=NULL,allocation=NULL,revision=revision+1
 WHERE transfer_peer_id IS NOT NULL AND (id=$1 OR transfer_peer_id=$1)`, id)
	return err
}
