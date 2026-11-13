package banking

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"bankpiggy/api/internal/store"
)

var ErrUnavailable = errors.New("bank connections are unavailable")
var ErrNotFound = errors.New("bank connection not found")
var ErrConflict = errors.New("bank connection changed; reload and try again")
var ErrInvalid = errors.New("invalid bank connection request")
var ErrLimit = errors.New("the configured connection limit has been reached")

// Identity carries only server-verified membership and opaque session binding.
type Identity struct{ HouseholdID, MemberID, SessionHash string }

// Service owns bounded provider work, encryption, and durable connection state.
type Service struct {
	config       Config
	db           *store.Store
	vault        *Vault
	provider     Provider
	slots        chan struct{}
	keyMu        sync.Mutex
	keys         map[string]cachedKey
	lastKeyFetch time.Time
}

// New creates the optional service; disabled configuration keeps the manual plan usable.
func New(c Config, db *store.Store, provider Provider) (*Service, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	s := &Service{config: c, db: db, slots: make(chan struct{}, providerConcurrency), keys: map[string]cachedKey{}}
	if !c.Enabled {
		return s, nil
	}
	vault, err := NewVault(c)
	if err != nil {
		return nil, err
	}
	s.vault = vault
	if provider == nil {
		provider = NewProvider(c)
	}
	s.provider = provider
	return s, nil
}

func opaqueID() (string, error) {
	value := make([]byte, 24)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}
func (s *Service) acquire(ctx context.Context) error {
	if !s.config.Enabled {
		return ErrUnavailable
	}
	select {
	case s.slots <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (s *Service) release() { <-s.slots }
func credentialContext(household, id, purpose string) string {
	return household + "/" + id + "/" + purpose
}
func formatTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	return ptr(value.UTC().Format(time.RFC3339))
}
