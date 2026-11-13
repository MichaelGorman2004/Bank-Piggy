package banking

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/go-jose/go-jose/v4"
)

const WebhookBodyLimit = 64 * 1024
const WebhookSignatureLimit = 8192
const webhookAge = 5 * time.Minute
const keyCacheLimit = 8

var ErrInvalidWebhook = errors.New("invalid provider webhook")
var ErrWebhookBusy = errors.New("provider verification is busy")

type cachedKey struct {
	key     jose.JSONWebKey
	expires time.Time
}

// VerifyWebhook authenticates ES256, key identity, age, and the exact raw body hash.
// Verification rejects future signatures and never uses request-controlled URLs.
func (s *Service) VerifyWebhook(ctx context.Context, signature string, body []byte) error {
	if !s.config.Enabled {
		return ErrUnavailable
	}
	if !validWebhookSize(signature, body) {
		return ErrInvalidWebhook
	}
	signed, err := jose.ParseSignedCompact(signature, []jose.SignatureAlgorithm{jose.ES256})
	if err != nil || len(signed.Signatures) != 1 {
		return ErrInvalidWebhook
	}
	id := signed.Signatures[0].Header.KeyID
	if len(id) < 1 || len(id) > 128 {
		return ErrInvalidWebhook
	}
	key, err := s.webhookKey(ctx, id)
	if err != nil {
		return err
	}
	payload, err := signed.Verify(key.Key)
	if err != nil {
		return ErrInvalidWebhook
	}
	return validateWebhookClaims(payload, body)
}

func validateWebhookClaims(payload, body []byte) error {
	var claims struct {
		IssuedAt int64  `json:"iat"`
		BodyHash string `json:"request_body_sha256"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ErrInvalidWebhook
	}
	issued := time.Unix(claims.IssuedAt, 0)
	now := time.Now()
	if issued.After(now.Add(webhookClockTolerance)) || now.Sub(issued) > webhookAge {
		return ErrInvalidWebhook
	}
	hash, err := hex.DecodeString(claims.BodyHash)
	if err != nil {
		return ErrInvalidWebhook
	}
	actual := sha256.Sum256(body)
	if subtle.ConstantTimeCompare(hash, actual[:]) != 1 {
		return ErrInvalidWebhook
	}
	return nil
}

func (s *Service) webhookKey(ctx context.Context, id string) (jose.JSONWebKey, error) {
	if !s.keyMu.TryLock() {
		return jose.JSONWebKey{}, ErrWebhookBusy
	}
	defer s.keyMu.Unlock()
	now := time.Now()
	if cached, ok := s.keys[id]; ok && now.Before(cached.expires) {
		return cached.key, nil
	}
	if now.Sub(s.lastKeyFetch) < unknownKeyFetchInterval {
		return jose.JSONWebKey{}, ErrWebhookBusy
	}
	select {
	case s.slots <- struct{}{}:
		defer s.release()
	default:
		return jose.JSONWebKey{}, ErrWebhookBusy
	}
	s.lastKeyFetch = now
	return s.fetchVerificationKey(ctx, id, now)
}

func (s *Service) fetchVerificationKey(ctx context.Context, id string, now time.Time) (jose.JSONWebKey, error) {
	data, err := s.provider.VerificationKey(ctx, id)
	if err != nil {
		return jose.JSONWebKey{}, err
	}
	key, err := decodeKey(data, id, now)
	if err != nil {
		return key, err
	}
	if len(s.keys) >= keyCacheLimit {
		s.keys = map[string]cachedKey{}
	}
	expires, err := verificationExpiry(data, now)
	if err != nil {
		return jose.JSONWebKey{}, err
	}
	s.keys[id] = cachedKey{key: key, expires: expires}
	return key, nil
}

func decodeKey(data []byte, id string, now time.Time) (jose.JSONWebKey, error) {
	var key jose.JSONWebKey
	var metadata struct {
		Algorithm string `json:"alg"`
		Curve     string `json:"crv"`
		Type      string `json:"kty"`
		Use       string `json:"use"`
		ID        string `json:"kid"`
		Expired   *int64 `json:"expired_at"`
	}
	if err := json.Unmarshal(data, &metadata); err != nil {
		return key, ErrInvalidWebhook
	}
	if !validKeyMetadata(metadata.Algorithm, metadata.Curve, metadata.Type, metadata.Use, metadata.ID, id) {
		return key, ErrInvalidWebhook
	}
	if metadata.Expired != nil && !now.Before(time.Unix(*metadata.Expired, 0)) {
		return key, ErrInvalidWebhook
	}
	if err := json.Unmarshal(data, &key); err != nil || !key.Valid() {
		return key, ErrInvalidWebhook
	}
	public, ok := key.Key.(*ecdsa.PublicKey)
	if !ok || public.Curve.Params().BitSize != 256 {
		return key, ErrInvalidWebhook
	}
	return key, nil
}

// ApplyWebhook deduplicates a verified notification and only schedules authorized
// existing Items. Generation and state checks keep disconnect final.
func (s *Service) ApplyWebhook(ctx context.Context, signature string, body []byte) error {
	var event webhookEvent
	if err := json.Unmarshal(body, &event); err != nil || event.ItemID == "" || event.Environment != s.config.Environment {
		return ErrInvalidWebhook
	}
	segments := strings.Split(signature, ".")
	if len(segments) != 3 {
		return ErrInvalidWebhook
	}
	digest := sha256.Sum256(append([]byte(s.config.Environment+"/"+segments[0]+"."+segments[1]), body...))
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(tx)
	tag, err := tx.Exec(ctx, `INSERT INTO bank_webhook_receipts(digest) VALUES($1) ON CONFLICT DO NOTHING`, hex.EncodeToString(digest[:]))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return nil
	}
	if err := applyItemEvent(ctx, tx, event); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `DELETE FROM bank_webhook_receipts WHERE digest IN(SELECT digest FROM bank_webhook_receipts WHERE received_at<now()-$1::interval LIMIT $2)`, webhookReceiptRetention.String(), expiryCleanupBatch)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func validKeyMetadata(algorithm, curve, kind, use, keyID, expected string) bool {
	return algorithm == "ES256" && curve == "P-256" && kind == "EC" && use == "sig" && keyID == expected
}

func validWebhookSize(signature string, body []byte) bool {
	return len(signature) <= WebhookSignatureLimit && len(body) <= WebhookBodyLimit && !strings.ContainsAny(signature, " \t\r\n")
}

func verificationExpiry(data []byte, now time.Time) (time.Time, error) {
	var metadata struct {
		Expired *int64 `json:"expired_at"`
	}
	if err := json.Unmarshal(data, &metadata); err != nil {
		return time.Time{}, ErrInvalidWebhook
	}
	expires := now.Add(webhookAge)
	if metadata.Expired != nil && time.Unix(*metadata.Expired, 0).Before(expires) {
		expires = time.Unix(*metadata.Expired, 0)
	}
	return expires, nil
}
