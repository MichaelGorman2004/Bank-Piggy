import type { Dispatch } from 'react';
import { Plus } from 'lucide-react';
import type { PlanDraft } from './planDraft';
import type { PlanAction } from './planReducer';
import { PlanItemRow } from './PlanFields';
import styles from './Editors.module.css';

type ExpenseState = Pick<PlanDraft, 'members' | 'expenses' | 'defaultSplit'>;
interface ExpensesStepProps {
  readonly state: ExpenseState;
  readonly dispatch: Dispatch<PlanAction>;
}

/**
 * Selects the split that applies when a new expense is added.
 * @param props Current split and draft dispatch.
 * @returns Split selection field.
 */
function DefaultSplit(props: ExpensesStepProps) {
  const { state, dispatch } = props;
  return (
    <label>
      Default split for new shared expenses
      <select
        value={state.defaultSplit}
        onChange={(event) => {
          const value = event.target.value;
          if (value === 'equal' || value === 'salary' || value === 'custom')
            dispatch({ type: 'split', method: value });
        }}
      >
        <option value="equal">Evenly (50 / 50)</option>
        <option value="salary">By annual base salary</option>
        <option value="custom">Custom (start at 50 / 50)</option>
      </select>
    </label>
  );
}

/**
 * Edits expense drafts while preserving existing split settings.
 * @param props Expense drafts and draft dispatch.
 * @returns Expense setup step.
 */
export function PlanExpensesStep(props: ExpensesStepProps) {
  const { state, dispatch } = props;
  return (
    <div className={styles.section}>
      {state.members.length > 1 && (
        <>
          <DefaultSplit {...props} />
          <p className={styles.help}>
            Existing expense splits stay the same. Open an expense after saving to change its split
            or record who pays.
          </p>
        </>
      )}
      {state.expenses.map((draft, index) => (
        <PlanItemRow
          key={draft.expense.id}
          kind="Expense"
          item={draft.expense}
          amount={draft.amount}
          members={state.members}
          first={index === 0}
          onChange={(update) => {
            dispatch({ type: 'expense', index, update });
          }}
          onRemove={() => {
            dispatch({ type: 'removeExpense', index });
          }}
        />
      ))}
      <button
        type="button"
        className={styles.add}
        onClick={() => {
          dispatch({ type: 'addExpense', id: crypto.randomUUID() });
        }}
      >
        <Plus size={15} />
        Add expense
      </button>
      <p className={styles.help}>
        Include housing, groceries, regular bills, and minimum debt payments. Missing costs will
        make your available amount too high.
      </p>
    </div>
  );
}
