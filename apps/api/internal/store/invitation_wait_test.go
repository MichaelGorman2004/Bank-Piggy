package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"bankpiggy/api/internal/store"
	"github.com/jackc/pgx/v5"
)

func TestInvitationExpirationIsRecheckedAfterWaitingForHouseholdLock(t *testing.T) {
	database := invitationStore(t, "First")
	setInvitation(t, database, "hash")
	tx, err := database.Pool.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Error(err)
		}
	})
	var lockPID int
	if err := tx.QueryRow(t.Context(), "SELECT pg_backend_pid() FROM households WHERE id='household' FOR UPDATE").Scan(&lockPID); err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() {
		result <- database.ClaimInvitation(t.Context(), "household", "hash", "two@example.com", "second-subject", "Second")
	}()
	waitForHouseholdWaiter(t, database, lockPID)
	if _, err := tx.Exec(t.Context(), "UPDATE household_invitations SET expires_at=clock_timestamp()"); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := <-result; !errors.Is(err, store.ErrInvitationUnavailable) {
		t.Fatalf("invitation expiring while the claim waited returned %v", err)
	}
}

func waitForHouseholdWaiter(t *testing.T, database *store.Store, lockPID int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var waiting bool
		query := "SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE $1=ANY(pg_blocking_pids(pid)))"
		if err := database.Pool.QueryRow(t.Context(), query, lockPID).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting {
			return
		}
	}
	t.Fatal("invitation claim did not wait for the held household lock")
}
