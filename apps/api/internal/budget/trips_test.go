package budget

import "testing"

func tripGoal() Goal {
	return Goal{
		ID: "trip", Name: "Vacation", MonthlyAmountCents: 401, OwnerID: "shared",
		Trip: &TripBudget{
			TargetMonth: "2027-01",
			Costs:       []TripCost{{Category: "flights", AmountCents: 6000}, {Category: "lodging", AmountCents: 4000}},
		},
	}
}

func TestTripProjectionUsesInclusiveMonthsAndRetainsOverdueNeeds(t *testing.T) {
	tests := []struct {
		name, month, status       string
		saved, periods, suggested int64
	}{
		{"three periods", "2026-11", "saving", 0, 3, 3334},
		{"deadline month", "2027-01", "saving", 0, 1, 10000},
		{"past deadline", "2027-02", "overdue", 0, 0, 0},
		{"already funded", "2027-02", "funded", 11000, 0, 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			goal := tripGoal()
			goal.Trip.SavedAmountCents = test.saved
			result := tripProjection(goal, test.month)
			if result.TargetAmountCents != 10000 || result.RemainingAmountCents != max(0, 10000-test.saved) {
				t.Fatal("trip target or saved progress was counted incorrectly")
			}
			if result.Status != test.status || result.MonthsRemaining != test.periods || result.SuggestedMonthlyCents != test.suggested {
				t.Fatalf("incorrect deadline projection: %+v", result)
			}
		})
	}
}

func TestTripTargetDoesNotChangeMonthlyAllocationsOrAccountBalances(t *testing.T) {
	plan := scheduledPlan()
	goal := tripGoal()
	goal.Trip.SavedAmountCents = 1000
	plan.Goals = []Goal{goal}
	result := Response(plan, testMonth(t, "2026-11"))
	if result.Summary.Household.MonthlyGoalsCents != 401 || result.YearlySummary.Household.MonthlyGoalsCents != 4812 {
		t.Fatal("trip target or suggested contribution replaced the user's monthly allocation")
	}
	if result.Summary.Household.SavingsCents != 12345 || result.YearlySummary.Household.SavingsCents != 12345 {
		t.Fatal("trip progress was added as another account balance")
	}
	share := result.YearlySummary.GoalShares[0]
	if share.FirstMemberCents != 2412 || share.SecondMemberCents != 2400 {
		t.Fatal("shared trip contributions lost monthly cent reconciliation")
	}
	if result.Summary.Household.MonthlyRemainingCents != 9498 {
		t.Fatal("trip cost target was deducted again from cash flow")
	}
}

func TestTripValidationRejectsInvalidBudgetInputs(t *testing.T) {
	changes := []struct {
		name   string
		change func(*TripBudget)
	}{
		{"missing costs", func(trip *TripBudget) { trip.Costs = nil }},
		{"duplicate category", func(trip *TripBudget) { trip.Costs[1].Category = "flights" }},
		{"unknown category", func(trip *TripBudget) { trip.Costs[0].Category = "unknown" }},
		{"negative cost", func(trip *TripBudget) { trip.Costs[0].AmountCents = -1 }},
		{"total limit", func(trip *TripBudget) { trip.Costs[0].AmountCents = MaxMoneyCents }},
		{"negative savings", func(trip *TripBudget) { trip.SavedAmountCents = -1 }},
		{"invalid deadline", func(trip *TripBudget) { trip.TargetMonth = "2027-1" }},
	}
	for _, test := range changes {
		t.Run(test.name, func(t *testing.T) {
			plan := scheduledPlan()
			plan.Goals = []Goal{tripGoal()}
			test.change(plan.Goals[0].Trip)
			if Validate(plan) == nil {
				t.Fatal("invalid trip budget accepted")
			}
		})
	}
}
