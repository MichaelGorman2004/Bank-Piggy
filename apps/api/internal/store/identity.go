package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const queryIdentity = `
 SELECT member_id,invitation_accepted FROM household_users
 WHERE household_id=$1 AND email=$2 AND google_subject=$3
`

// Identity is a persisted Google identity; link acceptance survives configuration changes.
type Identity struct {
	MemberID           string
	InvitationAccepted bool
}

// Identity finds an exact verified email and subject pair in one household.
func (s *Store) Identity(ctx context.Context, householdID, email, subject string) (Identity, error) {
	var identity Identity
	err := s.Pool.QueryRow(ctx, queryIdentity, householdID, email, subject).Scan(&identity.MemberID, &identity.InvitationAccepted)
	if errors.Is(err, pgx.ErrNoRows) {
		return identity, ErrUnauthorized
	}
	return identity, err
}

// BindIdentity pins a bootstrap member while holding the same lock as invitation claims.
func (s *Store) BindIdentity(ctx context.Context, householdID, memberID, email, subject string) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(tx)
	if _, err := lockPlan(ctx, tx, householdID); err != nil {
		return err
	}
	if err := bindIdentity(ctx, tx, householdID, memberID, email, subject); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func bindIdentity(ctx context.Context, tx pgx.Tx, householdID, memberID, email, subject string) error {
	if email == "" || subject == "" {
		return ErrIdentityMismatch
	}
	tag, err := tx.Exec(ctx, queryBindIdentity, householdID, memberID, email, subject)
	if err != nil {
		var databaseError *pgconn.PgError
		if errors.As(err, &databaseError) && databaseError.Code == uniqueViolationSQLState {
			return ErrIdentityMismatch
		}
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrIdentityMismatch
	}
	return nil
}
