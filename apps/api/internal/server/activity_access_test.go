package server

import (
	"encoding/json"
	"testing"

	"bankpiggy/api/internal/budget"
)

func TestActivityHTTPRequiresSessionOriginAndCSRF(t *testing.T) {
	client, provider := newBankHTTPClient(t)
	endpoints := []struct{ method, path string }{
		{"GET", "/api/activity?month=2026-11"},
		{"PUT", "/api/connections/foreign/activity"},
		{"PUT", "/api/activity/records/foreign"},
		{"POST", "/api/activity/paychecks/foreign/apply?month=2026-11"},
	}
	for _, endpoint := range endpoints {
		requireStatus(t, client.request(endpoint.method, endpoint.path, []byte(`{}`), testOrigin, ""), 401)
	}
	client.signIn(t)
	for _, endpoint := range endpoints[1:] {
		requireStatus(t, client.request(endpoint.method, endpoint.path, []byte(`{}`), testOrigin, ""), 403)
		requireStatus(t, client.request(endpoint.method, endpoint.path, []byte(`{}`), "https://foreign.example", client.csrf), 403)
	}
	if provider.links.Load() != 0 || provider.exchanges.Load() != 0 {
		t.Fatal("rejected activity requests reached the provider")
	}
}

func TestActivityHTTPReadIsSnapshotOnly(t *testing.T) {
	client, provider := newBankHTTPClient(t)
	client.signIn(t)
	before := readPlan(t, client)
	response := client.request("GET", "/api/activity?month=2026-11", nil, "", "")
	requireStatus(t, response, 200)
	var result budget.ActivityResponse
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Month != "2026-11" || result.Scope != "household" || len(result.Records) != 0 {
		t.Fatal("empty activity response did not preserve month and scope")
	}
	if readPlan(t, client).Revision != before.Revision || provider.links.Load() != 0 || provider.exchanges.Load() != 0 {
		t.Fatal("reading activity changed the plan or reached the provider")
	}
	absent := bankMemberClient(t, client, "demo-household", "foreign-member")
	requireStatus(t, absent.request("GET", "/api/activity", nil, "", ""), 401)
}

func TestActivityHTTPSettingsRequireConnectingMemberAndCostApproval(t *testing.T) {
	client, _ := newBankHTTPClient(t)
	client.signIn(t)
	createBankAttempt(t, client)
	id := bankList(t, client).Connections[0].ID
	if _, err := client.app.store.Pool.Exec(t.Context(), `UPDATE bank_connections SET state='connected' WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	seedHTTPBankAccount(t, client, id)
	request := budget.ActivitySettingsRequest{Enabled: true, Accounts: []budget.ActivityAccountSelection{{AccountID: "http-account", Enabled: true, Ownership: "member-1"}}}
	path := "/api/connections/" + id + "/activity"
	other := bankMemberClient(t, client, "demo-household", "member-2")
	requireStatus(t, other.request("PUT", path, bankJSON(t, request), testOrigin, other.csrf), 404)
	requireStatus(t, client.request("PUT", path, bankJSON(t, request), testOrigin, client.csrf), 503)
	var count int
	if err := client.app.store.Pool.QueryRow(t.Context(), `SELECT count(*) FROM bank_activity_jobs`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("rejected settings created an activity subscription")
	}
	request.Enabled = false
	requireStatus(t, client.request("PUT", path, bankJSON(t, request), testOrigin, client.csrf), 200)
	requireStatus(t, client.request("PUT", path, bankJSON(t, request), testOrigin, client.csrf), 409)
}
