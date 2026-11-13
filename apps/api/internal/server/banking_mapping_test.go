package server

import (
	"reflect"
	"testing"
)

func TestBankHTTPMappingIsRevisionCheckedWithoutChangingPlan(t *testing.T) {
	client, _ := newBankHTTPClient(t)
	client.signIn(t)
	before := readPlan(t, client)
	createBankAttempt(t, client)
	connection := bankList(t, client).Connections[0].ID
	seedHTTPBankAccount(t, client, connection)
	path := "/api/connections/" + connection + "/accounts/http-account/mapping"
	body := []byte(`{"revision":1,"enabled":true,"ownership":"shared","classification":"savings"}`)
	requireStatus(t, client.request("PUT", path, body, testOrigin, client.csrf), 200)
	requireStatus(t, client.request("PUT", path, body, testOrigin, client.csrf), 409)
	result := bankList(t, client)
	if result.PlanRevision != before.Revision {
		t.Fatal("projection did not identify its saved plan revision")
	}
	if !result.Connections[0].Accounts[0].Mapping.Enabled || result.Connections[0].Accounts[0].MappingRevision != 2 {
		t.Fatal("mapping did not persist once")
	}
	first, second := result.EffectiveBalances.Members["member-1"], result.EffectiveBalances.Members["member-2"]
	if first.SavingsCents+second.SavingsCents != result.EffectiveBalances.Household.SavingsCents {
		t.Fatal("shared effective totals lost a cent")
	}
	if !reflect.DeepEqual(before, readPlan(t, client)) {
		t.Fatal("account mapping changed plan data or revision")
	}
}

func TestBankHTTPMappingRejectsForeignIdentifiersAndFields(t *testing.T) {
	client, _ := newBankHTTPClient(t)
	client.signIn(t)
	createBankAttempt(t, client)
	connection := bankList(t, client).Connections[0].ID
	seedHTTPBankAccount(t, client, connection)
	path := "/api/connections/" + connection + "/accounts/http-account/mapping"
	bodies := []string{
		`{"revision":1,"enabled":true,"ownership":"foreign","classification":"savings"}`,
		`{"revision":1,"enabled":true,"ownership":"member-1","classification":"savings","manualMemberId":"member-1","manualAccountId":"foreign"}`,
		`{"revision":1,"enabled":true,"ownership":"member-1","classification":"savings","householdId":"foreign"}`,
		`{"revision":1,"enabled":true,"ownership":"member-1","classification":"savings","manualMemberId":"member-1"}`,
		`{"revision":1,"enabled":true,"ownership":"member-1","classification":"not-real"}`,
	}
	for _, body := range bodies {
		requireStatus(t, client.request("PUT", path, []byte(body), testOrigin, client.csrf), 400)
	}
	valid := []byte(`{"revision":1,"enabled":false,"ownership":"member-1","classification":"unclassified"}`)
	requireStatus(t, client.request("PUT", "/api/connections/foreign/accounts/http-account/mapping", valid, testOrigin, client.csrf), 404)
	requireStatus(t, client.request("PUT", "/api/connections/"+connection+"/accounts/foreign/mapping", valid, testOrigin, client.csrf), 404)
	if bankList(t, client).Connections[0].Accounts[0].MappingRevision != 1 {
		t.Fatal("rejected mapping changed saved state")
	}
}

func TestBankHTTPDisabledIntegrationPreservesManualPlanning(t *testing.T) {
	client := newTestClient(t)
	client.signIn(t)
	before := readPlan(t, client)
	result := bankList(t, client)
	if result.Enabled || result.Environment != "disabled" || len(result.Connections) != 0 {
		t.Fatal("disabled integration did not expose a clear empty state")
	}
	requireStatus(t, client.request("POST", "/api/connections/link-token", []byte(`{"kind":"depository"}`), testOrigin, client.csrf), 503)
	if !reflect.DeepEqual(before, readPlan(t, client)) {
		t.Fatal("unavailable provider changed the manual plan")
	}
}
