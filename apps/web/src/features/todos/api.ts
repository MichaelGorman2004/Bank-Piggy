import type { TodoList, TodoUpdate } from '@bank-piggy/contracts';
import { contractValidator, request } from '../../api';

const isTodoList = contractValidator<TodoList>('TodoList');

/**
 * Reads tasks from the pay period that contains the selected local date.
 * @param date Local calendar date in YYYY-MM-DD form.
 * @returns The authorized household tasks and pay periods.
 */
export function loadTodos(date: string): Promise<TodoList> {
  return request(`/api/todos?date=${encodeURIComponent(date)}`, isTodoList);
}

/**
 * Changes one completion only when its plan and prior checkbox state still match.
 * @param id Opaque task identifier supplied by the server.
 * @param update Selected date, plan revision, and intended completion.
 * @param csrfToken Server-issued token for the session's writes.
 * @returns The updated tasks for the selected date.
 */
export function saveTodo(id: string, update: TodoUpdate, csrfToken: string): Promise<TodoList> {
  return request(`/api/todos/${encodeURIComponent(id)}`, isTodoList, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
    body: JSON.stringify(update),
  });
}
