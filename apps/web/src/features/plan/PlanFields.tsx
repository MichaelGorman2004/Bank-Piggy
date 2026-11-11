import type { Expense } from '@bank-piggy/contracts';
import { X } from 'lucide-react';
import type { MemberDraft } from './planDraft';
import type { PlanRowUpdate } from './planReducer';
import styles from './Editors.module.css';

interface MoneyFieldProps {
  readonly label: string;
  readonly value: string;
  readonly onChange: (value: string) => void;
  readonly required?: boolean;
  readonly pattern?: string;
  readonly help?: string;
}

/**
 * Keeps incomplete decimal text in the draft until submission.
 * @param props Field label, text, and edit callback.
 * @returns Money entry field.
 */
export function PlanMoneyField(props: MoneyFieldProps) {
  const { label, value, onChange, required, pattern, help } = props;
  return (
    <label>
      {label}
      <input
        inputMode="decimal"
        required={required}
        pattern={pattern}
        value={value}
        onChange={(event) => {
          onChange(event.target.value);
        }}
      />
      {help && <span className={styles.help}>{help}</span>}
    </label>
  );
}

interface PlanItemRowProps {
  readonly kind: 'Expense' | 'Goal';
  readonly item: Pick<Expense, 'name' | 'ownerId'>;
  readonly amount: string;
  readonly members: MemberDraft[];
  readonly first: boolean;
  readonly onChange: (update: PlanRowUpdate) => void;
  readonly onRemove: () => void;
}

/**
 * Edits common name, amount, and owner fields of an expense or goal.
 * @param props Row values and edit callbacks.
 * @returns Expense or goal draft row.
 */
export function PlanItemRow(props: PlanItemRowProps) {
  const { kind, item, amount, members, first, onChange, onRemove } = props;
  const { name, ownerId } = item;
  return (
    <div className={styles.row}>
      <label>
        {kind}
        <input
          autoFocus={first}
          required
          maxLength={80}
          value={name}
          onChange={(event) => {
            onChange({ name: event.target.value });
          }}
          placeholder={kind === 'Expense' ? 'e.g. Groceries' : 'Emergency fund'}
        />
      </label>
      <PlanMoneyField
        label="Monthly ($)"
        required
        value={amount}
        onChange={(value) => {
          onChange({ amount: value });
        }}
      />
      <PlanOwnerField
        ownerId={ownerId}
        members={members}
        onChange={(value) => {
          onChange({ ownerId: value });
        }}
      />
      <button
        className={styles.remove}
        type="button"
        aria-label={`Remove ${name || kind.toLowerCase()}`}
        onClick={onRemove}
      >
        <X size={17} />
      </button>
    </div>
  );
}

interface PlanOwnerFieldProps {
  readonly ownerId: Expense['ownerId'];
  readonly members: MemberDraft[];
  readonly onChange: (ownerId: Expense['ownerId']) => void;
}
/**
 * Lists the shared household and both canonical member identifiers.
 * @param props Selected owner and current member names.
 * @returns Owner selection field.
 */
function PlanOwnerField(props: PlanOwnerFieldProps) {
  const { ownerId, members, onChange } = props;
  return (
    <label>
      For
      <select
        value={ownerId}
        onChange={(event) => {
          const ownerId =
            event.target.value === 'shared'
              ? 'shared'
              : members.find((member) => member.member.id === event.target.value)?.member.id;
          if (ownerId) onChange(ownerId);
        }}
      >
        <option value="shared">Both of us</option>
        {members.map((member) => (
          <option key={member.member.id} value={member.member.id}>
            {member.name}
          </option>
        ))}
      </select>
    </label>
  );
}
