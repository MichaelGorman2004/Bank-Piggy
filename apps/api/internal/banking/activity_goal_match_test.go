package banking

import (
	"reflect"
	"testing"

	"bankpiggy/api/internal/budget"
)

func TestLegacyInvestmentAllocationKeepsManualProgressAfterGoalDeletion(t *testing.T) {
	s, identity, connection := serviceActivityFixture(t, true)
	saveServiceActivity(t, s, identity, connection, "funding", connection.Accounts[0].ID, "2026-11-03", -101, func(r *budget.ActivityRecord) { r.ActivitySubtype = "contribution" })
	seedLegacyAllocation(t, s, "funding", true, "contribution", -51)
	before, err := s.db.Read(t.Context(), identity.HouseholdID)
	if err != nil {
		t.Fatal(err)
	}
	if got := serviceActivityMonth(t, s, identity, "member-2"); got.Totals.NetInvestmentContributionsCents != 50 {
		t.Fatal("saved investment allocation did not reach the selected member")
	}
	after, err := s.db.Read(t.Context(), identity.HouseholdID)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("aggregate reads changed the saved plan or manual progress", err)
	}
	updateMatchPlan(t, s, identity, func(h *budget.Household) { h.Goals = []budget.Goal{} })
	if got := serviceActivityMonth(t, s, identity, "member-2"); got.Totals.NetInvestmentContributionsCents != 50 {
		t.Fatal("goal deletion changed historical net investment contributions")
	}
}
