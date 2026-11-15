package banking

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestTransactionPagesCommitOnlyAfterCompleteCursorUpdate(t *testing.T) {
	s, p, identity, connection := activityTestService(t)
	enableActivityForTest(t, s, identity, connection)
	p.transactions = func(_ context.Context, _, cursor string) (TransactionPage, error) {
		if cursor == "" {
			return TransactionPage{Records: []ProviderActivityRecord{simulatedActivity("first", time.Now().Format(time.DateOnly), 29)}, NextCursor: "middle", HasMore: true}, nil
		}
		return TransactionPage{}, &ProviderError{Code: "provider_unavailable"}
	}
	if err := s.processActivity(t.Context(), connection.ID); err == nil {
		t.Fatal("partial update reported success")
	}
	assertActivityCheckpoint(t, s, connection.ID, "", 0)
	if err := s.processActivity(t.Context(), connection.ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("failure bypassed 30-minute deadline", err)
	}
	source := activitySourcesForTest(t, s, identity)[0]
	if source.HistoryStart != nil || source.State != "delayed" {
		t.Fatal("failed page established history coverage")
	}
	var reserved bool
	if err := s.db.Pool.QueryRow(t.Context(), `SELECT next_sync_at>now()+interval '29 minutes' FROM bank_activity_jobs WHERE connection_id=$1`, connection.ID).Scan(&reserved); err != nil || !reserved {
		t.Fatal("activity did not reserve its next read", err)
	}
}

func TestTransactionMutationRestartsFromOriginalPersistedCursor(t *testing.T) {
	s, p, identity, connection := activityTestService(t)
	enableActivityForTest(t, s, identity, connection)
	execSQL(t, s, `UPDATE bank_activity_jobs SET cursor='original' WHERE connection_id=$1`, connection.ID)
	requested := []string{}
	p.transactions = func(_ context.Context, _, cursor string) (TransactionPage, error) {
		requested = append(requested, cursor)
		if len(requested) == 2 {
			return TransactionPage{}, &ProviderError{Code: "activity_pagination_changed"}
		}
		if cursor == "original" {
			return TransactionPage{Records: []ProviderActivityRecord{simulatedActivity("first", time.Now().Format(time.DateOnly), 29)}, NextCursor: "middle", HasMore: true}, nil
		}
		return TransactionPage{Records: []ProviderActivityRecord{simulatedActivity("second", time.Now().Format(time.DateOnly), -100)}, NextCursor: "complete", HistoryComplete: true}, nil
	}
	if err := s.processActivity(t.Context(), connection.ID); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(requested, []string{"original", "middle", "original", "middle"}) {
		t.Fatal("pagination did not restart from original cursor", requested)
	}
	assertActivityCheckpoint(t, s, connection.ID, "complete", 2)
	source := activitySourcesForTest(t, s, identity)[0]
	if source.HistoryStart == nil || source.State != "ready" {
		t.Fatal("historical completion was not recorded")
	}
}

func TestTransactionMutationRetriesAreBounded(t *testing.T) {
	s, p, identity, connection := activityTestService(t)
	enableActivityForTest(t, s, identity, connection)
	calls := 0
	p.transactions = func(context.Context, string, string) (TransactionPage, error) {
		calls++
		return TransactionPage{}, &ProviderError{Code: "activity_pagination_changed"}
	}
	if err := s.processActivity(t.Context(), connection.ID); err == nil || calls != activityPaginationRestarts+1 {
		t.Fatal("pagination restarts were not bounded", calls, err)
	}
	assertActivityCheckpoint(t, s, connection.ID, "", 0)
}

func TestTransactionHistoryCoverageWaitsForHistoricalStatus(t *testing.T) {
	s, p, identity, connection := activityTestService(t)
	enableActivityForTest(t, s, identity, connection)
	p.transactions = func(context.Context, string, string) (TransactionPage, error) {
		return TransactionPage{NextCursor: "initial"}, nil
	}
	if err := s.processActivity(t.Context(), connection.ID); err != nil {
		t.Fatal(err)
	}
	source := activitySourcesForTest(t, s, identity)[0]
	if source.HistoryStart != nil || source.HistoryEnd != nil || source.State != "initializing" {
		t.Fatal("initial update pretended full history was ready")
	}
	execSQL(t, s, `UPDATE bank_activity_jobs SET next_sync_at=now() WHERE connection_id=$1`, connection.ID)
	p.transactions = func(context.Context, string, string) (TransactionPage, error) {
		return TransactionPage{NextCursor: "historical", HistoryComplete: true}, nil
	}
	if err := s.processActivity(t.Context(), connection.ID); err != nil {
		t.Fatal(err)
	}
	if activitySourcesForTest(t, s, identity)[0].State != "ready" {
		t.Fatal("historical status did not establish coverage")
	}
}

func assertActivityCheckpoint(t *testing.T, s *Service, connection, cursor string, count int) {
	t.Helper()
	var saved string
	var records int
	err := s.db.Pool.QueryRow(t.Context(), `SELECT j.cursor,(SELECT count(*) FROM bank_activity_records r WHERE r.connection_id=j.connection_id) FROM bank_activity_jobs j WHERE j.connection_id=$1`, connection).Scan(&saved, &records)
	if err != nil || saved != cursor || records != count {
		t.Fatal("activity checkpoint changed partially", saved, records, err)
	}
}
