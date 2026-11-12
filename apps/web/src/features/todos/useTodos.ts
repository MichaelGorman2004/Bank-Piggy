import { useCallback, useEffect, useRef, useState } from 'react';
import type { TodoItem, TodoList } from '@bank-piggy/contracts';
import { ApiError } from '../../api';
import { errorMessage } from '../session/errors';
import { loadTodos, saveTodo } from './api';

type TodoFailure = { readonly date: string; readonly revision: number; readonly message: string };

/**
 * Serializes checkbox writes and prevents older responses from replacing newer results.
 * @param date Selected local date.
 * @param revision Saved household-plan revision.
 * @param csrfToken Token owned by the authenticated session.
 * @returns Tasks, request feedback, and completion and reload actions.
 */
export function useTodos(date: string, revision: number, csrfToken: string) {
  const state = useTodoData(date, revision, csrfToken);
  const [saving, setSaving] = useState(false);
  const inFlight = useRef(false);
  const complete = useTodoCompletion({
    date,
    csrfToken,
    data: state.data,
    setData: state.setData,
    setError: state.setError,
    setSaving,
    generation: state.generation,
    inFlight,
  });
  return {
    data: state.data,
    loading: state.loading,
    error: state.error,
    reload: state.reload,
    saving,
    complete,
  };
}

/**
 * Keeps loaded tasks and failure feedback attached to their date and plan revision.
 * @param date Selected local date.
 * @param revision Current household-plan revision.
 * @param csrfToken Current session request identity.
 * @returns Scoped tasks, request feedback, ownership, and explicit reload action.
 */
function useTodoData(date: string, revision: number, csrfToken: string) {
  const [data, setData] = useState<TodoList | null>(null);
  const [loading, setLoading] = useState(true);
  const [failure, setFailure] = useState<TodoFailure | null>(null);
  const generation = useRef(0);
  const error = failure?.date === date && failure.revision === revision ? failure.message : null;
  const read = useCallback(() => {
    const version = ++generation.current;
    return loadTodos(date)
      .then((next) => {
        if (version === generation.current) {
          setData(next);
          setFailure(null);
        }
      })
      .catch((caught: unknown) => {
        if (version === generation.current)
          setFailure({ date, revision, message: errorMessage(caught) });
      })
      .finally(() => {
        if (version === generation.current) setLoading(false);
      });
  }, [date, revision]);
  useEffect(() => {
    void read();
    return () => {
      generation.current += 1;
    };
  }, [read, csrfToken]);
  const current = data?.date === date && data.planRevision >= revision ? data : null;
  return {
    data: current,
    setData,
    generation,
    loading: loading || (!current && !error),
    error,
    setError: (message: string | null) => {
      setFailure(message ? { date, revision, message } : null);
    },
    reload: async () => {
      setLoading(true);
      setFailure(null);
      await read();
    },
  };
}

interface CompletionOptions {
  readonly date: string;
  readonly csrfToken: string;
  readonly data: TodoList | null;
  readonly setData: (value: TodoList) => void;
  readonly setError: (value: string | null) => void;
  readonly setSaving: (value: boolean) => void;
  readonly generation: { current: number };
  readonly inFlight: { current: boolean };
}

/**
 * Saves a deliberate checkbox change without replaying it against a changed plan.
 * @param options Current list, session, generation, and feedback setters.
 * @returns Completion action with conflict refresh and visible failure feedback.
 */
function useTodoCompletion(options: CompletionOptions) {
  return async (task: TodoItem, completed: boolean) => {
    if (options.inFlight.current || !options.data || options.data.date !== options.date) return;
    options.inFlight.current = true;
    options.setSaving(true);
    options.setError(null);
    const version = ++options.generation.current;
    try {
      const next = await saveTodo(
        task.id,
        {
          date: options.date,
          planRevision: options.data.planRevision,
          expectedCompleted: task.completed,
          completed,
        },
        options.csrfToken,
      );
      if (version === options.generation.current) options.setData(next);
    } catch (failure) {
      await refreshConflict(options, failure, version);
    } finally {
      options.inFlight.current = false;
      options.setSaving(false);
    }
  };
}

/**
 * Refreshes a conflicting task while preserving the user's decision for a new explicit check.
 * @param options Current list and state setters.
 * @param failure Failed write result.
 * @param version Generation that owns the failed write.
 */
async function refreshConflict(options: CompletionOptions, failure: unknown, version: number) {
  if (version !== options.generation.current) return;
  if (!(failure instanceof ApiError) || failure.status !== 409) {
    options.setError(errorMessage(failure));
    return;
  }
  try {
    const next = await loadTodos(options.date);
    if (version !== options.generation.current) return;
    options.setData(next);
    options.setError(
      'This task or plan changed in another window. The list is updated. Check the amount and choose the checkbox again.',
    );
  } catch (refreshFailure) {
    if (version === options.generation.current)
      options.setError(
        `The task changed and the list could not refresh. ${errorMessage(refreshFailure)}`,
      );
  }
}
