package server

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"bankpiggy/api/internal/budget"
)

func decodeResponse(t *testing.T, response *httptest.ResponseRecorder) budget.HouseholdResponse {
	t.Helper()
	requireStatus(t, response, 200)
	var data budget.HouseholdResponse
	if err := json.Unmarshal(response.Body.Bytes(), &data); err != nil {
		t.Fatal(err)
	}
	return data
}

func scheduledTripPlan(plan budget.Household) budget.Household {
	plan.Expenses = []budget.Expense{{
		ID: "car", Name: "Car", Category: "Transport", Kind: "loan", ProviderID: "car-lender",
		MonthlyAmountCents: 101, OwnerID: "shared", Split: budget.Split{Method: "equal"},
		StartMonth: "2026-01", EndMonth: "2028-12",
	}}
	plan.Goals = []budget.Goal{{
		ID: "trip", Name: "Vacation", MonthlyAmountCents: 401, OwnerID: "shared",
		Trip: &budget.TripBudget{
			SavedAmountCents: 1000, TargetMonth: "2029-03",
			Costs: []budget.TripCost{{Category: "flights", AmountCents: 8000}, {Category: "lodging", AmountCents: 4000}},
		},
	}}
	return plan
}

func assertAcceptedProjection(t *testing.T, accepted budget.HouseholdResponse, revision int64) {
	t.Helper()
	if accepted.PlanningMonth != "2028-12" || accepted.Household.Revision != revision {
		t.Fatal("save did not preserve the selected month and advance the revision")
	}
	if accepted.Summary.Household.MonthlyExpensesCents != 101 || accepted.YearlySummary.Household.MonthlyExpensesCents != 101 {
		t.Fatal("final active loan month or yearly projection is incorrect")
	}
	if len(accepted.TripProjections) != 1 {
		t.Fatal("accepted trip projection is missing")
	}
	if accepted.TripProjections[0].TargetAmountCents != 12000 || accepted.TripProjections[0].SuggestedMonthlyCents != 2750 {
		t.Fatal("trip metadata did not reach the projection")
	}
}

func assertInactivePaymentRetained(t *testing.T, loaded budget.HouseholdResponse) {
	t.Helper()
	if loaded.Summary.Household.MonthlyExpensesCents != 0 || loaded.YearlySummary.Household.MonthlyExpensesCents != 0 {
		t.Fatal("loan continued after its thirty-sixth month")
	}
	if len(loaded.Household.Expenses) != 1 || loaded.Household.Expenses[0].ProviderID != "car-lender" || loaded.Household.Expenses[0].EndMonth != "2028-12" {
		t.Fatal("inactive payment metadata was not retained in PostgreSQL")
	}
	if len(loaded.Household.Goals) != 1 {
		t.Fatal("saved trip goal is missing")
	}
	if loaded.Household.Goals[0].Trip == nil || loaded.Household.Goals[0].Trip.SavedAmountCents != 1000 {
		t.Fatal("optional trip metadata did not persist")
	}
}

func TestScheduledTripPlanSurvivesHTTPJSONAndPostgreSQLRoundTrip(t *testing.T) {
	client := newTestClient(t)
	client.signIn(t)
	plan := scheduledTripPlan(readPlan(t, client))
	response := client.request("PUT", "/api/household?month=2028-12", planJSON(t, plan), testOrigin, client.csrf)
	accepted := decodeResponse(t, response)
	assertAcceptedProjection(t, accepted, plan.Revision+1)
	loaded := decodeResponse(t, client.request("GET", "/api/household?month=2029-01", nil, "", ""))
	assertInactivePaymentRetained(t, loaded)
	resaved := decodeResponse(t, client.request("PUT", "/api/household?month=2029-01", planJSON(t, loaded.Household), testOrigin, client.csrf))
	if resaved.PlanningMonth != "2029-01" || resaved.YearlySummary.Household.MonthlyGoalsCents != 4812 {
		t.Fatal("save-again changed the period or double counted the trip target")
	}
}

func TestInvalidPlanningMonthsRejectWritesAndDefaultUsesCurrentUTCMonth(t *testing.T) {
	client := newTestClient(t)
	client.signIn(t)
	plan := readPlan(t, client)
	body := planJSON(t, plan)
	for _, month := range []string{"2026-13", "2026-1", "0000-01", "9999-01"} {
		path := "/api/household?month=" + month
		requireStatus(t, client.request("GET", path, nil, "", ""), 400)
		requireStatus(t, client.request("PUT", path, body, testOrigin, client.csrf), 400)
	}
	before := time.Now().UTC().Format("2006-01")
	loaded := decodeResponse(t, client.request("GET", "/api/household", nil, "", ""))
	after := time.Now().UTC().Format("2006-01")
	if loaded.PlanningMonth != before && loaded.PlanningMonth != after {
		t.Fatal("default projection did not use the current UTC month")
	}
	if loaded.Household.Revision != plan.Revision {
		t.Fatal("invalid query mutated the household")
	}
}
