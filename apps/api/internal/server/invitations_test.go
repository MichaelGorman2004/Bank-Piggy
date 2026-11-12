package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"bankpiggy/api/internal/budget"
)

func createTestInvitation(t *testing.T, client *testClient) string {
	t.Helper()
	response := client.request("POST", "/api/household/invitation", []byte(`{}`), testOrigin, client.csrf)
	requireStatus(t, response, 201)
	var created budget.InvitationCreated
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(created.Url)
	if err != nil {
		t.Fatal(err)
	}
	token := strings.TrimPrefix(parsed.Fragment, "invite=")
	assertCreatedInvitation(t, client, created, parsed, token)
	return token
}

func assertCreatedInvitation(t *testing.T, client *testClient, created budget.InvitationCreated, parsed *url.URL, token string) {
	t.Helper()
	assertInvitationURL(t, parsed, token)
	expiresAt, err := time.Parse(time.RFC3339, created.ExpiresAt)
	if err != nil || time.Until(expiresAt) < invitationLifetime-time.Minute || time.Until(expiresAt) > invitationLifetime {
		t.Fatal("invitation does not expire in seven days")
	}
	var hash string
	if err := client.app.store.Pool.QueryRow(t.Context(), "SELECT token_hash FROM household_invitations WHERE household_id='private-household'").Scan(&hash); err != nil {
		t.Fatal(err)
	}
	if hash != tokenHash(token) {
		t.Fatal("invitation was not stored as a hash")
	}
}

func assertInvitationURL(t *testing.T, parsed *url.URL, token string) {
	t.Helper()
	if parsed.Scheme+"://"+parsed.Host != testOrigin || parsed.Path != "/join" || parsed.RawQuery != "" || !validInvitationToken(token) {
		t.Fatal("invitation did not use an origin-bound URL fragment")
	}
}

func invitationBody(t *testing.T, token string) []byte {
	t.Helper()
	data, err := json.Marshal(budget.InvitationPrepareRequest{Token: token})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestInvitationCreatePrepareRotateAndRevoke(t *testing.T) {
	client := newPrivateSoloClient(t)
	token := createTestInvitation(t, client)
	status := client.request("GET", "/api/household/invitation", nil, "", "")
	requireStatus(t, status, 200)
	assertPendingInvitationStatus(t, status, token)
	anonymous := &testClient{app: client.app}
	prepared := anonymous.request("POST", "/api/household/invitation/prepare", invitationBody(t, token), testOrigin, "")
	requireStatus(t, prepared, 200)
	assertInvitationPreview(t, prepared)
	createTestInvitation(t, client)
	requireStatus(t, anonymous.request("POST", "/api/household/invitation/prepare", invitationBody(t, token), testOrigin, ""), 410)
	requireStatus(t, client.request("DELETE", "/api/household/invitation", []byte(`{}`), testOrigin, client.csrf), 204)
	if status, err := client.app.store.InvitationStatus(t.Context(), "private-household"); err != nil || status.ExpiresAt != nil {
		t.Fatalf("revoke left a pending link: %v", err)
	}
}

func assertPendingInvitationStatus(t *testing.T, status *httptest.ResponseRecorder, token string) {
	t.Helper()
	var pending budget.InvitationStatus
	if err := json.Unmarshal(status.Body.Bytes(), &pending); err != nil {
		t.Fatal(err)
	}
	if !pending.CanInvite || !pending.Pending || pending.Linked || strings.Contains(status.Body.String(), token) {
		t.Fatal("status leaked the token or omitted pending state")
	}
}

func assertInvitationPreview(t *testing.T, prepared *httptest.ResponseRecorder) {
	t.Helper()
	var preview budget.InvitationPreview
	if err := json.Unmarshal(prepared.Body.Bytes(), &preview); err != nil {
		t.Fatal(err)
	}
	if preview.HouseholdName != "One’s household" || preview.MemberName != "New person" || strings.Contains(prepared.Body.String(), "Cents") {
		t.Fatal("preview did not restrict itself to invitation names and expiry")
	}
	assertPreparationCookie(t, responseCookies(t, prepared))
	if prepared.Header().Get("Cache-Control") != "no-store" || prepared.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Fatal("invitation response lacks privacy headers")
	}
}

func responseCookies(t *testing.T, response *httptest.ResponseRecorder) []*http.Cookie {
	t.Helper()
	result := response.Result()
	if err := result.Body.Close(); err != nil {
		t.Fatal(err)
	}
	return result.Cookies()
}

func assertPreparationCookie(t *testing.T, cookies []*http.Cookie) {
	t.Helper()
	if len(cookies) != 1 || cookies[0].Name != invitationCookieName || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode || cookies[0].MaxAge != 600 {
		t.Fatal("preparation cookie lacks transient, HttpOnly protection")
	}
}

func TestInvitationMutationRejectsMissingCSRFAndNonExactOrigins(t *testing.T) {
	client := newPrivateSoloClient(t)
	for _, method := range []string{"POST", "DELETE"} {
		requireStatus(t, client.request(method, "/api/household/invitation", []byte(`{}`), testOrigin, ""), 403)
		for _, origin := range []string{"", "https://attacker.example", testOrigin + "/"} {
			requireStatus(t, client.request(method, "/api/household/invitation", []byte(`{}`), origin, client.csrf), 403)
		}
	}
	token := createTestInvitation(t, client)
	for _, origin := range []string{"", "https://attacker.example", testOrigin + "/"} {
		requireStatus(t, client.request("POST", "/api/household/invitation/prepare", invitationBody(t, token), origin, ""), 403)
	}
}

func TestInvitationEndpointsRejectInvalidJSONAndTokens(t *testing.T) {
	client := newPrivateSoloClient(t)
	for _, body := range []string{"", "null", `[]`, `{} {}`, `{"memberId":"member-2"}`, strings.Repeat(" ", smallRequestLimit+1)} {
		for _, method := range []string{"POST", "DELETE"} {
			requireStatus(t, client.request(method, "/api/household/invitation", []byte(body), testOrigin, client.csrf), 400)
		}
	}
	for _, body := range []string{"", "null", `{}`, `{"token":null}`, `{"token":"invalid"}`, `{"token":"invalid","memberId":"member-1"}`, strings.Repeat(" ", smallRequestLimit+1)} {
		requireStatus(t, client.request("POST", "/api/household/invitation/prepare", []byte(body), testOrigin, ""), 400)
	}
}

func TestOnlyFirstMemberCanManageInvitationsAndDemoCannotInvite(t *testing.T) {
	client := newPrivateSoloClient(t)
	plan := readPlan(t, client)
	plan.Members = append(plan.Members, budget.Blank("One", "Two").Members[1])
	requireStatus(t, client.request("PUT", "/api/household", planJSON(t, plan), testOrigin, client.csrf), 200)
	if err := client.app.store.BindIdentity(t.Context(), "private-household", "member-2", "two@example.com", "second-subject"); err != nil {
		t.Fatal(err)
	}
	requireStatus(t, client.request("POST", "/api/household/invitation", []byte(`{}`), testOrigin, client.csrf), 409)
	setTestMemberSession(t, client, "member-2")
	for _, method := range []string{"POST", "DELETE"} {
		requireStatus(t, client.request(method, "/api/household/invitation", []byte(`{}`), testOrigin, client.csrf), 403)
	}
	demo := newTestClient(t)
	demo.signIn(t)
	requireStatus(t, demo.request("POST", "/api/household/invitation", []byte(`{}`), testOrigin, demo.csrf), 404)
	requireStatus(t, demo.request("POST", "/api/household/invitation/prepare", []byte(`{}`), testOrigin, ""), 404)
}
