import type { AccountMappingRequest, Household } from '@bank-piggy/contracts';
import { CONNECTED_TYPES, manualOptions } from './mappingDraft';
import styles from './Connections.module.css';

interface MappingFieldsProps {
  readonly draft: AccountMappingRequest;
  readonly household: Household;
  readonly onChange: (value: AccountMappingRequest) => void;
}

/**
 * Records confirmed ownership and classification with native selectors.
 * @param props Current mapping draft and saved household members.
 * @returns Explicit account ownership and account-type fields.
 */
export function AccountMappingFields(props: MappingFieldsProps) {
  const { draft, household, onChange } = props;
  return (
    <div className={styles.fields}>
      <label>
        Account owner
        <select
          value={draft.ownership}
          onChange={(event) => {
            const ownership = event.target.value;
            if (ownership === 'member-1' || ownership === 'member-2' || ownership === 'shared')
              onChange({ ...draft, ownership });
          }}
        >
          {household.members.map((member) => (
            <option key={member.id} value={member.id}>
              {member.name}
            </option>
          ))}
          <option value="shared">Joint / shared</option>
        </select>
      </label>
      <label>
        Confirm account type
        <select
          value={draft.classification}
          onChange={(event) => {
            const classification = CONNECTED_TYPES.find(
              (type) => type.value === event.target.value,
            )?.value;
            if (classification) onChange({ ...draft, classification });
          }}
        >
          {CONNECTED_TYPES.map((type) => (
            <option key={type.value} value={type.value}>
              {type.label}
            </option>
          ))}
        </select>
      </label>
      <ManualMappingField {...props} />
    </div>
  );
}

/**
 * Replaces a selected manual snapshot only after explicit user reconciliation.
 * @param props Current mapping draft, manual accounts, and update action.
 * @returns One native manual-account selector with a separate-account option.
 */
function ManualMappingField(props: MappingFieldsProps) {
  const options = manualOptions(props.household);
  const selected =
    props.draft.manualAccountId && props.draft.manualMemberId
      ? JSON.stringify([props.draft.manualMemberId, props.draft.manualAccountId])
      : '';
  return (
    <label>
      Replace a manual balance
      <select
        value={selected}
        onChange={(event) => {
          const match = options.find((option) => option.value === event.target.value);
          const { revision, enabled, ownership, classification } = props.draft;
          const fields = { revision, enabled, ownership, classification };
          props.onChange(
            match
              ? { ...fields, manualAccountId: match.accountId, manualMemberId: match.memberId }
              : fields,
          );
        }}
      >
        <option value="">Separate account · keep manual balances</option>
        {selected && !options.some((option) => option.value === selected) && (
          <option value={selected}>Manual account no longer exists</option>
        )}
        {options.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
    </label>
  );
}
