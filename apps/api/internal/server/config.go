// Package server provides authenticated HTTP access to a two-person monthly plan.
package server

import (
	"errors"
	"net"
	"net/mail"
	"net/url"
	"os"
	"strings"
)

// Config is validated before any listener or authentication endpoint starts.
type Config struct {
	Mode               string
	ListenAddr         string
	BaseURL            string
	DatabaseURL        string
	GoogleClientID     string
	GoogleClientSecret string
	WebDistDir         string
	Invites            []string
}

// LoadConfig defaults to private mode and rejects incomplete private deployments.
func LoadConfig() (Config, error) {
	c := Config{
		Mode:               os.Getenv("APP_MODE"),
		ListenAddr:         os.Getenv("LISTEN_ADDR"),
		BaseURL:            os.Getenv("APP_BASE_URL"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		WebDistDir:         os.Getenv("WEB_DIST_DIR"),
	}
	if c.Mode == "" {
		c.Mode = "private"
	}
	if c.ListenAddr == "" {
		c.ListenAddr = "127.0.0.1:8080"
	}
	if c.WebDistDir == "" {
		c.WebDistDir = "../web/dist"
	}
	if c.Mode == "demo" && c.BaseURL == "" {
		c.BaseURL = "http://127.0.0.1:5173"
	}
	c.Invites = strings.Split(strings.ToLower(strings.TrimSpace(os.Getenv("HOUSEHOLD_INVITES"))), ",")
	for i := range c.Invites {
		c.Invites[i] = strings.TrimSpace(c.Invites[i])
	}
	return c, c.Validate()
}

func loopback(host string) bool {
	ip := net.ParseIP(host)
	return host == "localhost" || (ip != nil && ip.IsLoopback())
}

func (c Config) baseOrigin() (*url.URL, error) {
	base, err := url.Parse(c.BaseURL)
	if err != nil || base.Host == "" || (base.Scheme != "http" && base.Scheme != "https") {
		return nil, errors.New("APP_BASE_URL must be an absolute HTTP origin")
	}
	if base.User != nil || base.RawQuery != "" || base.Fragment != "" || (base.Path != "" && base.Path != "/") {
		return nil, errors.New("APP_BASE_URL must contain only an origin")
	}
	return base, nil
}

func (c Config) validatePrivate(base *url.URL) error {
	if base.Scheme != "https" && !loopback(base.Hostname()) {
		return errors.New("private mode requires HTTPS outside loopback")
	}
	if c.GoogleClientID == "" || c.GoogleClientSecret == "" {
		return errors.New("private mode requires Google credentials")
	}
	if len(c.Invites) != 2 || c.Invites[0] == c.Invites[1] {
		return errors.New("HOUSEHOLD_INVITES must contain exactly two different email addresses")
	}
	for _, email := range c.Invites {
		parsed, err := mail.ParseAddress(email)
		if err != nil || parsed.Address != email {
			return errors.New("invalid invited email address")
		}
	}
	return nil
}

// Validate enforces demo isolation and the private household's invitation boundary.
func (c Config) Validate() error {
	if c.Mode != "demo" && c.Mode != "private" {
		return errors.New("APP_MODE must be demo or private")
	}
	if c.DatabaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}
	host, _, err := net.SplitHostPort(c.ListenAddr)
	if err != nil {
		return errors.New("LISTEN_ADDR must include host and port")
	}
	base, err := c.baseOrigin()
	if err != nil {
		return err
	}
	if c.Mode == "demo" {
		if !loopback(host) || !loopback(base.Hostname()) {
			return errors.New("demo mode requires loopback listener and origin")
		}
		return nil
	}
	return c.validatePrivate(base)
}
