package store_test

import (
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"testing"
	"time"

	"bankpiggy/api/internal/budget"
	"bankpiggy/api/internal/store"
	"bankpiggy/api/internal/testbudget"
	"bankpiggy/api/internal/testdb"
)

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func decodeJSON(t *testing.T, data []byte, target any) {
	t.Helper()
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatal(err)
	}
}

func legacyPlanJSON(t *testing.T, plan budget.Household, legacyMembers ...int) []byte {
	t.Helper()
	var object map[string]json.RawMessage
	decodeJSON(t, mustJSON(t, plan), &object)
	var members []map[string]json.RawMessage
	decodeJSON(t, object["members"], &members)
	summary := budget.Response(plan, time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)).Summary
	for _, index := range legacyMembers {
		member := members[index]
		delete(member, "accounts")
		totals := summary.Members[plan.Members[index].ID]
		member["savingsCents"] = mustJSON(t, totals.SavingsCents)
		member["investmentsCents"] = mustJSON(t, totals.InvestmentsCents)
	}
	object["members"] = mustJSON(t, members)
	return mustJSON(t, object)
}

func insertStoredJSON(t *testing.T, database *store.Store, id string, revision int64, data []byte) {
	t.Helper()
	_, err := database.Pool.Exec(t.Context(), "INSERT INTO households (id, revision, plan) VALUES ($1, $2, $3)", id, revision, data)
	if err != nil {
		t.Fatal(err)
	}
}

func reopenStore(t *testing.T, database *store.Store) *store.Store {
	t.Helper()
	reopened, err := store.Open(t.Context(), database.Pool.Config().ConnString())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reopened.Pool.Close)
	return reopened
}

func storedJSON(t *testing.T, database *store.Store, id string) ([]byte, int64) {
	t.Helper()
	var data []byte
	var revision int64
	if err := database.Pool.QueryRow(t.Context(), "SELECT plan, revision FROM households WHERE id=$1", id).Scan(&data, &revision); err != nil {
		t.Fatal(err)
	}
	return data, revision
}

func assertMigratedPlan(t *testing.T, database *store.Store, id string, original budget.Household) budget.Household {
	t.Helper()
	plan, err := database.Read(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	_, revision := storedJSON(t, database, id)
	if revision != original.Revision+1 || plan.Revision != revision {
		t.Fatal("migration must advance both revision fields exactly once")
	}
	if err := budget.Validate(plan); err != nil {
		t.Fatal("migrated plan is invalid", err)
	}
	month := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	if !reflect.DeepEqual(budget.Response(plan, month).Summary, budget.Response(original, month).Summary) {
		t.Fatal("migration changed balances, allocations, or member ownership")
	}
	if !reflect.DeepEqual(plan.Expenses, original.Expenses) || !reflect.DeepEqual(plan.Goals, original.Goals) {
		t.Fatal("migration changed expense or goal details")
	}
	return plan
}

func TestLegacyAccountsMigrateOnceAndRejectStaleRevisions(t *testing.T) {
	database := testdb.Open(t)
	original := testbudget.Sample()
	original.Revision = 7
	insertStoredJSON(t, database, "legacy", original.Revision, legacyPlanJSON(t, original, 0, 1))
	migrated := reopenStore(t, database)
	plan := assertMigratedPlan(t, migrated, "legacy", original)
	summary := budget.Response(original, time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)).Summary
	for index, member := range plan.Members {
		if member.Name != original.Members[index].Name {
			t.Fatal("member names, ordering, or legacy account count changed")
		}
		totals := summary.Members[member.ID]
		want := []budget.AccountBalance{
			{ID: "legacy-savings", Name: "Savings", Type: "savings", BalanceCents: totals.SavingsCents},
			{ID: "legacy-investments", Name: "Other investments", Type: "other_investment", BalanceCents: totals.InvestmentsCents},
		}
		if !reflect.DeepEqual(member.Accounts, want) {
			t.Fatal("migration assigned a specific savings or retirement type to unknown legacy money")
		}
	}
	if _, err := migrated.Update(t.Context(), "legacy", original); !errors.Is(err, store.ErrConflict) {
		t.Fatal("pre-migration revision was accepted", err)
	}
	before, _ := storedJSON(t, migrated, "legacy")
	reopened := reopenStore(t, migrated)
	after, revision := storedJSON(t, reopened, "legacy")
	if string(before) != string(after) || revision != plan.Revision {
		t.Fatal("repeated migration changed accounts or revision")
	}
}

func TestMixedAndTypedPlansPreserveTypedAccounts(t *testing.T) {
	database := testdb.Open(t)
	original := testbudget.Sample()
	insertStoredJSON(t, database, "mixed", original.Revision, legacyPlanJSON(t, original, 0))
	insertStoredJSON(t, database, "typed", original.Revision, mustJSON(t, original))
	typedBefore, _ := storedJSON(t, database, "typed")
	migrated := reopenStore(t, database)
	mixed := assertMigratedPlan(t, migrated, "mixed", original)
	if !reflect.DeepEqual(mixed.Members[1], original.Members[1]) {
		t.Fatal("migration changed a typed member's accounts, names, balances, or ordering")
	}
	typedAfter, revision := storedJSON(t, migrated, "typed")
	if string(typedBefore) != string(typedAfter) || revision != original.Revision {
		t.Fatal("migration changed a fully typed plan or its revision")
	}
}

func TestStaleAggregateKeysAreRemovedWithoutDuplicatingTypedAccounts(t *testing.T) {
	database := testdb.Open(t)
	original := testbudget.Sample()
	var object map[string]json.RawMessage
	decodeJSON(t, mustJSON(t, original), &object)
	var members []map[string]json.RawMessage
	decodeJSON(t, object["members"], &members)
	members[0]["savingsCents"] = mustJSON(t, int64(2450000))
	members[0]["investmentsCents"] = mustJSON(t, int64(6800000))
	object["members"] = mustJSON(t, members)
	insertStoredJSON(t, database, "mixed-fields", original.Revision, mustJSON(t, object))
	migrated := reopenStore(t, database)
	plan := assertMigratedPlan(t, migrated, "mixed-fields", original)
	if !reflect.DeepEqual(plan.Members, original.Members) {
		t.Fatal("stale aggregate cleanup changed the canonical account snapshots")
	}
	data, _ := storedJSON(t, migrated, "mixed-fields")
	decodeJSON(t, data, &object)
	members = nil
	decodeJSON(t, object["members"], &members)
	if _, exists := members[0]["savingsCents"]; exists {
		t.Fatal("stale aggregate fields remained persisted")
	}
}

func TestAccountMigrationRollsBackAllPlansOnFailure(t *testing.T) {
	database := testdb.Open(t)
	plan := testbudget.Sample()
	insertStoredJSON(t, database, "valid", plan.Revision, legacyPlanJSON(t, plan, 0, 1))
	before, _ := storedJSON(t, database, "valid")
	plan.Revision = math.MaxInt64
	insertStoredJSON(t, database, "overflow", plan.Revision, legacyPlanJSON(t, plan, 0, 1))
	failed, err := store.Open(t.Context(), database.Pool.Config().ConnString())
	if err == nil {
		failed.Pool.Close()
		t.Fatal("migration should fail when the revision cannot advance")
	}
	after, revision := storedJSON(t, database, "valid")
	if string(before) != string(after) || revision != 1 {
		t.Fatal("failed migration partially changed another household")
	}
}
