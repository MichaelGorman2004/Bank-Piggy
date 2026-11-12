package store_test

import (
	"errors"
	"reflect"
	"testing"

	"bankpiggy/api/internal/budget"
	"bankpiggy/api/internal/store"
	"bankpiggy/api/internal/testdb"
)

func TestMemberRemovalCannotChangeSavedPlan(t *testing.T) {
	database := testdb.Open(t)
	plan := budget.Blank("First", "Second")
	if err := database.Seed(t.Context(), "household", plan); err != nil {
		t.Fatal(err)
	}
	removal := plan
	removal.Members = plan.Members[:1]
	if _, err := database.Update(t.Context(), "household", removal); !errors.Is(err, store.ErrMemberRemoval) {
		t.Fatalf("member removal returned %v", err)
	}
	saved, err := database.Read(t.Context(), "household")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(saved, plan) {
		t.Fatal("rejected member removal changed the saved plan")
	}
}
