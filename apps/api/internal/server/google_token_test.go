package server

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bankpiggy/api/internal/store"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-jose/go-jose/v4"
	"golang.org/x/oauth2"
)

func signedIdentity(t *testing.T, key *rsa.PrivateKey, claims map[string]any) string {
	t.Helper()
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, nil)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	signed, err := signer.Sign(data)
	if err != nil {
		t.Fatal(err)
	}
	token, err := signed.CompactSerialize()
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func localIdentityProvider(t *testing.T, identity string, key *rsa.PrivateKey) *httptest.Server {
	t.Helper()
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/keys" {
			keys := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &key.PublicKey, Algorithm: string(jose.RS256), Use: "sig"}}}
			if err := json.NewEncoder(w).Encode(keys); err != nil {
				t.Error(err)
			}
			return
		}
		if err := r.ParseForm(); err != nil {
			t.Error(err)
			w.WriteHeader(400)
			return
		}
		if r.Form.Get("code_verifier") != "test-verifier" {
			t.Error("token exchange omitted the PKCE verifier")
			w.WriteHeader(400)
			return
		}
		response := map[string]any{"access_token": "test-access", "token_type": "Bearer", "id_token": identity}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(provider.Close)
	return provider
}

func checkGoogleClaims(t *testing.T, key *rsa.PrivateKey, claims map[string]any, wantError bool) {
	t.Helper()
	provider := localIdentityProvider(t, signedIdentity(t, key, claims), key)
	verifier := oidc.NewVerifier("https://accounts.google.com", oidc.NewRemoteKeySet(t.Context(), provider.URL+"/keys"), &oidc.Config{ClientID: "test-client"})
	app := &Server{
		verifier: verifier,
		oauth:    oauth2.Config{ClientID: "test-client", ClientSecret: "test-secret", Endpoint: oauth2.Endpoint{TokenURL: provider.URL + "/token"}},
	}
	request := httptest.NewRequest("GET", "/auth/google/callback?code=test-code", nil)
	request = request.WithContext(t.Context())
	_, _, err := app.verifiedGoogleClaims(request, store.OIDCAttempt{Nonce: "test-nonce", Verifier: "test-verifier"})
	if (err != nil) != wantError {
		t.Fatalf("verification error=%v, want rejection=%v", err, wantError)
	}
}

func TestGoogleVerifierRejectsInvalidIdentityClaims(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, field string
		value       any
		wantError   bool
	}{
		{"verified invited identity", "email", "one@example.com", false},
		{"wrong audience", "aud", "other-client", true},
		{"wrong issuer", "iss", "https://attacker.example", true},
		{"expired identity", "exp", time.Now().Add(-time.Hour).Unix(), true},
		{"wrong nonce", "nonce", "other-nonce", true},
		{"unverified email", "email_verified", false, true},
		{"missing subject", "sub", "", true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			claims := map[string]any{
				"iss": "https://accounts.google.com", "sub": "test-subject", "aud": "test-client",
				"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(),
				"nonce": "test-nonce", "email": "one@example.com", "email_verified": true,
			}
			claims[test.field] = test.value
			checkGoogleClaims(t, key, claims, test.wantError)
		})
	}
}
