package banking

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
)

func webhookSigner(t *testing.T) (jose.Signer, []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.ES256, Key: key}, (&jose.SignerOptions{}).WithHeader("kid", "simulated-key"))
	if err != nil {
		t.Fatal(err)
	}
	public, err := json.Marshal(jose.JSONWebKey{Key: &key.PublicKey, Algorithm: "ES256", Use: "sig", KeyID: "simulated-key"})
	if err != nil {
		t.Fatal(err)
	}
	return signer, public
}
func signedWebhook(t *testing.T, signer jose.Signer, body []byte, issued time.Time) string {
	t.Helper()
	digest := sha256.Sum256(body)
	payload, err := json.Marshal(map[string]any{"iat": issued.Unix(), "request_body_sha256": hex.EncodeToString(digest[:])})
	if err != nil {
		t.Fatal(err)
	}
	signed, err := signer.Sign(payload)
	if err != nil {
		t.Fatal(err)
	}
	compact, err := signed.CompactSerialize()
	if err != nil {
		t.Fatal(err)
	}
	return compact
}
func TestWebhookSignatureFreshnessHashAndKeyExpiry(t *testing.T) {
	s, p, _ := setupService(t)
	signer, key := webhookSigner(t)
	p.key = key
	body := []byte(`{"environment":"sandbox","item_id":"item-simulated","webhook_type":"ITEM","webhook_code":"ERROR"}`)
	valid := signedWebhook(t, signer, body, time.Now())
	if err := s.VerifyWebhook(t.Context(), valid, body); err != nil {
		t.Fatal(err)
	}
	invalid := []string{valid + " ", signedWebhook(t, signer, body, time.Now().Add(-6*time.Minute)), signedWebhook(t, signer, body, time.Now().Add(time.Minute)), "invalid.jwt.signature"}
	for _, signature := range invalid {
		if err := s.VerifyWebhook(t.Context(), signature, body); !errors.Is(err, ErrInvalidWebhook) {
			t.Fatal("invalid webhook accepted", err)
		}
	}
	if err := s.VerifyWebhook(t.Context(), valid, append(body, ' ')); !errors.Is(err, ErrInvalidWebhook) {
		t.Fatal("body tampering accepted")
	}
	now := time.Now()
	expires := now.Add(time.Minute).Unix()
	var jwk map[string]any
	if err := json.Unmarshal(key, &jwk); err != nil {
		t.Fatal(err)
	}
	jwk["expired_at"] = expires
	expiring, err := json.Marshal(jwk)
	if err != nil {
		t.Fatal(err)
	}
	deadline, err := verificationExpiry(expiring, now)
	if err != nil || deadline.Unix() != expires {
		t.Fatal("key cache outlived provider expiry")
	}
	if _, err := decodeKey(expiring, "simulated-key", now.Add(2*time.Minute)); !errors.Is(err, ErrInvalidWebhook) {
		t.Fatal("expired key accepted")
	}
}

func TestVerifiedWebhookReplayAndLifecycle(t *testing.T) {
	s, p, _, _ := connectedService(t)
	signer, key := webhookSigner(t)
	p.key = key
	body := []byte(`{"environment":"sandbox","item_id":"item-simulated","webhook_type":"ITEM","webhook_code":"USER_PERMISSION_REVOKED"}`)
	signature := signedWebhook(t, signer, body, time.Now())
	if err := s.VerifyWebhook(t.Context(), signature, body); err != nil {
		t.Fatal(err)
	}
	if err := s.ApplyWebhook(t.Context(), signature, body); err != nil {
		t.Fatal(err)
	}
	malleated := malleatedSignature(t, signature)
	if err := s.VerifyWebhook(t.Context(), malleated, body); err != nil {
		t.Fatal("valid alternate ECDSA representation rejected unexpectedly", err)
	}
	if err := s.ApplyWebhook(t.Context(), malleated, body); err != nil {
		t.Fatal(err)
	}
	var receipts int
	if err := s.db.Pool.QueryRow(t.Context(), `SELECT count(*) FROM bank_webhook_receipts`).Scan(&receipts); err != nil || receipts != 1 {
		t.Fatal("duplicate webhook received twice")
	}
}

func TestWebhookRevocationAndRepairKeepsExistingItem(t *testing.T) {
	s, p, identity, connection := connectedService(t)
	signer, key := webhookSigner(t)
	p.key = key
	body := []byte(`{"environment":"sandbox","item_id":"item-simulated","webhook_type":"ITEM","webhook_code":"USER_PERMISSION_REVOKED"}`)
	if err := s.ApplyWebhook(t.Context(), signedWebhook(t, signer, body, time.Now()), body); err != nil {
		t.Fatal(err)
	}
	list, err := s.List(t.Context(), identity)
	if err != nil || list.Connections[0].State != "reconnect" || list.Connections[0].Accounts[0].Active || p.removes != 0 {
		t.Fatal("revoked Item did not preserve update-mode recovery")
	}
	repaired := []byte(`{"environment":"sandbox","item_id":"item-simulated","webhook_type":"ITEM","webhook_code":"LOGIN_REPAIRED"}`)
	if err := s.ApplyWebhook(t.Context(), signedWebhook(t, signer, repaired, time.Now()), repaired); err != nil {
		t.Fatal(err)
	}
	if err := s.processOne(t.Context(), connection.ID); err != nil {
		t.Fatal(err)
	}
	list, err = s.List(t.Context(), identity)
	if err != nil || list.Connections[0].State != "connected" {
		t.Fatal("repaired login did not recover")
	}
}

func malleatedSignature(t *testing.T, signature string) string {
	t.Helper()
	parts := strings.Split(signature, ".")
	raw, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		t.Fatal(err)
	}
	alternate := new(big.Int).Sub(elliptic.P256().Params().N, new(big.Int).SetBytes(raw[32:]))
	alternate.FillBytes(raw[32:])
	parts[2] = base64.RawURLEncoding.EncodeToString(raw)
	return strings.Join(parts, ".")
}
