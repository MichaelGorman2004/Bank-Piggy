// Package store owns PostgreSQL persistence and household-scoped sessions.
package store

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"bankpiggy/api/internal/budget"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var schema string

// ErrConflict indicates that the submitted revision is no longer current.
var ErrConflict = errors.New("plan changed; reload before saving")

// ErrMemberRemoval rejects a save that would discard an existing person.
var ErrMemberRemoval = errors.New("household members cannot be removed")

// ErrUnauthorized indicates that an opaque session is absent or expired.
var ErrUnauthorized = errors.New("session not found")

// ErrIdentityMismatch indicates an invitation is already bound to another identity.
var ErrIdentityMismatch = errors.New("google identity does not match invitation")

// Store wraps one connection pool; every data query is household scoped.
type Store struct{ Pool *pgxpool.Pool }

// Open connects and atomically applies idempotent schema and account migrations.
func Open(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err = migrateSchema(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}
	return &Store{Pool: pool}, nil
}

// Seed creates a household once and never replaces a saved plan.
func (s *Store) Seed(ctx context.Context, id string, h budget.Household) error {
	data, err := json.Marshal(h)
	if err != nil {
		return err
	}
	_, err = s.Pool.Exec(ctx, querySeedHousehold, id, h.Revision, data)
	return err
}

// Read returns only the authorized household's plan.
func (s *Store) Read(ctx context.Context, id string) (budget.Household, error) {
	var h budget.Household
	var data []byte
	err := s.Pool.QueryRow(ctx, queryReadHousehold, id).Scan(&data)
	if err != nil {
		return h, err
	}
	err = json.Unmarshal(data, &h)
	return h, err
}

func lockPlan(ctx context.Context, tx pgx.Tx, householdID string) (budget.Household, error) {
	var h budget.Household
	var data []byte
	if err := tx.QueryRow(ctx, queryLockPlan, householdID).Scan(&data); err != nil {
		return h, err
	}
	err := json.Unmarshal(data, &h)
	return h, err
}

// Update locks one household and rejects stale browser revisions atomically.
func (s *Store) Update(ctx context.Context, id string, h budget.Household) (budget.Household, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return h, err
	}
	defer rollback(tx)
	var revision int64
	var memberCount int
	if err = tx.QueryRow(ctx, queryLockHousehold, id).Scan(&revision, &memberCount); err != nil {
		return h, err
	}
	if revision != h.Revision {
		return h, ErrConflict
	}
	if len(h.Members) < memberCount {
		return h, ErrMemberRemoval
	}
	h.Revision++
	data, err := json.Marshal(h)
	if err != nil {
		return h, err
	}
	if _, err = tx.Exec(ctx, queryUpdateHousehold, id, h.Revision, data); err != nil {
		return h, err
	}
	return h, tx.Commit(ctx)
}

const expiryCleanupBatchSize = 100
const uniqueViolationSQLState = "23505"

// Session stores only a hash of the browser's opaque session token.
type Session struct {
	HouseholdID, MemberID, CSRF, Email string
	ExpiresAt                          time.Time
	InvitationAccepted                 bool
}

// CreateSession persists an authorized server-owned identity.
func (s *Store) CreateSession(ctx context.Context, hash string, session Session) error {
	if _, err := s.Pool.Exec(ctx, queryPruneExpiredSessions, expiryCleanupBatchSize); err != nil {
		return err
	}
	_, err := s.Pool.Exec(ctx, queryCreateSession, hash, session.HouseholdID, session.MemberID, session.CSRF, session.ExpiresAt)
	return err
}

// Session returns an unexpired identity. Client input cannot select a household.
func (s *Store) Session(ctx context.Context, hash string) (Session, error) {
	var session Session
	err := s.Pool.QueryRow(ctx, queryReadSession, hash).Scan(&session.HouseholdID, &session.MemberID, &session.CSRF, &session.ExpiresAt, &session.Email, &session.InvitationAccepted)
	if errors.Is(err, pgx.ErrNoRows) {
		return session, ErrUnauthorized
	}
	return session, err
}

// DeleteSession revokes a session immediately.
func (s *Store) DeleteSession(ctx context.Context, hash string) error {
	_, err := s.Pool.Exec(ctx, queryDeleteSession, hash)
	return err
}

func rollback(transaction pgx.Tx) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := transaction.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		slog.Error("transaction rollback failed")
	}
}
