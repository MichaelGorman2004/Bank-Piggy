package server

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"bankpiggy/api/internal/budget"
	"bankpiggy/api/internal/store"
)

const sessionCookieName = "bank_piggy_session"
const sessionLifetime = 7 * 24 * time.Hour

func randomToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
func (s *Server) secureCookies() bool { return strings.HasPrefix(s.config.BaseURL, "https://") }

func (s *Server) setCookie(w http.ResponseWriter, name, value string, maxAge int) {
	cookie := &http.Cookie{Name: name, Value: value, Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, MaxAge: maxAge}
	// Validated loopback HTTP is the only exception to secure transport.
	if base, err := s.config.baseOrigin(); err == nil && loopback(base.Hostname()) {
		cookie.Secure = s.secureCookies()
	}
	http.SetCookie(w, cookie)
}

func (s *Server) lookupSession(r *http.Request) (store.Session, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return store.Session{}, store.ErrUnauthorized
	}
	session, err := s.store.Session(r.Context(), tokenHash(cookie.Value))
	if err != nil {
		return session, err
	}
	if session.HouseholdID != s.householdID() || (session.MemberID != "member-1" && session.MemberID != "member-2") {
		return store.Session{}, store.ErrUnauthorized
	}
	if s.config.Mode == "private" && session.MemberID != s.invitedMember(session.Email) {
		return store.Session{}, store.ErrUnauthorized
	}
	return session, nil
}

func (s *Server) requireSession(w http.ResponseWriter, r *http.Request) (store.Session, bool) {
	session, err := s.lookupSession(r)
	if errors.Is(err, store.ErrUnauthorized) {
		fail(w, 401, "unauthenticated", "Sign in to continue.")
		return session, false
	}
	if err != nil {
		internal(w, err)
		return session, false
	}
	return session, true
}

func (s *Server) sameOrigin(r *http.Request) bool {
	return r.Header.Get("Origin") == strings.TrimRight(s.config.BaseURL, "/")
}

func (s *Server) requireMutation(w http.ResponseWriter, r *http.Request) (store.Session, bool) {
	session, ok := s.requireSession(w, r)
	if !ok {
		return session, false
	}
	provided := r.Header.Get("X-CSRF-Token")
	if !s.sameOrigin(r) || provided == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(session.CSRF)) != 1 {
		fail(w, 403, "csrf_rejected", "Reload the page and try again.")
		return session, false
	}
	return session, true
}

func (s *Server) sessionResponse(r *http.Request, session *store.Session) (budget.Session, error) {
	response := budget.Session{Mode: s.config.Mode, GoogleConfigured: s.verifier != nil}
	if session == nil {
		return response, nil
	}
	h, err := s.store.Read(r.Context(), session.HouseholdID)
	if err != nil {
		return response, err
	}
	for _, m := range h.Members {
		if m.ID == session.MemberID {
			response.Authenticated = true
			response.Viewer = &budget.Viewer{ID: m.ID, Name: m.Name}
			response.CSRFToken = &session.CSRF
			return response, nil
		}
	}
	return response, store.ErrUnauthorized
}

func (s *Server) session(w http.ResponseWriter, r *http.Request) {
	identity, err := s.lookupSession(r)
	if errors.Is(err, store.ErrUnauthorized) {
		writeJSON(w, 200, budget.Session{Mode: s.config.Mode, GoogleConfigured: s.verifier != nil})
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	response, err := s.sessionResponse(r, &identity)
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, 200, response)
}

func (s *Server) issueSession(w http.ResponseWriter, r *http.Request, memberID string) (store.Session, error) {
	identity := store.Session{HouseholdID: s.householdID(), MemberID: memberID, ExpiresAt: time.Now().Add(sessionLifetime)}
	token, err := randomToken()
	if err != nil {
		return identity, err
	}
	identity.CSRF, err = randomToken()
	if err != nil {
		return identity, err
	}
	if previous, err := r.Cookie(sessionCookieName); err == nil {
		if err := s.store.DeleteSession(r.Context(), tokenHash(previous.Value)); err != nil {
			return identity, err
		}
	}
	if err := s.store.CreateSession(r.Context(), tokenHash(token), identity); err != nil {
		return identity, err
	}
	s.setCookie(w, sessionCookieName, token, int(sessionLifetime.Seconds()))
	return identity, nil
}

func (s *Server) demoSession(w http.ResponseWriter, r *http.Request) {
	if s.config.Mode != "demo" {
		fail(w, 404, "not_found", "Not found.")
		return
	}
	if !s.sameOrigin(r) {
		fail(w, 403, "origin_rejected", "Open Bank Piggy at its configured local address to continue.")
		return
	}
	identity, err := s.issueSession(w, r, "member-1")
	if err != nil {
		internal(w, err)
		return
	}
	response, err := s.sessionResponse(r, &identity)
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, 200, response)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireMutation(w, r); !ok {
		return
	}
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		fail(w, 401, "unauthenticated", "Sign in to continue.")
		return
	}
	if err := s.store.DeleteSession(r.Context(), tokenHash(cookie.Value)); err != nil {
		internal(w, err)
		return
	}
	s.setCookie(w, sessionCookieName, "", -1)
	w.WriteHeader(http.StatusNoContent)
}
