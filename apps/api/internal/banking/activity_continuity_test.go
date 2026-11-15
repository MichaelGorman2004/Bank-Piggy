package banking

import (
	"testing"
	"time"

	"bankpiggy/api/internal/budget"
)

func TestInvestmentCoverageRequiresContinuousHistoryWindows(t *testing.T) {
	for _, test := range []struct {
		name      string
		endOffset int
		gap       bool
	}{{"overlap", 1, false}, {"adjacent", -1, false}, {"one_missing_day", -2, true}} {
		t.Run(test.name, func(t *testing.T) {
			s, _, identity, connection := investmentActivityTestService(t)
			source := enableActivityForTest(t, s, identity, connection)
			start, end := activityWindow()
			first, err := time.Parse(time.DateOnly, start)
			if err != nil {
				t.Fatal(err)
			}
			oldStart := first.AddDate(0, 0, -120).Format(time.DateOnly)
			oldEnd := first.AddDate(0, 0, test.endOffset).Format(time.DateOnly)
			saveActivityBatchForTest(t, s, connection.ID, []ProviderActivityRecord{investmentActivity("retained", oldStart, -100)}, nil)
			execSQL(t, s, `UPDATE bank_activity_jobs SET history_complete=true,history_start=$2::date,history_end=$3::date WHERE connection_id=$1`, connection.ID, oldStart, oldEnd)
			saveInvestmentWindowAfterOptOutForTest(t, s, identity, source, activityBatch{Start: start, End: end, Complete: true})
			wantStart, wantCoverage := oldStart, "ready"
			if test.gap {
				wantStart, wantCoverage = start, "incomplete"
			}
			saved := activitySourcesForTest(t, s, identity)[0]
			if saved.HistoryStart == nil || *saved.HistoryStart != wantStart || saved.HistoryEnd == nil || *saved.HistoryEnd != end {
				t.Fatal("saved coverage did not preserve only a continuous history range")
			}
			month := first.AddDate(0, 0, -1)
			result, err := s.MonthlyActivity(t.Context(), identity, month, "household", "")
			if err != nil || result.Coverage != wantCoverage {
				t.Fatal("coverage certified a missing day or lost continuous history", err, result.Coverage)
			}
			assertActivityCheckpoint(t, s, connection.ID, "", 1)
		})
	}
}

func saveInvestmentWindowAfterOptOutForTest(t *testing.T, s *Service, identity Identity, source budget.ActivitySource, batch activityBatch) {
	t.Helper()
	accounts := []budget.ActivityAccountSelection{{AccountID: source.Accounts[0].AccountID, Enabled: true, Ownership: identity.MemberID}}
	for _, enabled := range []bool{false, true} {
		var err error
		source, err = s.SetActivitySettings(t.Context(), identity, source.ConnectionID, budget.ActivitySettingsRequest{ExpectedRevision: source.Revision, Enabled: enabled, Accounts: accounts})
		if err != nil {
			t.Fatal(err)
		}
	}
	execSQL(t, s, `UPDATE bank_activity_jobs SET next_sync_at=now() WHERE connection_id=$1`, source.ConnectionID)
	job, err := s.claimActivityWork(t.Context(), source.ConnectionID)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.saveActivity(t.Context(), job, batch); err != nil {
		t.Fatal(err)
	}
}
