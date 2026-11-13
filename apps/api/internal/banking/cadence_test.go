package banking

import (
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestSuccessfulSyncWaitsThirtyMinutes(t *testing.T) {
	s, p, _, connection := connectedService(t)
	var scheduled bool
	err := s.db.Pool.QueryRow(t.Context(), `SELECT next_sync_at=last_synced_at+interval '30 minutes' FROM bank_connections WHERE id=$1`, connection.ID).Scan(&scheduled)
	if err != nil || !scheduled {
		t.Fatal("successful sync did not schedule the next read in 30 minutes", err)
	}
	if err := s.processOne(t.Context(), connection.ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("successful sync was repeated before its next deadline", err)
	}
	makeSyncDue(t, s, connection.ID)
	if err := s.processOne(t.Context(), connection.ID); err != nil {
		t.Fatal("scheduled sync did not run at its deadline", err)
	}
	if p.reads != 2 {
		t.Fatal("scheduled sync made an unexpected number of provider reads")
	}
}

func TestManualRefreshHonorsThirtyMinuteBoundary(t *testing.T) {
	for _, elapsed := range []string{"29 minutes", "30 minutes"} {
		t.Run(elapsed, func(t *testing.T) {
			s, _, identity, connection := connectedService(t)
			execSQL(t, s, `UPDATE bank_connections SET last_synced_at=now()-$2::interval,next_sync_at=now() WHERE id=$1`, connection.ID, elapsed)
			err := s.ScheduleRefresh(t.Context(), identity, connection.ID)
			if elapsed == "29 minutes" && !errors.Is(err, ErrConflict) {
				t.Fatal("manual refresh bypassed the 30-minute cooldown", err)
			}
			if elapsed == "30 minutes" && err != nil {
				t.Fatal("manual refresh remained blocked after 30 minutes", err)
			}
		})
	}
}

func TestFailedSyncWaitsThirtyMinutesAtEveryAttempt(t *testing.T) {
	s, p, identity, connection := connectedService(t)
	p.accountsErr = &ProviderError{Code: "provider_unavailable"}
	for attempt := 0; attempt < MaxFailures; attempt++ {
		makeSyncDue(t, s, connection.ID)
		before := bankDatabaseNow(t, s)
		if err := s.processOne(t.Context(), connection.ID); err != nil {
			t.Fatal(err)
		}
		assertThirtyMinuteDeadline(t, s, connection.ID, before, bankDatabaseNow(t, s))
		if err := s.ScheduleRefresh(t.Context(), identity, connection.ID); !errors.Is(err, ErrConflict) {
			t.Fatal("manual refresh bypassed the failed read's retry deadline", err)
		}
		if err := s.processOne(t.Context(), connection.ID); !errors.Is(err, pgx.ErrNoRows) {
			t.Fatal("failed sync retried before 30 minutes", err)
		}
	}
	makeSyncDue(t, s, connection.ID)
	if err := s.processOne(t.Context(), connection.ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("30-minute scheduling bypassed the failure limit", err)
	}
	if p.reads != 1+MaxFailures {
		t.Fatal("failure handling made an unexpected number of provider reads")
	}
}

func TestExpiredLeaseKeepsThirtyMinuteReadDeadline(t *testing.T) {
	s, p, identity, connection := connectedService(t)
	makeSyncDue(t, s, connection.ID)
	before := bankDatabaseNow(t, s)
	if _, err := s.claimWork(t.Context(), connection.ID); err != nil {
		t.Fatal(err)
	}
	assertThirtyMinuteDeadline(t, s, connection.ID, before, bankDatabaseNow(t, s))
	execSQL(t, s, `UPDATE bank_connections SET lease_until=now()-interval '1 second' WHERE id=$1`, connection.ID)
	if err := s.processOne(t.Context(), connection.ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("expired ownership lease bypassed the next read deadline", err)
	}
	if err := s.ScheduleRefresh(t.Context(), identity, connection.ID); !errors.Is(err, ErrConflict) {
		t.Fatal("manual refresh bypassed the interrupted read's deadline", err)
	}
	makeSyncDue(t, s, connection.ID)
	if err := s.processOne(t.Context(), connection.ID); err != nil {
		t.Fatal("interrupted read did not recover at its deadline", err)
	}
	if p.reads != 2 {
		t.Fatal("lease recovery made an unexpected number of provider reads")
	}
}

func TestWebhookCannotAdvanceReadOrRetryDeadline(t *testing.T) {
	for _, event := range []webhookEvent{{Type: "BALANCE", Code: "DEFAULT_UPDATE"}, {Type: "ITEM", Code: "ERROR"}} {
		t.Run(event.Code, func(t *testing.T) {
			s, _, _, connection := connectedService(t)
			execSQL(t, s, `UPDATE bank_connections SET state='delayed',last_synced_at=now()-interval '1 hour' WHERE id=$1`, connection.ID)
			before := bankNextSync(t, s, connection.ID)
			event.ItemID, event.Environment = "item-simulated", "sandbox"
			tx, err := s.db.Pool.Begin(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			defer rollback(tx)
			if err := applyItemEvent(t.Context(), tx, event); err != nil {
				t.Fatal(err)
			}
			if err := tx.Commit(t.Context()); err != nil {
				t.Fatal(err)
			}
			if !bankNextSync(t, s, connection.ID).Equal(before) {
				t.Fatal("automatic webhook advanced a future provider read deadline")
			}
		})
	}
}

func TestDisconnectLeaseRecoveryDoesNotWaitForSyncCadence(t *testing.T) {
	s, _, _, connection := connectedService(t)
	execSQL(t, s, `UPDATE bank_connections SET state='disconnecting',next_sync_at=now() WHERE id=$1`, connection.ID)
	if _, err := s.claimWork(t.Context(), connection.ID); err != nil {
		t.Fatal(err)
	}
	execSQL(t, s, `UPDATE bank_connections SET lease_until=now()-interval '1 second' WHERE id=$1`, connection.ID)
	if _, err := s.claimWork(t.Context(), connection.ID); err != nil {
		t.Fatal("sync cooldown delayed provider removal recovery", err)
	}
}

func TestDisconnectFailureKeepsPromptRevocationRetry(t *testing.T) {
	s, p, identity, connection := connectedService(t)
	p.removeErr = errors.New("simulated revocation failure")
	before := bankDatabaseNow(t, s)
	if err := s.Disconnect(t.Context(), identity, connection.ID); !errors.Is(err, p.removeErr) {
		t.Fatal("provider removal lost the original error", err)
	}
	after := bankDatabaseNow(t, s)
	next := bankNextSync(t, s, connection.ID)
	if next.Before(before.Add(time.Minute)) || next.After(after.Add(time.Minute)) {
		t.Fatal("balance sync cadence delayed the existing one-minute removal retry")
	}
}

func makeSyncDue(t *testing.T, s *Service, id string) {
	t.Helper()
	execSQL(t, s, `UPDATE bank_connections SET last_synced_at=now()-interval '30 minutes',next_sync_at=now() WHERE id=$1`, id)
}

func bankDatabaseNow(t *testing.T, s *Service) time.Time {
	t.Helper()
	var now time.Time
	if err := s.db.Pool.QueryRow(t.Context(), `SELECT now()`).Scan(&now); err != nil {
		t.Fatal(err)
	}
	return now
}

func bankNextSync(t *testing.T, s *Service, id string) time.Time {
	t.Helper()
	var next time.Time
	if err := s.db.Pool.QueryRow(t.Context(), `SELECT next_sync_at FROM bank_connections WHERE id=$1`, id).Scan(&next); err != nil {
		t.Fatal(err)
	}
	return next
}

func assertThirtyMinuteDeadline(t *testing.T, s *Service, id string, before, after time.Time) {
	t.Helper()
	next := bankNextSync(t, s, id)
	if next.Before(before.Add(30*time.Minute)) || next.After(after.Add(30*time.Minute)) {
		t.Fatal("provider read deadline is not 30 minutes after the attempt")
	}
}
