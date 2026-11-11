package server

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"bankpiggy/api/internal/budget"
)

func TestDecodePlanEnforcesRequiredFieldsAndSafeMoney(t *testing.T) {
	valid := planJSON(t, budget.Sample())
	unknown := append(bytes.TrimSuffix(valid, []byte("}")), []byte(`,"householdId":"foreign"}`)...)
	tests := []struct {
		name string
		body []byte
	}{
		{"unknown field", unknown},
		{"missing salary", bytes.Replace(valid, []byte(`"annualBaseSalaryCents":10000000,`), nil, 1)},
		{"null salary", bytes.Replace(valid, []byte(`"annualBaseSalaryCents":10000000`), []byte(`"annualBaseSalaryCents":null`), 1)},
		{"fractional cents", bytes.Replace(valid, []byte(`"monthlyTakeHomeCents":620000`), []byte(`"monthlyTakeHomeCents":1.5`), 1)},
		{"trailing object", append(bytes.Clone(valid), []byte(" {}")...)},
		{"oversized body", bytes.Repeat([]byte(" "), 129*1024)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest("PUT", "/api/household", bytes.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			if _, err := decodePlan(httptest.NewRecorder(), request); err == nil {
				t.Fatal("invalid request accepted")
			}
		})
	}
}

func TestOptionalTripRequiresItsCompleteNestedContract(t *testing.T) {
	plan := scheduledTripPlan(budget.Sample())
	valid := planJSON(t, plan)
	tripJSON, err := json.Marshal(plan.Goals[0].Trip)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		body []byte
	}{
		{"missing savings", bytes.Replace(valid, []byte(`"savedAmountCents":1000,`), nil, 1)},
		{"null savings", bytes.Replace(valid, []byte(`"savedAmountCents":1000`), []byte(`"savedAmountCents":null`), 1)},
		{"missing cost amount", bytes.Replace(valid, []byte(`,"amountCents":8000`), nil, 1)},
		{"null trip", bytes.Replace(valid, tripJSON, []byte(`null`), 1)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest("PUT", "/api/household", bytes.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			if _, err := decodePlan(httptest.NewRecorder(), request); err == nil {
				t.Fatal("incomplete nested trip accepted")
			}
		})
	}
}

func TestLegacyGoalsOmitTripAndEnteredZeroTripSavingsSurviveDecoding(t *testing.T) {
	legacyPlan := budget.Blank("First", "Second")
	legacyPlan.Goals = []budget.Goal{{ID: "legacy", Name: "Savings", MonthlyAmountCents: 100, OwnerID: "shared"}}
	legacy := planJSON(t, legacyPlan)
	if bytes.Contains(legacy, []byte(`"trip"`)) {
		t.Fatal("a non-trip goal serialized an empty trip object")
	}
	plan := scheduledTripPlan(budget.Sample())
	plan.Goals[0].Trip.SavedAmountCents = 0
	request := httptest.NewRequest("PUT", "/api/household", bytes.NewReader(planJSON(t, plan)))
	request.Header.Set("Content-Type", "application/json")
	decoded, err := decodePlan(httptest.NewRecorder(), request)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Goals[0].Trip == nil || decoded.Goals[0].Trip.SavedAmountCents != 0 {
		t.Fatal("explicit zero trip savings was lost")
	}
}
