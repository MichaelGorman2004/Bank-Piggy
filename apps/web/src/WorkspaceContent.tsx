import type { HouseholdResponse, Member, Summary } from '@bank-piggy/contracts';
import { useMemo } from 'react';
import { TodoPanel } from './features/todos/TodoPanel';
import { TodoDateControl } from './features/todos/TodoDateControl';
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
import { AccountsDashboard } from './features/connections/AccountsDashboard';
import { ActivityDashboard } from './features/activity/ActivityDashboard';
import { ActualReports } from './features/reports/ActualReports';
import { ReportModeControl } from './features/reports/ReportMode';
import { reportingPeriod, supportsActuals } from './features/reports/reporting';

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
  const period = reportingPeriod(ui.view, ui.reportMode, displayPeriod(ui.period, member));
  const yearlyData = useMemo(() => ({ ...app.data, summary: app.data.yearlySummary }), [app.data]);
  const data = period === 'yearly' ? yearlyData : app.data;
  const summary =
    ui.scope === HOUSEHOLD_SCOPE ? data.summary.household : data.summary.members[ui.scope];
  const factor = periodFactor(period, member);
  return (
    <main id="main" className={styles.main} tabIndex={-1}>
      <WorkspaceFilters app={app} ui={ui} period={period} />
      <ReloadNotice conflict={app.conflict} error={app.error} onReload={app.onReload} />
      {!['todos', 'accounts', 'activity', 'overview'].includes(ui.view) &&
        !(ui.reportMode === 'actual' && supportsActuals(ui.view)) && (
          <PlanCompleteness
            incomeCents={app.data.summary.household.monthlyTakeHomeCents}
            expenseCount={app.data.household.expenses.length}
            onEdit={() => {
              ui.open({ type: 'plan' });
            }}
          />
        )}
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
  return (
    <section
      key={props.ui.view}
      className={styles.view}
      aria-label={`${VIEW_DETAILS[props.ui.view].label} content`}
    >
      <WorkspaceTask {...props} />
    </section>
  );
}

/**
 * Chooses pay-period tasks or the shared financial dashboard.
 * @param props Selected view, canonical totals, and editor actions.
 * @returns The bounded task content.
 */
function WorkspaceTask(
  props: WorkspaceContentProps & {
    readonly data: HouseholdResponse;
    readonly summary: Summary;
    readonly factor: number;
  },
) {
  const { ui, data } = props;
  if (ui.view === 'accounts') return <AccountsDashboard household={data.household} />;
  if (ui.view === 'activity') return <WorkspaceActivity app={props.app} ui={ui} />;
  if (ui.reportMode === 'actual' && supportsActuals(ui.view))
    return (
      <ActualReports
        view={ui.view}
        plan={props.app.data}
        scope={ui.scope}
        onActivity={() => {
          ui.setActivitySection('summary');
          ui.setView('activity');
        }}
      />
    );
  return ui.view === 'todos' ? (
    <TodoPanel
      household={data.household}
      scope={ui.scope}
      date={ui.todoDate}
      csrfToken={props.app.session.csrfToken ?? ''}
      onPlan={() => {
        ui.open({ type: 'plan' });
      }}
      onExpenses={() => {
        ui.changeScope(HOUSEHOLD_SCOPE);
        ui.setView('expenses');
      }}
    />
  ) : (
    <WorkspacePlannedDashboard {...props} view={ui.view} />
  );
}

/** Binds planned views to their existing saved-record editors.
 * @param props Selected forecast and a view that supports its saved plan.
 * @param props.view Current planned dashboard task.
 * @returns The planned dashboard with stable financial inputs and edit actions.
 */
function WorkspacePlannedDashboard(
  props: Parameters<typeof WorkspaceTask>[0] & {
    readonly view: Exclude<View, 'accounts' | 'activity' | 'todos'>;
  },
) {
  const { ui } = props;
  return (
    <Dashboard
      data={props.data}
      summary={props.summary}
      scope={ui.scope}
      factor={props.factor}
      view={props.view}
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
        ui.open({ type: 'goal', goal, initialKind: props.view === 'targets' ? 'trip' : 'goal' });
      }}
    />
  );
}

/**
 * Keeps scope, reporting period, and calendar preview controls together.
 * @param props Workspace state and the supported reporting period.
 * @returns Header controls and month selector.
 */
function WorkspaceFilters(props: WorkspaceContentProps & { readonly period: Period }) {
  const { app, ui, period } = props;
  const view = ui.view;
  if (view === 'accounts' || view === 'activity') return null;
  return (
    <div className={styles.filters}>
      <div className={styles.headingArea}>
        <DashboardHeader
          session={app.session}
          members={app.data.household.members}
          scope={ui.scope}
          period={period}
          actual={ui.reportMode === 'actual' && supportsActuals(view)}
          view={view}
          onScope={ui.changeScope}
          onPeriod={ui.setPeriod}
          onAdd={() => {
            ui.open(newRecordDialog(view));
          }}
        />
      </div>
      <div className={styles.monthArea}>
        {ui.view === 'todos' ? (
          <TodoDateControl date={ui.todoDate} onDate={ui.setTodoDate} />
        ) : (
          <PlanningMonth
            month={app.data.planningMonth}
            period={period}
            disabled={app.saving}
            loading={app.loading}
            onMonth={app.onMonth}
          />
        )}
        {supportsActuals(view) && (
          <ReportModeControl mode={ui.reportMode} onChange={ui.setReportMode} />
        )}
      </div>
    </div>
  );
}

/**
 * Opens the editor that belongs to the active task without creating a saved record.
 * @param view Selected finance task.
 * @returns A draft dialog with the appropriate new-record defaults.
 */
function newRecordDialog(view: Exclude<View, 'accounts' | 'activity'>): ActiveDialog {
  if (view === 'overview' || view === 'flow' || view === 'todos') return { type: 'plan' };
  if (view === 'goals' || view === 'targets')
    return { type: 'goal', initialKind: view === 'targets' ? 'trip' : 'goal' };
  return { type: 'expense', initialKind: 'bill' };
}

/** Uses the monthly Activity view without plan period multipliers.
 * @param props Saved workspace and shared scope.
 * @returns Actual monthly activity.
 */
function WorkspaceActivity(props: WorkspaceContentProps) {
  const { app, ui } = props;
  return (
    <ActivityDashboard
      household={app.data.household}
      section={ui.activitySection}
      onSection={ui.setActivitySection}
      session={app.session}
      month={app.data.planningMonth}
      scope={ui.scope}
      monthLoading={app.loading}
      onMonth={app.onMonth}
      onScope={ui.changeScope}
      onReload={app.onReload}
      onAccounts={() => {
        ui.setView('accounts');
      }}
    />
  );
}
