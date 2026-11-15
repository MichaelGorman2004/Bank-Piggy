package banking

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// invalidateExpandedActivityCoverage prevents an older complete Item snapshot from
// certifying a newly shared or reactivated account. The caller holds the household lock.
func invalidateExpandedActivityCoverage(ctx context.Context, tx pgx.Tx, connection string, accounts []ProviderAccount) error {
	ids := make([]string, 0, len(accounts))
	for _, account := range accounts {
		ids = append(ids, account.ProviderID)
	}
	var expanded bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM unnest($2::text[]) incoming(id)
 WHERE NOT EXISTS(SELECT 1 FROM bank_accounts a WHERE a.connection_id=$1 AND a.provider_id=incoming.id AND a.active))`, connection, ids).Scan(&expanded)
	if err != nil || !expanded {
		return err
	}
	return resetActivityCoverage(ctx, tx, connection)
}

// resetActivityCoverage invalidates in-flight certification while preserving the cursor
// and the thirty-minute read deadline. Provider history must become complete again.
func resetActivityCoverage(ctx context.Context, tx pgx.Tx, connection string) error {
	_, err := tx.Exec(ctx, `UPDATE bank_activity_jobs SET history_complete=false,history_start=NULL,history_end=NULL,
 generation=generation+1,lease_owner=NULL,lease_until=NULL WHERE connection_id=$1`, connection)
	return err
}
