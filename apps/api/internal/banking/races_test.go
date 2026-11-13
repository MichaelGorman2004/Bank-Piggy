package banking

import (
	"errors"
	"testing"

	"bankpiggy/api/internal/budget"
)

func TestDisconnectWinsAgainstLateSync(t *testing.T) {
	s, p, identity, connection := connectedService(t)
	p.readStarted, p.readRelease = make(chan struct{}), make(chan struct{})
	execSQL(t, s, `UPDATE bank_connections SET next_sync_at=now() WHERE id=$1`, connection.ID)
	finished := make(chan error, 1)
	go func() { finished <- s.processOne(t.Context(), connection.ID) }()
	<-p.readStarted
	if err := s.Disconnect(t.Context(), identity, connection.ID); err != nil {
		t.Fatal(err)
	}
	close(p.readRelease)
	if err := <-finished; !errors.Is(err, ErrConflict) {
		t.Fatal("late worker did not reject disconnected generation", err)
	}
	list, err := s.List(t.Context(), identity)
	if err != nil {
		t.Fatal(err)
	}
	if list.Connections[0].State != "disconnected" || len(list.Connections[0].Accounts) != 0 {
		t.Fatal("late sync restored disconnected data")
	}
	var credential *string
	if err := s.db.Pool.QueryRow(t.Context(), `SELECT token_cipher FROM bank_connections WHERE id=$1`, connection.ID).Scan(&credential); err != nil || credential != nil {
		t.Fatal("revoked token remained")
	}
}

func TestDisconnectDuringExchangeRetainsRevocationCredential(t *testing.T) {
	s, p, identity := setupService(t)
	link := createTestLink(t, s, identity)
	p.exchangeStarted, p.exchangeRelease = make(chan struct{}), make(chan struct{})
	p.removeErr = &ProviderError{Code: "provider_unavailable"}
	finished := make(chan error, 1)
	go func() {
		finished <- s.CompleteLink(t.Context(), identity, budget.ExchangeRequest{AttemptID: link.AttemptID, PublicToken: ptr("public-simulated-token")})
	}()
	<-p.exchangeStarted
	var id string
	if err := s.db.Pool.QueryRow(t.Context(), `SELECT connection_id FROM bank_link_attempts WHERE id=$1`, link.AttemptID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if err := s.Disconnect(t.Context(), identity, id); err != nil {
		t.Fatal(err)
	}
	close(p.exchangeRelease)
	if err := <-finished; !errors.Is(err, ErrConflict) {
		t.Fatal("exchange did not detect concurrent disconnect", err)
	}
	var state, cipher string
	if err := s.db.Pool.QueryRow(t.Context(), `SELECT state,token_cipher FROM bank_connections WHERE id=$1`, id).Scan(&state, &cipher); err != nil || state != "disconnecting" || cipher == "" {
		t.Fatal("only revocation credential was discarded")
	}
	if err := s.processOne(t.Context(), id); err == nil {
		t.Fatal("failed provider revocation reported success")
	}
	p.removeErr = nil
	execSQL(t, s, `UPDATE bank_connections SET next_sync_at=now() WHERE id=$1`, id)
	if err := s.processOne(t.Context(), id); err != nil {
		t.Fatal("restart cleanup did not recover", err)
	}
}

func TestFailedDisconnectKeepsCredentialAndExcludesBalances(t *testing.T) {
	s, p, identity, connection := connectedService(t)
	p.removeErr = &ProviderError{Code: "provider_unavailable"}
	if err := s.Disconnect(t.Context(), identity, connection.ID); err == nil {
		t.Fatal("revocation failure reported successful disconnect")
	}
	list, err := s.List(t.Context(), identity)
	if err != nil || list.Connections[0].State != "disconnecting" {
		t.Fatal("disconnect intent was not durable")
	}
	if list.Connections[0].Accounts[0].Active {
		t.Fatal("failed revoke left financial data active")
	}
	var cipher string
	if err := s.db.Pool.QueryRow(t.Context(), `SELECT token_cipher FROM bank_connections WHERE id=$1`, connection.ID).Scan(&cipher); err != nil || cipher == "" {
		t.Fatal("only revocation credential was lost")
	}
	p.removeErr = nil
	if err := s.Disconnect(t.Context(), identity, connection.ID); err != nil {
		t.Fatal("explicit revocation retry failed", err)
	}
}

func TestDisconnectDuringLinkCreationNeverReturnsUsableToken(t *testing.T) {
	s, p, identity := setupService(t)
	p.linkStarted, p.linkRelease = make(chan struct{}), make(chan struct{})
	t.Cleanup(func() {
		select {
		case <-p.linkRelease:
		default:
			close(p.linkRelease)
		}
	})
	type outcome struct {
		response budget.LinkTokenResponse
		err      error
	}
	finished := make(chan outcome, 1)
	go func() {
		response, err := s.CreateLink(t.Context(), identity, budget.LinkTokenRequest{Kind: ptr(budget.ConnectionKind("depository"))})
		finished <- outcome{response, err}
	}()
	<-p.linkStarted
	var id string
	if err := s.db.Pool.QueryRow(t.Context(), `SELECT id FROM bank_connections`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if err := s.Disconnect(t.Context(), identity, id); err != nil {
		t.Fatal(err)
	}
	close(p.linkRelease)
	result := <-finished
	if !errors.Is(result.err, ErrConflict) || result.response.LinkToken != "" {
		t.Fatal("cancelled connection received usable Link token", result.err)
	}
}
