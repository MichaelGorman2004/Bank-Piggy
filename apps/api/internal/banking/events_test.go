package banking

import (
	"errors"
	"testing"
	"time"

	"bankpiggy/api/internal/budget"
)

func TestAccountRevocationAndRenewalNoticeLifecycle(t *testing.T) {
	s, _, identity, connection := connectedService(t)
	signer, _ := webhookSigner(t)
	notice := []byte(`{"environment":"sandbox","item_id":"item-simulated","webhook_type":"ITEM","webhook_code":"NEW_ACCOUNTS_AVAILABLE"}`)
	if err := s.ApplyWebhook(t.Context(), signedWebhook(t, signer, notice, time.Now()), notice); err != nil {
		t.Fatal(err)
	}
	execSQL(t, s, `UPDATE bank_connections SET next_sync_at=now()`)
	if err := s.processOne(t.Context(), connection.ID); err != nil {
		t.Fatal(err)
	}
	list, err := s.List(t.Context(), identity)
	if err != nil {
		t.Fatal(err)
	}
	current := list.Connections[0]
	assertConsentNotice(t, current)
	link, err := s.CreateLink(t.Context(), identity, budget.LinkTokenRequest{ConnectionID: &connection.ID})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.CompleteLink(t.Context(), identity, budget.ExchangeRequest{AttemptID: link.AttemptID, PublicToken: ptr("ignored-update-token")}); err != nil {
		t.Fatal(err)
	}
	revoked := []byte(`{"environment":"sandbox","item_id":"item-simulated","webhook_type":"ITEM","webhook_code":"USER_ACCOUNT_REVOKED","account_id":"provider-account-1"}`)
	if err := s.ApplyWebhook(t.Context(), signedWebhook(t, signer, revoked, time.Now()), revoked); err != nil {
		t.Fatal(err)
	}
	list, err = s.List(t.Context(), identity)
	if err != nil || len(list.Connections[0].Accounts) != 0 || list.Connections[0].ErrorCode != nil {
		t.Fatal("revoked account data remained or explicit update retained old prompt")
	}
}

func TestEnvironmentIsolationAndDisabledManualFallback(t *testing.T) {
	s, p, identity, connection := connectedService(t)
	production := testConfig()
	production.Environment = "production"
	production.ProductionApproved = true
	production.LifetimeItemLimit = 1
	production.RedirectURI = "https://example.test/plaid-return"
	production.WebhookURL = "https://example.test/api/plaid/webhook"
	other, err := New(production, s.db, p)
	if err != nil {
		t.Fatal(err)
	}
	list, err := other.List(t.Context(), identity)
	if err != nil || len(list.Connections) != 0 {
		t.Fatal("Sandbox data crossed into Production")
	}
	if err := other.Disconnect(t.Context(), identity, connection.ID); !errors.Is(err, ErrNotFound) {
		t.Fatal("Production operated on Sandbox credential", err)
	}
	if err := other.ScheduleRefresh(t.Context(), identity, connection.ID); !errors.Is(err, ErrNotFound) {
		t.Fatal("cross environment refresh accepted", err)
	}
}

func TestDisabledProviderKeepsManualFallback(t *testing.T) {
	s, _, identity := setupService(t)
	disabled, err := New(Config{}, s.db, nil)
	if err != nil {
		t.Fatal(err)
	}
	list, err := disabled.List(t.Context(), identity)
	if err != nil || list.Environment != "disabled" || len(list.Connections) != 0 {
		t.Fatal("disabled provider broke manual-only fallback")
	}
	if len(list.EffectiveBalances.Accounts) == 0 {
		t.Fatal("disabled provider removed manual balances")
	}
}

func TestExpiredAndAmbiguousAttemptsStayHonest(t *testing.T) {
	s, _, identity := setupService(t)
	link := createTestLink(t, s, identity)
	execSQL(t, s, `UPDATE bank_link_attempts SET expires_at=now()-interval '1 second' WHERE id=$1`, link.AttemptID)
	if err := s.retireExpiredAttempts(t.Context()); err != nil {
		t.Fatal(err)
	}
	list, err := s.List(t.Context(), identity)
	if err != nil {
		t.Fatal(err)
	}
	if list.Connections[0].State != "disconnected" || list.Connections[0].ErrorCode == nil || *list.Connections[0].ErrorCode != "removal_unverified" {
		t.Fatal("expired Link claimed provider removal")
	}
	next := createTestLink(t, s, identity)
	execSQL(t, s, `UPDATE bank_link_attempts SET state='exchanging',created_at=now()-interval '3 minutes' WHERE id=$1`, next.AttemptID)
	if err := s.retireExpiredAttempts(t.Context()); err != nil {
		t.Fatal(err)
	}
	var code *string
	if err := s.db.Pool.QueryRow(t.Context(), `SELECT error_code FROM bank_connections WHERE id=$1`, next.ConnectionID).Scan(&code); err != nil || code == nil || *code != "exchange_uncertain" {
		t.Fatal("ambiguous exchange silently retried or removed")
	}
}

func assertConsentNotice(t *testing.T, current budget.Connection) {
	t.Helper()
	if current.State != "connected" || current.ErrorCode == nil || *current.ErrorCode != "new_accounts_available" {
		t.Fatal("normal sync froze balances or erased account consent prompt")
	}
}

func TestConsentNoticeSurvivesTransientReadFailure(t *testing.T) {
	s, p, identity, connection := connectedService(t)
	signer, _ := webhookSigner(t)
	notice := []byte(`{"environment":"sandbox","item_id":"item-simulated","webhook_type":"ITEM","webhook_code":"NEW_ACCOUNTS_AVAILABLE"}`)
	if err := s.ApplyWebhook(t.Context(), signedWebhook(t, signer, notice, time.Now()), notice); err != nil {
		t.Fatal(err)
	}
	p.accountsErr = &ProviderError{Code: "provider_unavailable"}
	execSQL(t, s, `UPDATE bank_connections SET next_sync_at=now()`)
	if err := s.processOne(t.Context(), connection.ID); err != nil {
		t.Fatal(err)
	}
	p.accountsErr = nil
	execSQL(t, s, `UPDATE bank_connections SET next_sync_at=now()`)
	if err := s.processOne(t.Context(), connection.ID); err != nil {
		t.Fatal(err)
	}
	list, err := s.List(t.Context(), identity)
	if err != nil {
		t.Fatal(err)
	}
	assertConsentNotice(t, list.Connections[0])
}
