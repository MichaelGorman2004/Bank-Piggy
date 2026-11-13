package banking

import "context"

func (s *Service) retireExpiredAttempts(ctx context.Context) error {
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(tx)
	if _, err := tx.Exec(ctx, `SELECT h.id FROM households h WHERE EXISTS(SELECT 1 FROM bank_connections c WHERE c.household_id=h.id AND c.environment=$1) ORDER BY h.id FOR UPDATE`, s.config.Environment); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `WITH expired AS (
 UPDATE bank_link_attempts SET state='expired' WHERE id IN(
 SELECT a.id FROM bank_link_attempts a JOIN bank_connections c ON c.id=a.connection_id WHERE a.state='pending' AND a.expires_at<=now() AND c.environment=$2 ORDER BY a.expires_at LIMIT $1)
 RETURNING connection_id,is_update)
 UPDATE bank_connections c SET state='disconnected',error_code='removal_unverified'
 FROM expired WHERE c.id=expired.connection_id AND NOT expired.is_update AND c.state='pending' AND c.token_cipher IS NULL AND c.environment=$2`, expiryCleanupBatch, s.config.Environment)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE bank_connections c SET error_code='exchange_uncertain' WHERE c.state='pending' AND c.token_cipher IS NULL AND EXISTS(SELECT 1 FROM bank_link_attempts a WHERE a.connection_id=c.id AND a.state='exchanging' AND a.created_at<now()-$2::interval) AND c.environment=$1`, s.config.Environment, LeaseDuration.String())
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
