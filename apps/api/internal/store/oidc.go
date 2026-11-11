package store

import (
	"context"
	"time"
)

// OIDCAttempt binds one browser to a short-lived OAuth request and PKCE verifier.
type OIDCAttempt struct {
	State     string
	Nonce     string
	Verifier  string
	ExpiresAt time.Time
}

// CreateOIDCAttempt persists secrets on the server, never in the browser cookie.
func (s *Store) CreateOIDCAttempt(ctx context.Context, hash string, attempt OIDCAttempt) error {
	if _, err := s.Pool.Exec(ctx, queryPruneExpiredOIDCAttempts, expiryCleanupBatchSize); err != nil {
		return err
	}
	_, err := s.Pool.Exec(ctx, queryCreateOIDCAttempt, hash, attempt.State, attempt.Nonce, attempt.Verifier, attempt.ExpiresAt)
	return err
}

// ConsumeOIDCAttempt atomically removes an attempt so callbacks cannot be replayed.
func (s *Store) ConsumeOIDCAttempt(ctx context.Context, hash string) (OIDCAttempt, error) {
	var attempt OIDCAttempt
	err := s.Pool.QueryRow(ctx, queryConsumeOIDCAttempt, hash).Scan(&attempt.State, &attempt.Nonce, &attempt.Verifier, &attempt.ExpiresAt)
	return attempt, err
}
