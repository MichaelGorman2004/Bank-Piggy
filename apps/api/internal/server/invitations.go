package server

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	"bankpiggy/api/internal/budget"
	"bankpiggy/api/internal/store"
)

const invitationCookieName = "bank_piggy_invitation"
const invitationLifetime = 7 * 24 * time.Hour
const invitationPreparationLifetime = 10 * time.Minute

func (s *Server) invitationStatus(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireSession(w, r)
	if !ok {
		return
	}
	status, err := s.store.InvitationStatus(r.Context(), session.HouseholdID)
	if err != nil {
		internal(w, err)
		return
	}
	h, err := s.store.Read(r.Context(), session.HouseholdID)
	if err != nil {
		internal(w, err)
		return
	}
	response := budget.InvitationStatus{Linked: status.Linked, CanInvite: s.config.Mode == "private" && session.MemberID == "member-1" && !status.Linked}
	if len(h.Members) == budget.MaxMembers {
		response.MemberName = h.Members[1].Name
	}
	if status.ExpiresAt != nil && !status.Linked {
		response.Pending = true
		response.ExpiresAt = status.ExpiresAt.UTC().Format(time.RFC3339)
	}
	writeJSON(w, 200, response)
}

func (s *Server) invitationMutation(w http.ResponseWriter, r *http.Request) (store.Session, bool) {
	session, ok := s.requireMutation(w, r)
	if !ok {
		return session, false
	}
	if s.config.Mode != "private" {
		fail(w, 404, "invitation_unavailable", "Invitation links are available in private mode.")
		return session, false
	}
	if session.MemberID != "member-1" {
		fail(w, 403, "invitation_forbidden", "Only the first household member can manage invitation links.")
		return session, false
	}
	if err := decodeJSON(w, r, &struct{}{}); err != nil {
		fail(w, 400, "invalid_invitation", err.Error())
		return session, false
	}
	return session, true
}

func (s *Server) createInvitation(w http.ResponseWriter, r *http.Request) {
	session, ok := s.invitationMutation(w, r)
	if !ok {
		return
	}
	token, err := randomToken()
	if err != nil {
		internal(w, err)
		return
	}
	expiresAt := time.Now().Add(invitationLifetime)
	if err := s.store.SetInvitation(r.Context(), session.HouseholdID, tokenHash(token), expiresAt); err != nil {
		if errors.Is(err, store.ErrPartnerLinked) {
			fail(w, 409, "partner_linked", "The second person already has sign-in access.")
		} else {
			internal(w, err)
		}
		return
	}
	response := budget.InvitationCreated{Url: strings.TrimRight(s.config.BaseURL, "/") + "/join#invite=" + token, ExpiresAt: expiresAt.UTC().Format(time.RFC3339)}
	writeJSON(w, 201, response)
}

func (s *Server) revokeInvitation(w http.ResponseWriter, r *http.Request) {
	session, ok := s.invitationMutation(w, r)
	if !ok {
		return
	}
	if err := s.store.RevokeInvitation(r.Context(), session.HouseholdID); err != nil {
		internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func validInvitationToken(token string) bool {
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	return err == nil && len(decoded) == opaqueTokenBytes && base64.RawURLEncoding.EncodeToString(decoded) == token
}

func (s *Server) prepareInvitation(w http.ResponseWriter, r *http.Request) {
	if s.config.Mode != "private" {
		fail(w, 404, "invitation_unavailable", "Invitation links are available in private mode.")
		return
	}
	if !s.sameOrigin(r) {
		fail(w, 403, "origin_rejected", "Open the invitation at the configured Bank Piggy address.")
		return
	}
	if !s.allowPublicAuth(w) {
		return
	}
	var input budget.InvitationPrepareRequest
	if err := decodeJSON(w, r, &input); err != nil {
		fail(w, 400, "invalid_invitation", err.Error())
		return
	}
	if !validInvitationToken(input.Token) {
		fail(w, 400, "invalid_invitation", "The invitation link is invalid.")
		return
	}
	preview, err := s.store.InvitationPreview(r.Context(), s.householdID(), tokenHash(input.Token))
	if err != nil {
		s.invitationFailure(w, err)
		return
	}
	s.setCookie(w, invitationCookieName, input.Token, int(invitationPreparationLifetime.Seconds()))
	writeJSON(w, 200, budget.InvitationPreview{HouseholdName: preview.HouseholdName + "’s household", MemberName: preview.MemberName, ExpiresAt: preview.ExpiresAt.UTC().Format(time.RFC3339)})
}

func (s *Server) invitationFailure(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrInvitationPlanConflict) {
		fail(w, 409, "invitation_plan_conflict", "Update salary-based splits in the household plan before adding another person. Then try the invitation again.")
		return
	}
	if errors.Is(err, store.ErrInvitationUnavailable) || errors.Is(err, store.ErrPartnerLinked) {
		fail(w, 410, "invitation_unavailable", "This invitation expired, was replaced, or was already used. Ask for a new link.")
		return
	}
	internal(w, err)
}
