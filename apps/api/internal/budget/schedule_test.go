package budget

import (
	"testing"
	"time"
)

func testMonth(t *testing.T, value string) time.Time {
	t.Helper()
	month, err := time.Parse("2006-01", value)
	if err != nil {
		t.Fatal(err)
	}
	return month
}

func scheduledPlan() Household {
	plan := Blank("First", "Second")
	plan.Members[0].MonthlyTakeHomeCents = 10000
	plan.Members[0].Accounts = []AccountBalance{{ID: "savings", Name: "Savings", Type: "savings", BalanceCents: 12345}}
	plan.Members[1].DebtCents = 6789
	plan.Expenses = []Expense{{
		ID: "car", Name: "Car payment", Category: "Transport", Kind: "loan",
		MonthlyAmountCents: 101, OwnerID: "shared", Split: Split{Method: "equal"},
		StartMonth: "2026-01", EndMonth: "2028-12",
	}}
	return plan
}

func TestFinitePaymentIncludesThirtySixMonthsAndRetainsTheRecord(t *testing.T) {
	plan := scheduledPlan()
	tests := []struct {
		month                string
		monthly, yearlyCount int64
	}{
		{"2025-12", 0, 11},
		{"2026-01", 101, 12},
		{"2028-07", 101, 6},
		{"2028-12", 101, 1},
		{"2029-01", 0, 0},
	}
	for _, test := range tests {
		t.Run(test.month, func(t *testing.T) {
			result := Response(plan, testMonth(t, test.month))
			if result.Summary.Household.MonthlyExpensesCents != test.monthly || result.YearlySummary.Household.MonthlyExpensesCents != 101*test.yearlyCount {
				t.Fatal("payment inclusion or twelve-month horizon is incorrect")
			}
			share := result.YearlySummary.ExpenseShares[0]
			if share.FirstMemberCents != 51*test.yearlyCount || share.SecondMemberCents != 50*test.yearlyCount {
				t.Fatal("yearly shares must add repeated monthly cent allocations")
			}
			if len(result.Household.Expenses) != 1 || result.Household.Expenses[0].EndMonth != "2028-12" {
				t.Fatal("projection removed or changed the saved payment")
			}
		})
	}
}

func TestLegacyPaymentsRemainActiveAndBalancesAreNotAnnualized(t *testing.T) {
	plan := scheduledPlan()
	plan.Expenses[0].Kind, plan.Expenses[0].StartMonth, plan.Expenses[0].EndMonth = "", "", ""
	result := Response(plan, testMonth(t, "2030-11"))
	if result.Summary.Household.MonthlyExpensesCents != 101 || result.YearlySummary.Household.MonthlyExpensesCents != 1212 {
		t.Fatal("a legacy payment without dates must remain active")
	}
	if result.YearlySummary.Household.MonthlyTakeHomeCents != 120000 {
		t.Fatal("income did not cover twelve monthly periods")
	}
	if result.YearlySummary.Household.SavingsCents != 12345 || result.YearlySummary.Household.DebtCents != 6789 {
		t.Fatal("balance snapshots were annualized")
	}
}

func TestScheduleValidationRejectsInvalidMetadata(t *testing.T) {
	changes := []struct {
		name   string
		change func(*Expense)
	}{
		{"kind", func(e *Expense) { e.Kind = "transfer" }},
		{"provider", func(e *Expense) { e.ProviderID = "bad_provider" }},
		{"month", func(e *Expense) { e.StartMonth = "2026-13" }},
		{"year zero", func(e *Expense) { e.StartMonth = "0000-01" }},
		{"reversed", func(e *Expense) { e.EndMonth = "2025-12" }},
	}
	for _, test := range changes {
		t.Run(test.name, func(t *testing.T) {
			plan := scheduledPlan()
			test.change(&plan.Expenses[0])
			if Validate(plan) == nil {
				t.Fatal("invalid payment schedule accepted")
			}
		})
	}
}
