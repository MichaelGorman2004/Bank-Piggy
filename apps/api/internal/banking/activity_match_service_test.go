package banking

import (
	"encoding/json"
	"reflect"
	"testing"

	"bankpiggy/api/internal/budget"
)

func updateMatchPlan(t *testing.T, s *Service, identity Identity, change func(*budget.Household)) budget.Household {
	t.Helper()
	plan, err := s.db.Read(t.Context(), identity.HouseholdID)
	if err != nil {
		t.Fatal(err)
	}
	change(&plan)
	if err := budget.Validate(plan); err != nil {
		t.Fatal(err)
	}
	plan, err = s.db.Update(t.Context(), identity.HouseholdID, plan)
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

// seedLegacyAllocation represents a saved explicit match from an earlier app version.
func seedLegacyAllocation(t *testing.T, s *Service, id string, goal bool, category budget.ActivityCategory, first int64) {
	t.Helper()
	allocation, err := json.Marshal(budget.ActivityAllocation{Ownership: "shared", FirstMemberCents: first})
	if err != nil {
		t.Fatal(err)
	}
	if goal {
		execSQL(t, s, `UPDATE bank_activity_records SET review_category=$2,goal_id='retirement',allocation=$3,revision=revision+1 WHERE id=$1`, id, category, allocation)
	} else {
		execSQL(t, s, `UPDATE bank_activity_records SET review_category=$2,expense_id='rent',allocation=$3,revision=revision+1 WHERE id=$1`, id, category, allocation)
	}
}

func TestLegacyExpenseAllocationPersistsAcrossPlanEditsAndDeletion(t *testing.T) {
	s, identity, connection := serviceActivityFixture(t, false)
	saveServiceActivity(t, s, identity, connection, "paid", connection.Accounts[0].ID, "2026-11-03", 101, nil)
	seedLegacyAllocation(t, s, "paid", false, "spending", 67)
	updateMatchPlan(t, s, identity, func(h *budget.Household) { h.Members[0].AnnualBaseSalaryCents = 0 })
	if got := serviceActivityMonth(t, s, identity, "member-2"); got.Totals.SpendingCents != 34 {
		t.Fatal("plan edits changed the existing saved expense share")
	}
	updateMatchPlan(t, s, identity, func(h *budget.Household) { h.Expenses = h.Expenses[1:] })
	if got := serviceActivityMonth(t, s, identity, "member-2"); got.Totals.SpendingCents != 34 {
		t.Fatal("deleting a plan target removed actual money or its saved share")
	}
	if present, _ := readSavedMatch(t, s, "paid"); !present {
		t.Fatal("aggregate reads discarded the saved legacy allocation")
	}
}

func TestLegacyRefundReversesSavedShareWithoutChangingPlan(t *testing.T) {
	s, identity, connection := serviceActivityFixture(t, false)
	saveServiceActivity(t, s, identity, connection, "refund", connection.Accounts[0].ID, "2026-11-03", -101, nil)
	seedLegacyAllocation(t, s, "refund", false, "spending", -67)
	before, err := s.db.Read(t.Context(), identity.HouseholdID)
	if err != nil {
		t.Fatal(err)
	}
	first := serviceActivityMonth(t, s, identity, "member-1")
	second := serviceActivityMonth(t, s, identity, "member-2")
	if first.Totals.SpendingCents != -67 || second.Totals.SpendingCents != -34 {
		t.Fatal("aggregate refund did not reverse the exact saved shares")
	}
	after, err := s.db.Read(t.Context(), identity.HouseholdID)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("reading aggregate activity changed the saved plan", err)
	}
}
