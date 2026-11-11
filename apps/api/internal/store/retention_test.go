package store_test

import (
	"testing"
	"time"

	"bankpiggy/api/internal/budget"
	"bankpiggy/api/internal/store"
	"bankpiggy/api/internal/testdb"
)

func TestNewAuthenticationRecordsPruneExpiredRows(t *testing.T) {
	database := testdb.Open(t)
	if err := database.Seed(t.Context(), "household", budget.Sample()); err != nil {
		t.Fatal(err)
	}
	session := store.Session{HouseholdID: "household", MemberID: "member-1", CSRF: "csrf", ExpiresAt: time.Now().Add(-time.Minute)}
	if err := database.CreateSession(t.Context(), "old", session); err != nil {
		t.Fatal(err)
	}
	session.ExpiresAt = time.Now().Add(time.Hour)
	if err := database.CreateSession(t.Context(), "new", session); err != nil {
		t.Fatal(err)
	}
	assertStoredCount(t, database, "sessions", 1)
	attempt := store.OIDCAttempt{State: "state", Nonce: "nonce", Verifier: "verifier", ExpiresAt: time.Now().Add(-time.Minute)}
	if err := database.CreateOIDCAttempt(t.Context(), "old", attempt); err != nil {
		t.Fatal(err)
	}
	attempt.ExpiresAt = time.Now().Add(time.Minute)
	if err := database.CreateOIDCAttempt(t.Context(), "new", attempt); err != nil {
		t.Fatal(err)
	}
	assertStoredCount(t, database, "oidc_attempts", 1)
	if _, err := database.Session(t.Context(), "new"); err != nil {
		t.Fatal("cleanup removed an active session", err)
	}
	if _, err := database.ConsumeOIDCAttempt(t.Context(), "new"); err != nil {
		t.Fatal("cleanup removed an active attempt", err)
	}
}

func assertStoredCount(t *testing.T, database *store.Store, table string, want int) {
	t.Helper()
	var count int
	// Table names here are fixed test literals, never request data.
	query := map[string]string{"sessions": "SELECT count(*) FROM sessions", "oidc_attempts": "SELECT count(*) FROM oidc_attempts"}[table]
	if err := database.Pool.QueryRow(t.Context(), query).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("%s retained %d rows, want %d", table, count, want)
	}
}
