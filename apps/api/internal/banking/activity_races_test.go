package banking

import (
	"context"
	"errors"
	"testing"
	"time"

	"bankpiggy/api/internal/budget"
	"github.com/jackc/pgx/v5"
)

func TestDisablingActivityWinsAgainstLateProviderCompletion(t *testing.T) {
	s, p, identity, connection := activityTestService(t)
	source := enableActivityForTest(t, s, identity, connection)
	started, release := make(chan struct{}), make(chan struct{})
	p.transactions = func(context.Context, string, string) (TransactionPage, error) {
		close(started)
		<-release
		return TransactionPage{Records: []ProviderActivityRecord{simulatedActivity("late", time.Now().Format(time.DateOnly), 29)}, NextCursor: "late", HistoryComplete: true}, nil
	}
	finished := make(chan error, 1)
	go func() { finished <- s.processActivity(t.Context(), connection.ID) }()
	<-started
	request := budget.ActivitySettingsRequest{ExpectedRevision: source.Revision, Enabled: false, Accounts: []budget.ActivityAccountSelection{}}
	if _, err := s.SetActivitySettings(t.Context(), identity, connection.ID, request); err != nil {
		t.Fatal(err)
	}
	close(release)
	if err := <-finished; !errors.Is(err, ErrConflict) {
		t.Fatal("late activity save overwrote disable", err)
	}
	assertActivityCheckpoint(t, s, connection.ID, "", 0)
}

func TestActivityLeaseAndConnectionGenerationsRejectLateWork(t *testing.T) {
	for _, change := range []string{"lease", "connection", "account"} {
		t.Run(change, func(t *testing.T) {
			s, _, identity, connection := activityTestService(t)
			enableActivityForTest(t, s, identity, connection)
			old, err := s.claimActivityWork(t.Context(), connection.ID)
			if err != nil {
				t.Fatal(err)
			}
			invalidateActivityWorkForTest(t, s, connection.ID, change)
			start, end := activityWindow()
			batch := activityBatch{Start: start, End: end, Cursor: "late", Complete: true}
			if err := s.saveActivity(t.Context(), old, batch); !errors.Is(err, ErrConflict) {
				t.Fatal("stale worker saved cursor", err)
			}
			assertActivityCheckpoint(t, s, connection.ID, "", 0)
		})
	}
}

func invalidateActivityWorkForTest(t *testing.T, s *Service, id, change string) {
	t.Helper()
	if change == "lease" {
		execSQL(t, s, `UPDATE bank_activity_jobs SET lease_until=now()-interval '1 second',next_sync_at=now() WHERE connection_id=$1`, id)
		if _, err := s.claimActivityWork(t.Context(), id); err != nil {
			t.Fatal(err)
		}
		return
	}
	if change == "account" {
		execSQL(t, s, `UPDATE bank_activity_jobs SET generation=generation+1,lease_owner=NULL,lease_until=NULL WHERE connection_id=$1`, id)
		return
	}
	execSQL(t, s, `UPDATE bank_connections SET generation=generation+1 WHERE id=$1`, id)
}

func TestBalanceFailureDoesNotBlockOrChangeActivitySchedule(t *testing.T) {
	s, _, identity, connection := activityTestService(t)
	enableActivityForTest(t, s, identity, connection)
	execSQL(t, s, `UPDATE bank_connections SET failures=$2,state='delayed',error_code='provider_unavailable',next_sync_at=now()+interval '1 hour' WHERE id=$1`, connection.ID, MaxFailures)
	if err := s.processActivity(t.Context(), connection.ID); err != nil {
		t.Fatal("balance failure blocked activity", err)
	}
	var failures int
	if err := s.db.Pool.QueryRow(t.Context(), `SELECT failures FROM bank_connections WHERE id=$1`, connection.ID).Scan(&failures); err != nil || failures != MaxFailures {
		t.Fatal("activity changed balance retries", err)
	}
	if err := s.processActivity(t.Context(), connection.ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("activity bypassed independent deadline", err)
	}
	s.config.BankActivityApproved = false
	execSQL(t, s, `UPDATE bank_activity_jobs SET next_sync_at=now() WHERE connection_id=$1`, connection.ID)
	if err := s.processActivity(t.Context(), connection.ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("closed operator gate permitted read", err)
	}
}
