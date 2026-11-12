package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sync"
	"testing"
	"time"

	"bankpiggy/api/internal/budget"
	"bankpiggy/api/internal/store"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

func invitationCallback(t *testing.T, client *testClient, cookie, hash string) *httptest.ResponseRecorder {
	t.Helper()
	request := invitationCallbackRequest(t, client, cookie, hash)
	response := httptest.NewRecorder()
	client.app.Handler().ServeHTTP(response, request)
	return response
}

func invitationCallbackRequest(t *testing.T, client *testClient, cookie, hash string) *http.Request {
	t.Helper()
	attempt := store.OIDCAttempt{State: "test-state", Nonce: "test-nonce", Verifier: "test-verifier", ExpiresAt: time.Now().Add(time.Minute), InvitationHash: hash}
	if err := client.app.store.CreateOIDCAttempt(t.Context(), tokenHash(cookie), attempt); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("GET", "/auth/google/callback?code=test-code&state=test-state&memberId=member-1", nil)
	request.AddCookie(&http.Cookie{Name: oidcCookieName, Value: cookie, Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	return request
}

func useCallbackSession(t *testing.T, client *testClient, response *httptest.ResponseRecorder) *testClient {
	t.Helper()
	joined := &testClient{app: client.app}
	result := response.Result()
	if err := result.Body.Close(); err != nil {
		t.Fatal(err)
	}
	for _, cookie := range result.Cookies() {
		if cookie.Name == sessionCookieName {
			joined.cookie = cookie
		}
	}
	if joined.cookie == nil {
		t.Fatal("successful callback did not issue a session")
	}
	return joined
}

func TestLinkCallbackAddsPartnerAndAllowsLaterSignInOutsideConfiguredInvites(t *testing.T) {
	client := newPrivateSoloClient(t)
	client.app.config.Invites = []string{"one@example.com"}
	configureSecondGoogleIdentity(t, client.app)
	token := createTestInvitation(t, client)
	joined := invitationCallback(t, client, "joining", tokenHash(token))
	requireStatus(t, joined, 302)
	second := useCallbackSession(t, client, joined)
	plan := readPlan(t, second)
	if len(plan.Members) != 2 || plan.Members[1].ID != "member-2" || plan.Members[1].Name != "two" || plan.Revision != 2 {
		t.Fatal("verified invitation did not append the partner to member-2")
	}
	requireStatus(t, secondGoogleCallback(t, client, "later-signin"), 302)
	client.app.config.Invites = []string{"replacement@example.com"}
	requireStatus(t, second.request("GET", "/api/household", nil, "", ""), 200)
	requireStatus(t, client.request("GET", "/api/household", nil, "", ""), 401)
	request := httptest.NewRequest("GET", "/auth/google/callback?code=test-code&state=test-state", nil)
	request.AddCookie(&http.Cookie{Name: oidcCookieName, Value: "joining", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	replay := httptest.NewRecorder()
	client.app.Handler().ServeHTTP(replay, request)
	requireStatus(t, replay, 400)
}

func TestPreparedOIDCAttemptPinsOnlyItsInvitationHash(t *testing.T) {
	client := newPrivateSoloClient(t)
	client.app.verifier = &oidc.IDTokenVerifier{}
	client.app.oauth = oauth2.Config{ClientID: "test-client", Endpoint: oauth2.Endpoint{AuthURL: "https://accounts.example/authorize"}}
	token := createTestInvitation(t, client)
	prepared := client.request("POST", "/api/household/invitation/prepare", invitationBody(t, token), testOrigin, "")
	requireStatus(t, prepared, 200)
	request := httptest.NewRequest("GET", "/auth/google?join=1&memberId=member-1&invite=attacker-value", nil)
	request.AddCookie(responseCookies(t, prepared)[0])
	started := httptest.NewRecorder()
	client.app.Handler().ServeHTTP(started, request)
	requireStatus(t, started, 302)
	target, err := url.Parse(started.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	callback := httptest.NewRequest("GET", "/auth/google/callback?state="+target.Query().Get("state"), nil)
	callback.AddCookie(responseCookies(t, started)[0])
	attempt, err := client.app.readOIDCAttempt(callback)
	if err != nil || attempt.InvitationHash != tokenHash(token) {
		t.Fatalf("OIDC attempt did not use its prepared invitation: %v", err)
	}
	if target.Query().Has("invite") || target.Query().Has("memberId") {
		t.Fatal("OAuth redirect exposed invitation or member inputs")
	}
}

func TestCallbackRechecksRevocationRotationAndExpiry(t *testing.T) {
	for _, invalidation := range []string{"revoke", "rotate", "expire"} {
		t.Run(invalidation, func(t *testing.T) {
			client := newPrivateSoloClient(t)
			configureSecondGoogleIdentity(t, client.app)
			token := createTestInvitation(t, client)
			switch invalidation {
			case "revoke":
				requireStatus(t, client.request("DELETE", "/api/household/invitation", []byte(`{}`), testOrigin, client.csrf), 204)
			case "rotate":
				createTestInvitation(t, client)
			case "expire":
				if _, err := client.app.store.Pool.Exec(t.Context(), "UPDATE household_invitations SET expires_at=now()-interval '1 second'"); err != nil {
					t.Fatal(err)
				}
			}
			requireStatus(t, invitationCallback(t, client, "invalidated", tokenHash(token)), 410)
			requireSecondIdentityCounts(t, client, 0)
			if plan := readPlan(t, client); len(plan.Members) != 1 || plan.Revision != 1 {
				t.Fatal("invalidated invitation changed the saved plan")
			}
		})
	}
}

func TestUnpreparedGoogleAccountCannotClaimPartnerByQuery(t *testing.T) {
	client := newPrivateSoloClient(t)
	client.app.config.Invites = []string{"one@example.com"}
	configureSecondGoogleIdentity(t, client.app)
	createTestInvitation(t, client)
	response := invitationCallback(t, client, "not-prepared", "")
	requireStatus(t, response, 403)
	var failure budget.ErrorResponse
	if err := json.Unmarshal(response.Body.Bytes(), &failure); err != nil {
		t.Fatal(err)
	}
	if failure.Code != "invite_required" {
		t.Fatalf("unexpected identity rejection: %+v", failure)
	}
	requireSecondIdentityCounts(t, client, 0)
}

func TestConcurrentGoogleCallbacksConsumeInvitationOnce(t *testing.T) {
	client := newPrivateSoloClient(t)
	configureSecondGoogleIdentity(t, client.app)
	token := createTestInvitation(t, client)
	requests := []*http.Request{
		invitationCallbackRequest(t, client, "first-callback", tokenHash(token)),
		invitationCallbackRequest(t, client, "second-callback", tokenHash(token)),
	}
	start := make(chan struct{})
	results := make(chan int, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	for _, request := range requests {
		go func() {
			ready.Done()
			<-start
			response := httptest.NewRecorder()
			client.app.Handler().ServeHTTP(response, request)
			results <- response.Code
		}()
	}
	ready.Wait()
	close(start)
	counts := map[int]int{}
	for index := 0; index < 2; index++ {
		counts[<-results]++
	}
	if counts[302] != 1 || counts[410] != 1 {
		t.Fatalf("unexpected concurrent callback results: %v", counts)
	}
	requireSecondIdentityCounts(t, client, 1)
}

func TestPartnerAppendReportsPlanConflictAndRetainsInvitationForRetry(t *testing.T) {
	client := newPrivateSoloClient(t)
	configureSecondGoogleIdentity(t, client.app)
	plan := readPlan(t, client)
	plan.Expenses = []budget.Expense{{ID: "salary-bill", Name: "Shared bill", Category: "Bills", MonthlyAmountCents: 10000, OwnerID: "shared", Split: budget.Split{Method: "salary"}}}
	requireStatus(t, client.request("PUT", "/api/household", planJSON(t, plan), testOrigin, client.csrf), 200)
	before := readPlan(t, client)
	token := createTestInvitation(t, client)
	rejected := invitationCallback(t, client, "invalid-plan", tokenHash(token))
	requireStatus(t, rejected, 409)
	var failure budget.ErrorResponse
	if err := json.Unmarshal(rejected.Body.Bytes(), &failure); err != nil {
		t.Fatal(err)
	}
	if failure.Code != "invitation_plan_conflict" || !reflect.DeepEqual(readPlan(t, client), before) {
		t.Fatal("partner append did not preserve the original plan with an actionable conflict")
	}
	status, err := client.app.store.InvitationStatus(t.Context(), "private-household")
	if err != nil || status.Linked || status.ExpiresAt == nil {
		t.Fatalf("failed append consumed the pending invitation: %+v %v", status, err)
	}
	requireSecondIdentityCounts(t, client, 0)
	before.Expenses[0].Split.Method = "equal"
	requireStatus(t, client.request("PUT", "/api/household", planJSON(t, before), testOrigin, client.csrf), 200)
	requireStatus(t, invitationCallback(t, client, "corrected-plan", tokenHash(token)), 302)
	requireSecondIdentityCounts(t, client, 1)
}
