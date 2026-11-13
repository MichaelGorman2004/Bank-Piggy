import { lazy, Suspense } from 'react';
import type { Expense, Goal, HouseholdResponse, Summary } from '@bank-piggy/contracts';
import { ExpenseList } from './ExpenseList';
import { ChartBoundary } from './ChartBoundary';
import { GoalPanel } from './GoalPanel';
import { HouseholdOverview } from './HouseholdOverview';
import type { View } from '../features/plan/views';
import styles from './Dashboard.module.css';

const MoneyFlow = lazy(() =>
  import('./MoneyFlow').then((module) => ({ default: module.MoneyFlow })),
);

interface DashboardProps {
  readonly data: HouseholdResponse;
  readonly summary: Summary;
  readonly scope: string;
  readonly factor: number;
  readonly view: Exclude<View, 'accounts' | 'todos'>;
  readonly onExpense: (expense?: Expense, initialKind?: Expense['kind']) => void;
  readonly onGoal: (goal?: Goal) => void;
  readonly onPlan: () => void;
}

/**
 * Shows the money flow for the selected scope and reporting period.
 * @param root0 Component values and event handlers.
 * @param root0.data Validated household response.
 * @param root0.summary Server-calculated totals.
 * @param root0.scope Selected household or member identifier.
 * @param root0.factor Scale for the selected estimated period.
 * @returns Rendered interface.
 */
function FlowPanel({
  data,
  summary,
  scope,
  factor,
}: Omit<DashboardProps, 'view' | 'onExpense' | 'onGoal' | 'onPlan'>) {
  return (
    <section className={styles.flowPanel} aria-label="Planned money flow">
      <div className={styles.flowBody} role="region" aria-label="Money flow diagram" tabIndex={0}>
        <ChartBoundary>
          <Suspense fallback={<p role="status">Loading money flow…</p>}>
            <MoneyFlow data={data} summary={summary} scope={scope} factor={factor} />
          </Suspense>
        </ChartBoundary>
      </div>
    </section>
  );
}

/**
 * Selects a focused view while keeping calculations shared across all views.
 * @param props Component values and event handlers.
 * @returns Rendered interface.
 */
export function Dashboard(props: DashboardProps) {
  const { data, summary, scope, factor, view, onExpense, onGoal, onPlan } = props;
  if (view === 'expenses')
    return (
      <div className={styles.focused}>
        <ExpenseList
          data={data}
          scope={scope}
          factor={factor}
          onEdit={onExpense}
          onAdd={(initialKind) => {
            onExpense(undefined, initialKind);
          }}
        />
      </div>
    );
  if (view === 'goals' || view === 'targets')
    return (
      <div className={styles.focused}>
        <GoalPanel
          data={data}
          summary={summary}
          scope={scope}
          factor={factor}
          view={view}
          onEdit={onGoal}
          onAdd={() => {
            onGoal();
          }}
        />
      </div>
    );
  if (view === 'flow')
    return (
      <div className={styles.home}>
        <FlowPanel data={data} summary={summary} scope={scope} factor={factor} />
      </div>
    );
  return (
    <HouseholdOverview
      household={data.household}
      payrollSummaries={data.payrollSummaries}
      householdIncomeCents={data.summary.household.monthlyTakeHomeCents}
      scope={scope}
      summary={summary}
      factor={factor}
      onEdit={onPlan}
    />
  );
}
