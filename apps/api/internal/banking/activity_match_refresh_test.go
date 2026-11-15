package banking

import (
	"testing"
	"time"

	"bankpiggy/api/internal/budget"
)

func matchedProviderCost(t *testing.T) (*Service, Identity, budget.Connection, ProviderActivityRecord, budget.ActivityRecord) {
	t.Helper()
	s, identity, connection := serviceActivityFixture(t, false)
	value := simulatedActivity("matched-cost", time.Now().Format(time.DateOnly), 101)
	value.Record.PrimaryCategory, value.Record.CategoryConfidence = "FOOD_AND_DRINK", "HIGH"
	saveActivityBatchForTest(t, s, connection.ID, []ProviderActivityRecord{value}, nil)
	var id string
	if err := s.db.Pool.QueryRow(t.Context(), `SELECT id FROM bank_activity_records WHERE provider_id='matched-cost'`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	record, err := s.ReviewActivity(t.Context(), identity, id, expenseReview(1, 1))
	if err != nil {
		t.Fatal(err)
	}
	return s, identity, connection, value, record
}

func readSavedMatch(t *testing.T, s *Service, id string) (bool, int64) {
	t.Helper()
	var present bool
	var revision int64
	if err := s.db.Pool.QueryRow(t.Context(), `SELECT expense_id IS NOT NULL OR goal_id IS NOT NULL OR allocation IS NOT NULL,revision FROM bank_activity_records WHERE id=$1`, id).Scan(&present, &revision); err != nil {
		t.Fatal(err)
	}
	return present, revision
}

func TestMatchedExpenseSurvivesDisplayOnlyRefresh(t *testing.T) {
	s, identity, connection, value, before := matchedProviderCost(t)
	value.Record.Name, value.Record.Counterparty = "Updated display", "Updated counterparty"
	saveActivityBatchForTest(t, s, connection.ID, []ProviderActivityRecord{value}, nil)
	if present, revision := readSavedMatch(t, s, before.ID); !present || revision <= before.Revision {
		t.Fatal("display update discarded match or did not version changed facts")
	}
	request := expenseReview(before.Revision+1, 1)
	after, err := s.ReviewActivity(t.Context(), identity, before.ID, request)
	if err != nil || after.Allocation == nil || *after.Allocation != *before.Allocation {
		t.Fatal("display refresh changed exact frozen shares", err)
	}
}

func TestMaterialRefreshClearsMatchesAndAllocation(t *testing.T) {
	cases := []struct {
		name   string
		change func(*ProviderActivityRecord)
	}{
		{"amount", func(v *ProviderActivityRecord) { v.Record.AmountCents = ptr(int64(102)) }},
		{"currency", func(v *ProviderActivityRecord) { v.Record.Currency = ptr("EUR") }},
		{"posting date", func(v *ProviderActivityRecord) {
			v.Record.Date = budget.CalendarDate(time.Now().AddDate(0, 0, -1).Format(time.DateOnly))
		}},
		{"pending", func(v *ProviderActivityRecord) { v.Record.Pending = true }},
		{"cancelled", func(v *ProviderActivityRecord) { v.Record.Cancelled = true }},
		{"meaning", func(v *ProviderActivityRecord) { v.Record.ActivitySubtype = "different meaning" }},
		{"category", func(v *ProviderActivityRecord) { v.Record.PrimaryCategory = "INCOME" }},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			s, _, connection, value, before := matchedProviderCost(t)
			item.change(&value)
			saveActivityBatchForTest(t, s, connection.ID, []ProviderActivityRecord{value}, nil)
			if present, revision := readSavedMatch(t, s, before.ID); present || revision <= before.Revision {
				t.Fatal("changed material facts retained a stale match")
			}
		})
	}
}

func TestRemovedTransactionClearsMatchAllocation(t *testing.T) {
	s, _, connection, _, before := matchedProviderCost(t)
	saveActivityBatchForTest(t, s, connection.ID, nil, []string{"matched-cost"})
	if present, revision := readSavedMatch(t, s, before.ID); present || revision <= before.Revision {
		t.Fatal("removed transaction retained a match allocation")
	}
}
