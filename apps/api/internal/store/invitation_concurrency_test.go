package store_test

import (
	"errors"
	"sync"
	"testing"

	"bankpiggy/api/internal/store"
)

func TestConcurrentInvitationClaimsHaveOneWinner(t *testing.T) {
	database := invitationStore(t, "First")
	setInvitation(t, database, "hash")
	start := make(chan struct{})
	results := make(chan error, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	for _, email := range []string{"two@example.com", "other@example.com"} {
		go func() {
			ready.Done()
			<-start
			results <- database.ClaimInvitation(t.Context(), "household", "hash", email, email, "Second")
		}()
	}
	ready.Wait()
	close(start)
	assertOneInvitationClaim(t, results)
	saved, err := database.Read(t.Context(), "household")
	if err != nil || len(saved.Members) != 2 || saved.Revision != 2 {
		t.Fatalf("concurrent claim corrupted profiles or revision: %v", err)
	}
}

func assertOneInvitationClaim(t *testing.T, results <-chan error) {
	t.Helper()
	success, rejected := 0, 0
	for index := 0; index < 2; index++ {
		err := <-results
		switch {
		case err == nil:
			success++
		case errors.Is(err, store.ErrPartnerLinked):
			rejected++
		default:
			t.Fatal(err)
		}
	}
	if success != 1 || rejected != 1 {
		t.Fatalf("got %d successful claims and %d rejected claims", success, rejected)
	}
}
