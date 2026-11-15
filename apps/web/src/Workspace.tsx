import type { Household, HouseholdResponse, Session } from '@bank-piggy/contracts';
import { SharingDialog } from './features/sharing/SharingDialog';
import { useWorkspaceSharing } from './useWorkspaceSharing';
import { Sidebar } from './components/Sidebar';
import { PlanDialogs } from './features/plan/PlanDialogs';
import { WorkspaceBar } from './WorkspaceChrome';
import { WorkspaceContent } from './WorkspaceContent';
import { useWorkspace } from './useWorkspace';
import styles from './Workspace.module.css';
import { plaidReturn } from './features/connections/oauthReturn';
import { ConnectionsProvider } from './features/connections/ConnectionsContext';
import { ActivityProvider } from './features/activity/ActivityContext';

export interface WorkspaceProps {
  readonly session: Session;
  readonly data: HouseholdResponse;
  readonly saving: boolean;
  readonly loading: boolean;
  readonly onMonth: (month: string) => Promise<boolean>;
  readonly saveError: string | null;
  readonly conflict: boolean;
  readonly error: string | null;
  readonly onSave: (household: Household) => Promise<boolean>;
  readonly onClearError: () => void;
  readonly onReload: () => void;
  readonly onSignout: () => void;
}

/**
 * Maintains the application shell around one persisted household plan.
 * @param props Component values and event handlers.
 * @returns Rendered interface.
 */
export function Workspace(props: WorkspaceProps) {
  const ui = useWorkspace({
    conflict: props.conflict || props.loading,
    onClearError: props.onClearError,
    blank: isBlankPlan(props.data) && !plaidReturn(),
  });
  const sharing = useWorkspaceSharing(props, ui);
  return (
    <ConnectionsProvider
      revision={props.data.household.revision}
      csrfToken={props.session.csrfToken ?? ''}
    >
      <ActivityProvider
        month={props.data.planningMonth}
        scope={ui.scope}
        revision={props.data.household.revision}
        onReloadPlan={props.onReload}
      >
        <WorkspaceLayout app={props} ui={ui} sharing={sharing} />
      </ActivityProvider>
    </ConnectionsProvider>
  );
}

interface WorkspaceLayoutProps {
  readonly app: WorkspaceProps;
  readonly ui: ReturnType<typeof useWorkspace>;
  readonly sharing: ReturnType<typeof useWorkspaceSharing>;
}

/** Keeps navigation and dialogs mounted while financial report content changes.
 * @param props Saved household, transient view state, and invitation actions.
 * @param props.app Saved household and persistence feedback.
 * @param props.ui Navigation, reporting mode, and editor state.
 * @param props.sharing Current invitation dialog and save actions.
 * @returns The bounded application frame inside its financial data providers.
 */
function WorkspaceLayout({ app, ui, sharing }: WorkspaceLayoutProps) {
  return (
    <div className={styles.app}>
      <a href="#main" className={styles.skip}>
        Skip to main content
      </a>
      <Sidebar
        session={app.session}
        members={app.data.household.members}
        view={ui.view}
        open={ui.mobileOpen}
        onView={ui.setView}
        onClose={ui.closeMobile}
        onSignout={app.onSignout}
        onShare={() => {
          void sharing.show();
        }}
      />
      <div className={styles.workspace}>
        <WorkspaceBar
          view={ui.view}
          saving={app.saving}
          mobileOpen={ui.mobileOpen}
          onMenu={() => {
            ui.setMobileOpen(true);
          }}
        />
        <WorkspaceContent app={app} ui={ui} />
      </div>
      <WorkspaceDialogs app={app} ui={ui} sharing={sharing} />
    </div>
  );
}

/**
 * Keeps plan and invitation dialogs within the household's shared balance source.
 * @param props Persisted household, editor state, and invitation actions.
 * @param props.app Saved household and persistence feedback.
 * @param props.ui Current editor and its close action.
 * @param props.sharing Household invitation and save actions.
 * @returns Household dialogs without replacing their editable drafts on balance refresh.
 */
function WorkspaceDialogs(props: WorkspaceLayoutProps) {
  return (
    <>
      <PlanDialogs
        active={props.ui.active}
        data={props.app.data}
        busy={props.app.saving}
        error={props.app.saveError}
        onClose={() => {
          props.ui.setActive(null);
        }}
        onSave={props.sharing.save}
      />
      <SharingDialog state={props.sharing} />
    </>
  );
}

/**
 * Opens setup automatically only while a household has no entered plan.
 * @param data Authorized saved household and calculated income.
 * @returns Whether the financial plan is still empty.
 */
function isBlankPlan(data: HouseholdResponse): boolean {
  return (
    data.summary.household.monthlyTakeHomeCents === 0 &&
    data.household.expenses.length === 0 &&
    data.household.goals.length === 0
  );
}
