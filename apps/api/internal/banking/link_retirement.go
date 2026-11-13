package banking

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
)

// A new POST is an explicit new Link session. Only OAuth resume reuses a token.
// Production reservations remain consumed because an issued token can create an Item.
func retireIssuedAttempt(ctx context.Context, tx pgx.Tx, identity Identity, environment string) error {
	var id, connection string
	var update bool
	var cipher *string
	err := tx.QueryRow(ctx, `SELECT a.id,a.connection_id,a.is_update,a.token_cipher FROM bank_link_attempts a JOIN bank_connections c ON c.id=a.connection_id WHERE a.household_id=$1 AND a.member_id=$2 AND a.session_hash=$3 AND a.state='pending' AND a.expires_at>now() AND c.environment=$4 FOR UPDATE OF a`, identity.HouseholdID, identity.MemberID, identity.SessionHash, environment).Scan(&id, &connection, &update, &cipher)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if cipher == nil {
		return ErrConflict
	}
	if _, err := tx.Exec(ctx, `UPDATE bank_link_attempts SET state='cancelled' WHERE id=$1`, id); err != nil {
		return err
	}
	if update {
		return nil
	}
	_, err = tx.Exec(ctx, `UPDATE bank_connections SET state='disconnected',error_code='removal_unverified' WHERE id=$1 AND state='pending' AND token_cipher IS NULL`, connection)
	return err
}
