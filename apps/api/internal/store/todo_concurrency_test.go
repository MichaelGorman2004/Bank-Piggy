package store_test

import (
	"errors"
	"sync"
	"testing"

	"bankpiggy/api/internal/budget"
	"bankpiggy/api/internal/store"
)

func TestConcurrentTodoChecksAreIdempotentAndStoredOnce(t *testing.T) {
	database, _ := todoStore(t)
	list := readTodos(t, database, "household")
	task := todoKind(t, list, "expense")
	input := budget.TodoUpdate{Date: list.Date, PlanRevision: list.PlanRevision, Completed: true}
	start := make(chan struct{})
	results := make(chan error, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	for index := 0; index < 2; index++ {
		go func() {
			ready.Done()
			<-start
			_, err := database.UpdateTodo(t.Context(), "household", task.ID, input, todoTestDate)
			results <- err
		}()
	}
	ready.Wait()
	close(start)
	for index := 0; index < 2; index++ {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	var count int
	if err := database.Pool.QueryRow(t.Context(), "SELECT count(*) FROM todo_completions").Scan(&count); err != nil || count != 1 {
		t.Fatalf("concurrent idempotent checks stored %d records: %v", count, err)
	}
	input.ExpectedCompleted = false
	input.Completed = false
	if _, err := database.UpdateTodo(t.Context(), "household", task.ID, input, todoTestDate); !errors.Is(err, store.ErrTodoConflict) {
		t.Fatalf("stale expected state returned %v", err)
	}
}
