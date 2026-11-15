import type {
  ActivityAccount,
  ActivityAccountSelection,
  ActivitySource,
  Member,
} from '@bank-piggy/contracts';
import { canToggleActivityAccount } from './activitySourceDraft';
import styles from './Activity.module.css';

/**
 * Uses native account inclusion and ownership controls without changing balance mappings.
 * @param props Saved accounts, member names, and the retained source draft.
 * @param props.source Current provider account identities and availability.
 * @param props.members Saved household member names.
 * @param props.accounts Draft account selections.
 * @param props.disabled Whether the current member can edit this source.
 * @param props.onChange Explicit account-selection action.
 * @returns Source account controls.
 */
export function SourceAccounts(props: {
  readonly source: ActivitySource;
  readonly members: ReadonlyArray<Member>;
  readonly accounts: ActivityAccountSelection[];
  readonly disabled: boolean;
  readonly onChange: (accounts: ActivityAccountSelection[]) => void;
}) {
  /** Updates one account without changing its sibling selections.
   * @param accountId Saved account identity.
   * @param patch Account selection changes.
   */
  function change(accountId: string, patch: Partial<ActivityAccountSelection>) {
    props.onChange(
      props.accounts.map((account) =>
        account.accountId === accountId ? { ...account, ...patch } : account,
      ),
    );
  }
  return (
    <div className={styles.sourceAccounts}>
      {props.source.accounts.map((account) => {
        const selection = props.accounts.find((item) => item.accountId === account.accountId);
        return (
          <SourceAccount
            key={account.accountId}
            account={account}
            selection={selection}
            members={props.members}
            disabled={props.disabled}
            onChange={(patch) => {
              change(account.accountId, patch);
            }}
          />
        );
      })}
    </div>
  );
}

interface AccountProps {
  readonly account: ActivityAccount;
  readonly selection: ActivityAccountSelection | undefined;
  readonly members: ReadonlyArray<Member>;
  readonly disabled: boolean;
  readonly onChange: (patch: Partial<ActivityAccountSelection>) => void;
}

/** Shows one account inclusion choice without changing its balance mapping.
 * @param props Provider account, selection draft, and ownership controls.
 * @returns Native account-inclusion checkbox and ownership selector.
 */
function SourceAccount(props: AccountProps) {
  const { account, selection } = props;
  return (
    <div className={styles.sourceAccount}>
      <label className={styles.checkbox}>
        <input
          type="checkbox"
          checked={selection?.enabled ?? false}
          disabled={props.disabled || !canToggleActivityAccount(account, selection)}
          onChange={(event) => {
            props.onChange({ enabled: event.target.checked });
          }}
        />
        <span>
          Include {account.name} in activity
          <small>
            {account.providerType} · {account.currency ?? 'Currency unavailable'}
            {!account.available ? ' · Unavailable' : ''}
          </small>
        </span>
      </label>
      <SourceAccountOwner {...props} />
    </div>
  );
}

/** Names the member allocation for one account activity selection.
 * @param props Account selection and saved household names.
 * @returns Native owner selector.
 */
function SourceAccountOwner(props: AccountProps) {
  const { account, selection } = props;
  return (
    <label>
      Activity owner
      <select
        value={selection?.ownership ?? account.ownership}
        disabled={props.disabled}
        onChange={(event) => {
          const ownership = event.target.value;
          if (ownership === 'shared' || ownership === 'member-1' || ownership === 'member-2')
            props.onChange({ ownership });
        }}
      >
        {props.members.map((member) => (
          <option key={member.id} value={member.id}>
            {member.name}
          </option>
        ))}
        {props.members.length > 1 && <option value="shared">Shared equally</option>}
      </select>
    </label>
  );
}
