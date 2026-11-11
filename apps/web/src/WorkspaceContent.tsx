import type { HouseholdResponse, Member, Summary } from '@bank-piggy/contracts';
import { useMemo } from 'react';
import { Dashboard } from './components/Dashboard';
import { DashboardHeader } from './components/DashboardHeader';
import { PlanningMonth } from './components/PlanningMonth';
import { HOUSEHOLD_SCOPE, periodFactor, type Period } from './features/plan/format';
import { PlanCompleteness, ReloadNotice } from './WorkspaceChrome';
import type { WorkspaceProps } from './Workspace';
import type { useWorkspace } from './useWorkspace';
import type { ActiveDialog } from './features/plan/PlanDialogs';
import { VIEW_DETAILS, type View } from './features/plan/views';
import styles from './Workspace.module.css';

interface WorkspaceContentProps {
  readonly app: WorkspaceProps;
  readonly ui: ReturnType<typeof useWorkspace>;
}

/**
 * Prevents stale paycheck selection after the user changes to irregular pay.
 * @param requested Current period selection.
 * @param member Selected member, if any.
 * @returns A period supported by the current scope and pay frequency.
 */
function displayPeriod(requested: Period, member: Member | undefined): Period {
  if (requested !== 'paycheck') return requested;
  return member && member.payFrequency !== 'irregular' ? 'paycheck' : 'monthly';
}

/**
 * Applies one scope and period factor to all dashboard totals and charts.
 * @param root0 Component values and event handlers.
 * @param root0.app Value used by this operation.
 * @param root0.ui Value used by this operation.
 * @returns Rendered interface.
 */
export function WorkspaceContent({ app, ui }: WorkspaceContentProps) {
  const member = app.data.household.members.find((item) => item.id === ui.scope);
  const period = displayPeriod(ui.period, member);
  const yearlyData = useMemo(() => ({ ...app.data, summary: app.data.yearlySummary }), [app.data]);
  const data = period === 'yearly' ? yearlyData : app.data;
  const summary =
    ui.scope === HOUSEHOLD_SCOPE ? data.summary.household : data.summary.members[ui.scope];
  const factor = periodFactor(period, member);
  return (
    <main id="main" className={styles.main} tabIndex={-1}>
      <WorkspaceFilters app={app} ui={ui} period={period} />
      <ReloadNotice conflict={app.conflict} error={app.error} onReload={app.onReload} />
      <PlanCompleteness
        incomeCents={app.data.summary.household.monthlyTakeHomeCents}
        expenseCount={app.data.household.expenses.length}
        onEdit={() => {
          ui.open({ type: 'plan' });
        }}
      />
      <WorkspaceView app={app} ui={ui} data={data} summary={summary} factor={factor} />
    </main>
  );
}

/**
 * Allocates the remaining workspace height to one task with card-owned scrolling.
 * @param props Current task, selected totals, and authorized editor actions.
 * @returns The active task, with balances and summary cards on the household plan.
 */
function WorkspaceView(
  props: WorkspaceContentProps & {
    readonly data: HouseholdResponse;
    readonly summary: Summary;
    readonly factor: number;
  },
) {
  const { ui, data, summary, factor } = props;
  return (
    <section
      key={ui.view}
      className={styles.view}
      aria-label={`${VIEW_DETAILS[ui.view].label} content`}
    >
      <Dashboard
        data={data}
        summary={summary}
        scope={ui.scope}
        factor={factor}
        view={ui.view}
        onPlan={() => {
          ui.open({ type: 'plan' });
        }}
        onExpense={(expense, initialKind) => {
          ui.open({
            type: 'expense',
            expense,
            initialKind: initialKind ?? 'bill',
          });
        }}
        onGoal={(goal) => {
          ui.open({ type: 'goal', goal, initialKind: ui.view === 'targets' ? 'trip' : 'goal' });
        }}
      />
    </section>
  );
}

/**
 * Keeps scope, reporting period, and calendar preview controls together.
 * @param props Workspace state and the supported reporting period.
 * @returns Header controls and month selector.
 */
function WorkspaceFilters(props: WorkspaceContentProps & { readonly period: Period }) {
  const { app, ui, period } = props;
  return (
    <div className={styles.filters}>
      <div className={styles.headingArea}>
        <DashboardHeader
          session={app.session}
          members={app.data.household.members}
          scope={ui.scope}
          period={period}
          view={ui.view}
          onScope={ui.changeScope}
          onPeriod={ui.setPeriod}
          onAdd={() => {
            ui.open(newRecordDialog(ui.view));
          }}
        />
      </div>
      <div className={styles.monthArea}>
        <PlanningMonth
          month={app.data.planningMonth}
          period={period}
          disabled={app.saving}
          loading={app.loading}
          onMonth={app.onMonth}
        />
      </div>
    </div>
  );
}

/**
 * Opens the editor that belongs to the active task without creating a saved record.
 * @param view Selected finance task.
 * @returns A draft dialog with the appropriate new-record defaults.
 */
function newRecordDialog(view: View): ActiveDialog {
  if (view === 'overview' || view === 'flow') return { type: 'plan' };
  if (view === 'goals' || view === 'targets')
    return { type: 'goal', initialKind: view === 'targets' ? 'trip' : 'goal' };
  return { type: 'expense', initialKind: 'bill' };
}
