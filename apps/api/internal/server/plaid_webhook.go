package server

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"bankpiggy/api/internal/banking"
)

func (s *Server) plaidWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, banking.WebhookBodyLimit))
	if err != nil {
		fail(w, 413, "webhook_too_large", "Webhook exceeds the request limit.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	signature := r.Header.Get("Plaid-Verification")
	if err := s.banking.VerifyWebhook(ctx, signature, body); err != nil {
		if errors.Is(err, banking.ErrWebhookBusy) {
			w.Header().Set("Retry-After", "5")
			fail(w, 429, "verification_busy", "Try again later.")
			return
		}
		if errors.Is(err, banking.ErrInvalidWebhook) {
			fail(w, 401, "invalid_webhook", "Webhook verification failed.")
			return
		}
		fail(w, 503, "verification_unavailable", "Webhook verification is unavailable.")
		return
	}
	if err := s.banking.ApplyWebhook(ctx, signature, body); err != nil {
		bankFailure(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
