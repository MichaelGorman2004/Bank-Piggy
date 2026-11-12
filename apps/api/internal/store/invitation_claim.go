package store

import (
	"context"
	"encoding/json"
	"fmt"

	"bankpiggy/api/internal/budget"
	"github.com/jackc/pgx/v5"
)

// ClaimInvitation consumes a link, pins member-2, and adds a missing profile atomically.
// Every failure restores the pending link and the original saved plan.
func (s *Store) ClaimInvitation(ctx context.Context, householdID, hash, email, subject, name string) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(tx)
	h, err := lockPlan(ctx, tx, householdID)
	if err != nil {
		return err
	}
	if err := requireUnlinkedPartner(ctx, tx, householdID); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, queryConsumeInvitation, householdID, hash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrInvitationUnavailable
	}
	if err := bindIdentity(ctx, tx, householdID, "member-2", email, subject); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, queryAcceptInvitation, householdID); err != nil {
		return err
	}
	if err := appendInvitedProfile(ctx, tx, householdID, h, name); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func appendInvitedProfile(ctx context.Context, tx pgx.Tx, householdID string, h budget.Household, name string) error {
	if budget.HasMember(h, "member-2") {
		return nil
	}
	h.Members = append(h.Members, budget.Blank("", name).Members[1])
	h.Revision++
	if err := budget.Validate(h); err != nil {
		return fmt.Errorf("%w: %w", ErrInvitationPlanConflict, err)
	}
	data, err := json.Marshal(h)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, queryUpdateHousehold, householdID, h.Revision, data)
	return err
}
