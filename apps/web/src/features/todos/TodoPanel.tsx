import type { Household } from '@bank-piggy/contracts';
import { FinanceCard } from '../../components/FinanceCard';
import { HOUSEHOLD_SCOPE } from '../plan/format';
import { useTodos } from './useTodos';
import { TodoGroup } from './TodoRows';
import { unscheduledExpensesForScope } from './todoWarnings';
import styles from './Todos.module.css';
import dashboardStyles from '../../components/Dashboard.module.css';

interface TodoPanelProps {
  readonly household: Household;
  readonly scope: string;
  readonly date: string;
  readonly csrfToken: string;
  readonly onPlan: () => void;
  readonly onExpenses: () => void;
}
interface TodoStateProps {
  readonly state: ReturnType<typeof useTodos>;
}
interface TodoLoadedProps extends TodoPanelProps, TodoStateProps {}
interface UnscheduledProps {
  readonly count: number;
  readonly onExpenses: () => void;
}

/**
 * Groups each member's pay-period tasks in one independently scrolling card.
 * @param props Saved people, selected date and scope, and authorized edit actions.
 * @returns Pay-period tasks with persistent completion controls.
 */
export function TodoPanel(props: TodoPanelProps) {
  const state = useTodos(props.date, props.household.revision, props.csrfToken);
  return (
    <FinanceCard
      title="This pay period"
      description="Pay dates follow the entered schedule. Weekends, holidays, and employer changes are not adjusted."
      actions={null}
      footer={
        <p className={styles.note}>
          Checking a task records completion. It does not move money or update balances or goal
          progress.
        </p>
      }
    >
      <TodoFeedback state={state} />
      {!state.loading && state.data && <TodoLoaded {...props} state={state} />}
      <span className={styles.note} role="status">
        {state.saving ? 'Saving completion…' : ''}
      </span>
    </FinanceCard>
  );
}

/**
 * Makes failed loads and writes visible without replacing the saved checkbox state.
 * @param props Current To-do request feedback and retry action.
 * @returns Loading or error messages within the card body.
 */
function TodoFeedback(props: TodoStateProps) {
  const state = props.state;
  return (
    <>
      {state.error && (
        <div className={styles.error} role="alert">
          <p>{state.error}</p>
          <button
            className={dashboardStyles.textButton}
            disabled={state.loading || state.saving}
            onClick={() => {
              void state.reload();
            }}
          >
            Reload tasks
          </button>
        </div>
      )}
      {state.loading && (
        <p className={styles.message} role="status">
          Loading pay-period tasks…
        </p>
      )}
    </>
  );
}

/**
 * Filters authorized server tasks and incomplete schedules by the selected scope.
 * @param props Household records, selected scope, and loaded task state.
 * @returns Member periods, missing-schedule guidance, and manual-payment guidance.
 */
function TodoLoaded(props: TodoLoadedProps) {
  const data = props.state.data;
  if (!data) return null;
  const members = props.household.members.filter(
    (member) => props.scope === HOUSEHOLD_SCOPE || member.id === props.scope,
  );
  const periods = data.periods.filter((period) =>
    members.some((member) => member.id === period.memberId),
  );
  const missing = members.filter(
    (member) => !periods.some((period) => period.memberId === member.id),
  );
  const unscheduled = unscheduledExpensesForScope(
    props.household.expenses,
    data.unscheduledExpenseIds,
    props.scope,
  );
  return (
    <>
      {missing.map((member) => (
        <div key={member.id} className={styles.message}>
          <p>
            {member.paySchedule
              ? `No pay period is available for ${member.name} on this date. Check the As-of date or recorded schedule.`
              : `Add a regular pay frequency and pay date for ${member.name} to start a pay-period list.`}
          </p>
          <button className={dashboardStyles.textButton} onClick={props.onPlan}>
            Edit pay plan
          </button>
        </div>
      ))}
      <UnscheduledPayments count={unscheduled.length} onExpenses={props.onExpenses} />
      {periods.map((period) => (
        <TodoGroup
          key={period.memberId}
          period={period}
          tasks={data.tasks.filter((task) => task.memberId === period.memberId)}
          busy={props.state.saving}
          onComplete={props.state.complete}
        />
      ))}
      <p className={styles.note}>
        Monthly goal contributions are divided across pay dates in that calendar month.
      </p>
    </>
  );
}

/**
 * Identifies manual payments that need an explicit due day before becoming tasks.
 * @param props Count in the selected scope and the expense-management action.
 * @returns A direct prompt to add missing due days.
 */
function UnscheduledPayments(props: UnscheduledProps) {
  if (props.count === 0) return null;
  return (
    <div className={styles.message}>
      <p>
        {String(props.count)} manual {props.count === 1 ? 'payment needs' : 'payments need'} a due
        day for To-do setup.
      </p>
      <button className={dashboardStyles.textButton} onClick={props.onExpenses}>
        Edit household expenses
      </button>
    </div>
  );
}
