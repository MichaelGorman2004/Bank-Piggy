import type { Household, HouseholdResponse, Session } from '@bank-piggy/contracts';
import { Sidebar } from './components/Sidebar';
import { PlanDialogs } from './features/plan/PlanDialogs';
import { WorkspaceBar } from './WorkspaceChrome';
import { WorkspaceContent } from './WorkspaceContent';
import { useWorkspace } from './useWorkspace';
import styles from './Workspace.module.css';

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
  const blank =
    props.data.summary.household.monthlyTakeHomeCents === 0 &&
    props.data.household.expenses.length === 0 &&
    props.data.household.goals.length === 0;
  const ui = useWorkspace({
    conflict: props.conflict || props.loading,
    onClearError: props.onClearError,
    blank,
  });
  return (
    <div className={styles.app}>
      <a href="#main" className={styles.skip}>
        Skip to main content
      </a>
      <Sidebar
        session={props.session}
        members={props.data.household.members}
        view={ui.view}
        open={ui.mobileOpen}
        onView={ui.setView}
        onClose={ui.closeMobile}
        onSignout={props.onSignout}
      />
      <div className={styles.workspace}>
        <WorkspaceBar
          view={ui.view}
          saving={props.saving}
          mobileOpen={ui.mobileOpen}
          onMenu={() => {
            ui.setMobileOpen(true);
          }}
        />
        <WorkspaceContent app={props} ui={ui} />
      </div>
      <PlanDialogs
        active={ui.active}
        data={props.data}
        busy={props.saving}
        error={props.saveError}
        onClose={() => {
          ui.setActive(null);
        }}
        onSave={props.onSave}
      />
    </div>
  );
}
