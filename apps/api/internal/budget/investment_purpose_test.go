package budget

import "testing"

func TestInvestmentPurposeIsExplicitAndCannotCoexistWithDatedTargets(t *testing.T) {
	plan := Blank("One")
	goal := Goal{ID: "goal", Name: "Retirement", OwnerID: "member-1", MonthlyAmountCents: 100}
	if IsInvestmentGoal(goal) {
		t.Fatal("editable goal name identified an investment")
	}
	goal.Purpose = "investment"
	plan.Goals = []Goal{goal}
	if err := Validate(plan); err != nil || !IsInvestmentGoal(goal) {
		t.Fatal("explicit investment goal was rejected", err)
	}
	plan.Goals[0].Purchase = &PurchaseTarget{TargetAmountCents: 100, TargetMonth: "2026-11"}
	if err := Validate(plan); err == nil {
		t.Fatal("purchase target accepted investment purpose")
	}
	plan.Goals[0].Purchase, plan.Goals[0].Purpose = nil, "unknown"
	if err := Validate(plan); err == nil {
		t.Fatal("unknown goal purpose was accepted")
	}
}

func TestSignedExpenseSplitReversesEqualSalaryAndCustomShares(t *testing.T) {
	plan := Blank("One", "Two")
	plan.Members[0].AnnualBaseSalaryCents, plan.Members[1].AnnualBaseSalaryCents = 10000, 20000
	for _, split := range []Split{{Method: "equal"}, {Method: "salary"}, {Method: "custom", CustomFirstShareBps: 3333}} {
		expense := Expense{OwnerID: "shared", MonthlyAmountCents: 101, Split: split}
		first, second := SplitExpense(plan, expense)
		expense.MonthlyAmountCents = -101
		refundFirst, refundSecond := SplitExpense(plan, expense)
		if first+second != 101 || refundFirst != -first || refundSecond != -second {
			t.Fatal("refund shares did not reverse the original rounded allocation")
		}
	}
}
