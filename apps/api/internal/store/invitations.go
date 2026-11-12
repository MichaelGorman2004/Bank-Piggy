package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// ErrInvitationUnavailable rejects a missing, rotated, revoked, expired, or used link.
var ErrInvitationUnavailable = errors.New("invitation is no longer available")

// ErrPartnerLinked rejects replacing the identity already linked to member-2.
var ErrPartnerLinked = errors.New("the second person already has sign-in access")

// ErrInvitationPlanConflict means adding the partner would invalidate the saved plan.
var ErrInvitationPlanConflict = errors.New("the household plan needs an adjustment before adding another person")

// InvitationStatus exposes link state without the stored token hash.
type InvitationStatus struct {
	Linked    bool
	ExpiresAt *time.Time
}

// InvitationPreview gives a link holder only the inviter's name and expiration.
type InvitationPreview struct {
	HouseholdName string
	MemberName    string
	ExpiresAt     time.Time
}

// InvitationStatus returns the partner's linked state and a live pending expiration.
func (s *Store) InvitationStatus(ctx context.Context, householdID string) (InvitationStatus, error) {
	var status InvitationStatus
	err := s.Pool.QueryRow(ctx, queryInvitationStatus, householdID).Scan(&status.Linked, &status.ExpiresAt)
	return status, err
}

// InvitationPreview validates a link without consuming it or revealing plan data.
func (s *Store) InvitationPreview(ctx context.Context, householdID, hash string) (InvitationPreview, error) {
	var preview InvitationPreview
	err := s.Pool.QueryRow(ctx, queryInvitationPreview, householdID, hash).Scan(&preview.HouseholdName, &preview.MemberName, &preview.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return preview, ErrInvitationUnavailable
	}
	return preview, err
}

// SetInvitation creates or rotates the one pending partner link under the household lock.
func (s *Store) SetInvitation(ctx context.Context, householdID, hash string, expiresAt time.Time) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(tx)
	if _, err := lockPlan(ctx, tx, householdID); err != nil {
		return err
	}
	if err := requireUnlinkedPartner(ctx, tx, householdID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, querySetInvitation, householdID, hash, expiresAt); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// RevokeInvitation invalidates any pending link, including prepared sign-in attempts.
func (s *Store) RevokeInvitation(ctx context.Context, householdID string) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(tx)
	if _, err := lockPlan(ctx, tx, householdID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, queryRevokeInvitation, householdID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func requireUnlinkedPartner(ctx context.Context, tx pgx.Tx, householdID string) error {
	var linked bool
	if err := tx.QueryRow(ctx, queryLinkedPartner, householdID).Scan(&linked); err != nil {
		return err
	}
	if linked {
		return ErrPartnerLinked
	}
	return nil
}
