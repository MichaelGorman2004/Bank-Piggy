import { ProviderField } from '../providers/ProviderField';
import { MonthPicker } from '../../components/MonthPicker';
import type { ExpenseDraft } from './expenseDraft';
import { MAX_PAYMENT_MONTHS, MIN_MONTH, monthLabel, termEndMonth } from './month';
import styles from './Editors.module.css';

interface PaymentFieldsProps {
  readonly draft: ExpenseDraft;
  readonly onChange: (update: Partial<ExpenseDraft>) => void;
}

/**
 * Identifies a service locally without a connection to its provider.
 * @param props Payment draft and field update action.
 * @returns Payment kind and service controls.
 */
function PaymentIdentity(props: PaymentFieldsProps) {
  const { draft, onChange } = props;
  return (
    <>
      <label>
        Payment type
        <select
          value={draft.kind}
          onChange={(event) => {
            const kind = event.target.value;
            if (kind === 'bill' || kind === 'subscription' || kind === 'loan') onChange({ kind });
          }}
        >
          <option value="bill">Bill or regular expense</option>
          <option value="subscription">Subscription</option>
          <option value="loan">Loan payment</option>
        </select>
      </label>
      <ProviderField
        value={draft.providerId}
        onChange={(providerId) => {
          onChange({ providerId });
        }}
      />
    </>
  );
}

/**
 * Offers ongoing payments, a final calendar month, or a finite monthly term.
 * @param props Payment draft and field update action.
 * @returns End condition controls.
 */
function PaymentEnd(props: PaymentFieldsProps) {
  const { draft, onChange } = props;
  return (
    <>
      <label>
        Payment ends
        <select
          value={draft.endMode}
          onChange={(event) => {
            const endMode = event.target.value;
            if (endMode === 'ongoing' || endMode === 'month' || endMode === 'term')
              onChange({ endMode });
          }}
        >
          <option value="ongoing">No end date</option>
          <option value="month">After a final month</option>
          <option value="term">After a number of months</option>
        </select>
      </label>
      {draft.endMode === 'month' && (
        <MonthPicker
          label="Final payment month"
          required
          min={draft.startMonth || MIN_MONTH}
          value={draft.endMonth}
          onChange={(endMonth) => {
            onChange({ endMonth });
          }}
        />
      )}
      <PaymentTerm {...props} />
    </>
  );
}

/**
 * Explains inclusive schedule dates before the user saves a payment.
 * @param props Payment draft and field update action.
 * @returns Schedule controls with the calculated final month.
 */
export function PaymentFields(props: PaymentFieldsProps) {
  const { draft, onChange } = props;
  const termEnd = termEndMonth(draft.startMonth, draft.termMonths);
  return (
    <>
      <PaymentIdentity {...props} />
      <MonthPicker
        className={styles.wide}
        label="First payment month (optional for ongoing costs)"
        required={draft.endMode === 'term'}
        value={draft.startMonth}
        onChange={(startMonth) => {
          onChange({ startMonth });
        }}
      />
      <PaymentEnd {...props} />
      <p className={`${styles.help} ${styles.wide}`}>
        {draft.endMode === 'term' && termEnd ? `Final payment: ${monthLabel(termEnd)}. ` : ''}
        The final month is included. Later months exclude this cost. This only changes your plan; it
        does not cancel a service or repay a loan.
      </p>
    </>
  );
}

/**
 * Keeps finite monthly terms separate from calendar end-date inputs.
 * @param props Payment draft and field update action.
 * @returns A term count when the finite-term mode is selected.
 */
function PaymentTerm(props: PaymentFieldsProps) {
  const { draft, onChange } = props;
  if (draft.endMode !== 'term') return null;
  return (
    <label>
      Number of monthly payments
      <input
        required
        type="number"
        min="1"
        max={MAX_PAYMENT_MONTHS}
        step="1"
        value={draft.termMonths}
        onChange={(event) => {
          onChange({ termMonths: event.target.value });
        }}
      />
    </label>
  );
}
