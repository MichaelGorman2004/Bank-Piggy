package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"bankpiggy/api/internal/banking"
	"bankpiggy/api/internal/budget"
	"bankpiggy/api/internal/store"
)

// httpBankProvider is an isolated provider simulation; it never makes network calls.
type httpBankProvider struct {
	exchanges atomic.Int64
	links     atomic.Int64
}

func (p *httpBankProvider) Link(context.Context, string, string, string) (string, time.Time, error) {
	return fmt.Sprintf("simulated-link-token-%d", p.links.Add(1)), time.Now().Add(20 * time.Minute), nil
}
func (p *httpBankProvider) Exchange(context.Context, string) (string, string, error) {
	p.exchanges.Add(1)
	return "simulated-access-token", "simulated-item-id", nil
}
func (p *httpBankProvider) Accounts(context.Context, string, string) (banking.Snapshot, error) {
	return banking.Snapshot{}, errors.New("HTTP boundary test does not run a worker")
}
func (p *httpBankProvider) Remove(context.Context, string) error { return nil }
func (p *httpBankProvider) VerificationKey(context.Context, string) ([]byte, error) {
	return nil, errors.New("HTTP boundary test does not fetch provider keys")
}

func newBankHTTPClient(t *testing.T) (*testClient, *httpBankProvider) {
	t.Helper()
	client := newTestClient(t)
	provider := &httpBankProvider{}
	config := banking.Config{Enabled: true, Environment: "sandbox", ClientID: "simulation", Secret: "simulation", ActiveKeyID: "simulation", Keys: map[string][]byte{"simulation": make([]byte, 32)}}
	service, err := banking.New(config, client.app.store, provider)
	if err != nil {
		t.Fatal(err)
	}
	client.app.banking = service
	return client, provider
}

func bankList(t *testing.T, client *testClient) budget.ConnectionList {
	t.Helper()
	response := client.request("GET", "/api/connections", nil, "", "")
	requireStatus(t, response, 200)
	var result budget.ConnectionList
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func createBankAttempt(t *testing.T, client *testClient) budget.LinkTokenResponse {
	t.Helper()
	response := client.request("POST", "/api/connections/link-token", []byte(`{"kind":"depository"}`), testOrigin, client.csrf)
	requireStatus(t, response, 200)
	var result budget.LinkTokenResponse
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func bankJSON(t *testing.T, value any) []byte {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func bankMemberClient(t *testing.T, client *testClient, household, member string) *testClient {
	t.Helper()
	token := "simulated-" + household + "-" + member
	session := store.Session{HouseholdID: household, MemberID: member, CSRF: "simulated-csrf", ExpiresAt: time.Now().Add(time.Hour)}
	if err := client.app.store.CreateSession(t.Context(), tokenHash(token), session); err != nil {
		t.Fatal(err)
	}
	return &testClient{app: client.app, cookie: &http.Cookie{Name: sessionCookieName, Value: token, Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode}, csrf: session.CSRF}
}

func seedHTTPBankAccount(t *testing.T, client *testClient, connection string) {
	t.Helper()
	cents, currency := int64(12345), "USD"
	snapshot := budget.ConnectedAccount{Name: "Simulated savings", ProviderType: "depository", Currency: &currency, CurrentBalanceCents: &cents, Active: true}
	body := bankJSON(t, snapshot)
	_, err := client.app.store.Pool.Exec(t.Context(), `INSERT INTO bank_accounts(id,household_id,connection_id,provider_id,snapshot) VALUES('http-account','demo-household',$1,'simulated-provider-account',$2)`, connection, body)
	if err != nil {
		t.Fatal(err)
	}
}
