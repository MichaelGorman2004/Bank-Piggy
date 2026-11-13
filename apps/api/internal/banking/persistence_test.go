package banking

import (
	"errors"
	"testing"

	"bankpiggy/api/internal/budget"
)

func TestFailedCredentialAttachRetainsCleanupUntilRevoked(t *testing.T) {
	s, p, identity := setupService(t)
	link := createTestLink(t, s, identity)
	execSQL(t, s, `CREATE FUNCTION reject_attach() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.state='delayed' AND NEW.token_cipher IS NOT NULL THEN RAISE EXCEPTION 'simulated attach failure'; END IF; RETURN NEW; END $$`)
	execSQL(t, s, `CREATE TRIGGER reject_attach BEFORE UPDATE ON bank_connections FOR EACH ROW EXECUTE FUNCTION reject_attach()`)
	p.removeErr = &ProviderError{Code: "provider_unavailable"}
	if err := s.CompleteLink(t.Context(), identity, budget.ExchangeRequest{AttemptID: link.AttemptID, PublicToken: ptr("public-simulated-token")}); err == nil {
		t.Fatal("failed attachment reported success")
	}
	var state, cipher string
	if err := s.db.Pool.QueryRow(t.Context(), `SELECT state,token_cipher FROM bank_connections`).Scan(&state, &cipher); err != nil || state != "disconnecting" || cipher == "" {
		t.Fatal("failed attachment discarded cleanup credential")
	}
	if err := s.processOne(t.Context(), ""); err == nil {
		t.Fatal("failed cleanup reported success")
	}
	p.removeErr = nil
	execSQL(t, s, `UPDATE bank_connections SET next_sync_at=now()`)
	if err := s.processOne(t.Context(), ""); err != nil {
		t.Fatal(err)
	}
}

func TestManualMappingForeignDuplicateAndUnsupportedAccounts(t *testing.T) {
	s, p, identity, connection := connectedService(t)
	plan, err := s.db.Read(t.Context(), identity.HouseholdID)
	if err != nil {
		t.Fatal(err)
	}
	mapping := budget.AccountMappingRequest{Revision: 1, Enabled: true, Ownership: "shared", Classification: "savings", ManualMemberID: ptr("member-1"), ManualAccountID: ptr(plan.Members[0].Accounts[0].ID)}
	account := connection.Accounts[0]
	foreign := mapping
	foreign.ManualMemberID = ptr("foreign-member")
	if err := s.MapAccount(t.Context(), identity, connection.ID, account.ID, foreign); !errors.Is(err, ErrInvalid) {
		t.Fatal("foreign manual member accepted")
	}
	if err := s.MapAccount(t.Context(), identity, "foreign-connection", account.ID, mapping); !errors.Is(err, ErrNotFound) {
		t.Fatal("foreign connection accepted")
	}
	if err := s.MapAccount(t.Context(), identity, connection.ID, account.ID, mapping); err != nil {
		t.Fatal(err)
	}
	p.snapshot.Accounts = append(p.snapshot.Accounts, ProviderAccount{ProviderID: "second-provider-account", Account: budget.ConnectedAccount{Name: "Second", ProviderType: "depository", Currency: ptr("USD"), CurrentBalanceCents: ptr(int64(25))}})
	execSQL(t, s, `UPDATE bank_connections SET next_sync_at=now()`)
	if err := s.processOne(t.Context(), ""); err != nil {
		t.Fatal(err)
	}
	list, err := s.List(t.Context(), identity)
	if err != nil {
		t.Fatal(err)
	}
	var second string
	for _, candidate := range list.Connections[0].Accounts {
		if candidate.ID != account.ID {
			second = candidate.ID
		}
	}
	if err := s.MapAccount(t.Context(), identity, connection.ID, second, mapping); !errors.Is(err, ErrConflict) {
		t.Fatal("same manual account counted twice", err)
	}

}

func TestProductionBudgetReservedBeforeLinkAndNeverRefunded(t *testing.T) {
	s, p, identity := setupService(t)
	s.config.Environment = "production"
	s.config.ProductionApproved = true
	s.config.LifetimeItemLimit = 1
	p.linkErr = &ProviderError{Code: "provider_unavailable"}
	if _, err := s.CreateLink(t.Context(), identity, budget.LinkTokenRequest{Kind: ptr(budget.ConnectionKind("depository"))}); err == nil {
		t.Fatal("provider error hidden")
	}
	if _, err := s.CreateLink(t.Context(), identity, budget.LinkTokenRequest{Kind: ptr(budget.ConnectionKind("depository"))}); !errors.Is(err, ErrLimit) {
		t.Fatal("failed Link refunded a possible created Item", err)
	}
	if p.links != 1 {
		t.Fatal("provider issued a Link token beyond approved budget")
	}
}

func TestMappingRejectsUnsupportedAndUnclassifiedAccounts(t *testing.T) {
	plan := budget.Blank("First")
	request := budget.AccountMappingRequest{Revision: 1, Enabled: true, Ownership: "member-1", Classification: "unclassified"}
	account := budget.ConnectedAccount{ProviderType: "depository", Active: true}
	if err := validateMapping(plan, account, request); !errors.Is(err, ErrInvalid) {
		t.Fatal("unclassified balance included")
	}
	request.Classification = "savings"
	account.ProviderType = "credit"
	if err := validateMapping(plan, account, request); !errors.Is(err, ErrInvalid) {
		t.Fatal("unsupported credit account included")
	}
}
