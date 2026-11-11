package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"bankpiggy/api/internal/budget"
	"bankpiggy/api/internal/store"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// Server owns routes, authentication configuration, and its persistence boundary.
type Server struct {
	config   Config
	store    *store.Store
	oauth    oauth2.Config
	verifier *oidc.IDTokenVerifier
}

// New initializes the Google verifier only in explicitly configured private mode.
func New(ctx context.Context, c Config, s *store.Store) (*Server, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	app := &Server{config: c, store: s}
	if c.Mode == "private" {
		providerContext := oidc.ClientContext(ctx, &http.Client{Timeout: 10 * time.Second})
		provider, err := oidc.NewProvider(providerContext, "https://accounts.google.com")
		if err != nil {
			return nil, err
		}
		app.verifier = provider.Verifier(&oidc.Config{ClientID: c.GoogleClientID})
		app.oauth = oauth2.Config{
			ClientID:     c.GoogleClientID,
			ClientSecret: c.GoogleClientSecret,
			Endpoint:     provider.Endpoint(),
			RedirectURL:  strings.TrimRight(c.BaseURL, "/") + "/auth/google/callback",
			Scopes:       []string{oidc.ScopeOpenID, "email", "profile"},
		}
	}
	h := budget.Blank("Member 1", "Member 2")
	if c.Mode == "private" {
		names := []string{strings.Split(c.Invites[0], "@")[0], strings.Split(c.Invites[1], "@")[0]}
		h = budget.Blank(names[0], names[1])
	}
	if err := s.Seed(ctx, app.householdID(), h); err != nil {
		return nil, err
	}
	return app, nil
}

func (s *Server) householdID() string { return s.config.Mode + "-household" }

// Handler returns the API and optional built frontend with security headers.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /api/session", s.session)
	mux.HandleFunc("POST /api/demo/session", s.demoSession)
	mux.HandleFunc("POST /api/logout", s.logout)
	mux.HandleFunc("GET /api/household", s.getHousehold)
	mux.HandleFunc("PUT /api/household", s.putHousehold)
	mux.HandleFunc("GET /auth/google", s.googleStart)
	mux.HandleFunc("GET /auth/google/callback", s.googleCallback)
	mux.HandleFunc("/", s.frontend)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Cache-Control", "no-store")
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		mux.ServeHTTP(w, r.WithContext(ctx))
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		slog.Error("response encoding failed")
	}
}

func fail(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, budget.ErrorResponse{Error: message, Code: code})
}
func internal(w http.ResponseWriter, err error) {
	slog.Error("request failed", "errorType", fmt.Sprintf("%T", err))
	fail(w, 500, "internal_error", "The request could not be completed.")
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	if err := s.store.Pool.Ping(r.Context()); err != nil {
		fail(w, 503, "database_unavailable", "Database unavailable.")
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func (s *Server) getHousehold(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireSession(w, r)
	if !ok {
		return
	}
	month, err := planningMonth(r)
	if err != nil {
		fail(w, 400, "invalid_month", err.Error())
		return
	}
	h, err := s.store.Read(r.Context(), session.HouseholdID)
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, 200, budget.Response(h, month))
}

func decodePlan(w http.ResponseWriter, r *http.Request) (budget.Household, error) {
	var h budget.Household
	if strings.Split(r.Header.Get("Content-Type"), ";")[0] != "application/json" {
		return h, errors.New("Content-Type must be application/json")
	}
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 128*1024))
	if err != nil {
		return h, errors.New("request exceeds 128 KiB")
	}
	if err := requireFields(data, reflect.TypeFor[budget.Household]()); err != nil {
		return h, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&h); err != nil {
		return h, errors.New("invalid JSON plan")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return h, errors.New("request must contain one JSON object")
	}
	return h, budget.Validate(h)
}

func (s *Server) putHousehold(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireMutation(w, r)
	if !ok {
		return
	}
	month, err := planningMonth(r)
	if err != nil {
		fail(w, 400, "invalid_month", err.Error())
		return
	}
	h, err := decodePlan(w, r)
	if err != nil {
		fail(w, 400, "invalid_plan", err.Error())
		return
	}
	h, err = s.store.Update(r.Context(), session.HouseholdID, h)
	if errors.Is(err, store.ErrConflict) {
		fail(w, 409, "revision_conflict", err.Error())
		return
	}

	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, 200, budget.Response(h, month))
}

func (s *Server) frontend(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/auth/") || r.Method != "GET" {
		fail(w, 404, "not_found", "Not found.")
		return
	}
	root := s.config.WebDistDir
	if strings.Contains(path.Base(r.URL.Path), ".") {
		http.FileServerFS(os.DirFS(root)).ServeHTTP(w, r)
		return
	}
	if _, err := os.Stat(filepath.Join(root, "index.html")); err != nil {
		fail(w, 404, "frontend_unavailable", "Run the web development server or build the frontend.")
		return
	}
	http.ServeFile(w, r, filepath.Join(root, "index.html"))
}
