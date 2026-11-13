package banking

import (
	"bankpiggy/api/internal/budget"
	"context"
	"errors"

	"time"
)

func (s *Service) failLink(pending attempt, original error) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return errors.Join(original, err)
	}
	defer rollback(tx)
	if err := lockBankHousehold(ctx, tx, pending.HouseholdID); err != nil {
		return errors.Join(original, err)
	}
	if _, err := tx.Exec(ctx, `UPDATE bank_link_attempts SET state='failed',token_cipher=NULL WHERE id=$1 AND state='pending'`, pending.ID); err != nil {
		return errors.Join(original, err)
	}
	if !pending.Update {
		if _, err := tx.Exec(ctx, `UPDATE bank_connections SET state='disconnected',error_code='link_failed' WHERE id=$1 AND token_cipher IS NULL`, pending.ConnectionID); err != nil {
			return errors.Join(original, err)
		}
	}
	return errors.Join(original, tx.Commit(ctx))
}

func validateLinkRequest(request budget.LinkTokenRequest) error {
	if request.ConnectionID != nil {
		if request.Kind != nil || *request.ConnectionID == "" {
			return ErrInvalid
		}
		return nil
	}
	if request.Kind == nil {
		return ErrInvalid
	}
	if *request.Kind != "depository" && *request.Kind != "investment" {
		return ErrInvalid
	}
	return nil
}
