package banking

import (
	"context"
	"errors"
	"log/slog"
)

func (s *Service) compensate(pending attempt, token string, original error) error {
	ctx, cancel := context.WithTimeout(context.Background(), ProviderTimeout)
	defer cancel()
	encrypted, sealErr := s.vault.Seal(token, credentialContext(pending.HouseholdID, pending.ConnectionID, "access"))
	if sealErr == nil {
		_, saveErr := s.db.Pool.Exec(ctx, `UPDATE bank_connections SET token_cipher=$1,state='disconnecting',error_code='cleanup_pending',generation=generation+1,next_sync_at=now() WHERE id=$2`, encrypted, pending.ConnectionID)
		if saveErr == nil {
			return original
		}
	}
	revokeCtx, revokeCancel := context.WithTimeout(context.Background(), ProviderTimeout)
	defer revokeCancel()
	if revokeErr := s.provider.Remove(revokeCtx, token); revokeErr != nil {
		slog.Error("bank exchange recovery requires provider dashboard review")
	}
	return original
}

// Disconnect first removes data from use, then revokes the Item. Its encrypted
// credential remains durable until Plaid confirms removal; a late sync cannot restore it.
func (s *Service) Disconnect(ctx context.Context, identity Identity, id string) error {
	cipher, done, err := s.disconnectIntent(ctx, identity, id)
	if err != nil {
		return err
	}
	if done {
		return nil
	}

	if cipher == nil {
		return s.finalizeEmptyDisconnect(ctx, id)
	}
	if err := s.acquire(ctx); err != nil {
		return err
	}
	defer s.release()
	return s.processOne(ctx, id)
}

func (s *Service) disconnectIntent(ctx context.Context, identity Identity, id string) (*string, bool, error) {
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return nil, false, err
	}
	defer rollback(tx)
	if err := lockBankHousehold(ctx, tx, identity.HouseholdID); err != nil {
		return nil, false, err
	}
	var cipher *string
	var state string
	err = tx.QueryRow(ctx, `SELECT token_cipher,state FROM bank_connections WHERE id=$1 AND household_id=$2 AND member_id=$3 AND environment=$4 FOR UPDATE`, id, identity.HouseholdID, identity.MemberID, s.config.Environment).Scan(&cipher, &state)
	if err != nil {
		return nil, false, notFound(err)
	}
	if state == "disconnected" {
		return nil, true, nil
	}
	if _, err = tx.Exec(ctx, `UPDATE bank_connections SET state='disconnecting',generation=generation+1,failures=0,next_sync_at=now(),lease_owner=NULL,lease_until=NULL WHERE id=$1`, id); err != nil {
		return nil, false, err
	}
	if _, err = tx.Exec(ctx, `UPDATE bank_accounts SET active=false,manual_member_id=NULL,manual_account_id=NULL WHERE connection_id=$1`, id); err != nil {
		return nil, false, err
	}
	if _, err = tx.Exec(ctx, `UPDATE bank_link_attempts SET state='cancelled' WHERE connection_id=$1 AND state='pending'`, id); err != nil {
		return nil, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, false, err
	}
	return cipher, false, nil
}

func (s *Service) finalizeEmptyDisconnect(ctx context.Context, id string) error {
	_, err := s.db.Pool.Exec(ctx, `UPDATE bank_connections SET state='disconnected',error_code='removal_unverified' WHERE id=$1 AND state='disconnecting' AND token_cipher IS NULL`, id)
	return err
}

func (s *Service) revokeJob(ctx context.Context, job work, token string) error {
	err := s.provider.Remove(ctx, token)
	var provider *ProviderError
	alreadyRemoved := errors.As(err, &provider) && provider.Code == "item_removed"
	if err != nil && !alreadyRemoved {
		return errors.Join(err, s.failJob(ctx, job, err))
	}
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(tx)
	tag, err := tx.Exec(ctx, `UPDATE bank_connections SET state='disconnected',token_cipher=NULL,error_code=NULL,lease_owner=NULL,lease_until=NULL WHERE id=$1 AND generation=$2 AND lease_owner=$3 AND state='disconnecting' AND lease_until>now()`, job.ID, job.Generation, job.Owner)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrConflict
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bank_accounts WHERE connection_id=$1`, job.ID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ScheduleRefresh requests bounded durable work without creating overlapping calls.
func (s *Service) ScheduleRefresh(ctx context.Context, identity Identity, id string) error {
	if !s.config.Enabled {
		return ErrUnavailable
	}
	tag, err := s.db.Pool.Exec(ctx, `UPDATE bank_connections SET next_sync_at=LEAST(next_sync_at,now()),failures=0 WHERE id=$1 AND household_id=$2 AND token_cipher IS NOT NULL AND state IN ('connected','delayed') AND (last_synced_at IS NULL OR last_synced_at<now()-$3::interval) AND environment=$4`, id, identity.HouseholdID, minimumRefreshInterval.String(), s.config.Environment)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return refreshRejection(ctx, s, id, identity)
	}
	return nil
}

func refreshRejection(ctx context.Context, s *Service, id string, identity Identity) error {
	var exists bool
	err := s.db.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM bank_connections WHERE id=$1 AND household_id=$2 AND environment=$3)`, id, identity.HouseholdID, s.config.Environment).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	return ErrConflict
}
