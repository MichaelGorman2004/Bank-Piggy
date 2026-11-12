package budget_test

import (
	"testing"

	"bankpiggy/api/internal/budget"
)

func soloPlan() budget.Household {
	plan := budget.Blank("First")
	plan.Members[0].MonthlyTakeHomeCents = 10000
	plan.Members[0].DebtCents = 456
	plan.Members[0].Accounts = []budget.AccountBalance{{ID: "saved", Name: "Saved", Type: "savings", BalanceCents: 123}}
	plan.Expenses = []budget.Expense{{ID: "bill", Name: "Bill", Category: "Bills", OwnerID: "shared", MonthlyAmountCents: 101, Split: budget.Split{Method: "equal"}}}
	plan.Goals = []budget.Goal{{ID: "goal", Name: "Goal", OwnerID: "shared", MonthlyAmountCents: 103}}
	return plan
}

func TestSoloSharedAllocationsKeepEveryCent(t *testing.T) {
	for _, method := range []string{"equal", "custom", "salary"} {
		t.Run(method, func(t *testing.T) {
			plan := soloPlan()
			plan.Expenses[0].Split = budget.Split{Method: method, CustomFirstShareBps: 0}
			if err := budget.Validate(plan); err != nil {
				t.Fatal(err)
			}
			response := budget.Response(plan, testMonth(t, "2026-11"))
			requireSoloSummary(t, response.Summary, 1)
			requireSoloSummary(t, response.YearlySummary, 12)
		})
	}
}

func requireSoloSummary(t *testing.T, summary budget.BudgetSummary, periods int64) {
	t.Helper()
	if len(summary.Members) != 1 || summary.Members["member-1"] != summary.Household {
		t.Fatal("solo totals contain an absent member")
	}
	if summary.ExpenseShares[0].FirstMemberCents != 101*periods || summary.ExpenseShares[0].SecondMemberCents != 0 {
		t.Fatalf("solo expense lost cents: %+v", summary.ExpenseShares[0])
	}
	if summary.GoalShares[0].FirstMemberCents != 103*periods || summary.GoalShares[0].SecondMemberCents != 0 {
		t.Fatalf("solo goal lost cents: %+v", summary.GoalShares[0])
	}
	if summary.Household.MonthlyRemainingCents != 9796*periods || summary.Household.SavingsCents != 123 || summary.Household.DebtCents != 456 {
		t.Fatalf("solo totals changed cash flow or snapshots: %+v", summary.Household)
	}
}

func TestAddingPartnerPreservesPersonalAllocations(t *testing.T) {
	plan := soloPlan()
	plan.Expenses[0].OwnerID = "member-1"
	plan.Expenses[0].PaidByMemberID = "member-1"
	plan.Goals[0].OwnerID = "member-1"
	before := budget.Response(plan, testMonth(t, "2026-11")).Summary
	plan.Members = append(plan.Members, budget.Blank("First", "Second").Members[1])
	if err := budget.Validate(plan); err != nil {
		t.Fatal(err)
	}
	after := budget.Response(plan, testMonth(t, "2026-11")).Summary
	if after.Members["member-1"] != before.Members["member-1"] || after.Household != before.Household {
		t.Fatal("adding a partner redistributed personal money")
	}
	if after.ExpenseShares[0] != before.ExpenseShares[0] || after.GoalShares[0] != before.GoalShares[0] || after.Members["member-2"] != (budget.Summary{}) {
		t.Fatal("adding an empty partner changed personal allocations")
	}
}

func TestSoloValidationRejectsAbsentAndInvalidMembers(t *testing.T) {
	tests := []struct {
		name   string
		change func(*budget.Household)
	}{
		{"no members", func(h *budget.Household) { h.Members = nil }},
		{"empty members", func(h *budget.Household) { h.Members = []budget.Member{} }},
		{"too many members", func(h *budget.Household) { h.Members = budget.Blank("One", "Two", "Three").Members }},
		{"wrong first ID", func(h *budget.Household) { h.Members[0].ID = "member-2" }},
		{"duplicate member", func(h *budget.Household) { h.Members = append(h.Members, h.Members[0]) }},
		{"absent expense owner", func(h *budget.Household) { h.Expenses[0].OwnerID = "member-2" }},
		{"absent expense payer", func(h *budget.Household) { h.Expenses[0].PaidByMemberID = "member-2" }},
		{"absent goal owner", func(h *budget.Household) { h.Goals[0].OwnerID = "member-2" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan := soloPlan()
			test.change(&plan)
			if err := budget.Validate(plan); err == nil {
				t.Fatal("invalid solo plan was accepted")
			}
		})
	}
}
