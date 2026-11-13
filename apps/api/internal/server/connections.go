package server

import (
	"context"
	"errors"
	"net/http"

	"bankpiggy/api/internal/banking"
	"bankpiggy/api/internal/budget"
	"bankpiggy/api/internal/store"
)

func bankingIdentity(r *http.Request, session store.Session) banking.Identity {
	cookie, err := r.Cookie(sessionCookieName)
	hash := ""
	if err == nil {
		hash = tokenHash(cookie.Value)
	}
	return banking.Identity{HouseholdID: session.HouseholdID, MemberID: session.MemberID, SessionHash: hash}
}

func bankFailure(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, banking.ErrInvalidWebhook):
		fail(w, 400, "invalid_webhook", "Webhook body is invalid.")
	case errors.Is(err, banking.ErrInvalid):
		fail(w, 400, "invalid_connection", "Check the account settings and try again.")
	case errors.Is(err, banking.ErrNotFound):
		fail(w, 404, "connection_not_found", "This connection or Link attempt is not available.")
	case errors.Is(err, banking.ErrConflict):
		fail(w, 409, "connection_conflict", "The connection changed or an update is already in progress. Reload and try again.")
	case errors.Is(err, banking.ErrLimit):
		fail(w, 409, "connection_limit", "The configured free connection limit has been reached. Keep using manual accounts.")
	case errors.Is(err, banking.ErrUnavailable):
		fail(w, 503, "connections_unavailable", "Bank connections are unavailable. Manual accounts still work.")
	default:
		var provider *banking.ProviderError
		if errors.As(err, &provider) {
			fail(w, 502, provider.Code, "The bank could not complete this request. Try again or use a manual account.")
			return
		}
		internal(w, err)
	}
}

func (s *Server) getConnections(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireSession(w, r)
	if !ok {
		return
	}
	result, err := s.banking.List(r.Context(), bankingIdentity(r, session))
	if err != nil {
		bankFailure(w, err)
		return
	}
	writeJSON(w, 200, result)
}

func (s *Server) createBankLink(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireMutation(w, r)
	if !ok {
		return
	}
	var request budget.LinkTokenRequest
	if err := decodeJSON(w, r, &request); err != nil {
		fail(w, 400, "invalid_request", err.Error())
		return
	}
	result, err := s.banking.CreateLink(r.Context(), bankingIdentity(r, session), request)
	if err != nil {
		bankFailure(w, err)
		return
	}
	writeJSON(w, 200, result)
}

func (s *Server) pendingBankLink(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireSession(w, r)
	if !ok {
		return
	}
	result, err := s.banking.PendingLink(r.Context(), bankingIdentity(r, session))
	if err != nil {
		bankFailure(w, err)
		return
	}
	writeJSON(w, 200, result)
}

func (s *Server) exchangeBankLink(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireMutation(w, r)
	if !ok {
		return
	}
	var request budget.ExchangeRequest
	if err := decodeJSON(w, r, &request); err != nil {
		fail(w, 400, "invalid_request", err.Error())
		return
	}
	if err := s.banking.CompleteLink(r.Context(), bankingIdentity(r, session), request); err != nil {
		bankFailure(w, err)
		return
	}
	s.getConnections(w, r)
}

func (s *Server) refreshBankConnection(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireMutation(w, r)
	if !ok {
		return
	}
	if err := s.banking.ScheduleRefresh(r.Context(), bankingIdentity(r, session), r.PathValue("id")); err != nil {
		bankFailure(w, err)
		return
	}
	s.getConnections(w, r)
}

func (s *Server) disconnectBankConnection(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireMutation(w, r)
	if !ok {
		return
	}
	if err := s.banking.Disconnect(r.Context(), bankingIdentity(r, session), r.PathValue("id")); err != nil {
		bankFailure(w, err)
		return
	}
	s.getConnections(w, r)
}

func (s *Server) mapBankAccount(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireMutation(w, r)
	if !ok {
		return
	}
	var request budget.AccountMappingRequest
	if err := decodeJSON(w, r, &request); err != nil {
		fail(w, 400, "invalid_request", err.Error())
		return
	}
	if err := s.banking.MapAccount(r.Context(), bankingIdentity(r, session), r.PathValue("id"), r.PathValue("accountId"), request); err != nil {
		bankFailure(w, err)
		return
	}
	s.getConnections(w, r)
}

// RunBankWorker runs optional leased synchronization until process shutdown.
func (s *Server) RunBankWorker(ctx context.Context) { s.banking.RunWorker(ctx) }
