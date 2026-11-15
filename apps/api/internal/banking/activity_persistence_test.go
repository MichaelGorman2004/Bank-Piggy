package banking

import (
	"testing"
	"time"
)

func TestActivityRawRefreshPreservesReviewAndOnlyVersionsChangedFacts(t *testing.T) {
	s, _, identity, connection := activityTestService(t)
	enableActivityForTest(t, s, identity, connection)
	value := simulatedActivity("record", time.Now().Format(time.DateOnly), 29)
	saveActivityBatchForTest(t, s, connection.ID, []ProviderActivityRecord{value}, nil)
	execSQL(t, s, `UPDATE bank_activity_records SET review_category='spending' WHERE provider_id='record'`)
	before := activityRecordStateForTest(t, s, "record")
	saveActivityBatchForTest(t, s, connection.ID, []ProviderActivityRecord{value}, nil)
	if after := activityRecordStateForTest(t, s, "record"); after != before {
		t.Fatal("identical raw refresh changed revision or review")
	}
	value.Record.Name = "Updated provider name"
	saveActivityBatchForTest(t, s, connection.ID, []ProviderActivityRecord{value}, nil)
	after := activityRecordStateForTest(t, s, "record")
	if after.revision != before.revision+1 || after.review != "spending" {
		t.Fatal("raw modification erased a review or failed to version changed facts")
	}
}

func TestRemovedActivityInvalidatesBothTransferReviews(t *testing.T) {
	s, _, identity, connection := activityTestService(t)
	enableActivityForTest(t, s, identity, connection)
	date := time.Now().Format(time.DateOnly)
	saveActivityBatchForTest(t, s, connection.ID, []ProviderActivityRecord{simulatedActivity("out", date, 100), simulatedActivity("in", date, -100)}, nil)
	pairActivityForTest(t, s, "out", "in")
	before := activityRecordStateForTest(t, s, "in")
	saveActivityBatchForTest(t, s, connection.ID, nil, []string{"out"})
	out, in := activityRecordStateForTest(t, s, "out"), activityRecordStateForTest(t, s, "in")
	if !out.cancelled || out.review != "" || out.peer != "" || in.review != "" || in.peer != "" || in.revision <= before.revision {
		t.Fatal("removed transaction left a stale paired review")
	}
}

func TestTransferReviewSurvivesNameRefreshButRejectsChangedMoney(t *testing.T) {
	s, _, identity, connection := activityTestService(t)
	enableActivityForTest(t, s, identity, connection)
	date := time.Now().Format(time.DateOnly)
	out := simulatedActivity("out", date, 100)
	in := simulatedActivity("in", date, -100)
	saveActivityBatchForTest(t, s, connection.ID, []ProviderActivityRecord{out, in}, nil)
	pairActivityForTest(t, s, "out", "in")
	out.Record.Name = "Updated counterparty"
	saveActivityBatchForTest(t, s, connection.ID, []ProviderActivityRecord{out}, nil)
	if saved := activityRecordStateForTest(t, s, "out"); saved.peer == "" || saved.review != "transfer" {
		t.Fatal("display-only provider update erased transfer review")
	}
	out.Record.AmountCents = ptr(int64(101))
	saveActivityBatchForTest(t, s, connection.ID, []ProviderActivityRecord{out}, nil)
	if saved := activityRecordStateForTest(t, s, "out"); saved.peer != "" || saved.review != "" {
		t.Fatal("changed transfer money retained a stale pair")
	}
	if saved := activityRecordStateForTest(t, s, "in"); saved.peer != "" || saved.review != "" {
		t.Fatal("changed transfer money retained the paired review")
	}
}

func TestActivityRetentionPurgesOldPayloadAndInvalidatesSurvivingPeer(t *testing.T) {
	s, _, identity, connection := activityTestService(t)
	enableActivityForTest(t, s, identity, connection)
	now := time.Now()
	old := simulatedActivity("old", now.AddDate(0, 0, -729).Format(time.DateOnly), 100)
	current := simulatedActivity("current", now.Format(time.DateOnly), -100)
	saveActivityBatchForTest(t, s, connection.ID, []ProviderActivityRecord{old, current}, nil)
	pairActivityForTest(t, s, "old", "current")
	execSQL(t, s, `UPDATE bank_activity_records SET posted_on=CURRENT_DATE-731 WHERE provider_id='old'`)
	saveActivityBatchForTest(t, s, connection.ID, nil, nil)
	assertActivityCheckpoint(t, s, connection.ID, "saved", 1)
	if saved := activityRecordStateForTest(t, s, "current"); saved.peer != "" || saved.review != "" {
		t.Fatal("retention left a stale transfer review")
	}
}

func TestDisabledActivityStillRemovesExpiredPayloadWithoutProviderReads(t *testing.T) {
	s, p, identity, connection := activityTestService(t)
	enableActivityForTest(t, s, identity, connection)
	saveActivityBatchForTest(t, s, connection.ID, []ProviderActivityRecord{simulatedActivity("old", time.Now().Format(time.DateOnly), 100)}, nil)
	execSQL(t, s, `UPDATE bank_activity_records SET posted_on=CURRENT_DATE-731`)
	execSQL(t, s, `UPDATE bank_activity_jobs SET enabled=false`)
	before := p.reads
	if err := s.retireExpiredAttempts(t.Context()); err != nil {
		t.Fatal(err)
	}
	assertActivityCheckpoint(t, s, connection.ID, "saved", 0)
	if p.reads != before {
		t.Fatal("retention started a provider read")
	}
}

type savedActivityState struct {
	revision     int64
	review, peer string
	cancelled    bool
}

func activityRecordStateForTest(t *testing.T, s *Service, providerID string) savedActivityState {
	t.Helper()
	var result savedActivityState
	err := s.db.Pool.QueryRow(t.Context(), `SELECT revision,COALESCE(review_category,''),COALESCE(transfer_peer_id,''),cancelled FROM bank_activity_records WHERE provider_id=$1`, providerID).Scan(&result.revision, &result.review, &result.peer, &result.cancelled)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func pairActivityForTest(t *testing.T, s *Service, first, second string) {
	t.Helper()
	execSQL(t, s, `UPDATE bank_activity_records SET review_category='transfer',transfer_peer_id=(SELECT id FROM bank_activity_records WHERE provider_id=$2) WHERE provider_id=$1`, first, second)
	execSQL(t, s, `UPDATE bank_activity_records SET review_category='transfer',transfer_peer_id=(SELECT id FROM bank_activity_records WHERE provider_id=$2) WHERE provider_id=$1`, second, first)
}

func saveActivityBatchForTest(t *testing.T, s *Service, id string, records []ProviderActivityRecord, removed []string) {
	t.Helper()
	execSQL(t, s, `UPDATE bank_activity_jobs SET next_sync_at=now() WHERE connection_id=$1`, id)
	job, err := s.claimActivityWork(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	start, end := activityWindow()
	batch := activityBatch{Records: records, Removed: removed, Cursor: "saved", Start: start, End: end, Complete: true}
	if err := s.saveActivity(t.Context(), job, batch); err != nil {
		t.Fatal(err)
	}
}
