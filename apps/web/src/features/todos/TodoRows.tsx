import type { PayPeriod, TodoItem } from '@bank-piggy/contracts';
import { money } from '../plan/format';
import styles from './Todos.module.css';

interface TodoRowsProps {
  readonly tasks: readonly TodoItem[];
  readonly busy: boolean;
  readonly onComplete: (task: TodoItem, completed: boolean) => Promise<void>;
}
interface TodoRowProps extends Omit<TodoRowsProps, 'tasks'> {
  readonly task: TodoItem;
}
interface TodoGroupProps extends TodoRowsProps {
  readonly period: PayPeriod;
}

/**
 * Formats a calendar date without a timezone shift.
 * @param date Canonical local calendar date.
 * @returns A short readable date with its year.
 */
function dateLabel(date: string): string {
  return new Date(`${date}T12:00:00`).toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  });
}

/**
 * Names one member's entered pay-period cadence and its tasks.
 * @param props Server-calculated period, tasks, and completion action.
 * @returns A named group with visible period boundaries and expected next pay.
 */
export function TodoGroup(props: TodoGroupProps) {
  return (
    <section className={styles.group} aria-label={`${props.period.memberName}’s pay-period tasks`}>
      <div className={styles.period}>
        <h3>{props.period.memberName}</h3>
        <p>
          {dateLabel(props.period.startDate)} – {dateLabel(props.period.dueDate)}
        </p>
        <small>Expected next pay: {dateLabel(props.period.nextPayDate)}</small>
      </div>
      <TodoRows tasks={props.tasks} busy={props.busy} onComplete={props.onComplete} />
    </section>
  );
}

/**
 * Keeps keyboard focus during serialized saves without changing funds or goal progress.
 * @param props Task amount, state, and completion action.
 * @returns A native checkbox with a visible due date and completed text.
 */
function TodoRow(props: TodoRowProps) {
  const { task } = props;
  return (
    <label className={styles.task} data-completed={task.completed} data-busy={props.busy}>
      <input
        type="checkbox"
        checked={task.completed}
        aria-disabled={props.busy}
        onChange={(event) => {
          if (props.busy) return;
          void props.onComplete(task, event.target.checked);
        }}
      />
      <span className={styles.taskText}>
        <strong>{task.name}</strong>
        <small>
          Due {dateLabel(task.dueDate)} ·{' '}
          {task.kind === 'goal' ? 'Goal contribution' : 'Manual payment'}
        </small>
      </span>
      <span className={styles.amount}>{money(task.amountCents)}</span>
    </label>
  );
}

/**
 * Keeps an empty period clear without adding automatic subscriptions.
 * @param props Period's tasks and serialized completion action.
 * @returns Task rows or an empty-period message.
 */
function TodoRows(props: TodoRowsProps) {
  if (props.tasks.length === 0)
    return (
      <p className={styles.message}>
        No manual payments or goal contributions are due in this pay period. Automatic payments stay
        in Expenses.
      </p>
    );
  return (
    <div>
      {props.tasks.map((task) => (
        <TodoRow key={task.id} task={task} busy={props.busy} onComplete={props.onComplete} />
      ))}
    </div>
  );
}
