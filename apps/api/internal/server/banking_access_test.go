package server

import (
	"bytes"
	"encoding/json"
	"testing"

	"bankpiggy/api/internal/budget"
)

func TestBankHTTPRequiresSessionOriginAndCSRF(t *testing.T) {
	client, provider := newBankHTTPClient(t)
	endpoints := []struct{ method, path string }{
		{"GET", "/api/connections"}, {"GET", "/api/connections/link-token/pending"},
		{"POST", "/api/connections/link-token"}, {"POST", "/api/connections/exchange"},
		{"POST", "/api/connections/foreign/refresh"}, {"DELETE", "/api/connections/foreign"},
		{"PUT", "/api/connections/foreign/accounts/foreign/mapping"},
	}
	for _, endpoint := range endpoints {
		requireStatus(t, client.request(endpoint.method, endpoint.path, []byte(`{}`), testOrigin, ""), 401)
	}
	client.signIn(t)
	for _, endpoint := range endpoints[2:] {
		requireStatus(t, client.request(endpoint.method, endpoint.path, []byte(`{}`), testOrigin, ""), 403)
		requireStatus(t, client.request(endpoint.method, endpoint.path, []byte(`{}`), "https://foreign.example", client.csrf), 403)
	}
	if provider.exchanges.Load() != 0 {
		t.Fatal("rejected mutations reached the provider")
	}
}

func TestBankHTTPExchangeBindsAttemptToSessionAndIsOneUse(t *testing.T) {
	client, provider := newBankHTTPClient(t)
	client.signIn(t)
	attempt := createBankAttempt(t, client)
	request := budget.ExchangeRequest{AttemptID: attempt.AttemptID, PublicToken: httpString("simulated-public-token")}
	other := &testClient{app: client.app}
	other.signIn(t)
	requireStatus(t, other.request("GET", "/api/connections/link-token/pending", nil, "", ""), 404)
	requireStatus(t, other.request("POST", "/api/connections/exchange", bankJSON(t, request), testOrigin, other.csrf), 404)
	requireStatus(t, client.request("GET", "/api/connections/link-token/pending", nil, "", ""), 200)
	requireStatus(t, client.request("POST", "/api/connections/exchange", bankJSON(t, request), testOrigin, client.csrf), 200)
	requireStatus(t, client.request("POST", "/api/connections/exchange", bankJSON(t, request), testOrigin, client.csrf), 404)
	if provider.exchanges.Load() != 1 {
		t.Fatal("exchange was repeated or ran for a foreign session")
	}
	response := client.request("GET", "/api/connections", nil, "", "")
	for _, secret := range []string{"simulated-access-token", "simulated-item-id", "token_cipher", "session_hash"} {
		if bytes.Contains(response.Body.Bytes(), []byte(secret)) {
			t.Fatal("connection response exposed provider credential data")
		}
	}
}

func TestBankHTTPConnectionManagementStaysWithConnectingMember(t *testing.T) {
	client, _ := newBankHTTPClient(t)
	client.signIn(t)
	createBankAttempt(t, client)
	connection := bankList(t, client).Connections[0].ID
	other := bankMemberClient(t, client, "demo-household", "member-2")
	visible := bankList(t, other)
	if len(visible.Connections) != 1 || visible.Connections[0].CanManage {
		t.Fatal("household read or connector permission changed")
	}
	requireStatus(t, other.request("POST", "/api/connections/link-token", bankJSON(t, budget.LinkTokenRequest{ConnectionID: &connection}), testOrigin, other.csrf), 404)
	requireStatus(t, other.request("DELETE", "/api/connections/"+connection, nil, testOrigin, other.csrf), 404)
	requireStatus(t, client.request("POST", "/api/connections/foreign/refresh", nil, testOrigin, client.csrf), 404)
	requireStatus(t, client.request("DELETE", "/api/connections/foreign", nil, testOrigin, client.csrf), 404)
	if err := client.app.store.Seed(t.Context(), "foreign-household", budget.Blank("Foreign")); err != nil {
		t.Fatal(err)
	}
	foreign := bankMemberClient(t, client, "foreign-household", "member-1")
	requireStatus(t, foreign.request("GET", "/api/connections", nil, "", ""), 401)
	absent := bankMemberClient(t, client, "demo-household", "foreign-member")
	requireStatus(t, absent.request("GET", "/api/connections", nil, "", ""), 401)
}

func httpString(value string) *string { return &value }

func TestBankHTTPFreshRetryKeepsOAuthResumeAndRejectsExpiredExchange(t *testing.T) {
	client, provider := newBankHTTPClient(t)
	client.signIn(t)
	first := createBankAttempt(t, client)
	second := createBankAttempt(t, client)
	if first.AttemptID == second.AttemptID || first.LinkToken == second.LinkToken {
		t.Fatal("deliberate retry reused a potentially consumed production Link token")
	}
	response := client.request("GET", "/api/connections/link-token/pending", nil, "", "")
	requireStatus(t, response, 200)
	var resumed budget.LinkTokenResponse
	if err := json.Unmarshal(response.Body.Bytes(), &resumed); err != nil {
		t.Fatal(err)
	}
	if resumed.AttemptID != second.AttemptID || resumed.LinkToken != second.LinkToken {
		t.Fatal("OAuth resume replaced the current Link session")
	}
	_, err := client.app.store.Pool.Exec(t.Context(), `UPDATE bank_link_attempts SET expires_at=now()-interval '1 second' WHERE id=$1`, second.AttemptID)
	if err != nil {
		t.Fatal(err)
	}
	request := budget.ExchangeRequest{AttemptID: second.AttemptID, PublicToken: httpString("simulated-public-token")}
	requireStatus(t, client.request("POST", "/api/connections/exchange", bankJSON(t, request), testOrigin, client.csrf), 404)
	if provider.exchanges.Load() != 0 {
		t.Fatal("expired attempt reached the provider")
	}
}
