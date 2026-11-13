import type { ConnectedAccount, Household } from '@bank-piggy/contracts';
import { AccountMappingFields } from './AccountMappingFields';
import { accountProblem } from './mappingDraft';
import { accountSnapshotMoney } from './accountSnapshotMoney';
import { accountSnapshotBalanceLabel, accountSnapshotType } from './accountSnapshotLabels';
import { useAccountMapping, type AccountMappingOptions } from './useAccountMapping';
import styles from './Connections.module.css';
import formStyles from '../plan/Editors.module.css';

interface AccountRowProps extends AccountMappingOptions {
  readonly household: Household;
}
type MappingState = ReturnType<typeof useAccountMapping>;

/**
 * Keeps mapping drafts mounted through refreshes and provider failures.
 * @param props Authoritative account, saved household, and session mutation token.
 * @returns A compact account disclosure with explicit balance selection.
 */
export function ConnectedAccountRow(props: AccountRowProps) {
  const { account } = props;
  const state = useAccountMapping(props);
  const balance = accountSnapshotMoney(account.currentBalanceCents, account.currency);
  return (
    <details className={styles.account}>
      <summary>
        <strong>
          {account.name}
          {account.mask ? ` · ${account.mask}` : ''}
        </strong>
        <span>
          {accountSnapshotBalanceLabel(account)}:{' '}
          {balance === 'Unavailable' ? 'Not provided' : balance} ·{' '}
          {account.mapping.enabled ? 'Selected' : 'Excluded'}
        </span>
      </summary>
      <p className={styles.meta}>{accountSnapshotType(account)}</p>
      <AccountMappingForm state={state} account={account} household={props.household} />
    </details>
  );
}

/**
 * Keeps native mapping fields editable during requests and reports stale saved revisions.
 * @param props Mapping draft and authoritative account.
 * @param props.state Mapping draft owner and save actions.
 * @param props.account Current provider snapshot.
 * @param props.household Saved ownership and manual-account options.
 * @returns A draft-safe explicit account selection form.
 */
function AccountMappingForm(props: {
  readonly state: MappingState;
  readonly account: ConnectedAccount;
  readonly household: Household;
}) {
  const { state } = props;
  const problem = accountProblem(props.account);
  return (
    <form
      onSubmit={(event) => {
        event.preventDefault();
        void state.save();
      }}
    >
      <label className={styles.checkbox}>
        <input
          type="checkbox"
          checked={state.draft.enabled}
          disabled={Boolean(problem) && !state.draft.enabled}
          onChange={(event) => {
            state.setDraft({ ...state.draft, enabled: event.target.checked });
          }}
        />
        Include in household balances
      </label>
      {problem && <p className={styles.notice}>{problem} This balance is excluded from totals.</p>}
      <AccountMappingFields
        draft={state.draft}
        household={props.household}
        onChange={state.setDraft}
      />
      <p className={formStyles.help}>
        Confirm Traditional, Roth, and high-yield types yourself. A joint account counts once in the
        household; personal views split its balance equally. Choose a manual balance only when it is
        the same account. The connected current balance replaces that manual snapshot in totals.
        Your manual plan stays saved. If the same joint account appears through both logins, include
        one copy and exclude the other.
      </p>
      <MappingActions state={state} />
    </form>
  );
}

/**
 * Offers explicit revision recovery without replacing the local draft automatically.
 * @param props Mapping state and save feedback.
 * @param props.state Current mapping draft owner.
 * @returns Save, reset, validation, and stale revision feedback.
 */
function MappingActions(props: { readonly state: MappingState }) {
  const state = props.state;
  return (
    <>
      {state.error && (
        <p className={formStyles.error} role="alert">
          {state.error}
        </p>
      )}
      <div className={styles.actions}>
        {state.changed && (
          <button
            type="button"
            className={formStyles.secondary}
            disabled={state.busy}
            onClick={state.reset}
          >
            Load saved settings
          </button>
        )}
        <button className={formStyles.button} disabled={state.busy || state.changed}>
          {state.busy ? 'Saving…' : 'Save account settings'}
        </button>
      </div>
      {state.changed && (
        <p className={styles.notice}>
          Saved settings changed. Your draft is kept. Load saved settings before you save again.
        </p>
      )}
    </>
  );
}
