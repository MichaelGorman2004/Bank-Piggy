package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

func intentClient(t *testing.T) *testClient {
	t.Helper()
	client := newPrivateSoloClient(t)
	client.app.verifier = &oidc.IDTokenVerifier{}
	client.app.oauth = oauth2.Config{ClientID: "test-client", Endpoint: oauth2.Endpoint{AuthURL: "https://accounts.example/authorize"}}
	return client
}

func startWithInvitation(t *testing.T, client *testClient, path, token string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest("GET", path, nil)
	if token != "" {
		request.AddCookie(&http.Cookie{Name: invitationCookieName, Value: token, Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	}
	response := httptest.NewRecorder()
	client.app.Handler().ServeHTTP(response, request)
	return response
}

func TestNormalGoogleSigninIgnoresAndClearsAnyPreparedInvitation(t *testing.T) {
	client := intentClient(t)
	token := createTestInvitation(t, client)
	for _, value := range []string{token, "invalid-cookie"} {
		started := startWithInvitation(t, client, "/auth/google", value)
		requireStatus(t, started, 302)
		cookies := responseCookies(t, started)
		assertClearedInvitation(t, cookies)
		target, err := url.Parse(started.Header().Get("Location"))
		if err != nil {
			t.Fatal(err)
		}
		callback := httptest.NewRequest("GET", "/auth/google/callback?state="+target.Query().Get("state"), nil)
		callback.AddCookie(cookies[0])
		attempt, err := client.app.readOIDCAttempt(callback)
		if err != nil || attempt.InvitationHash != "" {
			t.Fatalf("normal sign-in adopted a prepared invitation: %v", err)
		}
	}
}

func TestJoinRequiresPreparedCookieAndClearsAnInvalidCookie(t *testing.T) {
	client := intentClient(t)
	token := createTestInvitation(t, client)
	requireStatus(t, client.request("DELETE", "/api/household/invitation", []byte(`{}`), testOrigin, client.csrf), 204)
	for _, value := range []string{"", "invalid-cookie", token} {
		rejected := startWithInvitation(t, client, "/auth/google?join=1&memberId=member-2&invite="+token, value)
		requireStatus(t, rejected, 410)
		assertClearedInvitation(t, responseCookies(t, rejected))
		requireStatus(t, startWithInvitation(t, client, "/auth/google", value), 302)
	}
}

func assertClearedInvitation(t *testing.T, cookies []*http.Cookie) {
	t.Helper()
	for _, cookie := range cookies {
		if cookie.Name == invitationCookieName && cookie.MaxAge < 0 && cookie.Value == "" {
			return
		}
	}
	t.Fatal("pending invitation cookie was not cleared")
}

func TestPreparedInvitationCookieRequiresSecureTransportOutsideLoopback(t *testing.T) {
	client := newPrivateSoloClient(t)
	token := createTestInvitation(t, client)
	client.app.config.BaseURL = "https://bankpiggy.example"
	prepared := client.request("POST", "/api/household/invitation/prepare", invitationBody(t, token), client.app.config.BaseURL, "")
	requireStatus(t, prepared, 200)
	cookies := responseCookies(t, prepared)
	assertPreparationCookie(t, cookies)
	if !cookies[0].Secure {
		t.Fatal("prepared invitation cookie permitted insecure transport")
	}
}
