package server

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"testing"

	"bankpiggy/api/internal/budget"
)

func accountPlan() budget.Household {
	plan := budget.Blank("First", "Second")
	plan.Members[0].Accounts = []budget.AccountBalance{
		{ID: "savings", Name: "Emergency savings", Type: "high_yield_savings", BalanceCents: 12345},
		{ID: "roth", Name: "Roth account", Type: "roth_ira", BalanceCents: 54321},
	}
	return plan
}

func TestAccountsRequireCompleteWholeCentJSON(t *testing.T) {
	plan := accountPlan()
	valid := planJSON(t, plan)
	account, err := json.Marshal(plan.Members[0].Accounts[0])
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		body []byte
	}{
		{"missing accounts", bytes.Replace(valid, []byte(`,"accounts":[]`), nil, 1)},
		{"null accounts", bytes.Replace(valid, []byte(`"accounts":[]`), []byte(`"accounts":null`), 1)},
		{"null account", bytes.Replace(valid, account, []byte(`null`), 1)},
		{"missing ID", bytes.Replace(valid, []byte(`"id":"savings",`), nil, 1)},
		{"missing name", bytes.Replace(valid, []byte(`"name":"Emergency savings",`), nil, 1)},
		{"missing type", bytes.Replace(valid, []byte(`"type":"high_yield_savings",`), nil, 1)},
		{"missing balance", bytes.Replace(valid, []byte(`,"balanceCents":12345`), nil, 1)},
		{"null balance", bytes.Replace(valid, []byte(`"balanceCents":12345`), []byte(`"balanceCents":null`), 1)},
		{"fractional balance", bytes.Replace(valid, []byte(`"balanceCents":12345`), []byte(`"balanceCents":12.5`), 1)},
		{"legacy aggregate", bytes.Replace(valid, []byte(`"accounts":[]`), []byte(`"accounts":[],"savingsCents":0`), 1)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest("PUT", "/api/household", bytes.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			if _, err := decodePlan(httptest.NewRecorder(), request); err == nil {
				t.Fatal("incomplete account or noncanonical aggregate accepted")
			}
		})
	}
}

func TestTypedAccountBalancesSurviveHTTPAndPostgreSQL(t *testing.T) {
	client := newTestClient(t)
	client.signIn(t)
	plan := readPlan(t, client)
	plan.Members[0].Accounts = accountPlan().Members[0].Accounts
	plan.Members[1].Accounts = []budget.AccountBalance{}
	accepted := decodeResponse(t, client.request("PUT", "/api/household", planJSON(t, plan), testOrigin, client.csrf))
	loaded := decodeResponse(t, client.request("GET", "/api/household", nil, "", ""))
	if !reflect.DeepEqual(loaded.Household.Members, plan.Members) {
		t.Fatal("account identities, names, types, balances, or ordering did not persist")
	}
	if loaded.Summary.Household.SavingsCents != 12345 || loaded.Summary.Household.InvestmentsCents != 54321 || loaded.Summary.Members["member-2"].SavingsCents != 0 {
		t.Fatal("saved account totals did not retain ownership and type grouping")
	}
	loaded.Household.Members[0].Accounts[0].Type = "brokerage"
	changed := decodeResponse(t, client.request("PUT", "/api/household", planJSON(t, loaded.Household), testOrigin, client.csrf))
	if changed.Summary.Household.SavingsCents != 0 || changed.Summary.Household.InvestmentsCents != 66666 {
		t.Fatal("changing account type duplicated or lost the balance")
	}
	if changed.Summary.Household.MonthlyRemainingCents != accepted.Summary.Household.MonthlyRemainingCents {
		t.Fatal("account snapshots changed the monthly plan")
	}
}

func TestInvalidAccountsLeaveSavedHouseholdIntact(t *testing.T) {
	client := newTestClient(t)
	client.signIn(t)
	before := readPlan(t, client)
	changes := []func(*budget.Household){
		func(h *budget.Household) { h.Members[0].Accounts[0].Type = "unknown" },
		func(h *budget.Household) { h.Members[0].Accounts[1].ID = h.Members[0].Accounts[0].ID },
		func(h *budget.Household) { h.Members[0].Accounts[0].BalanceCents = budget.MaxMoneyCents },
	}
	for _, change := range changes {
		plan := readPlan(t, client)
		change(&plan)
		requireStatus(t, client.request("PUT", "/api/household", planJSON(t, plan), testOrigin, client.csrf), 400)
	}
	if !reflect.DeepEqual(before, readPlan(t, client)) {
		t.Fatal("rejected account edits changed the saved household or revision")
	}
}
