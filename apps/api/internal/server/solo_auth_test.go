package server

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bankpiggy/api/internal/budget"
	"bankpiggy/api/internal/store"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

func setTestMemberSession(t *testing.T, client *testClient, memberID string) {
	t.Helper()
	response := httptest.NewRecorder()
	identity, err := client.app.issueSession(response, httptest.NewRequest("GET", "/", nil), memberID)
	if err != nil {
		t.Fatal(err)
	}
	result := response.Result()
	if err := result.Body.Close(); err != nil {
		t.Fatal(err)
	}
	client.cookie = result.Cookies()[0]
	client.csrf = identity.CSRF
}

func newPrivateSoloClient(t *testing.T) *testClient {
	t.Helper()
	client := newSoloTestClient(t)
	client.app.config.Mode = "private"
	client.app.config.Invites = []string{"one@example.com", "two@example.com"}
	if err := client.app.store.Seed(t.Context(), "private-household", budget.Blank("One")); err != nil {
		t.Fatal(err)
	}
	if err := client.app.store.BindIdentity(t.Context(), "private-household", "member-1", "one@example.com", "first-subject"); err != nil {
		t.Fatal(err)
	}
	setTestMemberSession(t, client, "member-1")
	return client
}

func configureSecondGoogleIdentity(t *testing.T, app *Server) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	claims := map[string]any{
		"iss": "https://accounts.google.com", "sub": "second-subject", "aud": "test-client",
		"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(),
		"nonce": "test-nonce", "email": "two@example.com", "email_verified": true,
	}
	provider := localIdentityProvider(t, signedIdentity(t, key, claims), key)
	app.verifier = oidc.NewVerifier("https://accounts.google.com", oidc.NewRemoteKeySet(t.Context(), provider.URL+"/keys"), &oidc.Config{ClientID: "test-client"})
	app.oauth = oauth2.Config{ClientID: "test-client", ClientSecret: "test-secret", Endpoint: oauth2.Endpoint{TokenURL: provider.URL + "/token"}}
}

func secondGoogleCallback(t *testing.T, client *testClient, token string) *httptest.ResponseRecorder {
	t.Helper()
	attempt := store.OIDCAttempt{State: "test-state", Nonce: "test-nonce", Verifier: "test-verifier", ExpiresAt: time.Now().Add(time.Minute)}
	if err := client.app.store.CreateOIDCAttempt(t.Context(), tokenHash(token), attempt); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("GET", "/auth/google/callback?code=test-code&state=test-state", nil)
	request.AddCookie(&http.Cookie{Name: oidcCookieName, Value: token, Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	response := httptest.NewRecorder()
	client.app.Handler().ServeHTTP(response, request)
	return response
}

func requireSecondIdentityCounts(t *testing.T, client *testClient, want int) {
	t.Helper()
	queries := []string{
		"SELECT count(*) FROM household_users WHERE household_id='private-household' AND member_id='member-2'",
		"SELECT count(*) FROM sessions WHERE household_id='private-household' AND member_id='member-2'",
	}
	for _, query := range queries {
		var count int
		if err := client.app.store.Pool.QueryRow(t.Context(), query).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != want {
			t.Fatalf("second-member row count=%d, want %d", count, want)
		}
	}
}

func TestSecondInviteRequiresSavedPersonBeforeIdentityOrSession(t *testing.T) {
	client := newPrivateSoloClient(t)
	configureSecondGoogleIdentity(t, client.app)
	rejected := secondGoogleCallback(t, client, "before-person")
	requireStatus(t, rejected, 403)
	var failure budget.ErrorResponse
	if err := json.Unmarshal(rejected.Body.Bytes(), &failure); err != nil {
		t.Fatal(err)
	}
	if failure.Code != "member_not_added" {
		t.Fatalf("unexpected second-invite rejection: %+v", failure)
	}
	requireSecondIdentityCounts(t, client, 0)
	plan := readPlan(t, client)
	plan.Members = append(plan.Members, budget.Blank("One", "Two").Members[1])
	requireStatus(t, client.request("PUT", "/api/household", planJSON(t, plan), testOrigin, client.csrf), 200)
	accepted := secondGoogleCallback(t, client, "after-person")
	requireStatus(t, accepted, 302)
	requireSecondIdentityCounts(t, client, 1)
	result := accepted.Result()
	if err := result.Body.Close(); err != nil {
		t.Fatal(err)
	}
	second := &testClient{app: client.app}
	for _, cookie := range result.Cookies() {
		if cookie.Name == sessionCookieName {
			second.cookie = cookie
		}
	}
	var session budget.Session
	if err := json.Unmarshal(second.request("GET", "/api/session", nil, "", "").Body.Bytes(), &session); err != nil {
		t.Fatal(err)
	}
	if !session.Authenticated || session.Viewer == nil || session.Viewer.ID != "member-2" {
		t.Fatal("saved second person could not use the issued session")
	}
	requireStatus(t, second.request("GET", "/api/household", nil, "", ""), 200)
}

func TestAbsentPersonSessionCannotReadOrChangePlan(t *testing.T) {
	client := newPrivateSoloClient(t)
	if err := client.app.store.BindIdentity(t.Context(), "private-household", "member-2", "two@example.com", "second-subject"); err != nil {
		t.Fatal(err)
	}
	plan := readPlan(t, client)
	setTestMemberSession(t, client, "member-2")
	response := client.request("GET", "/api/session", nil, "", "")
	requireStatus(t, response, 200)
	var session budget.Session
	if err := json.Unmarshal(response.Body.Bytes(), &session); err != nil {
		t.Fatal(err)
	}
	if session.Authenticated || session.Viewer != nil || session.CSRFToken != nil {
		t.Fatal("absent person received an authenticated session response")
	}
	requireStatus(t, client.request("GET", "/api/household", nil, "", ""), 401)
	requireStatus(t, client.request("PUT", "/api/household", planJSON(t, plan), testOrigin, client.csrf), 401)
	requireStatus(t, client.request("POST", "/api/logout", nil, testOrigin, client.csrf), 401)
	setTestMemberSession(t, client, "member-1")
	if saved := readPlan(t, client); saved.Revision != plan.Revision || len(saved.Members) != 1 {
		t.Fatal("absent person changed the saved plan")
	}
}
