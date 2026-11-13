package store_test

import (
	"reflect"
	"testing"
	"time"

	"bankpiggy/api/internal/store"
	"bankpiggy/api/internal/testbudget"
	"bankpiggy/api/internal/testdb"
)

type scheduleMigrationCase struct {
	id, adjustment string
	advances       bool
}

type scheduleMigrationState struct {
	last     *time.Time
	next     time.Time
	metadata string
}

func TestBankScheduleMigrationAdvancesOnlyOldHealthyDeadlinesOnce(t *testing.T) {
	database := testdb.Open(t)
	if err := database.Seed(t.Context(), "household", testbudget.Sample()); err != nil {
		t.Fatal(err)
	}
	cases := scheduleMigrationCases()
	for _, tc := range cases {
		seedScheduleConnection(t, database, tc)
	}
	seedSchedulePayload(t, database)
	payloadBefore := scheduleFinancialPayload(t, database)
	before := readScheduleConnections(t, database)
	start := scheduleDatabaseTime(t, database)
	migrated := reopenStore(t, database)
	end := scheduleDatabaseTime(t, migrated)
	after := readScheduleConnections(t, migrated)
	for _, tc := range cases {
		assertScheduleMigration(t, tc, before[tc.id], after[tc.id], start, end)
	}
	if scheduleFinancialPayload(t, migrated) != payloadBefore {
		t.Fatal("schedule migration changed financial records or plan revisions")
	}
	for range 2 {
		reopened := reopenStore(t, migrated)
		if !reflect.DeepEqual(readScheduleConnections(t, reopened), after) || scheduleFinancialPayload(t, reopened) != payloadBefore {
			t.Fatal("repeated migration changed a schedule or financial record")
		}
	}
}

func scheduleMigrationCases() []scheduleMigrationCase {
	return []scheduleMigrationCase{
		{"past", "", true},
		{"production-recent", "environment='production',last_synced_at=now()-interval '10 minutes',next_sync_at=now()+interval '23 hours 50 minutes'", true},
		{"expired-lease", "lease_until=now()-interval '1 minute'", true},
		{"already-due", "last_synced_at=now()-interval '25 hours',next_sync_at=now()-interval '1 hour'", false},
		{"active-lease", "lease_until=now()+interval '1 minute'", false},
		{"lease-owner", "lease_owner='simulated-worker',lease_until=now()-interval '1 minute'", false},
		{"failed", "failures=1", false},
		{"exhausted-retries", "failures=8", false},
		{"delayed", "state='delayed'", false},
		{"reconnect", "state='reconnect'", false},
		{"disconnecting", "state='disconnecting'", false},
		{"disconnected", "state='disconnected'", false},
		{"pending", "state='pending'", false},
		{"missing-token", "token_cipher=NULL", false},
		{"never-synced", "last_synced_at=NULL", false},
		{"current-cadence", "next_sync_at=last_synced_at+interval '30 minutes'", false},
		{"retry", "next_sync_at=now()+interval '30 minutes'", false},
		{"future", "next_sync_at='9999-12-31 00:00:00+00'", false},
		{"custom-daily", "next_sync_at=last_synced_at+interval '24 hours 1 second'", false},
	}
}

func seedScheduleConnection(t *testing.T, database *store.Store, tc scheduleMigrationCase) {
	t.Helper()
	query := `INSERT INTO bank_connections(id,household_id,member_id,kind,state,token_cipher,last_synced_at,next_sync_at)
 VALUES ($1,'household','member-1','depository','connected','simulated-encrypted-token',now()-interval '2 hours',now()+interval '22 hours')`
	if _, err := database.Pool.Exec(t.Context(), query, tc.id); err != nil {
		t.Fatal(err)
	}
	if tc.adjustment != "" {
		if _, err := database.Pool.Exec(t.Context(), "UPDATE bank_connections SET "+tc.adjustment+" WHERE id=$1", tc.id); err != nil {
			t.Fatal(err)
		}
	}
}

func readScheduleConnections(t *testing.T, database *store.Store) map[string]scheduleMigrationState {
	t.Helper()
	rows, err := database.Pool.Query(t.Context(), "SELECT id,last_synced_at,next_sync_at,(to_jsonb(c)-'next_sync_at')::text FROM bank_connections c")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	result := map[string]scheduleMigrationState{}
	for rows.Next() {
		var id string
		var state scheduleMigrationState
		if err := rows.Scan(&id, &state.last, &state.next, &state.metadata); err != nil {
			t.Fatal(err)
		}
		result[id] = state
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return result
}

func assertScheduleMigration(t *testing.T, tc scheduleMigrationCase, before, after scheduleMigrationState, start, end time.Time) {
	t.Helper()
	if after.metadata != before.metadata {
		t.Fatalf("%s: migration changed connection state beyond the deadline", tc.id)
	}
	if !tc.advances {
		if !after.next.Equal(before.next) {
			t.Fatalf("%s: migration replaced a protected deadline", tc.id)
		}
		return
	}
	earliest := before.last.Add(30 * time.Minute)
	if earliest.After(start) {
		if !after.next.Equal(earliest) {
			t.Fatalf("%s: recent success was not scheduled 30 minutes after synchronization", tc.id)
		}
		return
	}
	if after.next.Before(start) || after.next.After(end) {
		t.Fatalf("%s: overdue success was not made due at migration time", tc.id)
	}
}

func scheduleDatabaseTime(t *testing.T, database *store.Store) time.Time {
	t.Helper()
	var now time.Time
	if err := database.Pool.QueryRow(t.Context(), "SELECT now()").Scan(&now); err != nil {
		t.Fatal(err)
	}
	return now
}

func seedSchedulePayload(t *testing.T, database *store.Store) {
	t.Helper()
	query := `INSERT INTO bank_accounts(id,household_id,connection_id,provider_id,snapshot,mapping,mapping_revision)
 VALUES ('sample-account','household','past','simulated-account','{"currentBalanceCents":12345}',
 '{"enabled":true,"ownership":"member-1","classification":"savings"}',2);
 INSERT INTO bank_item_budget(environment,reservations) VALUES ('production',3);
 INSERT INTO todo_completions(household_id,member_id,task_id,completed,task_signature)
 VALUES ('household','member-1','sample-task',true,'sample-signature')`
	if _, err := database.Pool.Exec(t.Context(), query); err != nil {
		t.Fatal(err)
	}
}

func scheduleFinancialPayload(t *testing.T, database *store.Store) string {
	t.Helper()
	query := `SELECT jsonb_build_object(
 'households',(SELECT jsonb_agg(to_jsonb(h) ORDER BY id) FROM households h),
 'accounts',(SELECT jsonb_agg(to_jsonb(a) ORDER BY id) FROM bank_accounts a),
 'budget',(SELECT jsonb_agg(to_jsonb(b) ORDER BY environment) FROM bank_item_budget b),
 'todos',(SELECT jsonb_agg(to_jsonb(c) ORDER BY task_id) FROM todo_completions c))::text`
	var payload string
	if err := database.Pool.QueryRow(t.Context(), query).Scan(&payload); err != nil {
		t.Fatal(err)
	}
	return payload
}
