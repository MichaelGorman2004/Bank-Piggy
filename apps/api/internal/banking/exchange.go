package banking

import (
	"context"
	"time"

	"bankpiggy/api/internal/budget"
)

// CompleteLink consumes an attempt once before contacting Plaid. The returned access
// credential is saved before snapshot work; retries never exchange the same attempt.
func (s *Service) CompleteLink(ctx context.Context, identity Identity, request budget.ExchangeRequest) error {
	if err := s.acquire(ctx); err != nil {
		return err
	}
	defer s.release()
	pending, err := s.claimAttempt(ctx, identity, request)
	if err != nil {
		return err
	}
	if pending.Update {
		return s.finishUpdate(ctx, pending)
	}
	token, item, exchangeErr := s.provider.Exchange(ctx, *request.PublicToken)
	if token == "" {
		return s.failExchange(pending, exchangeErr)
	}
	encrypted, err := s.vault.Seal(token, credentialContext(identity.HouseholdID, pending.ConnectionID, "access"))
	if err != nil {
		return s.compensate(pending, token, err)
	}
	durable, cancel := context.WithTimeout(context.Background(), ProviderTimeout)
	defer cancel()
	tag, err := s.db.Pool.Exec(durable, `UPDATE bank_connections SET token_cipher=$1,item_id=$2,state='delayed',error_code=NULL,next_sync_at=now() WHERE id=$3 AND state='pending'`, encrypted, item, pending.ConnectionID)
	if err != nil {
		return s.compensate(pending, token, err)
	}
	if tag.RowsAffected() != 1 {
		return s.compensate(pending, token, ErrConflict)
	}
	_, err = s.db.Pool.Exec(durable, `UPDATE bank_link_attempts SET state='completed',token_cipher=NULL,public_token_cipher=NULL WHERE id=$1`, pending.ID)
	return err
}

func (s *Service) claimAttempt(ctx context.Context, identity Identity, request budget.ExchangeRequest) (attempt, error) {
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return attempt{}, err
	}
	defer rollback(tx)
	if err := lockBankHousehold(ctx, tx, identity.HouseholdID); err != nil {
		return attempt{}, err
	}
	var pending attempt
	err = tx.QueryRow(ctx, `SELECT a.id,a.connection_id,a.is_update,c.kind FROM bank_link_attempts a JOIN bank_connections c ON c.id=a.connection_id WHERE a.id=$1 AND a.household_id=$2 AND a.member_id=$3 AND a.session_hash=$4 AND a.state='pending' AND a.expires_at>now() AND c.state NOT IN ('disconnecting','disconnected') AND c.environment=$5 FOR UPDATE OF a,c`, request.AttemptID, identity.HouseholdID, identity.MemberID, identity.SessionHash, s.config.Environment).Scan(&pending.ID, &pending.ConnectionID, &pending.Update, &pending.Kind)
	if err != nil {
		return pending, notFound(err)
	}
	pending.HouseholdID, pending.MemberID = identity.HouseholdID, identity.MemberID
	public, err := s.encryptPublicToken(pending, request.PublicToken)
	if err != nil {
		return pending, err
	}
	_, err = tx.Exec(ctx, `UPDATE bank_link_attempts SET state='exchanging',public_token_cipher=$1 WHERE id=$2`, public, pending.ID)
	if err != nil {
		return pending, err
	}
	return pending, tx.Commit(ctx)
}

func (s *Service) finishUpdate(ctx context.Context, pending attempt) error {
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(tx)
	if err := lockBankHousehold(ctx, tx, pending.HouseholdID); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `UPDATE bank_connections SET state='delayed',error_code=NULL,failures=0,next_sync_at=now(),generation=generation+1,lease_owner=NULL,lease_until=NULL WHERE id=$1 AND household_id=$2 AND member_id=$3 AND state NOT IN ('disconnecting','disconnected')`, pending.ConnectionID, pending.HouseholdID, pending.MemberID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrConflict
	}
	_, err = tx.Exec(ctx, `UPDATE bank_link_attempts SET state='completed',token_cipher=NULL WHERE id=$1`, pending.ID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Service) failExchange(pending attempt, original error) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.db.Pool.Exec(ctx, `UPDATE bank_connections SET error_code='exchange_uncertain' WHERE id=$1 AND state='pending'`, pending.ConnectionID)
	if original != nil {
		return original
	}
	if err != nil {
		return err
	}
	return ErrUnavailable
}

func (s *Service) encryptPublicToken(pending attempt, token *string) (*string, error) {
	if pending.Update {
		return nil, nil
	}
	if token == nil || len(*token) < 10 || len(*token) > 1024 {
		return nil, ErrInvalid
	}
	encrypted, err := s.vault.Seal(*token, credentialContext(pending.HouseholdID, pending.ID, "public"))
	return &encrypted, err
}
