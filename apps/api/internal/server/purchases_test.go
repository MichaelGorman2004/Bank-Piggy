package server

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"testing"

	"bankpiggy/api/internal/budget"
	"bankpiggy/api/internal/testbudget"
)

func purchasePlan(plan budget.Household) budget.Household {
	plan = scheduledTripPlan(plan)
	plan.Goals = append(plan.Goals, budget.Goal{
		ID: "laptop", Name: "Laptop", MonthlyAmountCents: 251, OwnerID: "member-1",
		Purchase: &budget.PurchaseTarget{TargetAmountCents: 125001, SavedAmountCents: 25000, TargetMonth: "2029-03"},
	})
	return plan
}

func assertSavedPurchaseProjection(t *testing.T, accepted budget.HouseholdResponse) {
	t.Helper()
	if len(accepted.TripProjections) != 1 || len(accepted.PurchaseProjections) != 1 {
		t.Fatal("trip and purchase projections must both be returned")
	}
	projection := accepted.PurchaseProjections[0]
	if projection.GoalID != "laptop" || projection.RemainingAmountCents != 100001 || projection.MonthsRemaining != 4 || projection.SuggestedMonthlyCents != 25001 {
		t.Fatal("purchase target did not reach the inclusive whole-cent projection")
	}
}

func TestPurchaseAndOlderTripSurviveHTTPAndPostgreSQLRoundTrip(t *testing.T) {
	client := newTestClient(t)
	client.signIn(t)
	plan := purchasePlan(readPlan(t, client))
	accepted := decodeResponse(t, client.request("PUT", "/api/household?month=2028-12", planJSON(t, plan), testOrigin, client.csrf))
	assertSavedPurchaseProjection(t, accepted)
	loaded := decodeResponse(t, client.request("GET", "/api/household?month=2029-04", nil, "", ""))
	if !reflect.DeepEqual(loaded.Household.Goals, plan.Goals) {
		t.Fatal("purchase or older trip details changed in PostgreSQL")
	}
	if loaded.PurchaseProjections[0].Status != "overdue" || loaded.PurchaseProjections[0].RemainingAmountCents != 100001 {
		t.Fatal("past purchase target was removed or marked funded")
	}
	resaved := decodeResponse(t, client.request("PUT", "/api/household?month=2029-04", planJSON(t, loaded.Household), testOrigin, client.csrf))
	if resaved.Summary.Household.MonthlyGoalsCents != 652 || resaved.YearlySummary.Household.MonthlyGoalsCents != 7824 {
		t.Fatal("saved targets changed the chosen budget allocations")
	}
	if !reflect.DeepEqual(resaved.Household.Goals, plan.Goals) {
		t.Fatal("saving again changed older trip or purchase details")
	}
}

func TestOptionalPurchaseRequiresCompleteWholeCentJSON(t *testing.T) {
	plan := purchasePlan(testbudget.Sample())
	valid := planJSON(t, plan)
	purchaseJSON, err := json.Marshal(plan.Goals[1].Purchase)
	if err != nil {
		t.Fatal(err)
	}
	withoutMonth := bytes.Replace(purchaseJSON, []byte(`,"targetMonth":"2029-03"`), nil, 1)
	tests := []struct {
		name string
		body []byte
	}{
		{"missing target", bytes.Replace(valid, []byte(`"targetAmountCents":125001,`), nil, 1)},
		{"missing savings", bytes.Replace(valid, []byte(`"savedAmountCents":25000,`), nil, 1)},
		{"missing month", bytes.Replace(valid, purchaseJSON, withoutMonth, 1)},
		{"null savings", bytes.Replace(valid, []byte(`"savedAmountCents":25000`), []byte(`"savedAmountCents":null`), 1)},
		{"fractional cents", bytes.Replace(valid, []byte(`"targetAmountCents":125001`), []byte(`"targetAmountCents":1.5`), 1)},
		{"null purchase", bytes.Replace(valid, purchaseJSON, []byte(`null`), 1)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest("PUT", "/api/household", bytes.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			if _, err := decodePlan(httptest.NewRecorder(), request); err == nil {
				t.Fatal("incomplete purchase target accepted")
			}
		})
	}
}

func TestInvalidPurchaseWritePreservesSavedRevision(t *testing.T) {
	client := newTestClient(t)
	client.signIn(t)
	before := readPlan(t, client)
	plan := purchasePlan(before)
	plan.Goals[0].Purchase = plan.Goals[1].Purchase
	requireStatus(t, client.request("PUT", "/api/household", planJSON(t, plan), testOrigin, client.csrf), 400)
	plan.Goals[0].Purchase = nil
	plan.Goals[1].Purchase.TargetAmountCents = budget.MaxMoneyCents + 1
	requireStatus(t, client.request("PUT", "/api/household", planJSON(t, plan), testOrigin, client.csrf), 400)
	if !reflect.DeepEqual(readPlan(t, client), before) {
		t.Fatal("invalid purchase write changed the saved household")
	}
}
