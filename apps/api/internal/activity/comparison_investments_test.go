package activity

import (
	"testing"

	"bankpiggy/api/internal/budget"
)

func matchedFunding(id string, amount, first int64, category budget.ActivityCategory) budget.ActivityRecord {
	record := investmentRecord(id, "checking", "2026-11-02", "cash", string(category), amount)
	record.IsReviewed, record.Category, record.GoalID = true, category, pointer("invest")
	record.Allocation = &budget.ActivityAllocation{Ownership: "shared", FirstMemberCents: first}
	return record
}

func investmentComparisonPlan() budget.Household {
	plan := budget.Blank("One", "Two")
	plan.Members[0].AnnualBaseSalaryCents = 1200000
	plan.Members[0].Payroll = &budget.Payroll{Traditional401KBps: 1000, Roth401KBps: 500, MatchMethod: "fixed", EmployerMonthlyCents: 2000, MatchTiers: []budget.MatchTier{}}
	plan.Goals = []budget.Goal{
		{ID: "general", Name: "Retirement", MonthlyAmountCents: 99000, OwnerID: "member-1"},
		{ID: "invest", Name: "Invest", Purpose: "investment", MonthlyAmountCents: 5000, OwnerID: "shared"},
		{ID: "ira", Name: "IRA", MonthlyAmountCents: 3000, OwnerID: "member-1", ContributionYear: pointer(budget.ContributionYear(2025)), SavingsTarget: &budget.SavingsTarget{TargetAmountCents: 10000}},
	}
	return plan
}

func TestInvestmentComparisonUsesExplicitPlansAndSignedFunding(t *testing.T) {
	plan := investmentComparisonPlan()
	records := []budget.ActivityRecord{
		matchedFunding("deposit", -101, -51, "contribution"), matchedFunding("withdraw", 201, 101, "withdrawal"),
		investmentRecord("unmatched", "checking", "2026-11-03", "cash", "deposit", -77),
	}
	first := comparisonProjection(t, plan, records, "investment", "member-1")
	second := comparisonProjection(t, plan, records, "investment", "member-2")
	if first.Totals.NetInvestmentContributionsCents != 27 || second.Totals.NetInvestmentContributionsCents != -50 {
		t.Fatal("investment deposits and withdrawals lost recorded shares")
	}
	comparison := first.Comparison
	amounts := [4]int64{comparison.InvestmentGoalBudgetCents, comparison.PayrollBudgetCents, comparison.InvestmentBudgetCents, comparison.InvestmentRemainingCents}
	if amounts != [4]int64{5500, 17000, 22500, 22473} {
		t.Fatalf("incorrect explicit investment plan: %+v", comparison)
	}
	if len(comparison.InvestmentGoals) != 2 || comparison.InvestmentGoals[0].NetContributionsCents != -50 || comparison.UnmatchedInvestmentNetCents != 77 {
		t.Fatal("ordinary goal name entered investments or unmatched funding disappeared")
	}
	requireSeparateInvestmentHistory(t, comparison)
	if plan.Goals[2].SavingsTarget.SavedAmountCents != 0 {
		t.Fatal("comparison changed manual goal progress")
	}
}

func requireSeparateInvestmentHistory(t *testing.T, comparison budget.ActivityComparison) {
	t.Helper()
	if comparison.BankFlow.TotalCents != 0 || len(comparison.BankFlow.Inflows) != 0 || !comparison.InvestmentAvailable || comparison.BankAvailable {
		t.Fatal("investment funding was added to bank cash flow")
	}
}

func TestDeletedInvestmentGoalRemainsUnmatchedFunding(t *testing.T) {
	plan := investmentComparisonPlan()
	plan.Goals = []budget.Goal{}
	result := comparisonProjection(t, plan, []budget.ActivityRecord{matchedFunding("withdraw", 201, 101, "withdrawal")}, "investment", "member-2")
	if result.Comparison.UnmatchedInvestmentNetCents != -100 || result.Totals.NetInvestmentContributionsCents != -100 {
		t.Fatal("deleted investment goal erased a recorded withdrawal")
	}
}
