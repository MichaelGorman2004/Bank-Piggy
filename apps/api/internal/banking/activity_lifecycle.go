package banking

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func stopActivity(ctx context.Context, tx pgx.Tx, id string) error {
	_, err := tx.Exec(ctx, `UPDATE bank_activity_jobs SET enabled=false,revision=revision+1,
 generation=generation+1,lease_owner=NULL,lease_until=NULL WHERE connection_id=$1`, id)
	return err
}

func purgeActivity(ctx context.Context, tx pgx.Tx, id string) error {
	if err := purgeActivityRecords(ctx, tx, id); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `DELETE FROM bank_activity_jobs WHERE connection_id=$1`, id)
	return err
}

func purgeActivityRecords(ctx context.Context, tx pgx.Tx, id string) error {
	if err := invalidateConnectionActivityPairs(ctx, tx, id); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bank_activity_cancellations WHERE connection_id=$1`, id); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `DELETE FROM bank_activity_records WHERE connection_id=$1`, id)
	return err
}

func revokeActivityConsent(ctx context.Context, tx pgx.Tx, id string) error {
	if err := purgeActivityRecords(ctx, tx, id); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `UPDATE bank_activity_jobs SET authorized_at=NULL,cursor='',history_start=NULL,
 history_end=NULL,history_complete=false,generation=generation+1,revision=revision+1,
 lease_owner=NULL,lease_until=NULL,error_code='permission_revoked' WHERE connection_id=$1`, id)
	return err
}

func invalidateConnectionActivityPairs(ctx context.Context, tx pgx.Tx, id string) error {
	_, err := tx.Exec(ctx, `UPDATE bank_activity_records SET transfer_peer_id=NULL,review_category=NULL,expense_id=NULL,goal_id=NULL,allocation=NULL,revision=revision+1
 WHERE transfer_peer_id IN (SELECT id FROM bank_activity_records WHERE connection_id=$1)`, id)
	return err
}

func revokeAccountActivity(ctx context.Context, tx pgx.Tx, connection, providerID string) error {
	_, err := tx.Exec(ctx, `UPDATE bank_activity_records SET transfer_peer_id=NULL,review_category=NULL,expense_id=NULL,goal_id=NULL,allocation=NULL,revision=revision+1
 WHERE transfer_peer_id IN (SELECT r.id FROM bank_activity_records r JOIN bank_accounts a ON a.id=r.account_id
 WHERE a.connection_id=$1 AND a.provider_id=$2)`, connection, providerID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE bank_activity_jobs SET generation=generation+1,lease_owner=NULL,lease_until=NULL WHERE connection_id=$1`, connection)
	return err
}
