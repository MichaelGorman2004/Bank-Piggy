package server

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"
	"time"

	"bankpiggy/api/internal/store"
	"golang.org/x/oauth2"
)

const oidcCookieName = "bank_piggy_oidc"
const oidcAttemptLifetime = 10 * time.Minute

func (s *Server) googleStart(w http.ResponseWriter, r *http.Request) {
	if s.verifier == nil {
		fail(w, 404, "google_unavailable", "Google sign-in is not available in demo mode.")
		return
	}
	if !s.allowPublicAuth(w) {
		return
	}
	attempt, token, err := s.newOIDCAttempt(r)
	if err != nil {
		if r.URL.Query().Get("join") == "1" {
			s.setCookie(w, invitationCookieName, "", -1)
		}
		s.invitationFailure(w, err)
		return
	}
	if err := s.store.CreateOIDCAttempt(r.Context(), tokenHash(token), attempt); err != nil {
		internal(w, err)
		return
	}
	s.setCookie(w, oidcCookieName, token, int(oidcAttemptLifetime.Seconds()))
	if _, err := r.Cookie(invitationCookieName); err == nil {
		s.setCookie(w, invitationCookieName, "", -1)
	}
	target := s.oauth.AuthCodeURL(attempt.State, oauth2.SetAuthURLParam("nonce", attempt.Nonce), oauth2.S256ChallengeOption(attempt.Verifier))
	http.Redirect(w, r, target, http.StatusFound)
}

func (s *Server) newOIDCAttempt(r *http.Request) (store.OIDCAttempt, string, error) {
	var attempt store.OIDCAttempt
	invitationHash, err := s.preparedInvitation(r)
	if err != nil {
		return attempt, "", err
	}
	state, err := randomToken()
	if err != nil {
		return attempt, "", err
	}
	nonce, err := randomToken()
	if err != nil {
		return attempt, "", err
	}
	attempt = store.OIDCAttempt{State: state, Nonce: nonce, Verifier: oauth2.GenerateVerifier(), ExpiresAt: time.Now().Add(oidcAttemptLifetime), InvitationHash: invitationHash}
	token, err := randomToken()
	return attempt, token, err
}

func (s *Server) preparedInvitation(r *http.Request) (string, error) {
	if r.URL.Query().Get("join") != "1" {
		return "", nil
	}
	cookie, err := r.Cookie(invitationCookieName)
	if err != nil || !validInvitationToken(cookie.Value) {
		return "", store.ErrInvitationUnavailable
	}
	hash := tokenHash(cookie.Value)
	_, err = s.store.InvitationPreview(r.Context(), s.householdID(), hash)
	return hash, err
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
	memberID, err := s.googleMember(r, attempt, claims, subject)
	if err != nil {
		s.googleMemberFailure(w, err)
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
	Name          string `json:"name"`
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
	if !claims.EmailVerified || !validVerifiedEmail(claims.Email) {
		return claims, "", fmt.Errorf("email is not verified")
	}
	return claims, identity.Subject, nil
}
