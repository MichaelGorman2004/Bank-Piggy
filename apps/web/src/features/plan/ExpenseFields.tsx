import type { Household } from '@bank-piggy/contracts';
import { EXPENSE_CATEGORIES } from './format';
import type { ExpenseDraft } from './expenseDraft';
import { PaymentFields } from './PaymentFields';
import styles from './Editors.module.css';

interface ExpenseFieldsProps {
  readonly draft: ExpenseDraft;
  readonly household: Household;
  readonly onChange: (update: Partial<ExpenseDraft>) => void;
}

/**
 * Collects the expense's whole monthly amount and its category.
 * @param props Component values and event handlers.
 * @returns Rendered interface.
 */
function ExpenseBasics(props: ExpenseFieldsProps) {
  const { draft, onChange } = props;
  return (
    <>
      <label className={styles.wide}>
        Expense name
        <input
          autoFocus
          required
          maxLength={80}
          value={draft.name}
          onChange={(event) => {
            onChange({ name: event.target.value });
          }}
          placeholder="e.g. Rent"
        />
      </label>
      <label>
        Monthly amount ($)
        <input
          required
          inputMode="decimal"
          value={draft.amount}
          onChange={(event) => {
            onChange({ amount: event.target.value });
          }}
          placeholder="0.00"
        />
      </label>
      <label>
        Category
        <select
          value={draft.category}
          onChange={(event) => {
            onChange({ category: event.target.value });
          }}
        >
          {[...new Set([...EXPENSE_CATEGORIES, draft.category])].map((item) => (
            <option key={item}>{item}</option>
          ))}
        </select>
      </label>
    </>
  );
}

/**
 * Keeps expense ownership and payment responsibility separate.
 * @param props Component values and event handlers.
 * @returns Rendered interface.
 */
function ExpenseOwnership(props: ExpenseFieldsProps) {
  const { draft, household, onChange } = props;
  return (
    <>
      <label className={styles.wide}>
        Who is this expense for?
        <select
          value={draft.ownerId}
          onChange={(event) => {
            const ownerId =
              event.target.value === 'shared'
                ? 'shared'
                : household.members.find((member) => member.id === event.target.value)?.id;
            if (ownerId) onChange({ ownerId });
          }}
        >
          <option value="shared">
            {household.members.length === 1 ? 'Household' : 'Both of us'}
          </option>
          {household.members.map((member) => (
            <option key={member.id} value={member.id}>
              {member.name}
            </option>
          ))}
        </select>
      </label>
      <ExpenseSplit {...props} />
      <ExpensePayer {...props} />
    </>
  );
}

/**
 * Records the payer separately from the people who share an expense.
 * @param props Expense draft and available household people.
 * @returns Optional payer selection.
 */
function ExpensePayer(props: ExpenseFieldsProps) {
  const { draft, household, onChange } = props;
  return (
    <label className={styles.wide}>
      Who usually pays?
      <select
        value={draft.paidBy}
        onChange={(event) => {
          const paidBy =
            event.target.value === ''
              ? ''
              : household.members.find((member) => member.id === event.target.value)?.id;
          if (paidBy !== undefined) onChange({ paidBy });
        }}
      >
        <option value="">Not recorded</option>
        {household.members.map((member) => (
          <option key={member.id} value={member.id}>
            {member.name}
          </option>
        ))}
      </select>
      {household.members.length > 1 && (
        <span className={styles.help}>Who pays does not change the agreed split.</span>
      )}
    </label>
  );
}

/**
 * Supports even, salary, and exact custom-percentage shared expense splits.
 * @param props Component values and event handlers.
 * @returns Rendered interface.
 */
function ExpenseSplit(props: ExpenseFieldsProps) {
  const { draft, household, onChange } = props;
  if (draft.ownerId !== 'shared' || household.members.length === 1) return null;
  const salaryAvailable = household.members.some((member) => member.annualBaseSalaryCents > 0);
  return (
    <>
      <label className={draft.method === 'custom' ? undefined : styles.wide}>
        How do you split it?
        <select
          value={draft.method}
          onChange={(event) => {
            const method = event.target.value;
            if (method === 'equal' || method === 'salary' || method === 'custom')
              onChange({ method });
          }}
        >
          <option value="equal">Evenly (50 / 50)</option>
          <option value="salary" disabled={!salaryAvailable}>
            By annual base salary
          </option>
          <option value="custom">Custom percentage</option>
        </select>
      </label>
      {draft.method === 'custom' && (
        <label>
          {household.members[0]?.name ?? 'First member'}'s share (%)
          <input
            required
            inputMode="decimal"
            value={draft.share}
            onChange={(event) => {
              onChange({ share: event.target.value });
            }}
          />
          <span className={styles.help}>The other member pays the remainder.</span>
        </label>
      )}
    </>
  );
}

/**
 * Groups recurring expense fields into one accessible form layout.
 * @param props Component values and event handlers.
 * @returns Rendered interface.
 */
export function ExpenseFields(props: ExpenseFieldsProps) {
  return (
    <div className={styles.grid}>
      <ExpenseBasics {...props} />
      <PaymentFields {...props} />
      <ExpenseOwnership {...props} />
    </div>
  );
}
