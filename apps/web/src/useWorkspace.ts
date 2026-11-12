import { useCallback, useState } from 'react';
import { HOUSEHOLD_SCOPE, type Period } from './features/plan/format';
import { localDate } from './features/plan/payScheduleDraft';
import type { View } from './features/plan/views';
import type { ActiveDialog } from './features/plan/PlanDialogs';

interface WorkspaceStateOptions {
  readonly conflict: boolean;
  readonly blank: boolean;
  readonly onClearError: () => void;
}

/**
 * Keeps transient UI state separate from the persisted household plan.
 * @param options State initialization options.
 * @returns Current draft state and event handlers.
 */
export function useWorkspace(options: WorkspaceStateOptions) {
  const [view, setView] = useState<View>('overview');
  const [todoDate, setTodoDate] = useState(() => localDate(new Date()));
  const [scope, setScope] = useState(HOUSEHOLD_SCOPE);
  const [period, setPeriod] = useState<Period>('monthly');
  const [mobileOpen, setMobileOpen] = useState(false);
  const [active, setActive] = useState<ActiveDialog>(options.blank ? { type: 'plan' } : null);
  const closeMobile = useCallback(() => {
    setMobileOpen(false);
  }, []);
  /**
   * Opens an editor only when the saved plan is current.
   * @param dialog Value used by this operation.
   */
  function open(dialog: ActiveDialog) {
    if (options.conflict) return;
    options.onClearError();
    setActive(dialog);
  }
  /**
   * Switches scope and resets household paycheck estimates.
   * @param next Complete next household plan.
   */
  function changeScope(next: string) {
    setScope(next);
    if (next === HOUSEHOLD_SCOPE)
      setPeriod((current) => (current === 'paycheck' ? 'monthly' : current));
  }
  return {
    todoDate,
    setTodoDate,
    view,
    setView,
    scope,
    period,
    setPeriod,
    mobileOpen,
    setMobileOpen,
    closeMobile,
    active,
    setActive,
    open,
    changeScope,
  };
}
