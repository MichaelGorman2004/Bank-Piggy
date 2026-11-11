package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bankpiggy/api/internal/budget"
	"bankpiggy/api/internal/testdb"
)

const testOrigin = "http://127.0.0.1:5173"

type testClient struct {
	app    *Server
	cookie *http.Cookie
	csrf   string
}

func newTestClient(t *testing.T) *testClient {
	t.Helper()
	database := testdb.Open(t)
	config := Config{Mode: "demo", ListenAddr: "127.0.0.1:8080", BaseURL: testOrigin, DatabaseURL: "configured", WebDistDir: t.TempDir()}
	app, err := New(t.Context(), config, database)
	if err != nil {
		t.Fatal(err)
	}
	return &testClient{app: app}
}

func (client *testClient) request(method, path string, body []byte, origin, csrf string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	request.Header.Set("Origin", origin)
	request.Header.Set("X-CSRF-Token", csrf)
	request.Header.Set("Content-Type", "application/json")
	if client.cookie != nil {
		request.AddCookie(client.cookie)
	}
	response := httptest.NewRecorder()
	client.app.Handler().ServeHTTP(response, request)
	return response
}

func (client *testClient) signIn(t *testing.T) {
	t.Helper()
	response := client.request("POST", "/api/demo/session", nil, testOrigin, "")
	requireStatus(t, response, 200)
	var session budget.Session
	if err := json.Unmarshal(response.Body.Bytes(), &session); err != nil {
		t.Fatal(err)
	}
	if !session.Authenticated || session.CSRFToken == nil {
		t.Fatal("demo did not issue an authenticated session")
	}
	result := response.Result()
	defer func() {
		if err := result.Body.Close(); err != nil {
			t.Error(err)
		}
	}()
	client.cookie = result.Cookies()[0]
	client.csrf = *session.CSRFToken
	if !client.cookie.HttpOnly || client.cookie.SameSite != http.SameSiteLaxMode {
		t.Fatal("session cookie lacks protection")
	}
}

func requireStatus(t *testing.T, response *httptest.ResponseRecorder, want int) {
	t.Helper()
	if response.Code != want {
		t.Fatalf("got status %d, want %d: %s", response.Code, want, response.Body.String())
	}
}

func readPlan(t *testing.T, client *testClient) budget.Household {
	t.Helper()
	response := client.request("GET", "/api/household", nil, "", "")
	requireStatus(t, response, 200)
	var data budget.HouseholdResponse
	if err := json.Unmarshal(response.Body.Bytes(), &data); err != nil {
		t.Fatal(err)
	}
	return data.Household
}

func planJSON(t *testing.T, plan budget.Household) []byte {
	t.Helper()
	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestDemoAuthenticationAndCSRF(t *testing.T) {
	client := newTestClient(t)
	requireStatus(t, client.request("GET", "/api/household", nil, "", ""), 401)
	requireStatus(t, client.request("POST", "/api/demo/session", nil, "http://evil.example", ""), 403)
	client.signIn(t)
	plan := readPlan(t, client)
	body := planJSON(t, plan)
	requireStatus(t, client.request("PUT", "/api/household", body, testOrigin, ""), 403)
	requireStatus(t, client.request("PUT", "/api/household", body, "http://evil.example", client.csrf), 403)
	requireStatus(t, client.request("POST", "/api/logout", nil, testOrigin, client.csrf), 204)
	requireStatus(t, client.request("GET", "/api/household", nil, "", ""), 401)
}

func TestPlanPersistenceConflictAndOwnerChange(t *testing.T) {
	client := newTestClient(t)
	client.signIn(t)
	plan := readPlan(t, client)
	plan.Expenses[0].OwnerID = "member-2"
	plan.Members[0].Accounts[0].BalanceCents = 12345
	body := planJSON(t, plan)
	requireStatus(t, client.request("PUT", "/api/household", body, testOrigin, client.csrf), 200)
	requireStatus(t, client.request("PUT", "/api/household", body, testOrigin, client.csrf), 409)
	saved := readPlan(t, client)
	if saved.Revision != plan.Revision+1 || saved.Expenses[0].OwnerID != "member-2" || saved.Members[0].Accounts[0].BalanceCents != 12345 {
		t.Fatal("plan did not persist exactly")
	}
	if err := client.app.store.Seed(t.Context(), "demo-household", budget.Sample()); err != nil {
		t.Fatal(err)
	}
	if readPlan(t, client).Revision != saved.Revision {
		t.Fatal("demo seed replaced the saved plan")
	}
}

func TestPlanRejectsForeignOwnerAndMalformedJSON(t *testing.T) {
	client := newTestClient(t)
	client.signIn(t)
	plan := readPlan(t, client)
	plan.Expenses[0].OwnerID = "foreign-household"
	requireStatus(t, client.request("PUT", "/api/household", planJSON(t, plan), testOrigin, client.csrf), 400)
	for _, body := range []string{`{}`, `{"householdId":"someone-else"}`, `null`, `{} {}`, `{"revision":1.2}`} {
		requireStatus(t, client.request("PUT", "/api/household", []byte(body), testOrigin, client.csrf), 400)
	}
}
