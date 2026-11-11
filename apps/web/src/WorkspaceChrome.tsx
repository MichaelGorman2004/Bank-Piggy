import { Menu, RefreshCw } from 'lucide-react';
import { VIEW_DETAILS, type View } from './features/plan/views';
import styles from './Workspace.module.css';
import dashboardStyles from './components/Dashboard.module.css';

interface WorkspaceBarProps {
  readonly view: View;
  readonly saving: boolean;
  readonly mobileOpen: boolean;
  readonly onMenu: () => void;
}
interface ReloadNoticeProps {
  readonly conflict: boolean;
  readonly error: string | null;
  readonly onReload: () => void;
}

/**
 * Keeps navigation and transient save feedback above every finance view.
 * @param props Component values and event handlers.
 * @returns Rendered interface.
 */
export function WorkspaceBar(props: WorkspaceBarProps) {
  return (
    <>
      <div className={styles.mobileBar}>
        <button
          onClick={props.onMenu}
          aria-label="Open navigation"
          aria-expanded={props.mobileOpen}
        >
          <Menu size={20} />
        </button>
        <span>
          Bank Piggy
          <small className={styles.mobileSaving} role="status">
            {props.saving ? 'Saving…' : ''}
          </small>
        </span>
        <img src="/assets/banker-piggy.png" alt="" />
      </div>
      <div className={styles.topBar}>
        <span className={styles.breadcrumb}>
          Our household<span>/</span>
          {VIEW_DETAILS[props.view].label}
        </span>
        <span className={styles.saving} role="status">
          {props.saving ? 'Saving…' : ''}
        </span>
      </div>
    </>
  );
}

/**
 * Makes revision conflicts and loading failures actionable without discarding a draft.
 * @param props Component values and event handlers.
 * @returns Rendered interface.
 */
export function ReloadNotice(props: ReloadNoticeProps) {
  if (!props.conflict && !props.error) return null;
  return (
    <div className={dashboardStyles.notice} role="alert">
      <p>
        {props.conflict
          ? 'The plan changed in another window. Reload to get the latest version.'
          : props.error}
      </p>
      <button className={dashboardStyles.textButton} onClick={props.onReload}>
        <RefreshCw size={13} />
        Reload plan
      </button>
    </div>
  );
}

interface PlanCompletenessProps {
  readonly incomeCents: number;
  readonly expenseCount: number;
  readonly onEdit: () => void;
}

/**
 * Warns that a plan without income or living costs cannot show a useful available amount.
 * @param props Component values and event handlers.
 * @returns Rendered interface.
 */
export function PlanCompleteness(props: PlanCompletenessProps) {
  const missingIncome = props.incomeCents === 0;
  const missingExpenses = props.expenseCount === 0;
  if (!missingIncome && !missingExpenses) return null;
  const message = missingIncome
    ? 'No take-home pay is recorded. Add your income to build your plan.'
    : 'No living expenses are recorded. Add regular costs before using the amount left to plan.';
  return (
    <div className={dashboardStyles.notice} role="status">
      <p>{message}</p>
      <button className={dashboardStyles.textButton} onClick={props.onEdit}>
        Complete your plan
      </button>
    </div>
  );
}
