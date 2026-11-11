package server

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"testing"

	"bankpiggy/api/internal/budget"
)

func ordinarySavingsGoal() budget.Goal {
	return budget.Goal{ID: "emergency", Name: "Emergency fund", MonthlyAmountCents: 101, OwnerID: "shared",
		SavingsTarget: &budget.SavingsTarget{TargetAmountCents: 10000, SavedAmountCents: 1234}}
}

func TestOrdinarySavingsProgressSurvivesHTTPAndPostgreSQL(t *testing.T) {
	client := newTestClient(t)
	client.signIn(t)
	plan := purchasePlan(readPlan(t, client))
	plan.Goals = append(plan.Goals, ordinarySavingsGoal())
	accepted := decodeResponse(t, client.request("PUT", "/api/household", planJSON(t, plan), testOrigin, client.csrf))
	loaded := decodeResponse(t, client.request("GET", "/api/household", nil, "", ""))
	if !reflect.DeepEqual(plan.Goals, loaded.Household.Goals) || !reflect.DeepEqual(plan.Members, loaded.Household.Members) {
		t.Fatal("ordinary progress, existing targets, or account snapshots changed in persistence")
	}
	loaded.Household.Goals[2].SavingsTarget.SavedAmountCents = 20000
	resaved := decodeResponse(t, client.request("PUT", "/api/household", planJSON(t, loaded.Household), testOrigin, client.csrf))
	if !reflect.DeepEqual(resaved.Summary, accepted.Summary) || !reflect.DeepEqual(resaved.YearlySummary, accepted.YearlySummary) {
		t.Fatal("ordinary progress changed budget allocations, shares, or account balances")
	}
	if len(resaved.TripProjections) != 1 || len(resaved.PurchaseProjections) != 1 || resaved.Household.Goals[2].SavingsTarget.SavedAmountCents != 20000 {
		t.Fatal("overfunded ordinary progress was lost or created a deadline projection")
	}
}

func TestOptionalOrdinaryProgressRequiresCompleteWholeCentJSON(t *testing.T) {
	plan := budget.Blank("First", "Second")
	plan.Goals = []budget.Goal{ordinarySavingsGoal()}
	valid := planJSON(t, plan)
	target, err := json.Marshal(plan.Goals[0].SavingsTarget)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		body []byte
	}{
		{"missing target", bytes.Replace(valid, []byte(`"targetAmountCents":10000,`), nil, 1)},
		{"missing saved", bytes.Replace(valid, []byte(`,"savedAmountCents":1234`), nil, 1)},
		{"null target", bytes.Replace(valid, []byte(`"targetAmountCents":10000`), []byte(`"targetAmountCents":null`), 1)},
		{"null saved", bytes.Replace(valid, []byte(`"savedAmountCents":1234`), []byte(`"savedAmountCents":null`), 1)},
		{"fractional cents", bytes.Replace(valid, []byte(`"targetAmountCents":10000`), []byte(`"targetAmountCents":1.5`), 1)},
		{"null progress", bytes.Replace(valid, target, []byte(`null`), 1)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest("PUT", "/api/household", bytes.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			if _, err := decodePlan(httptest.NewRecorder(), request); err == nil {
				t.Fatal("incomplete ordinary savings progress accepted")
			}
		})
	}
}

func TestInvalidOrdinaryProgressLeavesSavedRevisionIntact(t *testing.T) {
	client := newTestClient(t)
	client.signIn(t)
	before := readPlan(t, client)
	plan := purchasePlan(readPlan(t, client))
	plan.Goals[0].SavingsTarget = ordinarySavingsGoal().SavingsTarget
	requireStatus(t, client.request("PUT", "/api/household", planJSON(t, plan), testOrigin, client.csrf), 400)
	plan.Goals[0].SavingsTarget = nil
	goal := ordinarySavingsGoal()
	goal.SavingsTarget.TargetAmountCents = 0
	plan.Goals = append(plan.Goals, goal)
	requireStatus(t, client.request("PUT", "/api/household", planJSON(t, plan), testOrigin, client.csrf), 400)
	if !reflect.DeepEqual(before, readPlan(t, client)) {
		t.Fatal("rejected ordinary progress changed the saved household")
	}
}
