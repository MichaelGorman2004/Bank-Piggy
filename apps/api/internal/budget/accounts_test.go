package budget

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestAccountTypesProduceExactPersonalAndHouseholdBalances(t *testing.T) {
	plan := Blank("First", "Second")
	plan.Members[0].Accounts = []AccountBalance{
		{ID: "a", Name: "Savings", Type: "savings", BalanceCents: 1},
		{ID: "b", Name: "High-yield", Type: "high_yield_savings", BalanceCents: 2},
		{ID: "c", Name: "401k", Type: "traditional_401k", BalanceCents: 4},
		{ID: "d", Name: "Roth 401k", Type: "roth_401k", BalanceCents: 8},
		{ID: "e", Name: "IRA", Type: "traditional_ira", BalanceCents: 16},
		{ID: "f", Name: "Roth IRA", Type: "roth_ira", BalanceCents: 32},
		{ID: "g", Name: "Brokerage", Type: "brokerage", BalanceCents: 64},
		{ID: "h", Name: "Other", Type: "other_investment", BalanceCents: 128},
	}
	plan.Members[1].Accounts = []AccountBalance{{ID: "a", Name: "Separate owner", Type: "savings", BalanceCents: 1000}}
	if err := Validate(plan); err != nil {
		t.Fatal(err)
	}
	response := Response(plan, testMonth(t, "2026-11"))
	first := response.Summary.Members["member-1"]
	if first.SavingsCents != 3 || first.InvestmentsCents != 252 || response.Summary.Members["member-2"].SavingsCents != 1000 {
		t.Fatal("account type grouping or member ownership is incorrect")
	}
	want := Summary{SavingsCents: 1003, InvestmentsCents: 252}
	if !reflect.DeepEqual(response.Summary.Household, want) || !reflect.DeepEqual(response.YearlySummary.Household, want) {
		t.Fatal("account balances changed cash flow, were doubled, or were annualized")
	}
}

func TestAccountValidationRejectsInvalidSnapshots(t *testing.T) {
	tests := []struct {
		name   string
		change func(*Household)
	}{
		{"null accounts", func(h *Household) { h.Members[0].Accounts = nil }},
		{"too many", func(h *Household) { h.Members[0].Accounts = makeAccounts(MaxAccounts + 1) }},
		{"duplicate ID", func(h *Household) { h.Members[0].Accounts[1].ID = h.Members[0].Accounts[0].ID }},
		{"invalid ID", func(h *Household) { h.Members[0].Accounts[0].ID = "not an id" }},
		{"long ID", func(h *Household) { h.Members[0].Accounts[0].ID = strings.Repeat("a", 65) }},
		{"blank name", func(h *Household) { h.Members[0].Accounts[0].Name = "  " }},
		{"long name", func(h *Household) { h.Members[0].Accounts[0].Name = strings.Repeat("a", 101) }},
		{"unknown type", func(h *Household) { h.Members[0].Accounts[0].Type = "checking" }},
		{"negative balance", func(h *Household) { h.Members[0].Accounts[0].BalanceCents = -1 }},
		{"balance limit", func(h *Household) { h.Members[0].Accounts[0].BalanceCents = MaxMoneyCents + 1 }},
		{"savings group limit", func(h *Household) { h.Members[0].Accounts[0].BalanceCents = MaxMoneyCents }},
		{"investment group limit", func(h *Household) { h.Members[0].Accounts[2].BalanceCents = MaxMoneyCents }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan := Sample()
			test.change(&plan)
			if Validate(plan) == nil {
				t.Fatal("invalid account snapshot accepted")
			}
		})
	}
}

func makeAccounts(count int) []AccountBalance {
	accounts := make([]AccountBalance, count)
	for index := range accounts {
		accounts[index] = AccountBalance{ID: fmt.Sprintf("account-%d", index), Name: "Savings", Type: "savings"}
	}
	return accounts
}

func TestAccountLimitsAllowEmptyAndMaximumGroups(t *testing.T) {
	plan := Blank("First", "Second")
	if err := Validate(plan); err != nil {
		t.Fatal("empty account arrays must be valid", err)
	}
	plan.Members[0].Accounts = makeAccounts(MaxAccounts)
	plan.Members[0].Accounts[0].BalanceCents = MaxMoneyCents
	plan.Members[0].Accounts[1].Type = "brokerage"
	plan.Members[0].Accounts[1].BalanceCents = MaxMoneyCents
	if err := Validate(plan); err != nil {
		t.Fatal("maximum account count and independent group limits must be valid", err)
	}
}

func TestTypedSamplePreservesEstablishedTotals(t *testing.T) {
	plan := Sample()
	if err := Validate(plan); err != nil {
		t.Fatal(err)
	}
	result := Response(plan, testMonth(t, "2026-11")).Summary.Household
	if result.SavingsCents != 3830000 || result.InvestmentsCents != 9750000 || result.MonthlyGoalsCents != 220000 || result.MonthlyTakeHomeCents != 960000 {
		t.Fatal("typed sample changed established balance or allocation totals")
	}
}
