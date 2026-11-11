package budget

import (
	"reflect"
	"testing"
)

func savingsGoal() Goal {
	return Goal{ID: "emergency", Name: "Emergency fund", MonthlyAmountCents: 101, OwnerID: "shared",
		SavingsTarget: &SavingsTarget{TargetAmountCents: 10000, SavedAmountCents: 1234}}
}

func TestOrdinarySavingsProgressDoesNotChangeAllocationsOrAccountBalances(t *testing.T) {
	plan := scheduledPlan()
	goal := savingsGoal()
	goal.SavingsTarget = nil
	plan.Goals = []Goal{goal}
	before := Response(plan, testMonth(t, "2026-11"))
	plan.Goals[0].SavingsTarget = savingsGoal().SavingsTarget
	after := Response(plan, testMonth(t, "2026-11"))
	if !reflect.DeepEqual(before.Summary, after.Summary) || !reflect.DeepEqual(before.YearlySummary, after.YearlySummary) {
		t.Fatal("manual ordinary progress changed allocations, shares, or account balances")
	}
	if len(after.TripProjections) != 0 || len(after.PurchaseProjections) != 0 {
		t.Fatal("ordinary progress created a deadline projection")
	}
}

func TestOrdinarySavingsTargetRejectsInvalidOrConflictingProgress(t *testing.T) {
	tests := []struct {
		name   string
		change func(*Goal)
	}{
		{"zero target", func(g *Goal) { g.SavingsTarget.TargetAmountCents = 0 }},
		{"negative target", func(g *Goal) { g.SavingsTarget.TargetAmountCents = -1 }},
		{"target limit", func(g *Goal) { g.SavingsTarget.TargetAmountCents = MaxMoneyCents + 1 }},
		{"negative saved", func(g *Goal) { g.SavingsTarget.SavedAmountCents = -1 }},
		{"saved limit", func(g *Goal) { g.SavingsTarget.SavedAmountCents = MaxMoneyCents + 1 }},
		{"trip conflict", func(g *Goal) { g.Trip = tripGoal().Trip }},
		{"purchase conflict", func(g *Goal) { g.Purchase = purchaseGoal().Purchase }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan := scheduledPlan()
			plan.Goals = []Goal{savingsGoal()}
			test.change(&plan.Goals[0])
			if Validate(plan) == nil {
				t.Fatal("invalid ordinary savings progress accepted")
			}
		})
	}
}

func TestOrdinarySavingsTargetAllowsWholeCentBoundariesAndOverfunding(t *testing.T) {
	for _, target := range []SavingsTarget{{TargetAmountCents: 1}, {TargetAmountCents: 1, SavedAmountCents: MaxMoneyCents}, {TargetAmountCents: MaxMoneyCents, SavedAmountCents: MaxMoneyCents}} {
		plan := scheduledPlan()
		goal := savingsGoal()
		goal.MonthlyAmountCents = 0
		goal.SavingsTarget = &target
		plan.Goals = []Goal{goal}
		if err := Validate(plan); err != nil {
			t.Fatal("valid whole-cent ordinary progress rejected", err)
		}
	}
}
