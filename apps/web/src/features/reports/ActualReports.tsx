import { lazy, Suspense } from 'react';
import type { ActivityResponse, Expense, Goal, HouseholdResponse } from '@bank-piggy/contracts';
import { FinanceCard } from '../../components/FinanceCard';
import { ChartBoundary } from '../../components/ChartBoundary';
import { useActivityData } from '../activity/ActivityContext';
import { monthLabel } from '../plan/month';
import { ExpenseActuals } from './ExpenseActuals';
import { InvestmentActuals } from './InvestmentActuals';
import { currentReport, type ActualView } from './reporting';
import styles from './Reports.module.css';

const ActualMoneyFlow = lazy(() =>
  import('../../components/ActualMoneyFlow').then((module) => ({
    default: module.ActualMoneyFlow,
  })),
);
const TITLES: Readonly<Record<ActualView, string>> = {
  flow: 'Recorded money flow',
  expenses: 'Monthly spending',
  goals: 'Investment contributions',
};

interface ActualReportsProps {
  readonly view: ActualView;
  readonly plan: HouseholdResponse;
  readonly scope: string;
  readonly onExpense: (expense: Expense) => void;
  readonly onGoal: (goal: Goal) => void;
  readonly onReview: () => void;
}

/** Presents one server-owned actual projection across flow, expenses, and investments.
 * @param props Current saved plan and the selected monthly report.
 * @returns A bounded report with stale-data guards and explicit coverage feedback.
 */
export function ActualReports(props: ActualReportsProps) {
  const state = useActivityData();
  const { plan, scope } = props;
  const selection = { month: plan.planningMonth, scope, revision: plan.household.revision };
  const current = currentReport(state.data, selection, state.totalsStale);
  return (
    <div className={styles.page}>
      <FinanceCard
        title={TITLES[props.view]}
        description={`${monthLabel(plan.planningMonth)} · Selected accounts · Recorded monthly activity`}
        actions={
          <button
            type="button"
            className={styles.action}
            aria-label="Reload recorded activity"
            disabled={state.loading || state.busy}
            onClick={() => {
              void state.reload();
            }}
          >
            {state.loading ? 'Loading…' : 'Reload'}
          </button>
        }
        footer={<ActualReportFooter onReview={props.onReview} />}
      >
        {state.error && (
          <p className={styles.error} role="alert">
            {state.error} Your saved plan and unsaved review choices are kept.
          </p>
        )}
        {state.loading && (
          <p className={styles.note} role="status">
            Reading saved monthly activity…
          </p>
        )}
        {current ? (
          <ActualReportContent {...props} data={current} />
        ) : (
          <p className={styles.empty}>
            The current monthly report is unavailable until activity and the saved plan are read
            together. Reload activity to update it.
          </p>
        )}
      </FinanceCard>
    </div>
  );
}

/** Keeps activity access reachable outside the scrolling report body.
 * @param props Explicit navigation to transaction review and source settings.
 * @param props.onReview Opens the existing Activity workspace.
 * @returns The saved-data cadence and persistent review action.
 */
function ActualReportFooter(props: { readonly onReview: () => void }) {
  return (
    <div className={styles.footer}>
      <p className={styles.note}>Saved activity checked every 30 minutes.</p>
      <button type="button" className={styles.action} onClick={props.onReview}>
        Activity & matches
      </button>
    </div>
  );
}

/** Selects presentation only; every monetary amount comes from the same recorded snapshot.
 * @param props Coherent monthly actuals and explicit review/edit actions.
 * @param props.data Current month, scope, saved plan revision, and full-ledger comparisons.
 * @returns Recorded cash flow or matched expense/investment comparisons.
 */
function ActualReportContent(props: ActualReportsProps & { readonly data: ActivityResponse }) {
  return (
    <>
      <p className={styles.notice}>
        {props.data.notice}{' '}
        {props.data.coverage === 'incomplete'
          ? 'Recorded amounts are partial; missing history is not a confirmed zero.'
          : ''}
      </p>
      {props.data.environment === 'sandbox' && (
        <p className={styles.notice}>Plaid Sandbox · Sample data.</p>
      )}
      {props.scope !== 'household' && (
        <p className={styles.note}>
          Matched activity uses the split saved with that match. Unmatched activity follows account
          ownership.
        </p>
      )}
      {props.view === 'flow' ? (
        <ChartBoundary>
          <Suspense fallback={<p role="status">Loading recorded money flow…</p>}>
            <ActualMoneyFlow data={props.data} />
          </Suspense>
        </ChartBoundary>
      ) : props.view === 'expenses' ? (
        <ExpenseActuals
          data={props.data}
          household={props.plan.household}
          onExpense={props.onExpense}
          onReview={props.onReview}
        />
      ) : (
        <InvestmentActuals
          data={props.data}
          household={props.plan.household}
          onGoal={props.onGoal}
          onReview={props.onReview}
        />
      )}
    </>
  );
}
