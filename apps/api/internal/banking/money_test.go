package banking

import (
	"encoding/json"
	"testing"

	"bankpiggy/api/internal/budget"
)

func TestExactProviderCents(t *testing.T) {
	cases := map[string]int64{"0": 0, "19.99": 1999, "-1.01": -101, "1e2": 10000, "999999999.99": 99999999999, "0.29": 29}
	for input, want := range cases {
		number := json.Number(input)
		actual, err := MoneyCents(&number)
		if err != nil || *actual != want {
			t.Fatalf("%s did not convert exactly", input)
		}
	}
	for _, input := range []string{"0.001", "1000000000.01", "-1000000000.01", "NaN", "1e1000000"} {
		number := json.Number(input)
		if _, err := MoneyCents(&number); err == nil {
			t.Fatalf("invalid money accepted: %s", input)
		}
	}
	if value, err := MoneyCents(nil); err != nil || value != nil {
		t.Fatal("missing balance became zero")
	}
}

func TestEffectiveBalanceMappingAndSignedJointRounding(t *testing.T) {
	plan := budget.Blank("First")
	plan.Members[0].Accounts = []budget.AccountBalance{{ID: "manual", Name: "Manual savings", Type: "savings", BalanceCents: 500}}
	plan.Members = append(plan.Members, budget.Member{ID: "member-2", Accounts: []budget.AccountBalance{}})
	account := budget.ConnectedAccount{ID: "connected", Name: "Joint", ProviderType: "depository", Currency: ptr("USD"), CurrentBalanceCents: ptr(int64(-101)), Active: true, Mapping: budget.AccountMapping{Enabled: true, Ownership: "shared", Classification: "savings", ManualMemberID: ptr("member-1"), ManualAccountID: ptr("manual")}}
	connections := []budget.Connection{{State: "connected", Accounts: []budget.ConnectedAccount{account}}}
	result := EffectiveBalances(plan, connections)
	if result.Household.SavingsCents != -101 || result.Members["member-1"].SavingsCents != -51 || result.Members["member-2"].SavingsCents != -50 {
		t.Fatal("joint account was duplicated or rounded incorrectly")
	}
	if plan.Members[0].Accounts[0].BalanceCents != 500 {
		t.Fatal("provider changed manual plan")
	}
	connections[0].Accounts[0].Currency = ptr("CAD")
	result = EffectiveBalances(plan, connections)
	if result.Household.SavingsCents != 500 {
		t.Fatal("foreign currency replaced USD manual account")
	}
	connections[0].Accounts[0].Currency = ptr("USD")
	connections[0].Accounts[0].CurrentBalanceCents = nil
	if EffectiveBalances(plan, connections).Household.SavingsCents != 500 {
		t.Fatal("missing value suppressed manual account")
	}
	connections[0].Accounts[0].CurrentBalanceCents = ptr(int64(10))
	connections[0].State = "disconnecting"
	if EffectiveBalances(plan, connections).Household.SavingsCents != 500 {
		t.Fatal("disconnected account stayed in totals")
	}
}
