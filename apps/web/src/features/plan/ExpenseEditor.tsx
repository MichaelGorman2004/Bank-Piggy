import { useState, type SubmitEvent } from 'react';
import type { Expense, Household } from '@bank-piggy/contracts';
import { Modal } from '../../components/Modal';
import { EditorActions } from './EditorActions';
import { ExpenseFields } from './ExpenseFields';
import { expenseDraft, validateExpense, type ExpenseDraft } from './expenseDraft';
import styles from './Editors.module.css';

interface ExpenseEditorProps {
  readonly household: Household;
  readonly expense?: Expense;
  readonly initialKind?: Expense['kind'];
  readonly busy: boolean;
  readonly error: string | null;
  readonly onClose: () => void;
  readonly onSave: (expense: Expense) => Promise<void>;
  readonly onDelete: (id: string) => Promise<void>;
}

/**
 * Retains incomplete field text and validates the complete request before saving.
 * @param props Component values and event handlers.
 * @returns Current draft state and event handlers.
 */
function useExpenseForm(props: ExpenseEditorProps) {
  const [draft, setDraft] = useState(() =>
    expenseDraft(props.household, props.expense, props.initialKind),
  );
  const [localError, setLocalError] = useState<string | null>(null);
  /**
   * Merges edited fields into the latest local draft.
   * @param change Fields to merge into the current draft.
   */
  function update(change: Partial<ExpenseDraft>) {
    setDraft((current) => ({ ...current, ...change }));
  }
  /**
   * Validates the draft and awaits the requested save.
   * @param event Browser form or keyboard event.
   */
  async function submit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    const result = validateExpense(draft, props.household);
    if (result.error) {
      setLocalError(result.error);
      return;
    }
    setLocalError(null);
    if (result.expense) await props.onSave(result.expense);
  }
  return { draft, localError, submit, update };
}

/**
 * Edits one recurring expense and retains its draft when saving fails.
 * @param props Component values and event handlers.
 * @returns Rendered interface.
 */
export function ExpenseEditor(props: ExpenseEditorProps) {
  const { draft, localError, submit, update } = useExpenseForm(props);
  return (
    <Modal
      title={
        props.expense
          ? 'Edit expense'
          : props.initialKind === 'subscription'
            ? 'Add a subscription'
            : 'Add an expense'
      }
      description="Set a monthly estimate. You can change it as your plans change."
      onClose={props.onClose}
      busy={props.busy}
    >
      <form
        className={styles.form}
        onSubmit={(event) => {
          void submit(event);
        }}
      >
        <fieldset disabled={props.busy} className={styles.fields}>
          <ExpenseFields draft={draft} household={props.household} onChange={update} />
          {(localError || props.error) && (
            <p className={styles.error} role="alert">
              {localError ?? props.error}
            </p>
          )}
          <EditorActions
            kind="expense"
            busy={props.busy}
            onClose={props.onClose}
            onDelete={
              props.expense
                ? () => {
                    void props.onDelete(draft.id);
                  }
                : undefined
            }
          />
        </fieldset>
      </form>
    </Modal>
  );
}
