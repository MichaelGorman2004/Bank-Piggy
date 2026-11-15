package server

import (
	"errors"
	"net/http"
	"net/url"

	"bankpiggy/api/internal/banking"
	"bankpiggy/api/internal/budget"
	"bankpiggy/api/internal/store"
)

func activityFailure(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, banking.ErrInvalid):
		fail(w, 400, "invalid_activity", "Check the activity settings and try again.")
	case errors.Is(err, banking.ErrNotFound):
		fail(w, 404, "activity_not_found", "This activity is not available.")
	case errors.Is(err, banking.ErrConflict), errors.Is(err, store.ErrConflict):
		fail(w, 409, "activity_conflict", "The activity or plan changed. Reload and try again.")
	case errors.Is(err, banking.ErrUnavailable), errors.Is(err, banking.ErrLimit):
		fail(w, 503, "activity_unavailable", "Activity is unavailable. Your saved plan still works.")
	default:
		bankFailure(w, err)
	}
}

func activityQuery(r *http.Request, allowed ...string) (url.Values, error) {
	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return nil, banking.ErrInvalid
	}
	for key, entries := range values {
		valid := false
		for _, name := range allowed {
			valid = valid || key == name
		}
		if !valid || len(entries) != 1 || entries[0] == "" {
			return nil, banking.ErrInvalid
		}
	}
	return values, nil
}

func (s *Server) getActivity(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireSession(w, r)
	if !ok {
		return
	}
	query, err := activityQuery(r, "month", "scope")
	if err != nil {
		activityFailure(w, err)
		return
	}
	month, err := planningMonth(r)
	if err != nil {
		activityFailure(w, banking.ErrInvalid)
		return
	}
	scope := query.Get("scope")
	if scope == "" {
		scope = "household"
	}
	result, err := s.banking.MonthlyActivity(r.Context(), bankingIdentity(r, session), month, scope)
	if err != nil {
		activityFailure(w, err)
		return
	}
	writeJSON(w, 200, result)
}

func (s *Server) putActivitySettings(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireMutation(w, r)
	if !ok {
		return
	}
	var request budget.ActivitySettingsRequest
	if err := decodeJSON(w, r, &request); err != nil {
		activityFailure(w, banking.ErrInvalid)
		return
	}
	result, err := s.banking.SetActivitySettings(r.Context(), bankingIdentity(r, session), r.PathValue("id"), request)
	if err != nil {
		activityFailure(w, err)
		return
	}
	writeJSON(w, 200, result)
}

func (s *Server) applyActivityPaycheck(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireMutation(w, r)
	if !ok {
		return
	}
	if _, err := activityQuery(r, "month"); err != nil {
		activityFailure(w, err)
		return
	}
	month, err := planningMonth(r)
	if err != nil {
		activityFailure(w, banking.ErrInvalid)
		return
	}
	var request budget.ApplyPaycheckRequest
	if err := decodeJSON(w, r, &request); err != nil {
		activityFailure(w, banking.ErrInvalid)
		return
	}
	result, err := s.banking.ApplyPaycheck(r.Context(), bankingIdentity(r, session), r.PathValue("id"), request)
	if err != nil {
		activityFailure(w, err)
		return
	}
	writeJSON(w, 200, budget.Response(result, month))
}
