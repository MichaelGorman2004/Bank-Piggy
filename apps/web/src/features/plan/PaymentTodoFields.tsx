import type { ExpenseDraft } from './expenseDraft';
import styles from './Editors.module.css';

interface PaymentTodoFieldsProps {
  readonly draft: ExpenseDraft;
  readonly onChange: (change: Partial<ExpenseDraft>) => void;
}

/**
 * Includes only manual payments in the pay-period To-do list.
 * @param props Editable payment mode, due day, and field update action.
 * @returns Payment handling and optional calendar due day.
 */
export function PaymentTodoFields(props: PaymentTodoFieldsProps) {
  const { draft, onChange } = props;
  return (
    <>
      <label>
        How is this payment made?
        <select
          value={draft.paymentMode}
          onChange={(event) => {
            const paymentMode = event.target.value;
            if (paymentMode === 'manual' || paymentMode === 'automatic') onChange({ paymentMode });
          }}
        >
          <option value="automatic">Automatic payment</option>
          <option value="manual">I make this payment</option>
        </select>
      </label>
      {draft.paymentMode === 'manual' && (
        <label>
          Due day of the month (optional)
          <input
            type="number"
            min="1"
            max="31"
            step="1"
            value={draft.dueDay}
            onChange={(event) => {
              onChange({ dueDay: event.target.value });
            }}
          />
          <small className={styles.help}>
            Add a day to show this payment in To-do. Short months use the last day.
          </small>
        </label>
      )}
      {draft.paymentMode === 'automatic' && (
        <p className={styles.help}>
          Automatic payments stay in your budget and do not appear in To-do.
        </p>
      )}
    </>
  );
}
