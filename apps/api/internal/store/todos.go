package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"bankpiggy/api/internal/budget"
	"github.com/jackc/pgx/v5"
)

// ErrTodoUnavailable rejects a task absent from the current saved plan and period.
var ErrTodoUnavailable = errors.New("this task is no longer available")

// ErrTodoConflict rejects a toggle based on a checkbox state another request changed.
var ErrTodoConflict = errors.New("this task changed; reload before saving")

const queryReadTodoStates = `
 SELECT task_id,completed,task_signature FROM todo_completions WHERE household_id=$1 AND task_id=ANY($2)
`

const querySetTodoState = `
 INSERT INTO todo_completions (household_id,member_id,task_id,completed,task_signature) VALUES ($1,$2,$3,$4,$5)
 ON CONFLICT (household_id,member_id,task_id) DO UPDATE SET completed=EXCLUDED.completed,task_signature=EXCLUDED.task_signature
`

// Todos combines a consistent plan snapshot with its shared persisted checkboxes.
func (s *Store) Todos(ctx context.Context, householdID string, onDate time.Time) (budget.TodoList, error) {
	var list budget.TodoList
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return list, err
	}
	defer rollback(tx)
	var data []byte
	if err := tx.QueryRow(ctx, queryReadHousehold, householdID).Scan(&data); err != nil {
		return list, err
	}
	var h budget.Household
	if err := json.Unmarshal(data, &h); err != nil {
		return list, err
	}
	list = budget.Todos(h, onDate)
	if err := applyTodoStates(ctx, tx, householdID, &list); err != nil {
		return list, err
	}
	return list, tx.Commit(ctx)
}

// UpdateTodo validates the current source and expected checkbox state under the plan lock.
func (s *Store) UpdateTodo(ctx context.Context, householdID, taskID string, input budget.TodoUpdate, onDate time.Time) (budget.TodoList, error) {
	var list budget.TodoList
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return list, err
	}
	defer rollback(tx)
	h, err := lockPlan(ctx, tx, householdID)
	if err != nil {
		return list, err
	}
	if h.Revision != input.PlanRevision {
		return list, ErrConflict
	}
	list = budget.Todos(h, onDate)
	if err := applyTodoStates(ctx, tx, householdID, &list); err != nil {
		return list, err
	}
	if err := setTodoState(ctx, tx, householdID, taskID, input, &list); err != nil {
		return list, err
	}
	return list, tx.Commit(ctx)
}

func applyTodoStates(ctx context.Context, tx pgx.Tx, householdID string, list *budget.TodoList) error {
	ids := make([]string, len(list.Tasks))
	for index, task := range list.Tasks {
		ids[index] = task.ID
	}
	rows, err := tx.Query(ctx, queryReadTodoStates, householdID, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	type completion struct {
		completed bool
		signature string
	}
	states := map[string]completion{}
	for rows.Next() {
		var id string
		var state completion
		if err := rows.Scan(&id, &state.completed, &state.signature); err != nil {
			return err
		}
		states[id] = state
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for index, task := range list.Tasks {
		signature, err := todoSignature(task)
		if err != nil {
			return err
		}
		state := states[task.ID]
		list.Tasks[index].Completed = state.completed && state.signature == signature
	}
	return nil
}

func setTodoState(ctx context.Context, tx pgx.Tx, householdID, taskID string, input budget.TodoUpdate, list *budget.TodoList) error {
	for index, task := range list.Tasks {
		if task.ID != taskID {
			continue
		}
		if task.Completed != input.ExpectedCompleted && task.Completed != input.Completed {
			return ErrTodoConflict
		}
		if task.Completed == input.Completed {
			return nil
		}
		signature, err := todoSignature(task)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, querySetTodoState, householdID, task.MemberID, task.ID, input.Completed, signature); err != nil {
			return err
		}
		list.Tasks[index].Completed = input.Completed
		return nil
	}
	return ErrTodoUnavailable
}

// todoSignature retains acknowledgement only while the generated obligation is unchanged.
func todoSignature(task budget.TodoItem) (string, error) {
	task.Name = ""
	task.Completed = false
	data, err := json.Marshal(task)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
