package server

import (
	"encoding/json"
	"testing"

	"bankpiggy/api/internal/budget"
)

func TestCustomBoundarySharesSurviveHTTPJSONRoundTrip(t *testing.T) {
	client := newTestClient(t)
	client.signIn(t)
	for _, bps := range []int64{0, 10000} {
		plan := readPlan(t, client)
		plan.Expenses[0].Split = budget.Split{Method: "custom", CustomFirstShareBps: bps}
		response := client.request("PUT", "/api/household", planJSON(t, plan), testOrigin, client.csrf)
		requireStatus(t, response, 200)
		assertEncodedShare(t, response.Body.Bytes(), bps)
		persisted := readPlan(t, client)
		if persisted.Expenses[0].Split.CustomFirstShareBps != bps {
			t.Fatal("saved percentage changed")
		}
		// Saving the returned plan must preserve the selected share exactly.
		requireStatus(t, client.request("PUT", "/api/household", planJSON(t, persisted), testOrigin, client.csrf), 200)
	}
}

func assertEncodedShare(t *testing.T, data []byte, want int64) {
	t.Helper()
	var wire struct {
		Household struct {
			Expenses []struct {
				Split map[string]json.RawMessage `json:"split"`
			} `json:"expenses"`
		} `json:"household"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		t.Fatal(err)
	}
	encoded, present := wire.Household.Expenses[0].Split["customFirstShareBps"]
	if !present {
		t.Fatal("custom percentage was omitted from the HTTP response")
	}
	var actual int64
	if err := json.Unmarshal(encoded, &actual); err != nil {
		t.Fatal(err)
	}
	if actual != want {
		t.Fatalf("encoded percentage %d, want %d", actual, want)
	}
}
