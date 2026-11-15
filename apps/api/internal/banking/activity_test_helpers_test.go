package banking

import (
	"context"
	"testing"
	"time"

	"bankpiggy/api/internal/budget"
	"github.com/jackc/pgx/v5"
)

func (p *simulatedProvider) ActivityLink(ctx context.Context, user, token string) (string, time.Time, error) {
	return p.Link(ctx, user, "activity", token)
}

func (*simulatedProvider) TransactionsSync(context.Context, string, string) (TransactionPage, error) {
	return TransactionPage{NextCursor: "simulated-cursor", HistoryComplete: true}, nil
}

func (*simulatedProvider) InvestmentTransactions(context.Context, string, string, string, int) (InvestmentPage, error) {
	return InvestmentPage{}, nil
}

type activitySimulation struct {
	*simulatedProvider
	transactions func(context.Context, string, string) (TransactionPage, error)
	investments  func(context.Context, string, string, string, int) (InvestmentPage, error)
}

func (p *activitySimulation) TransactionsSync(ctx context.Context, token, cursor string) (TransactionPage, error) {
	if p.transactions != nil {
		return p.transactions(ctx, token, cursor)
	}
	return p.simulatedProvider.TransactionsSync(ctx, token, cursor)
}

func (p *activitySimulation) InvestmentTransactions(ctx context.Context, token, start, end string, offset int) (InvestmentPage, error) {
	if p.investments != nil {
		return p.investments(ctx, token, start, end, offset)
	}
	return p.simulatedProvider.InvestmentTransactions(ctx, token, start, end, offset)
}

func activityTestService(t *testing.T) (*Service, *activitySimulation, Identity, budget.Connection) {
	t.Helper()
	s, p, identity, connection := connectedService(t)
	s.config.BankActivityApproved, s.config.InvestmentActivityApproved = true, true
	simulation := &activitySimulation{simulatedProvider: p}
	s.provider = simulation
	return s, simulation, identity, connection
}

func activitySourcesForTest(t *testing.T, s *Service, identity Identity) []budget.ActivitySource {
	t.Helper()
	tx, err := s.db.Pool.BeginTx(t.Context(), pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		t.Fatal(err)
	}
	defer rollback(tx)
	result, err := s.readActivitySources(t.Context(), tx, identity)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(t.Context()); err != nil {
		t.Fatal(err)
	}
	return result
}

func enableActivityForTest(t *testing.T, s *Service, identity Identity, connection budget.Connection) budget.ActivitySource {
	t.Helper()
	request := budget.ActivitySettingsRequest{Enabled: true, Accounts: []budget.ActivityAccountSelection{{AccountID: connection.Accounts[0].ID, Enabled: true, Ownership: identity.MemberID}}}
	result, err := s.SetActivitySettings(t.Context(), identity, connection.ID, request)
	if err != nil {
		t.Fatal(err)
	}
	if connection.Kind == "investment" {
		return result
	}
	link, err := s.CreateLink(t.Context(), identity, budget.LinkTokenRequest{ConnectionID: &connection.ID, Purpose: ptr(budget.LinkPurpose("activity"))})
	if err != nil {
		t.Fatal(err)
	}
	completeTestLink(t, s, identity, link)
	return activitySourcesForTest(t, s, identity)[0]
}

func simulatedActivity(id, date string, amount int64) ProviderActivityRecord {
	return ProviderActivityRecord{ProviderID: id, ProviderAccountID: "provider-account-1", Record: budget.ActivityRecord{Kind: "bank", Date: budget.CalendarDate(date), Name: "Simulated activity", Currency: ptr("USD"), AmountCents: &amount}}
}
