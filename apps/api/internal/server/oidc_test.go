package server

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"bankpiggy/api/internal/store"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

func TestOIDCAttemptBindsStateNoncePKCEAndPreventsReplay(t *testing.T) {
	client := newTestClient(t)
	client.app.verifier = &oidc.IDTokenVerifier{}
	client.app.oauth = oauth2.Config{ClientID: "test-client", Endpoint: oauth2.Endpoint{AuthURL: "https://accounts.example/authorize"}}
	response := client.request("GET", "/auth/google", nil, "", "")
	requireStatus(t, response, 302)
	target, err := url.Parse(response.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	result := response.Result()
	if err := result.Body.Close(); err != nil {
		t.Fatal(err)
	}
	cookie := result.Cookies()[0]
	request := httptest.NewRequest("GET", "/auth/google/callback?state="+target.Query().Get("state"), nil)
	request.AddCookie(cookie)
	attempt, err := client.app.readOIDCAttempt(request)
	if err != nil {
		t.Fatal(err)
	}
	challenge := sha256.Sum256([]byte(attempt.Verifier))
	if target.Query().Get("code_challenge") != base64.RawURLEncoding.EncodeToString(challenge[:]) || target.Query().Get("code_challenge_method") != "S256" {
		t.Fatal("PKCE was not bound to the persisted verifier")
	}
	if target.Query().Get("nonce") == "" || target.Query().Get("nonce") != attempt.Nonce {
		t.Fatal("nonce was not bound to the sign-in attempt")
	}
	if _, err := client.app.readOIDCAttempt(request); err == nil {
		t.Fatal("one-use OAuth attempt replay succeeded")
	}
}

func TestOIDCRejectsWrongStateAndUninvitedIdentity(t *testing.T) {
	client := newTestClient(t)
	attempt := store.OIDCAttempt{State: "expected", Nonce: "nonce", Verifier: "verifier", ExpiresAt: time.Now().Add(time.Minute)}
	if err := client.app.store.CreateOIDCAttempt(t.Context(), tokenHash("cookie"), attempt); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("GET", "/auth/google/callback?state=wrong", nil)
	request.AddCookie(&http.Cookie{Name: oidcCookieName, Value: "cookie", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	if _, err := client.app.readOIDCAttempt(request); err == nil {
		t.Fatal("wrong OAuth state accepted")
	}
	client.app.config.Invites = []string{"one@example.com", "two@example.com"}
	if client.app.invitedMember("uninvited@example.com") != "" {
		t.Fatal("uninvited account was accepted")
	}
	if client.app.invitedMember("ONE@example.com") != "member-1" {
		t.Fatal("verified invitation was not matched")
	}
}
