// Package banking owns read-only provider access and independent balance snapshots.
package banking

import (
	"encoding/base64"
	"errors"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const ProviderTimeout = 15 * time.Second
const SyncInterval = 30 * time.Minute
const LeaseDuration = 2 * time.Minute
const MaxFailures = 5
const AttemptLifetime = 30 * time.Minute

// Config keeps provider credentials and encryption keys on the server.
type Config struct {
	Enabled                                                             bool
	Environment, ClientID, Secret, ActiveKeyID, RedirectURI, WebhookURL string
	Keys                                                                map[string][]byte
	ProductionApproved                                                  bool
	LifetimeItemLimit                                                   int
}

// LoadConfig reads explicit opt-in settings without enabling provider access by default.
func LoadConfig(baseURL string) (Config, error) {
	c := Config{Enabled: os.Getenv("PLAID_ENABLED") == "true", Environment: os.Getenv("PLAID_ENV"), ClientID: os.Getenv("PLAID_CLIENT_ID"), Secret: os.Getenv("PLAID_SECRET"), ActiveKeyID: os.Getenv("PLAID_ACTIVE_KEY_ID"), Keys: map[string][]byte{}, ProductionApproved: os.Getenv("PLAID_PRODUCTION_APPROVED") == "true"}
	if !c.Enabled {
		return c, nil
	}
	if c.Environment == "" {
		c.Environment = "sandbox"
	}
	for _, entry := range strings.Split(os.Getenv("PLAID_TOKEN_KEYS"), ",") {
		parts := strings.SplitN(entry, ":", 2)
		if len(parts) != 2 {
			return c, errors.New("PLAID_TOKEN_KEYS requires id:base64 entries")
		}
		key, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			return c, errors.New("invalid Plaid encryption key")
		}
		if _, exists := c.Keys[parts[0]]; exists {
			return c, errors.New("duplicate Plaid encryption key ID")
		}
		c.Keys[parts[0]] = key
	}
	limit, err := strconv.Atoi(os.Getenv("PLAID_LIFETIME_ITEM_LIMIT"))
	if err != nil && c.Environment == "production" {
		return c, errors.New("production Plaid requires an explicit lifetime Item limit")
	}
	c.LifetimeItemLimit = limit
	c.RedirectURI = strings.TrimRight(baseURL, "/") + "/plaid-return"
	c.WebhookURL = strings.TrimRight(baseURL, "/") + "/api/plaid/webhook"
	return c, c.Validate()
}

// Validate fails closed for incomplete encryption, credentials, or live-cost approval.
func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.ClientID == "" || c.Secret == "" {
		return errors.New("plaid requires server credentials")
	}
	if c.Environment != "sandbox" && c.Environment != "production" {
		return errors.New("plaid environment must be sandbox or production")
	}
	if c.ActiveKeyID == "" || len(c.Keys[c.ActiveKeyID]) != 32 {
		return errors.New("plaid requires an active 256-bit encryption key")
	}
	if err := c.validateKeyRing(); err != nil {
		return err
	}
	return c.validateProduction()
}

func (c Config) validateKeyRing() error {
	for id, key := range c.Keys {
		if id == "" || strings.ContainsAny(id, ".:") || len(key) != 32 {
			return errors.New("invalid plaid encryption key ring")
		}
	}
	return nil
}

func (c Config) validateProduction() error {
	if c.Environment != "production" {
		return nil
	}
	if !c.ProductionApproved || c.LifetimeItemLimit < 1 || c.LifetimeItemLimit > 10 {
		return errors.New("production plaid requires confirmed Trial access and a lifetime Item limit from 1 to 10")
	}
	redirect, err := secureProviderURL(c.RedirectURI)
	if err != nil {
		return err
	}
	webhook, err := secureProviderURL(c.WebhookURL)
	if err != nil {
		return err
	}
	if redirect.Host != webhook.Host || redirect.Path != "/plaid-return" || webhook.Path != "/api/plaid/webhook" {
		return errors.New("production plaid URLs must use the configured app origin")
	}
	return nil
}

func secureProviderURL(value string) (*url.URL, error) {
	parsed, err := url.Parse(value)
	if err != nil {
		return nil, errors.New("invalid production plaid URL")
	}
	if parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("production plaid requires HTTPS URLs")
	}
	return parsed, nil
}
