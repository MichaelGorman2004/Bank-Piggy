package banking

import (
	"context"

	"bankpiggy/api/internal/budget"
	"github.com/jackc/pgx/v5"
)

const activityRetentionDays = 730

func saveActivityCancellations(ctx context.Context, tx pgx.Tx, job activityWork, accounts map[string]budget.ConnectedAccount, records []ProviderActivityRecord) error {
	for _, record := range records {
		if record.CancelledProviderID == "" {
			continue
		}
		account, exists := accounts[record.ProviderAccountID]
		if !exists || !activityAccountAvailable(account, job.Kind) {
			continue
		}
		_, err := tx.Exec(ctx, `INSERT INTO bank_activity_cancellations(connection_id,account_id,provider_id)
 VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, job.ID, account.ID, record.CancelledProviderID)
		if err != nil {
			return err
		}
	}
	return nil
}

func pruneActivityHistory(ctx context.Context, tx pgx.Tx, id string) error {
	_, err := tx.Exec(ctx, `UPDATE bank_activity_records SET transfer_peer_id=NULL,review_category=NULL,expense_id=NULL,goal_id=NULL,allocation=NULL,revision=revision+1
 WHERE transfer_peer_id IN(SELECT id FROM bank_activity_records
 WHERE connection_id=$1 AND posted_on<CURRENT_DATE-$2::integer)`, id, activityRetentionDays)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `DELETE FROM bank_activity_records WHERE connection_id=$1 AND posted_on<CURRENT_DATE-$2::integer`, id, activityRetentionDays)
	return err
}

// pruneExpiredActivity runs without a provider read, including disabled sources.
// The caller holds household locks before this changes ledger or review rows.
func pruneExpiredActivity(ctx context.Context, tx pgx.Tx, environment string) error {
	_, err := tx.Exec(ctx, `UPDATE bank_activity_records SET transfer_peer_id=NULL,review_category=NULL,expense_id=NULL,goal_id=NULL,allocation=NULL,revision=revision+1
 WHERE transfer_peer_id IN(SELECT r.id FROM bank_activity_records r JOIN bank_connections c ON c.id=r.connection_id
 WHERE c.environment=$1 AND r.posted_on<CURRENT_DATE-$2::integer)`, environment, activityRetentionDays)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bank_activity_records r USING bank_connections c WHERE c.id=r.connection_id
 AND c.environment=$1 AND r.posted_on<CURRENT_DATE-$2::integer`, environment, activityRetentionDays); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE bank_activity_jobs j SET history_start=GREATEST(history_start,CURRENT_DATE-$2::integer)
 FROM bank_connections c WHERE c.id=j.connection_id AND c.environment=$1
 AND history_start<CURRENT_DATE-$2::integer`, environment, activityRetentionDays)
	return err
}
