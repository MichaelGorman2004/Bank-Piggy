package server

import (
	"encoding/json"
	"testing"

	"bankpiggy/api/internal/budget"
)

func TestBankHTTPReconnectProofRequiresExactCompletedAttempt(t *testing.T) {
	client, _ := newBankHTTPClient(t)
	client.signIn(t)
	first := createBankAttempt(t, client)
	requireStatus(t, client.request("POST", "/api/connections/exchange", bankJSON(t, budget.ExchangeRequest{AttemptID: first.AttemptID, PublicToken: httpString("simulated-public-token")}), testOrigin, client.csrf), 200)
	connection := bankList(t, client).Connections[0]
	if connection.LastCompletedAttemptID == nil || *connection.LastCompletedAttemptID != first.AttemptID {
		t.Fatal("new Item did not identify its completed attempt")
	}
	response := client.request("POST", "/api/connections/link-token", bankJSON(t, budget.LinkTokenRequest{ConnectionID: &connection.ID}), testOrigin, client.csrf)
	requireStatus(t, response, 200)
	var update budget.LinkTokenResponse
	if err := json.Unmarshal(response.Body.Bytes(), &update); err != nil {
		t.Fatal(err)
	}
	unchanged := bankList(t, client).Connections[0]
	if *unchanged.LastCompletedAttemptID != first.AttemptID {
		t.Fatal("issuing reconnect token falsely marked its attempt completed")
	}
	requireStatus(t, client.request("POST", "/api/connections/exchange", bankJSON(t, budget.ExchangeRequest{AttemptID: update.AttemptID}), testOrigin, client.csrf), 200)
	completed := bankList(t, client).Connections[0]
	if completed.LastCompletedAttemptID == nil || *completed.LastCompletedAttemptID != update.AttemptID {
		t.Fatal("accepted reconnect did not identify its exact completed attempt")
	}
}
