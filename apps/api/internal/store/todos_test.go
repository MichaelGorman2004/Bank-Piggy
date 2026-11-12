package store_test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"bankpiggy/api/internal/budget"
	"bankpiggy/api/internal/store"
	"bankpiggy/api/internal/testdb"
)

var todoTestDate = time.Date(2026, 11, 11, 0, 0, 0, 0, time.UTC)

func todoStore(t *testing.T) (*store.Store, budget.Household) {
	t.Helper()
	database := testdb.Open(t)
	h := budget.Blank("First")
	h.Members[0].PaySchedule = &budget.PaySchedule{AnchorDate: "2026-11-01", SemiMonthlyDays: []budget.DayOfMonth{}}
	day := budget.DayOfMonth(15)
	h.Expenses = []budget.Expense{{ID: "bill", Name: "Bill", Category: "Costs", Kind: "bill", MonthlyAmountCents: 50000, OwnerID: "member-1", Split: budget.Split{Method: "equal"}, DueDay: &day}}
	h.Goals = []budget.Goal{{ID: "goal", Name: "Savings", MonthlyAmountCents: 10000, OwnerID: "member-1"}}
	if err := budget.Validate(h); err != nil {
		t.Fatal(err)
	}
	if err := database.Seed(t.Context(), "household", h); err != nil {
		t.Fatal(err)
	}
	return database, h
}

func readTodos(t *testing.T, database *store.Store, householdID string) budget.TodoList {
	t.Helper()
	list, err := database.Todos(t.Context(), householdID, todoTestDate)
	if err != nil {
		t.Fatal(err)
	}
	return list
}

func todoKind(t *testing.T, list budget.TodoList, kind string) budget.TodoItem {
	t.Helper()
	for _, task := range list.Tasks {
		if task.Kind == kind {
			return task
		}
	}
	t.Fatalf("missing %s task", kind)
	return budget.TodoItem{}
}

func completeTodo(t *testing.T, database *store.Store, list budget.TodoList, task budget.TodoItem) budget.TodoList {
	t.Helper()
	input := budget.TodoUpdate{Date: list.Date, PlanRevision: list.PlanRevision, ExpectedCompleted: false, Completed: true}
	updated, err := database.UpdateTodo(t.Context(), "household", task.ID, input, todoTestDate)
	if err != nil {
		t.Fatal(err)
	}
	return updated
}

func TestTodoCheckboxPersistsWithoutChangingPlanAndStaysHouseholdScoped(t *testing.T) {
	database, h := todoStore(t)
	list := readTodos(t, database, "household")
	task := todoKind(t, list, "expense")
	completeTodo(t, database, list, task)
	if !todoKind(t, readTodos(t, database, "household"), "expense").Completed {
		t.Fatal("completion did not persist")
	}
	saved, err := database.Read(t.Context(), "household")
	if err != nil || !reflect.DeepEqual(saved, h) {
		t.Fatalf("checkbox changed financial plan or revision: %v", err)
	}
	if err := database.Seed(t.Context(), "other-household", h); err != nil {
		t.Fatal(err)
	}
	if todoKind(t, readTodos(t, database, "other-household"), "expense").Completed {
		t.Fatal("completion crossed household boundaries")
	}
	input := budget.TodoUpdate{Date: list.Date, PlanRevision: list.PlanRevision, ExpectedCompleted: true, Completed: false}
	if _, err := database.UpdateTodo(t.Context(), "household", task.ID, input, todoTestDate); err != nil {
		t.Fatal(err)
	}
	if todoKind(t, readTodos(t, database, "household"), "expense").Completed {
		t.Fatal("unchecking did not persist")
	}
}

func TestTodoCompletionClearsWhenTheObligationChanges(t *testing.T) {
	for _, change := range []string{"amount", "due date", "goal amount"} {
		t.Run(change, func(t *testing.T) {
			database, h := todoStore(t)
			list := readTodos(t, database, "household")
			completeTodo(t, database, list, todoKind(t, list, "expense"))
			completeTodo(t, database, list, todoKind(t, list, "goal"))
			kind := changeTodoObligation(&h, change)
			if _, err := database.Update(t.Context(), "household", h); err != nil {
				t.Fatal(err)
			}
			if todoKind(t, readTodos(t, database, "household"), kind).Completed {
				t.Fatal("changed obligation retained its completion")
			}
		})
	}
}

func changeTodoObligation(h *budget.Household, change string) string {
	switch change {
	case "amount":
		h.Expenses[0].MonthlyAmountCents++
	case "due date":
		day := budget.DayOfMonth(16)
		h.Expenses[0].DueDay = &day
	case "goal amount":
		h.Goals[0].MonthlyAmountCents++
		return "goal"
	}
	return "expense"
}

func TestTodoCompletionSurvivesNamesAndUnrelatedPlanChanges(t *testing.T) {
	database, h := todoStore(t)
	list := readTodos(t, database, "household")
	completeTodo(t, database, list, todoKind(t, list, "expense"))
	h.Expenses[0].Name = "Renamed bill"
	h.Goals[0].MonthlyAmountCents++
	h.Members[0].MonthlyTakeHomeCents = 100000
	if _, err := database.Update(t.Context(), "household", h); err != nil {
		t.Fatal(err)
	}
	if !todoKind(t, readTodos(t, database, "household"), "expense").Completed {
		t.Fatal("unrelated revision or display name cleared completion")
	}
}

func TestTodoUpdateRejectsOldRevisionDeletedSourcesAndUnknownIDs(t *testing.T) {
	database, h := todoStore(t)
	list := readTodos(t, database, "household")
	task := todoKind(t, list, "expense")
	h.Expenses = []budget.Expense{}
	saved, err := database.Update(t.Context(), "household", h)
	if err != nil {
		t.Fatal(err)
	}
	input := budget.TodoUpdate{Date: list.Date, PlanRevision: list.PlanRevision, Completed: true}
	if _, err := database.UpdateTodo(t.Context(), "household", task.ID, input, todoTestDate); !errors.Is(err, store.ErrConflict) {
		t.Fatalf("old source snapshot returned %v", err)
	}
	input.PlanRevision = saved.Revision
	for _, id := range []string{task.ID, "member-2:expense:unowned:2026-11-15"} {
		if _, err := database.UpdateTodo(t.Context(), "household", id, input, todoTestDate); !errors.Is(err, store.ErrTodoUnavailable) {
			t.Fatalf("unavailable source returned %v", err)
		}
	}
}
