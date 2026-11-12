package server

import (
	"errors"
	"net/http"
	"net/mail"
	"strings"

	"bankpiggy/api/internal/budget"
	"bankpiggy/api/internal/store"
)

var errMemberNotAdded = errors.New("the invited person has no saved profile")

const maxInvitedProfileNameRunes = 80

func (s *Server) googleMember(r *http.Request, attempt store.OIDCAttempt, claims googleClaims, subject string) (string, error) {
	email := strings.ToLower(claims.Email)
	if attempt.InvitationHash != "" {
		err := s.store.ClaimInvitation(r.Context(), s.householdID(), attempt.InvitationHash, email, subject, invitedProfileName(claims))
		return "member-2", err
	}
	identity, err := s.store.Identity(r.Context(), s.householdID(), email, subject)
	if err == nil && identity.InvitationAccepted {
		return s.savedGoogleMember(r, identity.MemberID)
	}
	if err != nil && !errors.Is(err, store.ErrUnauthorized) {
		return "", err
	}
	memberID := s.invitedMember(email)
	if memberID == "" {
		return "", store.ErrUnauthorized
	}
	if _, err := s.savedGoogleMember(r, memberID); err != nil {
		return "", err
	}
	err = s.store.BindIdentity(r.Context(), s.householdID(), memberID, email, subject)
	return memberID, err
}

func (s *Server) savedGoogleMember(r *http.Request, memberID string) (string, error) {
	h, err := s.store.Read(r.Context(), s.householdID())
	if err != nil {
		return "", err
	}
	if !budget.HasMember(h, memberID) {
		return "", errMemberNotAdded
	}
	return memberID, nil
}

func (s *Server) googleMemberFailure(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrIdentityMismatch):
		fail(w, 403, "identity_mismatch", "This invitation belongs to another Google identity.")
	case errors.Is(err, errMemberNotAdded):
		fail(w, 403, "member_not_added", "Add this person to the household plan before signing in.")
	case errors.Is(err, store.ErrUnauthorized):
		fail(w, 403, "invite_required", "This account is not invited to this household.")
	default:
		s.invitationFailure(w, err)
	}
}

func validVerifiedEmail(email string) bool {
	parsed, err := mail.ParseAddress(email)
	return err == nil && parsed.Address == email
}

func invitedProfileName(claims googleClaims) string {
	name := strings.TrimSpace(claims.Name)
	if name == "" {
		name = strings.Split(claims.Email, "@")[0]
	}
	runes := []rune(name)
	if len(runes) > maxInvitedProfileNameRunes {
		return string(runes[:maxInvitedProfileNameRunes])
	}
	return name
}
