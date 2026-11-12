package server

import (
	"reflect"
	"testing"

	"bankpiggy/api/internal/budget"
	"bankpiggy/api/internal/testbudget"
	"bankpiggy/api/internal/testdb"
)

func TestNewLocalHouseholdStartsEmpty(t *testing.T) {
	database := testdb.Open(t)
	config := Config{Mode: "demo", ListenAddr: "127.0.0.1:8080", BaseURL: testOrigin, DatabaseURL: "configured", WebDistDir: t.TempDir()}
	app, err := New(t.Context(), config, database)
	if err != nil {
		t.Fatal(err)
	}
	client := &testClient{app: app}
	client.signIn(t)
	plan := readPlan(t, client)
	if !reflect.DeepEqual(plan, budget.Blank("Member 1")) {
		t.Fatal("new household contains prefilled financial data")
	}
}

func TestStartupPreservesSavedHouseholds(t *testing.T) {
	database := testdb.Open(t)
	config := Config{Mode: "demo", ListenAddr: "127.0.0.1:8080", BaseURL: testOrigin, DatabaseURL: "configured", WebDistDir: t.TempDir()}
	saved := testbudget.Sample()
	saved.Members[0].Name = "Saved member"
	saved.Members[0].Accounts[0].BalanceCents = 12345
	saved.Expenses[0].MonthlyAmountCents = 11111
	saved.Goals[0].MonthlyAmountCents = 22222
	saved.Revision = 7
	for _, id := range []string{"demo-household", "private-household"} {
		if err := database.Seed(t.Context(), id, saved); err != nil {
			t.Fatal(err)
		}
	}
	for range 2 {
		if _, err := New(t.Context(), config, database); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []string{"demo-household", "private-household"} {
		actual, err := database.Read(t.Context(), id)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(actual, saved) {
			t.Fatalf("startup changed the saved %s plan", id)
		}
	}
}
