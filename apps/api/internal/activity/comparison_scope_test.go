package activity

import (
	"testing"

	"bankpiggy/api/internal/budget"
)

func personalComparisonPlan() budget.Household {
	plan := budget.Blank("One", "Two")
	plan.Expenses = []budget.Expense{
		{ID: "own", Name: "Own", Category: "Costs", OwnerID: "member-1", Split: budget.Split{Method: "equal"}},
		{ID: "shared", Name: "Shared", Category: "Costs", OwnerID: "shared", Split: budget.Split{Method: "equal"}},
		{ID: "foreign", Name: "Other member", Category: "Costs", OwnerID: "member-2", Split: budget.Split{Method: "equal"}},
	}
	plan.Goals = []budget.Goal{
		{ID: "own", Name: "Own", OwnerID: "member-1", Purpose: "investment"},
		{ID: "shared", Name: "Shared", OwnerID: "shared", Purpose: "investment"},
		{ID: "foreign", Name: "Other member", OwnerID: "member-2", Purpose: "investment"},
	}
	return plan
}

func TestPersonalComparisonsOmitUnattributedForeignRowsButKeepOwnZeroBudgets(t *testing.T) {
	plan := personalComparisonPlan()
	personal := comparisonProjection(t, plan, nil, "depository", "member-1").Comparison
	if len(personal.Expenses) != 2 || len(personal.InvestmentGoals) != 2 {
		t.Fatal("personal comparison included unattributed foreign rows or hid own/shared zero budgets")
	}
	if personal.Expenses[0].ExpenseID != "own" || personal.Expenses[1].ExpenseID != "shared" || personal.InvestmentGoals[1].GoalID != "shared" {
		t.Fatal("personal filtering changed saved order or ownership")
	}
	household := comparisonProjection(t, plan, nil, "depository", "household").Comparison
	if len(household.Expenses) != 3 || len(household.InvestmentGoals) != 3 {
		t.Fatal("personal comparison filtering changed household rows")
	}
}

func TestPersonalComparisonsKeepFrozenActualsAfterTargetOwnershipChanges(t *testing.T) {
	plan := personalComparisonPlan()
	cost := matchedCost("cost", 101, 34)
	cost.ExpenseID = pointer("foreign")
	result := comparisonProjection(t, plan, []budget.ActivityRecord{cost}, "depository", "member-1")
	row := result.Comparison.Expenses[2]
	if row.ExpenseID != "foreign" || row.BudgetCents != 0 || row.SpentCents != 34 || row.RecordCount != 1 {
		t.Fatal("new target owner hid a frozen expense allocation")
	}
	funding := matchedFunding("funding", -101, -51, "contribution")
	funding.GoalID = pointer("foreign")
	result = comparisonProjection(t, plan, []budget.ActivityRecord{funding}, "investment", "member-1")
	goal := result.Comparison.InvestmentGoals[2]
	if goal.GoalID != "foreign" || goal.BudgetCents != 0 || goal.NetContributionsCents != 51 || goal.RecordCount != 1 {
		t.Fatal("new target owner hid a frozen investment allocation")
	}
}
