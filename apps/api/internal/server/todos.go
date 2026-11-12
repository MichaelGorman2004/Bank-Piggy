package server

import (
	"errors"
	"net/http"
	"regexp"
	"time"

	"bankpiggy/api/internal/budget"
	"bankpiggy/api/internal/store"
)

const maxTodoIDLength = 160

var validTodoID = regexp.MustCompile(`^[a-zA-Z0-9_:-]+$`)

func validTodoUpdate(input budget.TodoUpdate, id string) bool {
	return input.PlanRevision > 0 && len(id) <= maxTodoIDLength && validTodoID.MatchString(id)
}

func todoDate(value string) (time.Time, error) {
	if !budget.ValidDate(value) {
		return time.Time{}, errors.New("date must be a valid YYYY-MM-DD between 2000 and 2100")
	}
	return time.Parse(time.DateOnly, value)
}

func (s *Server) getTodos(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireSession(w, r)
	if !ok {
		return
	}
	onDate, err := todoDate(r.URL.Query().Get("date"))
	if err != nil {
		fail(w, 400, "invalid_date", err.Error())
		return
	}
	list, err := s.store.Todos(r.Context(), session.HouseholdID, onDate)
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, 200, list)
}

func (s *Server) putTodo(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireMutation(w, r)
	if !ok {
		return
	}
	var input budget.TodoUpdate
	if err := decodeJSON(w, r, &input); err != nil {
		fail(w, 400, "invalid_todo", err.Error())
		return
	}
	if !validTodoUpdate(input, r.PathValue("id")) {
		fail(w, 400, "invalid_todo", "A current plan revision and a valid task ID are required.")
		return
	}
	onDate, err := todoDate(string(input.Date))
	if err != nil {
		fail(w, 400, "invalid_date", err.Error())
		return
	}
	list, err := s.store.UpdateTodo(r.Context(), session.HouseholdID, r.PathValue("id"), input, onDate)
	if errors.Is(err, store.ErrConflict) || errors.Is(err, store.ErrTodoConflict) {
		fail(w, 409, "todo_conflict", err.Error())
		return
	}
	if errors.Is(err, store.ErrTodoUnavailable) {
		fail(w, 404, "todo_unavailable", err.Error())
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, 200, list)
}
