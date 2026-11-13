package banking

import (
	"reflect"
	"testing"

	"bankpiggy/api/internal/budget"
)

func TestAccountDefaultsUseOnlyProviderTypes(t *testing.T) {
	cases := []struct {
		providerType, subtype string
		want                  budget.ConnectedAccountType
	}{
		{"depository", "checking", "checking"},
		{"depository", "savings", "savings"},
		{"depository", "hsa", "hsa"},
		{"investment", "hsa", "hsa"},
		{"investment", "brokerage", "brokerage"},
		{"investment", "401k", "other_investment"},
		{"investment", "roth", "other_investment"},
		{"investment", "ira", "other_investment"},
		{"investment", "", "other_investment"},
		{"depository", "", "unclassified"},
		{"depository", "money market", "unclassified"},
		{"credit", "credit card", "unclassified"},
		{"loan", "student", "unclassified"},
		{"other", "savings", "unclassified"},
	}
	for _, tc := range cases {
		t.Run(tc.providerType+"/"+tc.subtype, func(t *testing.T) {
			account := untouchedTestAccount()
			account.ProviderType, account.ProviderSubtype = tc.providerType, ptr(tc.subtype)
			mapping := defaultAccountMapping(account, "member-2")
			if mapping.Classification != tc.want || mapping.Enabled != (tc.want != "unclassified") {
				t.Fatalf("provider classification = %s, enabled = %t", mapping.Classification, mapping.Enabled)
			}
			if mapping.Enabled && mapping.Ownership != "member-2" {
				t.Fatal("default did not use the connecting member")
			}
		})
	}
}

func TestAccountDefaultsPreserveDecisionsAndUnsupportedBalances(t *testing.T) {
	cases := map[string]func(*budget.ConnectedAccount){
		"saved exclusion":    func(a *budget.ConnectedAccount) { a.MappingRevision = 2 },
		"enabled mapping":    func(a *budget.ConnectedAccount) { a.Mapping.Enabled = true },
		"classified mapping": func(a *budget.ConnectedAccount) { a.Mapping.Classification = "checking" },
		"manual member":      func(a *budget.ConnectedAccount) { a.Mapping.ManualMemberID = ptr("member-1") },
		"manual account":     func(a *budget.ConnectedAccount) { a.Mapping.ManualAccountID = ptr("manual") },
		"inactive":           func(a *budget.ConnectedAccount) { a.Active = false },
		"missing currency":   func(a *budget.ConnectedAccount) { a.Currency = nil },
		"foreign currency":   func(a *budget.ConnectedAccount) { a.Currency = ptr("CAD") },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			account := untouchedTestAccount()
			change(&account)
			if got := defaultAccountMapping(account, "member-2"); !reflect.DeepEqual(got, account.Mapping) {
				t.Fatal("default replaced a decision or unsupported account mapping")
			}
		})
	}
	account := untouchedTestAccount()
	if got := defaultAccountMapping(account, ""); !reflect.DeepEqual(got, account.Mapping) {
		t.Fatal("default assigned missing ownership")
	}
}

func untouchedTestAccount() budget.ConnectedAccount {
	return budget.ConnectedAccount{ID: "connected", Name: "Roth high-yield savings 401k", Mask: ptr("1234"), ProviderType: "depository", ProviderSubtype: ptr("savings"), Currency: ptr("USD"), CurrentBalanceCents: ptr(int64(10001)), Active: true, MappingRevision: initialMappingRevision, Mapping: budget.AccountMapping{Ownership: "member-1", Classification: "unclassified"}}
}
