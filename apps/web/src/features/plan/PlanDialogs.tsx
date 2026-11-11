import type { Expense, Goal, Household, HouseholdResponse } from '@bank-piggy/contracts';
import { ExpenseEditor } from './ExpenseEditor';
import { GoalEditor } from './GoalEditor';
import type { GoalKind } from './goalDraft';
import { PlanEditor } from './PlanEditor';
import { planActions } from './planActions';

export type ActiveDialog =
  | { readonly type: 'expense'; readonly expense?: Expense; readonly initialKind?: Expense['kind'] }
  | { readonly type: 'goal'; readonly goal?: Goal; readonly initialKind?: GoalKind }
  | { readonly type: 'plan' }
  | null;
interface PlanDialogsProps {
  readonly active: ActiveDialog;
  readonly data: HouseholdResponse;
  readonly busy: boolean;
  readonly error: string | null;
  readonly onClose: () => void;
  readonly onSave: (household: Household) => Promise<boolean>;
}

/**
 * Applies editor changes atomically to the last saved household revision.
 * @param root0 Component values and event handlers.
 * @param root0.active Value used by this operation.
 * @param root0.data Validated household response.
 * @param root0.busy Whether persistence is in progress.
 * @param root0.error Current actionable error message.
 * @param root0.onClose Closes the current dialog or drawer.
 * @param root0.onSave Persists the next household revision.
 * @returns Rendered interface.
 */
export function PlanDialogs({ active, data, busy, error, onClose, onSave }: PlanDialogsProps) {
  if (!active) return null;
  const household = data.household;
  const { persist, saveExpense, deleteExpense, saveGoal, deleteGoal } = planActions({
    household,
    onSave,
    onClose,
  });
  if (active.type === 'plan')
    return (
      <PlanEditor
        household={household}
        busy={busy}
        error={error}
        onClose={onClose}
        onSave={persist}
      />
    );
  if (active.type === 'expense')
    return (
      <ExpenseEditor
        household={household}
        expense={active.expense}
        initialKind={active.initialKind}
        busy={busy}
        error={error}
        onClose={onClose}
        onSave={saveExpense}
        onDelete={deleteExpense}
      />
    );
  return (
    <GoalEditor
      household={household}
      goal={active.goal}
      initialKind={active.initialKind}
      busy={busy}
      error={error}
      onClose={onClose}
      onSave={saveGoal}
      onDelete={deleteGoal}
    />
  );
}
