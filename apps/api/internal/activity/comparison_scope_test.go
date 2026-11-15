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

func TestAggregateBudgetsUseSelectedScopeAndReconcileCents(t *testing.T) {
	plan := personalComparisonPlan()
	for index := range plan.Expenses {
		plan.Expenses[index].MonthlyAmountCents = 101
		plan.Goals[index].MonthlyAmountCents = 101
	}
	first := comparisonProjection(t, plan, nil, "depository", "member-1").Comparison
	second := comparisonProjection(t, plan, nil, "depository", "member-2").Comparison
	household := comparisonProjection(t, plan, nil, "depository", "household").Comparison
	if first.ExpenseBudgetCents != 152 || second.ExpenseBudgetCents != 151 || household.ExpenseBudgetCents != 303 {
		t.Fatal("aggregate expense budget duplicated or lost shared cents")
	}
	if first.InvestmentGoalBudgetCents != 152 || second.InvestmentGoalBudgetCents != 151 || household.InvestmentGoalBudgetCents != 303 {
		t.Fatal("aggregate goal budget duplicated or lost shared cents")
	}
}

func TestPersonalComparisonsKeepFrozenActualsAfterTargetOwnershipChanges(t *testing.T) {
	plan := personalComparisonPlan()
	cost := matchedCost("cost", 101, 34)
	cost.ExpenseID = pointer("foreign")
	result := comparisonProjection(t, plan, []budget.ActivityRecord{cost}, "depository", "member-1")
	if result.Comparison.ExpenseBudgetCents != 0 || result.Totals.SpendingCents != 34 || result.Comparison.ExpenseRemainingCents != -34 {
		t.Fatal("new target owner hid a frozen expense allocation")
	}
	funding := matchedFunding("funding", -101, -51, "contribution")
	funding.GoalID = pointer("foreign")
	result = comparisonProjection(t, plan, []budget.ActivityRecord{funding}, "investment", "member-1")
	if result.Comparison.InvestmentGoalBudgetCents != 0 || result.Totals.NetInvestmentContributionsCents != 51 || result.Comparison.InvestmentRemainingCents != -51 {
		t.Fatal("new target owner hid a frozen investment allocation")
	}
}
