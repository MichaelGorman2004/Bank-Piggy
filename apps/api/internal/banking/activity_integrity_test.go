package banking

import (
	"testing"
	"time"
)

func TestUnknownActivityAccountKeepsCursorAndEarlierFactsUntilSnapshotArrives(t *testing.T) {
	s, _, identity, connection := activityTestService(t)
	enableActivityForTest(t, s, identity, connection)
	date := time.Now().Format(time.DateOnly)
	known := simulatedActivity("known", date, 100)
	saveActivityBatchForTest(t, s, connection.ID, []ProviderActivityRecord{known}, nil)
	before := activityRecordStateForTest(t, s, "known")
	known.Record.Name = "Staged display correction"
	unknown := simulatedActivity("new-account-record", date, 200)
	unknown.ProviderAccountID = "new-provider-account"
	execSQL(t, s, `UPDATE bank_activity_jobs SET next_sync_at=now() WHERE connection_id=$1`, connection.ID)
	job, err := s.claimActivityWork(t.Context(), connection.ID)
	if err != nil {
		t.Fatal(err)
	}
	start, end := activityWindow()
	batch := activityBatch{Records: []ProviderActivityRecord{known, unknown}, Cursor: "new-checkpoint", Start: start, End: end, Complete: true}
	if err := s.saveActivity(t.Context(), job, batch); err == nil {
		t.Fatal("unknown account advanced the activity cursor")
	}
	assertActivityCheckpoint(t, s, connection.ID, "saved", 1)
	if got := activityRecordStateForTest(t, s, "known"); got != before {
		t.Fatal("rejected batch persisted an earlier record modification")
	}
	execSQL(t, s, `INSERT INTO bank_accounts(id,household_id,connection_id,provider_id,snapshot)
 SELECT 'new-account',household_id,connection_id,'new-provider-account',snapshot FROM bank_accounts WHERE id=$1`, connection.Accounts[0].ID)
	if err := s.saveActivity(t.Context(), job, batch); err != nil {
		t.Fatal("saved account snapshot did not permit complete batch recovery", err)
	}
	assertActivityCheckpoint(t, s, connection.ID, "new-checkpoint", 2)
	if got := activityRecordStateForTest(t, s, "known"); got.revision != before.revision+1 {
		t.Fatal("recovered batch lost the staged modification")
	}
}

func TestKnownUnavailableActivityAccountDoesNotBlockCheckpoint(t *testing.T) {
	s, _, identity, connection := activityTestService(t)
	enableActivityForTest(t, s, identity, connection)
	execSQL(t, s, `UPDATE bank_accounts SET active=false WHERE id=$1`, connection.Accounts[0].ID)
	value := simulatedActivity("unavailable-account-record", time.Now().Format(time.DateOnly), 100)
	saveActivityBatchForTest(t, s, connection.ID, []ProviderActivityRecord{value}, nil)
	assertActivityCheckpoint(t, s, connection.ID, "saved", 0)
}

func TestChangedInvestmentMeaningInvalidatesBothTransferReviews(t *testing.T) {
	for _, change := range []string{"type", "subtype"} {
		t.Run(change, func(t *testing.T) {
			s, _, identity, connection := investmentActivityTestService(t)
			enableActivityForTest(t, s, identity, connection)
			date := time.Now().Format(time.DateOnly)
			out, in := investmentActivity("out", date, 100), investmentActivity("in", date, -100)
			out.Record.ActivitySubtype = "withdrawal"
			saveActivityBatchForTest(t, s, connection.ID, []ProviderActivityRecord{out, in}, nil)
			pairActivityForTest(t, s, "out", "in")
			if change == "type" {
				in.Record.ActivityType = "buy"
			} else {
				in.Record.ActivitySubtype = "dividend"
			}
			saveActivityBatchForTest(t, s, connection.ID, []ProviderActivityRecord{out, in}, nil)
			for _, providerID := range []string{"out", "in"} {
				if got := activityRecordStateForTest(t, s, providerID); got.peer != "" || got.review != "" || got.revision < 2 {
					t.Fatal("changed investment meaning retained a stale transfer review")
				}
			}
		})
	}
}
