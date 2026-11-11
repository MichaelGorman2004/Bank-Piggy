package budget

import "testing"

func purchaseGoal() Goal {
	return Goal{
		ID: "purchase", Name: "Laptop", MonthlyAmountCents: 401, OwnerID: "shared",
		Purchase: &PurchaseTarget{TargetAmountCents: 10000, TargetMonth: "2027-01"},
	}
}

func TestPurchaseProjectionUsesInclusiveMonthsAndWholeCents(t *testing.T) {
	tests := []struct {
		name, month, status               string
		target, saved, periods, suggested int64
	}{
		{"three periods", "2026-11", "saving", 10000, 0, 3, 3334},
		{"deadline month", "2027-01", "saving", 10000, 1000, 1, 9000},
		{"past deadline", "2027-02", "overdue", 10000, 1000, 0, 0},
		{"funded past deadline", "2027-02", "funded", 10000, 11000, 0, 0},
		{"zero target", "2026-11", "funded", 0, 0, 3, 0},
		{"maximum cents", "2026-11", "saving", MaxMoneyCents, 0, 3, 33333333334},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan := scheduledPlan()
			goal := purchaseGoal()
			goal.Purchase.TargetAmountCents, goal.Purchase.SavedAmountCents = test.target, test.saved
			plan.Goals = []Goal{goal}
			result := Response(plan, testMonth(t, test.month)).PurchaseProjections[0]
			if result.GoalID != goal.ID || result.TargetAmountCents != test.target || result.RemainingAmountCents != max(0, test.target-test.saved) {
				t.Fatal("purchase target or saved progress was counted incorrectly")
			}
			if result.Status != test.status || result.MonthsRemaining != test.periods || result.SuggestedMonthlyCents != test.suggested {
				t.Fatalf("incorrect purchase deadline projection: %+v", result)
			}
		})
	}
}

func TestPurchaseProgressDoesNotChangeAllocationsOrAccountBalances(t *testing.T) {
	plan := scheduledPlan()
	goal := purchaseGoal()
	goal.Purchase.SavedAmountCents = 1000
	plan.Goals = []Goal{goal}
	result := Response(plan, testMonth(t, "2026-11"))
	if result.Summary.Household.MonthlyGoalsCents != 401 || result.YearlySummary.Household.MonthlyGoalsCents != 4812 {
		t.Fatal("purchase target replaced or duplicated the monthly allocation")
	}
	if result.Summary.Household.SavingsCents != 12345 || result.YearlySummary.Household.SavingsCents != 12345 {
		t.Fatal("purchase progress changed the account balance")
	}
	share := result.YearlySummary.GoalShares[0]
	if share.FirstMemberCents != 2412 || share.SecondMemberCents != 2400 || result.Summary.Household.MonthlyRemainingCents != 9498 {
		t.Fatal("purchase contribution lost cent reconciliation or was deducted twice")
	}
	if len(result.TripProjections) != 0 || len(result.PurchaseProjections) != 1 {
		t.Fatal("purchase was presented as a trip or omitted")
	}
}

func TestPurchaseValidationRejectsInvalidInputsAndConflictingTargets(t *testing.T) {
	changes := []struct {
		name   string
		change func(*Goal)
	}{
		{"negative target", func(goal *Goal) { goal.Purchase.TargetAmountCents = -1 }},
		{"target limit", func(goal *Goal) { goal.Purchase.TargetAmountCents = MaxMoneyCents + 1 }},
		{"negative savings", func(goal *Goal) { goal.Purchase.SavedAmountCents = -1 }},
		{"savings limit", func(goal *Goal) { goal.Purchase.SavedAmountCents = MaxMoneyCents + 1 }},
		{"invalid deadline", func(goal *Goal) { goal.Purchase.TargetMonth = "2027-1" }},
		{"zero year", func(goal *Goal) { goal.Purchase.TargetMonth = "0000-01" }},
		{"conflicting targets", func(goal *Goal) { goal.Trip = tripGoal().Trip }},
	}
	for _, test := range changes {
		t.Run(test.name, func(t *testing.T) {
			plan := scheduledPlan()
			plan.Goals = []Goal{purchaseGoal()}
			test.change(&plan.Goals[0])
			if Validate(plan) == nil {
				t.Fatal("invalid purchase target accepted")
			}
		})
	}
}

func TestPurchaseValidationAllowsBoundaryValuesAndZeroAllocations(t *testing.T) {
	for _, month := range []string{"0001-01", "9999-12"} {
		plan := scheduledPlan()
		goal := purchaseGoal()
		goal.MonthlyAmountCents = 0
		goal.Purchase = &PurchaseTarget{TargetAmountCents: 0, SavedAmountCents: MaxMoneyCents, TargetMonth: month}
		plan.Goals = []Goal{goal, tripGoal()}
		if err := Validate(plan); err != nil {
			t.Fatal(err)
		}
		result := Response(plan, testMonth(t, "2026-11"))
		if len(result.TripProjections) != 1 || len(result.PurchaseProjections) != 1 {
			t.Fatal("mixed targets did not retain their separate projections")
		}
	}
}
