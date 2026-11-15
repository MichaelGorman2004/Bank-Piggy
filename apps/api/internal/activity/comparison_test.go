package activity

import (
	"testing"

	"bankpiggy/api/internal/budget"
)

func comparisonProjection(t *testing.T, plan budget.Household, records []budget.ActivityRecord, kind budget.ConnectionKind, scope string) budget.ActivityResponse {
	t.Helper()
	sources := sourcesFor("checking")
	sources[0].Kind = kind
	sources[0].Accounts[0].Ownership = "member-1"
	result, err := Project(plan, records, sources, testMonth, scope, testNow)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func matchedCost(id string, amount, first int64) budget.ActivityRecord {
	record := bankRecord(id, "2026-11-02", amount, "FOOD_AND_DRINK", "")
	record.IsReviewed, record.Category, record.ExpenseID = true, "spending", pointer("shared-cost")
	record.Allocation = &budget.ActivityAllocation{Ownership: "shared", FirstMemberCents: first}
	return record
}

func comparisonPlan() budget.Household {
	plan := budget.Blank("One", "Two")
	plan.Members[0].AnnualBaseSalaryCents, plan.Members[1].AnnualBaseSalaryCents = 10000, 20000
	plan.Expenses = []budget.Expense{{ID: "shared-cost", Name: "Shared cost", Category: "Costs", OwnerID: "shared", MonthlyAmountCents: 10000, Split: budget.Split{Method: "salary"}}}
	return plan
}

func TestMatchedCostsUseFrozenAllocationAcrossPayerScopes(t *testing.T) {
	plan := comparisonPlan()
	records := []budget.ActivityRecord{matchedCost("paid", 101, 34), matchedCost("refund", -2, -1), bankRecord("unmatched", "2026-11-03", 101, "FOOD_AND_DRINK", "")}
	household := comparisonProjection(t, plan, records, "depository", "household")
	first := comparisonProjection(t, plan, records, "depository", "member-1")
	second := comparisonProjection(t, plan, records, "depository", "member-2")
	if first.Totals.SpendingCents != 134 || second.Totals.SpendingCents != 66 || household.Totals.SpendingCents != 200 {
		t.Fatal("payer ownership replaced recorded cost shares")
	}
	if first.Comparison.ExpenseBudgetCents != 3333 || first.Comparison.ExpenseRemainingCents != 3199 || !second.Comparison.BankAvailable {
		t.Fatal("aggregate comparison lost unmatched spending or cross-payer availability")
	}
	if first.Totals.SpendingCents+second.Totals.SpendingCents != household.Totals.SpendingCents {
		t.Fatal("personal spending shares do not reconcile to the household")
	}
}

func TestChangedOrDeletedExpenseKeepsRecordedCostAndAllocation(t *testing.T) {
	plan := comparisonPlan()
	records := []budget.ActivityRecord{matchedCost("paid", 101, 34)}
	plan.Expenses[0].OwnerID = "member-2"
	result := comparisonProjection(t, plan, records, "depository", "member-1")
	if result.Comparison.ExpenseBudgetCents != 0 || result.Totals.SpendingCents != 34 || result.Comparison.ExpenseRemainingCents != -34 {
		t.Fatal("current plan ownership changed historical actual allocation")
	}
	plan.Expenses = []budget.Expense{}
	result = comparisonProjection(t, plan, records, "depository", "member-2")
	if result.Totals.SpendingCents != 67 || result.Comparison.ExpenseRemainingCents != -67 {
		t.Fatal("deleted expense discarded actual money or match context")
	}
}

func TestInactiveExpenseHasZeroBudgetAndRetainsActualCost(t *testing.T) {
	plan := comparisonPlan()
	plan.Expenses[0].EndMonth = "2026-10"
	result := comparisonProjection(t, plan, []budget.ActivityRecord{matchedCost("paid", 101, 34)}, "depository", "household")
	if result.Comparison.ExpenseBudgetCents != 0 || result.Totals.SpendingCents != 101 || result.Comparison.ExpenseRemainingCents != -101 {
		t.Fatal("ended expense hid actual spending")
	}
}

func TestComparisonRejectsUnsafeRemainingAmount(t *testing.T) {
	plan := comparisonPlan()
	record := matchedCost("refund", -maxSafeCents, -maxSafeCents)
	_, err := Project(plan, []budget.ActivityRecord{record}, sourcesFor("checking"), testMonth, "household", testNow)
	if err != ErrInvalid {
		t.Fatal("unsafe budget-minus-refund amount was returned")
	}
}

func TestAvailabilityDistinguishesRecordedZeroFromUnavailableHistory(t *testing.T) {
	plan := budget.Blank("One", "Two")
	result := comparisonProjection(t, plan, nil, "depository", "member-2")
	if !result.Comparison.BankAvailable || result.Comparison.InvestmentAvailable || result.Totals.SpendingCents != 0 {
		t.Fatal("recorded zero or household source availability was lost")
	}
	sources := sourcesFor("checking")
	sources[0].Kind, sources[0].HistoryStart = "depository", pointer("invalid")
	result, err := Project(plan, nil, sources, testMonth, "household", testNow)
	if err != nil || result.Comparison.BankAvailable {
		t.Fatal("invalid history dates established an available zero", err)
	}
}

func TestIncompleteStoredAllocationCannotFallBackToPayerShares(t *testing.T) {
	record := matchedCost("paid", 101, 34)
	record.Allocation = nil
	_, err := Project(comparisonPlan(), []budget.ActivityRecord{record}, sourcesFor("checking"), testMonth, "member-1", testNow)
	if err != ErrInvalid {
		t.Fatal("matched record without its allocation silently used payer shares")
	}
}
