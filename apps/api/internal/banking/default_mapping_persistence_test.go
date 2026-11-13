package banking

import (
	"encoding/json"
	"reflect"
	"testing"

	"bankpiggy/api/internal/budget"
)

func TestImportedDefaultsUseConnectingMemberWithoutSavingMapping(t *testing.T) {
	s, p, identity := setupService(t)
	identity.MemberID = "member-2"
	p.snapshot.Accounts[0].Account.ProviderSubtype = ptr("savings")
	plan, err := s.db.Read(t.Context(), identity.HouseholdID)
	if err != nil {
		t.Fatal(err)
	}
	completeTestLink(t, s, identity, createTestLink(t, s, identity))
	refreshDefaultAccounts(t, s)
	list := listDefaultAccounts(t, s, identity)
	account := list.Connections[0].Accounts[0]
	assertDefaultMapping(t, account, "member-2", "savings")
	assertUntouchedStoredMapping(t, s, account.ID)
	before := EffectiveBalances(plan, nil)
	if list.EffectiveBalances.Household.SavingsCents != before.Household.SavingsCents+10001 || list.EffectiveBalances.Members["member-2"].SavingsCents != before.Members["member-2"].SavingsCents+10001 {
		t.Fatal("import did not enter household and connecting-member totals")
	}
	actual, err := s.db.Read(t.Context(), identity.HouseholdID)
	if err != nil || !reflect.DeepEqual(actual, plan) {
		t.Fatal("import or list changed the saved plan", err)
	}
}

func TestLaterImportsUseConnectingMemberDefaults(t *testing.T) {
	s, p, identity := setupService(t)
	identity.MemberID = "member-2"
	completeTestLink(t, s, identity, createTestLink(t, s, identity))
	refreshDefaultAccounts(t, s)
	before := listDefaultAccounts(t, s, identity).EffectiveBalances.Household
	p.snapshot.Accounts = append(p.snapshot.Accounts, ProviderAccount{ProviderID: "new-provider-account", Account: budget.ConnectedAccount{Name: "New checking", ProviderType: "depository", ProviderSubtype: ptr("checking"), Currency: ptr("USD"), CurrentBalanceCents: ptr(int64(33))}})
	refreshDefaultAccounts(t, s)
	list := listDefaultAccounts(t, s, identity)
	for _, imported := range list.Connections[0].Accounts {
		if imported.ProviderSubtype != nil && *imported.ProviderSubtype == "checking" {
			assertDefaultMapping(t, imported, "member-2", "checking")
		}
	}
	if len(list.Connections[0].Accounts) != 2 || list.EffectiveBalances.Household.CheckingCents != before.CheckingCents+33 {
		t.Fatal("later import did not receive the same defaults")
	}
}

func TestExistingUntouchedImportsResolveDefaultsWithoutDatabaseWrites(t *testing.T) {
	s, _, identity, connection := connectedService(t)
	id := connection.Accounts[0].ID
	execSQL(t, s, `UPDATE bank_accounts SET snapshot=snapshot || '{"providerType":"investment","providerSubtype":"roth","currentBalanceCents":0}'::jsonb WHERE id=$1`, id)
	list := listDefaultAccounts(t, s, identity)
	assertDefaultMapping(t, list.Connections[0].Accounts[0], identity.MemberID, "other_investment")
	assertUntouchedStoredMapping(t, s, id)
	for _, row := range list.EffectiveBalances.Accounts {
		if row.ID == id && (!row.Included || row.BalanceCents == nil || *row.BalanceCents != 0) {
			t.Fatal("a known zero balance became unavailable")
		}
	}
}

func TestSavedMappingsSurviveProviderRefresh(t *testing.T) {
	cases := map[string]budget.AccountMappingRequest{
		"excluded":                 {Enabled: false, Ownership: "member-2", Classification: "unclassified"},
		"confirmed type and owner": {Enabled: true, Ownership: "member-2", Classification: "roth_ira"},
		"manual replacement":       {Enabled: true, Ownership: "shared", Classification: "high_yield_savings", ManualMemberID: ptr("member-1")},
	}
	for name, request := range cases {
		t.Run(name, func(t *testing.T) {
			assertSavedMappingSurvivesRefresh(t, request)
		})
	}
}

func assertSavedMappingSurvivesRefresh(t *testing.T, request budget.AccountMappingRequest) {
	t.Helper()
	s, p, identity, connection := connectedService(t)
	account := connection.Accounts[0]
	request.Revision = account.MappingRevision
	if request.ManualMemberID != nil {
		plan, err := s.db.Read(t.Context(), identity.HouseholdID)
		if err != nil {
			t.Fatal(err)
		}
		request.ManualAccountID = ptr(plan.Members[0].Accounts[0].ID)
	}
	if err := s.MapAccount(t.Context(), identity, connection.ID, account.ID, request); err != nil {
		t.Fatal(err)
	}
	before := listDefaultAccounts(t, s, identity).Connections[0].Accounts[0]
	p.snapshot.Accounts[0].Account.ProviderSubtype = ptr("checking")
	p.snapshot.Accounts[0].Account.CurrentBalanceCents = ptr(int64(20003))
	refreshDefaultAccounts(t, s)
	after := listDefaultAccounts(t, s, identity).Connections[0].Accounts[0]
	if after.MappingRevision != 2 || !reflect.DeepEqual(before.Mapping, after.Mapping) || after.ID != account.ID {
		t.Fatal("sync replaced the saved mapping or account identity")
	}
	if after.CurrentBalanceCents == nil || *after.CurrentBalanceCents != 20003 {
		t.Fatal("sync did not update the provider balance")
	}
}

func TestDefaultsKeepMissingForeignAndUnsupportedBalancesOutOfTotals(t *testing.T) {
	s, p, identity := setupService(t)
	accounts := []budget.ConnectedAccount{untouchedTestAccount(), untouchedTestAccount(), untouchedTestAccount(), untouchedTestAccount(), untouchedTestAccount(), untouchedTestAccount()}
	accounts[0].CurrentBalanceCents = nil
	accounts[1].Currency = nil
	accounts[2].Currency = ptr("CAD")
	accounts[3].ProviderType = "credit"
	accounts[4].ProviderType = "loan"
	accounts[5].ProviderSubtype = nil
	p.snapshot.Accounts = nil
	for index, account := range accounts {
		id := string(rune('a' + index))
		account.Name = id
		p.snapshot.Accounts = append(p.snapshot.Accounts, ProviderAccount{ProviderID: id, Account: account})
	}
	completeTestLink(t, s, identity, createTestLink(t, s, identity))
	refreshDefaultAccounts(t, s)
	list := listDefaultAccounts(t, s, identity)
	for _, account := range list.Connections[0].Accounts {
		if account.Mapping.Enabled != (account.Name == "a") {
			t.Fatal("unsupported account defaulted to included, or missing balance lost its classification")
		}
	}
	for _, row := range list.EffectiveBalances.Accounts {
		if row.Source == "plaid" && row.Included {
			t.Fatal("unavailable or unsupported balance entered totals")
		}
	}
}

func TestDefaultsDoNotMatchManualAccountsByNameOrMask(t *testing.T) {
	s, p, identity := setupService(t)
	plan, err := s.db.Read(t.Context(), identity.HouseholdID)
	if err != nil {
		t.Fatal(err)
	}
	account := untouchedTestAccount()
	account.Name = plan.Members[0].Accounts[0].Name
	p.snapshot.Accounts = []ProviderAccount{{ProviderID: "first", Account: account}, {ProviderID: "second", Account: account}}
	completeTestLink(t, s, identity, createTestLink(t, s, identity))
	refreshDefaultAccounts(t, s)
	list := listDefaultAccounts(t, s, identity)
	before := EffectiveBalances(plan, nil)
	if list.EffectiveBalances.Household.SavingsCents != before.Household.SavingsCents+20002 {
		t.Fatal("matching names or masks replaced or deduplicated account identities")
	}
	for _, imported := range list.Connections[0].Accounts {
		if imported.Mapping.ManualMemberID != nil || imported.Mapping.ManualAccountID != nil {
			t.Fatal("default guessed a manual account match")
		}
	}
}

func listDefaultAccounts(t *testing.T, s *Service, identity Identity) budget.ConnectionList {
	t.Helper()
	list, err := s.List(t.Context(), identity)
	if err != nil {
		t.Fatal(err)
	}
	return list
}

func refreshDefaultAccounts(t *testing.T, s *Service) {
	t.Helper()
	execSQL(t, s, `UPDATE bank_connections SET next_sync_at=now()`)
	if err := s.processOne(t.Context(), ""); err != nil {
		t.Fatal(err)
	}
}

func assertDefaultMapping(t *testing.T, account budget.ConnectedAccount, owner string, kind budget.ConnectedAccountType) {
	t.Helper()
	if !account.Mapping.Enabled || account.Mapping.Ownership != budget.AccountOwnership(owner) || account.Mapping.Classification != kind || account.MappingRevision != initialMappingRevision {
		t.Fatal("untouched import did not resolve its default mapping")
	}
}

func assertUntouchedStoredMapping(t *testing.T, s *Service, id string) {
	t.Helper()
	var data []byte
	var revision int64
	if err := s.db.Pool.QueryRow(t.Context(), `SELECT mapping,mapping_revision FROM bank_accounts WHERE id=$1`, id).Scan(&data, &revision); err != nil {
		t.Fatal(err)
	}
	var mapping budget.AccountMapping
	if err := json.Unmarshal(data, &mapping); err != nil {
		t.Fatal(err)
	}
	if mapping.Enabled || mapping.Classification != "unclassified" || revision != initialMappingRevision {
		t.Fatal("reading defaults saved a user decision")
	}
}
