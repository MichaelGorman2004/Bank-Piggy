package banking

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"bankpiggy/api/internal/budget"
	"bankpiggy/api/internal/testbudget"
	"bankpiggy/api/internal/testdb"
)

// simulatedProvider never contacts Plaid or creates Sandbox or Production Items.
type simulatedProvider struct {
	mu                                                         sync.Mutex
	linkErr, accountsErr, removeErr                            error
	links, exchanges, reads, removes                           int
	snapshot                                                   Snapshot
	exchangeStarted, exchangeRelease, readStarted, readRelease chan struct{}
	linkStarted, linkRelease                                   chan struct{}
	key                                                        []byte
}

func (p *simulatedProvider) Link(ctx context.Context, _, _, _ string) (string, time.Time, error) {
	p.mu.Lock()
	p.links++
	started, release, err := p.linkStarted, p.linkRelease, p.linkErr
	p.mu.Unlock()
	if started != nil {
		close(started)
		select {
		case <-release:
		case <-ctx.Done():
			return "", time.Time{}, ctx.Err()
		}
	}
	return "link-simulated", time.Now().Add(20 * time.Minute), err
}

func (p *simulatedProvider) Exchange(ctx context.Context, _ string) (string, string, error) {
	p.mu.Lock()
	p.exchanges++
	started, release := p.exchangeStarted, p.exchangeRelease
	p.mu.Unlock()
	if started != nil {
		close(started)
		select {
		case <-release:
		case <-ctx.Done():
			return "", "", ctx.Err()
		}
	}
	return "access-simulated-secret", "item-simulated", nil
}
func (p *simulatedProvider) Accounts(ctx context.Context, _, _ string) (Snapshot, error) {
	p.mu.Lock()
	p.reads++
	started, release, snapshot, err := p.readStarted, p.readRelease, p.snapshot, p.accountsErr
	p.mu.Unlock()
	if started != nil {
		close(started)
		select {
		case <-release:
		case <-ctx.Done():
			return Snapshot{}, ctx.Err()
		}
	}
	return snapshot, err
}
func (p *simulatedProvider) Remove(context.Context, string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.removes++
	return p.removeErr
}
func (p *simulatedProvider) VerificationKey(context.Context, string) ([]byte, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.key == nil {
		return nil, errors.New("simulation key absent")
	}
	return p.key, nil
}

func testConfig() Config {
	return Config{Enabled: true, Environment: "sandbox", ClientID: "simulated", Secret: "simulated", ActiveKeyID: "current", Keys: map[string][]byte{"current": make([]byte, 32)}}
}
func setupService(t *testing.T) (*Service, *simulatedProvider, Identity) {
	t.Helper()
	db := testdb.Open(t)
	if err := db.Seed(t.Context(), "test-household", testbudget.Sample()); err != nil {
		t.Fatal(err)
	}
	provider := &simulatedProvider{snapshot: Snapshot{InstitutionName: "Simulated bank", Accounts: []ProviderAccount{{ProviderID: "provider-account-1", Account: budget.ConnectedAccount{Name: "Simulated savings", ProviderType: "depository", Currency: ptr("USD"), CurrentBalanceCents: ptr(int64(10001)), Active: true}}}}}
	service, err := New(testConfig(), db, provider)
	if err != nil {
		t.Fatal(err)
	}
	return service, provider, Identity{HouseholdID: "test-household", MemberID: "member-1", SessionHash: "simulated-session-hash"}
}
func createTestLink(t *testing.T, s *Service, identity Identity) budget.LinkTokenResponse {
	t.Helper()
	result, err := s.CreateLink(t.Context(), identity, budget.LinkTokenRequest{Kind: ptr(budget.ConnectionKind("depository"))})
	if err != nil {
		t.Fatal(err)
	}
	return result
}
func completeTestLink(t *testing.T, s *Service, identity Identity, link budget.LinkTokenResponse) {
	t.Helper()
	if err := s.CompleteLink(t.Context(), identity, budget.ExchangeRequest{AttemptID: link.AttemptID, PublicToken: ptr("public-simulated-token")}); err != nil {
		t.Fatal(err)
	}
}
func connectedService(t *testing.T) (*Service, *simulatedProvider, Identity, budget.Connection) {
	t.Helper()
	s, p, identity := setupService(t)
	completeTestLink(t, s, identity, createTestLink(t, s, identity))
	if err := s.processOne(t.Context(), ""); err != nil {
		t.Fatal(err)
	}
	list, err := s.List(t.Context(), identity)
	if err != nil {
		t.Fatal(err)
	}
	return s, p, identity, list.Connections[0]
}
func execSQL(t *testing.T, s *Service, query string, args ...any) {
	t.Helper()
	if _, err := s.db.Pool.Exec(t.Context(), query, args...); err != nil {
		t.Fatal(err)
	}
}
