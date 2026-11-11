package store_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"bankpiggy/api/internal/budget"
	"bankpiggy/api/internal/store"
	"bankpiggy/api/internal/testdb"
)

func TestConcurrentUpdatesPreserveOneRevision(t *testing.T) {
	database := testdb.Open(t)
	plan := budget.Sample()
	if err := database.Seed(t.Context(), "household", plan); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	for index := 0; index < 2; index++ {
		go func() {
			ready.Done()
			<-start
			_, err := database.Update(t.Context(), "household", plan)
			results <- err
		}()
	}
	ready.Wait()
	close(start)
	assertOneSuccessOneConflict(t, results)
	saved, err := database.Read(t.Context(), "household")
	if err != nil {
		t.Fatal(err)
	}
	if saved.Revision != 2 {
		t.Fatal("concurrent update lost revision integrity")
	}
}

func TestIdentityCannotBeReboundAndExpiredSessionsAreRejected(t *testing.T) {
	database := testdb.Open(t)
	if err := database.Seed(t.Context(), "household", budget.Sample()); err != nil {
		t.Fatal(err)
	}
	if err := database.BindIdentity(t.Context(), "household", "member-1", "one@example.com", "original-subject"); err != nil {
		t.Fatal(err)
	}
	if err := database.BindIdentity(t.Context(), "household", "member-1", "one@example.com", "attacker-subject"); !errors.Is(err, store.ErrIdentityMismatch) {
		t.Fatal("Google subject was rebound")
	}
	session := store.Session{HouseholdID: "household", MemberID: "member-1", CSRF: "csrf", ExpiresAt: time.Now().Add(-time.Minute)}
	if err := database.CreateSession(t.Context(), "expired-hash", session); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Session(t.Context(), "expired-hash"); !errors.Is(err, store.ErrUnauthorized) {
		t.Fatal("expired session was accepted")
	}
}

func assertOneSuccessOneConflict(t *testing.T, results <-chan error) {
	t.Helper()
	success, conflict := 0, 0
	for index := 0; index < 2; index++ {
		err := <-results
		switch {
		case err == nil:
			success++
		case errors.Is(err, store.ErrConflict):
			conflict++
		default:
			t.Fatal(err)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("got %d successes and %d conflicts", success, conflict)
	}
}

func TestBindingDistinguishesIdentityConflictsFromDatabaseFailures(t *testing.T) {
	database := testdb.Open(t)
	if err := database.Seed(t.Context(), "household", budget.Sample()); err != nil {
		t.Fatal(err)
	}
	if err := database.BindIdentity(t.Context(), "household", "member-1", "one@example.com", "subject"); err != nil {
		t.Fatal(err)
	}
	err := database.BindIdentity(t.Context(), "household", "member-2", "two@example.com", "subject")
	if !errors.Is(err, store.ErrIdentityMismatch) {
		t.Fatalf("unique identity conflict not classified: %v", err)
	}
	database.Pool.Close()
	err = database.BindIdentity(t.Context(), "household", "member-1", "one@example.com", "subject")
	if err == nil || errors.Is(err, store.ErrIdentityMismatch) {
		t.Fatal("database failure was classified as an identity conflict")
	}
}
