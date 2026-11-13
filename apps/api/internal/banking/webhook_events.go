package banking

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
)

type webhookEvent struct {
	ItemID      string `json:"item_id"`
	Type        string `json:"webhook_type"`
	Code        string `json:"webhook_code"`
	AccountID   string `json:"account_id"`
	Environment string `json:"environment"`
}

func applyItemEvent(ctx context.Context, tx pgx.Tx, event webhookEvent) error {
	if event.Type != "ITEM" {
		return queueBalanceEvent(ctx, tx, event.ItemID, event.Environment)
	}
	switch event.Code {
	case "USER_PERMISSION_REVOKED":
		return reconnectRevokedItem(ctx, tx, event.ItemID, event.Environment)
	case "USER_ACCOUNT_REVOKED":
		return removeRevokedAccount(ctx, tx, event)
	case "NEW_ACCOUNTS_AVAILABLE", "PENDING_DISCONNECT", "PENDING_EXPIRATION":
		_, err := tx.Exec(ctx, `UPDATE bank_connections SET error_code=$2 WHERE item_id=$1 AND state IN ('connected','delayed','reconnect') AND environment=$3`, event.ItemID, strings.ToLower(event.Code), event.Environment)
		return err
	case "LOGIN_REPAIRED", "ERROR":
		_, err := tx.Exec(ctx, `UPDATE bank_connections SET state='delayed',failures=0,next_sync_at=CASE WHEN $3='LOGIN_REPAIRED' THEN now() ELSE GREATEST(next_sync_at,now()) END,generation=generation+1,lease_owner=NULL,lease_until=NULL WHERE item_id=$1 AND state IN ('connected','delayed','reconnect') AND environment=$2`, event.ItemID, event.Environment, event.Code)
		return err
	default:
		return queueBalanceEvent(ctx, tx, event.ItemID, event.Environment)
	}
}

func queueBalanceEvent(ctx context.Context, tx pgx.Tx, item, environment string) error {
	_, err := tx.Exec(ctx, `UPDATE bank_connections SET next_sync_at=LEAST(next_sync_at,now()) WHERE item_id=$1 AND state IN ('connected','delayed') AND next_sync_at<=now() AND (last_synced_at IS NULL OR last_synced_at<=now()-$2::interval) AND environment=$3`, item, minimumRefreshInterval.String(), environment)
	return err
}

func reconnectRevokedItem(ctx context.Context, tx pgx.Tx, item, environment string) error {
	var id string
	err := tx.QueryRow(ctx, `UPDATE bank_connections SET state='reconnect',generation=generation+1,failures=0,lease_owner=NULL,lease_until=NULL,error_code='permission_revoked' WHERE item_id=$1 AND state IN ('connected','delayed','reconnect') AND environment=$2 RETURNING id`, item, environment).Scan(&id)
	if err == pgx.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE bank_accounts SET active=false WHERE connection_id=$1`, id)
	return err
}

func removeRevokedAccount(ctx context.Context, tx pgx.Tx, event webhookEvent) error {
	if event.AccountID == "" {
		return ErrInvalidWebhook
	}
	var id string
	err := tx.QueryRow(ctx, `UPDATE bank_connections SET generation=generation+1,lease_owner=NULL,lease_until=NULL WHERE item_id=$1 AND state IN ('connected','delayed','reconnect') AND environment=$2 RETURNING id`, event.ItemID, event.Environment).Scan(&id)
	if err == pgx.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `DELETE FROM bank_accounts WHERE connection_id=$1 AND provider_id=$2`, id, event.AccountID)
	return err
}
