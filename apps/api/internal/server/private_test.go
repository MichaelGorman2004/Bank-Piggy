package server

import (
	"net/http/httptest"
	"testing"

	"bankpiggy/api/internal/budget"
)

func TestPrivateSessionRejectsChangedInvitationsAndDemoEndpoint(t *testing.T) {
	client := newTestClient(t)
	client.app.config.Mode = "private"
	client.app.config.Invites = []string{"one@example.com", "two@example.com"}
	database := client.app.store
	if err := database.Seed(t.Context(), "private-household", budget.Blank("One", "Two")); err != nil {
		t.Fatal(err)
	}
	if err := database.BindIdentity(t.Context(), "private-household", "member-1", "one@example.com", "subject"); err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/", nil)
	if _, err := client.app.issueSession(response, request, "member-1"); err != nil {
		t.Fatal(err)
	}
	result := response.Result()
	if err := result.Body.Close(); err != nil {
		t.Fatal(err)
	}
	client.cookie = result.Cookies()[0]
	requireStatus(t, client.request("GET", "/api/household", nil, "", ""), 200)
	requireStatus(t, client.request("POST", "/api/demo/session", nil, testOrigin, ""), 404)
	client.app.config.Invites[0] = "replacement@example.com"
	requireStatus(t, client.request("GET", "/api/household", nil, "", ""), 401)
}
