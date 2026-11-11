import type { AccountType } from '@bank-piggy/contracts';
import { X } from 'lucide-react';
import { ACCOUNT_TYPES } from '../accounts/accountTypes';
import type { AccountDraft, AccountUpdate } from './accountDraft';
import { PlanMoneyField } from './PlanFields';
import styles from './Accounts.module.css';

interface AccountFieldsProps {
  readonly account: AccountDraft;
  readonly index: number;
  readonly onChange: (update: AccountUpdate) => void;
  readonly onRemove: () => void;
}

/**
 * Edits each snapshot's explicit name, type, and balance with native controls.
 * @param props Editable snapshot, row position, and edit callbacks.
 * @returns One account row with an accessible removal action.
 */
export function AccountFields(props: AccountFieldsProps) {
  const { account, index, onChange, onRemove } = props;
  return (
    <fieldset className={styles.row}>
      <legend>Account {index + 1}</legend>
      <label>
        Account name
        <input
          autoFocus={!account.name && !account.balance}
          required
          value={account.name}
          placeholder="e.g. Workplace retirement"
          onChange={(event) => {
            onChange({ name: event.target.value });
          }}
        />
      </label>
      <AccountTypeField
        type={account.type}
        onChange={(type) => {
          onChange({ type });
        }}
      />
      <PlanMoneyField
        label="Balance ($)"
        required
        value={account.balance}
        onChange={(balance) => {
          onChange({ balance });
        }}
      />
      <RemoveAccountButton name={account.name || String(index + 1)} onRemove={onRemove} />
    </fieldset>
  );
}

/**
 * Keeps keyboard focus on a surviving control when an account row is removed.
 * @param props Account label and removal callback.
 * @param props.name Visible account name, or its row number when blank.
 * @param props.onRemove Removes this row from the current draft.
 * @returns Removal control with next-row or add-action focus.
 */
function RemoveAccountButton(props: { readonly name: string; readonly onRemove: () => void }) {
  const { name, onRemove } = props;
  return (
    <button
      className={styles.remove}
      type="button"
      aria-label={`Remove account ${name}`}
      onClick={(event) => {
        const row = event.currentTarget.closest('fieldset');
        const next = row?.nextElementSibling?.querySelector<HTMLElement>(
          'input:not(:disabled), button:not(:disabled)',
        );
        const previous = row?.previousElementSibling?.querySelector<HTMLElement>('input');
        onRemove();
        (next ?? previous)?.focus();
      }}
    >
      <X size={17} />
    </button>
  );
}

/**
 * Narrows selected values to the canonical account-type options.
 * @param props Current type and selection callback.
 * @param props.type Canonical account type.
 * @param props.onChange Receives a validated canonical option.
 * @returns Native account-type selector.
 */
function AccountTypeField(props: {
  readonly type: AccountType;
  readonly onChange: (type: AccountType) => void;
}) {
  const { type, onChange } = props;
  return (
    <label>
      Account type
      <select
        value={type}
        onChange={(event) => {
          const type = ACCOUNT_TYPES.find((option) => option.value === event.target.value)?.value;
          if (type) onChange(type);
        }}
      >
        {ACCOUNT_TYPES.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
    </label>
  );
}
