package budget

import "testing"

func TestSharedExpenseAllocations(t *testing.T) {
	tests := []struct {
		name, method       string
		amount, bps, first int64
	}{
		{"salary ratio", "salary", 300000, 0, 200000},
		{"salary fractional cent", "salary", 100, 0, 67},
		{"equal odd cent", "equal", 101, 0, 51},
		{"custom percent", "custom", 999, 2500, 250},
		{"custom zero", "custom", 999, 0, 0},
		{"custom full", "custom", 999, 10000, 999},
		{"maximum safe product", "salary", MaxMoneyCents, 0, 66666666667},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan := Sample()
			plan.Expenses = []Expense{{ID: "bill", Name: "Bill", Category: "Bills", OwnerID: "shared", MonthlyAmountCents: test.amount, Split: Split{Method: test.method, CustomFirstShareBps: test.bps}}}
			if err := Validate(plan); err != nil {
				t.Fatal(err)
			}
			summary := calculateMonths(plan, []string{"2026-11"})
			share := summary.ExpenseShares[0]
			if share.FirstMemberCents != test.first || share.SecondMemberCents != test.amount-test.first {
				t.Fatalf("unexpected split: %+v", share)
			}
			if summary.Household.MonthlyExpensesCents != test.amount {
				t.Fatal("shared expense was counted more than once")
			}
		})
	}
}

func TestSummarySeparatesSpendingGoalsAndBalances(t *testing.T) {
	plan := Blank("First", "Second")
	plan.Members[0].MonthlyTakeHomeCents = 10000
	plan.Members[1].MonthlyTakeHomeCents = 5000
	plan.Members[0].Accounts = []AccountBalance{{ID: "savings", Name: "Savings", Type: "savings", BalanceCents: 200000}}
	plan.Expenses = []Expense{{ID: "personal", Name: "Personal", Category: "Food", MonthlyAmountCents: 3000, OwnerID: "member-2", PaidByMemberID: "member-1", Split: Split{Method: "equal"}}}
	plan.Goals = []Goal{{ID: "shared", Name: "Save", MonthlyAmountCents: 1001, OwnerID: "shared"}}
	summary := calculateMonths(plan, []string{"2026-11"})
	if summary.Members["member-1"].MonthlyExpensesCents != 0 {
		t.Fatal("payer changed the owner allocation")
	}
	if summary.Members["member-2"].MonthlyExpensesCents != 3000 {
		t.Fatal("personal cost was not allocated to its owner")
	}
	if summary.Household.MonthlyAvailableCents != 12000 || summary.Household.MonthlyRemainingCents != 10999 {
		t.Fatalf("incorrect cash flow: %+v", summary.Household)
	}
	if summary.Household.SavingsCents != 200000 {
		t.Fatal("savings balance changed cash flow")
	}
	if summary.GoalShares[0].FirstMemberCents != 501 || summary.GoalShares[0].SecondMemberCents != 500 {
		t.Fatal("goal cents were lost")
	}
}

func TestValidationRejectsInvalidPlans(t *testing.T) {
	tests := []struct {
		name   string
		change func(*Household)
	}{
		{"zero salary", func(h *Household) {
			h.Members[0].AnnualBaseSalaryCents = 0
			h.Members[1].AnnualBaseSalaryCents = 0
		}},
		{"negative", func(h *Household) { h.Members[0].Accounts[0].BalanceCents = -1 }},
		{"excessive", func(h *Household) { h.Expenses[0].MonthlyAmountCents = MaxMoneyCents + 1 }},
		{"unknown owner", func(h *Household) { h.Goals[0].OwnerID = "outsider" }},
		{"duplicate ID", func(h *Household) { h.Expenses[1].ID = h.Expenses[0].ID }},
		{"unknown member", func(h *Household) { h.Members[0].ID = "outsider" }},
		{"custom percent", func(h *Household) { h.Expenses[0].Split = Split{Method: "custom", CustomFirstShareBps: 10001} }},
		{"nil list", func(h *Household) { h.Goals = nil }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan := Sample()
			test.change(&plan)
			if Validate(plan) == nil {
				t.Fatal("invalid plan accepted")
			}
		})
	}
}

func TestSalaryZeroForOneMemberIsValid(t *testing.T) {
	plan := Sample()
	plan.Members[0].AnnualBaseSalaryCents = 0
	if err := Validate(plan); err != nil {
		t.Fatal(err)
	}
	share := calculateMonths(plan, []string{"2026-11"}).ExpenseShares[0]
	if share.FirstMemberCents != 0 || share.SecondMemberCents != 300000 {
		t.Fatal("zero salary allocation incorrect")
	}
}
