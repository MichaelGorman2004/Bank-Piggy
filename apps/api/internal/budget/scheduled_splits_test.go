package budget

import "testing"

func TestScheduledSplitsKeepZeroSharesAndRepeatedMonthlyCents(t *testing.T) {
	tests := []struct {
		name, method string
		basis, first int64
	}{
		{"custom zero", "custom", 0, 0},
		{"custom full", "custom", 10000, 101},
		{"equal odd cent", "equal", 0, 51},
		{"salary ratio", "salary", 0, 67},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan := Sample()
			plan.Expenses = []Expense{{
				ID: "payment", Name: "Payment", Category: "Bills", OwnerID: "shared",
				MonthlyAmountCents: 101, StartMonth: "2026-11", EndMonth: "2026-12",
				Split: Split{Method: test.method, CustomFirstShareBps: test.basis},
			}}
			result := Response(plan, testMonth(t, "2026-11"))
			share := result.Summary.ExpenseShares[0]
			if share.FirstMemberCents != test.first || share.SecondMemberCents != 101-test.first {
				t.Fatal("schedule changed the exact monthly split")
			}
			yearly := result.YearlySummary.ExpenseShares[0]
			if yearly.FirstMemberCents != test.first*2 || yearly.SecondMemberCents != (101-test.first)*2 {
				t.Fatal("projection rounded a combined total instead of repeating monthly cents")
			}
			ended := Response(plan, testMonth(t, "2027-01"))
			if ended.Summary.Household.MonthlyExpensesCents != 0 || len(ended.Household.Expenses) != 1 {
				t.Fatal("ended payments must stop contributing without losing their records")
			}
		})
	}
}
