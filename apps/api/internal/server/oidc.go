package server

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"bankpiggy/api/internal/store"
	"golang.org/x/oauth2"
)

const oidcCookieName = "bank_piggy_oidc"

func (s *Server) googleStart(w http.ResponseWriter, r *http.Request) {
	if s.verifier == nil {
		fail(w, 404, "google_unavailable", "Google sign-in is not available in demo mode.")
		return
	}
	state, err := randomToken()
	if err != nil {
		internal(w, err)
		return
	}
	nonce, err := randomToken()
	if err != nil {
		internal(w, err)
		return
	}
	verifier := oauth2.GenerateVerifier()
	attempt := store.OIDCAttempt{State: state, Nonce: nonce, Verifier: verifier, ExpiresAt: time.Now().Add(10 * time.Minute)}
	token, err := randomToken()
	if err != nil {
		internal(w, err)
		return
	}
	if err := s.store.CreateOIDCAttempt(r.Context(), tokenHash(token), attempt); err != nil {
		internal(w, err)
		return
	}
	s.setCookie(w, oidcCookieName, token, 600)
	target := s.oauth.AuthCodeURL(state, oauth2.SetAuthURLParam("nonce", nonce), oauth2.S256ChallengeOption(verifier))
	http.Redirect(w, r, target, http.StatusFound)
}

func (s *Server) readOIDCAttempt(r *http.Request) (store.OIDCAttempt, error) {
	var attempt store.OIDCAttempt
	cookie, err := r.Cookie(oidcCookieName)
	if err != nil {
		return attempt, fmt.Errorf("sign-in cookie missing")
	}
	attempt, err = s.store.ConsumeOIDCAttempt(r.Context(), tokenHash(cookie.Value))
	if err != nil {
		return attempt, err
	}
	state := r.URL.Query().Get("state")
	if state == "" || attempt.State == "" || subtle.ConstantTimeCompare([]byte(state), []byte(attempt.State)) != 1 || attempt.ExpiresAt.Before(time.Now()) || attempt.Verifier == "" || attempt.Nonce == "" {
		return attempt, fmt.Errorf("invalid or expired sign-in attempt")
	}
	return attempt, nil
}

func (s *Server) googleCallback(w http.ResponseWriter, r *http.Request) {
	if s.verifier == nil {
		fail(w, 404, "google_unavailable", "Google sign-in is unavailable.")
		return
	}
	attempt, err := s.readOIDCAttempt(r)
	s.setCookie(w, oidcCookieName, "", -1)
	if err != nil {
		fail(w, 400, "invalid_signin", "Sign-in expired. Start again.")
		return
	}
	claims, subject, err := s.verifiedGoogleClaims(r, attempt)
	if err != nil {
		fail(w, 401, "invalid_signin", "Google identity could not be verified.")
		return
	}
	memberID := s.invitedMember(claims.Email)
	if memberID == "" {
		fail(w, 403, "invite_required", "This account is not invited to this household.")
		return
	}
	if err := s.store.BindIdentity(r.Context(), s.householdID(), memberID, strings.ToLower(claims.Email), subject); err != nil {
		if errors.Is(err, store.ErrIdentityMismatch) {
			fail(w, 403, "identity_mismatch", "This invitation belongs to another Google identity.")
		} else {
			internal(w, err)
		}
		return
	}
	if _, err := s.issueSession(w, r, memberID); err != nil {
		internal(w, err)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *Server) invitedMember(email string) string {
	for i, invited := range s.config.Invites {
		if strings.EqualFold(email, invited) {
			return fmt.Sprintf("member-%d", i+1)
		}
	}
	return ""
}

type googleClaims struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}

func (s *Server) verifiedGoogleClaims(r *http.Request, attempt store.OIDCAttempt) (googleClaims, string, error) {
	var claims googleClaims
	token, err := s.oauth.Exchange(r.Context(), r.URL.Query().Get("code"), oauth2.VerifierOption(attempt.Verifier))
	if err != nil {
		return claims, "", err
	}
	raw, ok := token.Extra("id_token").(string)
	if !ok {
		return claims, "", fmt.Errorf("missing identity token")
	}
	identity, err := s.verifier.Verify(r.Context(), raw)
	if err != nil {
		return claims, "", err
	}
	if identity.Subject == "" || subtle.ConstantTimeCompare([]byte(identity.Nonce), []byte(attempt.Nonce)) != 1 {
		return claims, "", fmt.Errorf("identity binding mismatch")
	}
	if err := identity.Claims(&claims); err != nil {
		return claims, "", err
	}
	if !claims.EmailVerified {
		return claims, "", fmt.Errorf("email is not verified")
	}
	return claims, identity.Subject, nil
}
