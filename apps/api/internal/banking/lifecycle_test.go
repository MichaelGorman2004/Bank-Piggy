package banking

import (
	"errors"
	"strings"
	"testing"
	"time"

	"bankpiggy/api/internal/budget"
	"github.com/jackc/pgx/v5"
)

func TestLinkBindingReplayExpiryAndOAuthResume(t *testing.T) {
	s, p, identity := setupService(t)
	link := createTestLink(t, s, identity)
	resumed, err := s.PendingLink(t.Context(), identity)
	if err != nil || resumed.AttemptID != link.AttemptID {
		t.Fatal("OAuth continuation did not restore original Link")
	}
	foreign := identity
	foreign.SessionHash = "foreign-session"
	request := budget.ExchangeRequest{AttemptID: link.AttemptID, PublicToken: ptr("public-simulated-token")}
	if err := s.CompleteLink(t.Context(), foreign, request); !errors.Is(err, ErrNotFound) {
		t.Fatal("another session consumed Link")
	}
	completeTestLink(t, s, identity, link)
	if err := s.CompleteLink(t.Context(), identity, request); !errors.Is(err, ErrNotFound) {
		t.Fatal("attempt exchanged twice")
	}
	var cipher string
	if err := s.db.Pool.QueryRow(t.Context(), `SELECT token_cipher FROM bank_connections`).Scan(&cipher); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(cipher, "access-simulated-secret") || p.exchanges != 1 {
		t.Fatal("access token leaked or duplicate exchange occurred")
	}
	next := createTestLink(t, s, identity)
	execSQL(t, s, `UPDATE bank_link_attempts SET expires_at=now()-interval '1 second' WHERE id=$1`, next.AttemptID)
	if err := s.CompleteLink(t.Context(), identity, budget.ExchangeRequest{AttemptID: next.AttemptID, PublicToken: ptr("public-simulated-token")}); !errors.Is(err, ErrNotFound) {
		t.Fatal("expired attempt consumed")
	}
}

func TestSnapshotRefreshPreservesPlanMappingAndStableIDs(t *testing.T) {
	s, p, identity, connection := connectedService(t)
	plan, err := s.db.Read(t.Context(), identity.HouseholdID)
	if err != nil {
		t.Fatal(err)
	}
	account := connection.Accounts[0]
	mapping := budget.AccountMappingRequest{Revision: account.MappingRevision, Enabled: true, Ownership: "shared", Classification: "high_yield_savings", ManualMemberID: ptr("member-1"), ManualAccountID: ptr(plan.Members[0].Accounts[0].ID)}
	if err := s.MapAccount(t.Context(), identity, connection.ID, account.ID, mapping); err != nil {
		t.Fatal(err)
	}
	plan.Members[0].MonthlyTakeHomeCents++
	saved, err := s.db.Update(t.Context(), identity.HouseholdID, plan)
	if err != nil {
		t.Fatal(err)
	}
	p.snapshot.Accounts[0].Account.CurrentBalanceCents = ptr(int64(20003))
	execSQL(t, s, `UPDATE bank_connections SET next_sync_at=now() WHERE id=$1`, connection.ID)
	if err := s.processOne(t.Context(), connection.ID); err != nil {
		t.Fatal(err)
	}
	list, err := s.List(t.Context(), identity)
	if err != nil {
		t.Fatal(err)
	}
	assertRefreshedMapping(t, list.Connections[0].Accounts[0], account.ID)
	actual, err := s.db.Read(t.Context(), identity.HouseholdID)
	if err != nil {
		t.Fatal(err)
	}
	if actual.Revision != saved.Revision || actual.Members[0].MonthlyTakeHomeCents != saved.Members[0].MonthlyTakeHomeCents {
		t.Fatal("sync changed a newer plan edit")
	}

	if err := s.MapAccount(t.Context(), identity, connection.ID, account.ID, mapping); !errors.Is(err, ErrConflict) {
		t.Fatal("stale mapping overwrote newer state")
	}
}

func TestWorkerLeaseRecoveryAndBoundedFailure(t *testing.T) {
	s, p, identity, connection := connectedService(t)
	execSQL(t, s, `UPDATE bank_connections SET next_sync_at=now(),lease_owner='dead-process',lease_until=now()+interval '1 minute' WHERE id=$1`, connection.ID)
	if _, err := s.claimWork(t.Context(), connection.ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("live lease overlapped")
	}
	execSQL(t, s, `UPDATE bank_connections SET lease_until=now()-interval '1 second' WHERE id=$1`, connection.ID)
	job, err := s.claimWork(t.Context(), connection.ID)
	if err != nil {
		t.Fatal("expired lease did not recover", err)
	}
	old := job
	old.Owner = "dead-process"
	if err := s.saveSnapshot(t.Context(), old, p.snapshot); !errors.Is(err, ErrConflict) {
		t.Fatal("stale worker published data")
	}
	if err := s.failJob(t.Context(), job, &ProviderError{Code: "reconnect_required"}); err != nil {
		t.Fatal(err)
	}
	list, err := s.List(t.Context(), identity)
	if err != nil || list.Connections[0].State != "reconnect" {
		t.Fatal("login failure did not require reconnect")
	}
	execSQL(t, s, `UPDATE bank_connections SET state='delayed',failures=$1,next_sync_at=$2 WHERE id=$3`, MaxFailures, time.Now().Add(-time.Minute), connection.ID)
	if _, err := s.claimWork(t.Context(), connection.ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("retry limit was ignored")
	}
}

func assertRefreshedMapping(t *testing.T, refreshed budget.ConnectedAccount, id string) {
	t.Helper()
	if refreshed.ID != id || refreshed.Mapping.Classification != "high_yield_savings" {
		t.Fatal("refresh replaced classification or stable ID")
	}
	if refreshed.MappingRevision != 2 || *refreshed.CurrentBalanceCents != 20003 {
		t.Fatal("refresh changed mapping revision or lost current balance")
	}
}
