package server

import (
	"encoding/json"
	"reflect"
	"testing"

	"bankpiggy/api/internal/budget"
	"bankpiggy/api/internal/testdb"
)

func newSoloTestClient(t *testing.T) *testClient {
	t.Helper()
	config := Config{Mode: "demo", ListenAddr: "127.0.0.1:8080", BaseURL: testOrigin, DatabaseURL: "configured", WebDistDir: t.TempDir()}
	app, err := New(t.Context(), config, testdb.Open(t))
	if err != nil {
		t.Fatal(err)
	}
	client := &testClient{app: app}
	client.signIn(t)
	return client
}

func TestSoloPlanAddsPartnerWithoutRedistributingPersonalRecords(t *testing.T) {
	client := newSoloTestClient(t)
	plan := readPlan(t, client)
	plan.Members[0].MonthlyTakeHomeCents = 10000
	plan.Expenses = []budget.Expense{{ID: "bill", Name: "Bill", Category: "Bills", OwnerID: "member-1", PaidByMemberID: "member-1", MonthlyAmountCents: 101, Split: budget.Split{Method: "equal"}}}
	plan.Goals = []budget.Goal{{ID: "goal", Name: "Goal", OwnerID: "member-1", MonthlyAmountCents: 103}}
	before := decodeResponse(t, client.request("PUT", "/api/household", planJSON(t, plan), testOrigin, client.csrf))
	addition := before.Household
	addition.Members = append(addition.Members, budget.Blank("First", "Second").Members[1])
	accepted := decodeResponse(t, client.request("PUT", "/api/household", planJSON(t, addition), testOrigin, client.csrf))
	if accepted.Summary.Members["member-1"] != before.Summary.Members["member-1"] || accepted.Summary.Household != before.Summary.Household {
		t.Fatal("adding a partner redistributed personal records")
	}
	addition.Members[1].Name = "Stale partner"
	requireStatus(t, client.request("PUT", "/api/household", planJSON(t, addition), testOrigin, client.csrf), 409)
	requireStatus(t, client.request("PUT", "/api/household", planJSON(t, before.Household), testOrigin, client.csrf), 409)
	if saved := readPlan(t, client); !reflect.DeepEqual(saved, accepted.Household) {
		t.Fatal("stale save changed the added partner")
	}
	removal := accepted.Household
	removal.Members = removal.Members[:1]
	response := client.request("PUT", "/api/household", planJSON(t, removal), testOrigin, client.csrf)
	requireStatus(t, response, 400)
	var failure budget.ErrorResponse
	if err := json.Unmarshal(response.Body.Bytes(), &failure); err != nil {
		t.Fatal(err)
	}
	if failure.Code != "member_removal_unsupported" || !reflect.DeepEqual(readPlan(t, client), accepted.Household) {
		t.Fatal("member removal was not rejected without changing the plan")
	}
}

func TestSoloPlanRejectsAbsentOwnersAndPayersWithoutSaving(t *testing.T) {
	client := newSoloTestClient(t)
	original := readPlan(t, client)
	tests := []struct {
		name   string
		change func(*budget.Household)
	}{
		{"expense owner", func(h *budget.Household) { h.Expenses[0].OwnerID = "member-2" }},
		{"expense payer", func(h *budget.Household) { h.Expenses[0].PaidByMemberID = "member-2" }},
		{"goal owner", func(h *budget.Household) { h.Goals[0].OwnerID = "member-2" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan := original
			plan.Expenses = []budget.Expense{{ID: "bill", Name: "Bill", Category: "Bills", OwnerID: "member-1", MonthlyAmountCents: 101, Split: budget.Split{Method: "equal"}}}
			plan.Goals = []budget.Goal{{ID: "goal", Name: "Goal", OwnerID: "member-1", MonthlyAmountCents: 103}}
			test.change(&plan)
			requireStatus(t, client.request("PUT", "/api/household", planJSON(t, plan), testOrigin, client.csrf), 400)
			if !reflect.DeepEqual(readPlan(t, client), original) {
				t.Fatal("invalid solo save changed the plan")
			}
		})
	}
}
