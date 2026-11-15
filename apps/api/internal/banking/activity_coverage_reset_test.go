package banking

import (
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

type savedCoverageState struct {
	complete   bool
	generation int64
	cursor     string
	next       time.Time
}

func TestExpandedSnapshotInvalidatesCoverageAndOldActivityWithoutEarlyRead(t *testing.T) {
	for _, mode := range []string{"new", "reactivated"} {
		t.Run(mode, func(t *testing.T) {
			runExpandedCoverageCase(t, mode)
		})
	}
}

func runExpandedCoverageCase(t *testing.T, mode string) {
	t.Helper()
	s, provider, identity, connection := activityTestService(t)
	enableActivityForTest(t, s, identity, connection)
	saveActivityBatchForTest(t, s, connection.ID, nil, nil)
	if mode == "reactivated" {
		execSQL(t, s, `UPDATE bank_accounts SET active=false WHERE connection_id=$1`, connection.ID)
	}
	execSQL(t, s, `UPDATE bank_activity_jobs SET next_sync_at=now() WHERE connection_id=$1`, connection.ID)
	old, err := s.claimActivityWork(t.Context(), connection.ID)
	if err != nil {
		t.Fatal(err)
	}
	before := activityCoverageStateForTest(t, s, connection.ID)
	if err := saveCoverageSnapshotForTest(t, s, connection.ID, expandedCoverageSnapshot(provider.snapshot, mode)); err != nil {
		t.Fatal(err)
	}
	after := activityCoverageStateForTest(t, s, connection.ID)
	assertExpandedCoverageState(t, before, after)
	start, end := activityWindow()
	if err := s.saveActivity(t.Context(), old, activityBatch{Cursor: "stale", Start: start, End: end, Complete: true}); !errors.Is(err, ErrConflict) {
		t.Fatal("activity claimed before expansion restored obsolete coverage", err)
	}
	if _, err := s.claimActivityWork(t.Context(), connection.ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("coverage invalidation bypassed the reserved provider deadline", err)
	}
	if got := activityCoverageStateForTest(t, s, connection.ID); got != after {
		t.Fatal("rejected late work changed the coverage checkpoint")
	}
}

func assertExpandedCoverageState(t *testing.T, before, after savedCoverageState) {
	t.Helper()
	if after.complete || after.generation <= before.generation || after.cursor != before.cursor || !after.next.Equal(before.next) {
		t.Fatal("expanded account scope retained coverage or changed its read deadline")
	}
}

func TestUnchangedAccountScopePreservesCompletedActivityCoverage(t *testing.T) {
	s, provider, identity, connection := activityTestService(t)
	enableActivityForTest(t, s, identity, connection)
	saveActivityBatchForTest(t, s, connection.ID, nil, nil)
	before := activityCoverageStateForTest(t, s, connection.ID)
	snapshot := expandedCoverageSnapshot(provider.snapshot, "unchanged")
	snapshot.Accounts[0].Account.Name = "Updated display name"
	snapshot.Accounts[0].Account.CurrentBalanceCents = ptr(int64(20002))
	if err := saveCoverageSnapshotForTest(t, s, connection.ID, snapshot); err != nil {
		t.Fatal(err)
	}
	if got := activityCoverageStateForTest(t, s, connection.ID); got != before || !got.complete {
		t.Fatal("ordinary balance or display update reset completed history")
	}
}

func TestRejectedSnapshotExpansionRollsBackActivityCoverageReset(t *testing.T) {
	s, provider, identity, connection := activityTestService(t)
	enableActivityForTest(t, s, identity, connection)
	saveActivityBatchForTest(t, s, connection.ID, nil, nil)
	before := activityCoverageStateForTest(t, s, connection.ID)
	execSQL(t, s, `CREATE FUNCTION reject_coverage_snapshot() RETURNS trigger LANGUAGE plpgsql AS $$
 BEGIN IF NEW.provider_id='added-provider-account' THEN RAISE EXCEPTION 'simulated snapshot failure'; END IF; RETURN NEW; END $$`)
	execSQL(t, s, `CREATE TRIGGER reject_coverage_snapshot BEFORE INSERT OR UPDATE ON bank_accounts FOR EACH ROW EXECUTE FUNCTION reject_coverage_snapshot()`)
	if err := saveCoverageSnapshotForTest(t, s, connection.ID, expandedCoverageSnapshot(provider.snapshot, "new")); err == nil {
		t.Fatal("snapshot rejection reported success")
	}
	if got := activityCoverageStateForTest(t, s, connection.ID); got != before {
		t.Fatal("failed snapshot persisted its activity coverage invalidation")
	}
}

func expandedCoverageSnapshot(original Snapshot, mode string) Snapshot {
	snapshot := Snapshot{InstitutionName: original.InstitutionName, Accounts: append([]ProviderAccount{}, original.Accounts...)}
	if mode == "new" {
		account := snapshot.Accounts[0]
		account.ProviderID = "added-provider-account"
		snapshot.Accounts = append(snapshot.Accounts, account)
	}
	return snapshot
}

func saveCoverageSnapshotForTest(t *testing.T, s *Service, id string, snapshot Snapshot) error {
	t.Helper()
	makeSyncDue(t, s, id)
	job, err := s.claimWork(t.Context(), id)
	if err != nil {
		return err
	}
	return s.saveSnapshot(t.Context(), job, snapshot)
}

func activityCoverageStateForTest(t *testing.T, s *Service, id string) savedCoverageState {
	t.Helper()
	var result savedCoverageState
	err := s.db.Pool.QueryRow(t.Context(), `SELECT history_complete,generation,cursor,next_sync_at
 FROM bank_activity_jobs WHERE connection_id=$1`, id).Scan(&result.complete, &result.generation, &result.cursor, &result.next)
	if err != nil {
		t.Fatal(err)
	}
	return result
}
