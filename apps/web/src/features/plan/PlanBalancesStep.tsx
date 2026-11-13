import type { Dispatch } from 'react';
import { Plus } from 'lucide-react';
import { MAX_ACCOUNTS } from '../accounts/accountTypes';
import { AccountFields } from './AccountFields';
import type { MemberDraft } from './planDraft';
import type { PlanAction } from './planReducer';
import { PlanMoneyField } from './PlanFields';
import styles from './Editors.module.css';
import accountStyles from './Accounts.module.css';
import { useConnectionData } from '../connections/ConnectionsContext';
import { manualAccountSource } from './manualAccountSource';

interface MemberBalancesProps {
  readonly draft: MemberDraft;
  readonly index: number;
  readonly dispatch: Dispatch<PlanAction>;
}

/**
 * Keeps one member's account snapshots and debt separate from monthly income.
 * @param props Member draft, position, and draft dispatch.
 * @returns Editable accounts and debt balance.
 */
function MemberBalances(props: MemberBalancesProps) {
  const { draft, index, dispatch } = props;
  const connections = useConnectionData();
  return (
    <section className={styles.member}>
      <h3>{draft.name || `Member ${String(index + 1)}`}</h3>
      <div className={accountStyles.accounts}>
        {draft.accounts.map((account, accountIndex) => (
          <AccountFields
            key={account.id}
            account={account}
            index={accountIndex}
            source={manualAccountSource(connections, draft.member.id, account.id)}
            onChange={(update) => {
              dispatch({ type: 'account', memberIndex: index, id: account.id, update });
            }}
            onRemove={() => {
              dispatch({ type: 'removeAccount', memberIndex: index, id: account.id });
            }}
          />
        ))}
        {!draft.accounts.length && <p className={styles.help}>No manual accounts entered.</p>}
        <AddAccountButton {...props} />
        <PlanMoneyField
          label="Outstanding debt ($)"
          value={draft.debt}
          onChange={(debt) => {
            dispatch({ type: 'member', index, update: { debt } });
          }}
        />
      </div>
    </section>
  );
}

/**
 * Creates a stable account identifier only when the user adds a row.
 * @param props Member draft, position, and draft dispatch.
 * @returns Add control and the account limit.
 */
function AddAccountButton(props: MemberBalancesProps) {
  const { draft, index, dispatch } = props;
  return (
    <div className={accountStyles.add}>
      <button
        type="button"
        className={styles.add}
        disabled={draft.accounts.length >= MAX_ACCOUNTS}
        aria-label={`Add account for ${draft.name || `member ${String(index + 1)}`}`}
        onClick={() => {
          dispatch({ type: 'addAccount', memberIndex: index, id: crypto.randomUUID() });
        }}
      >
        <Plus size={15} />
        Add account
      </button>
      <span className={styles.help}>
        {draft.accounts.length} of {MAX_ACCOUNTS} accounts
      </span>
    </div>
  );
}

/**
 * Edits optional manual account snapshots without changing target progress.
 * @param props Member drafts and draft dispatch.
 * @param props.members Current household member drafts.
 * @param props.dispatch Applies edits against current draft state.
 * @returns Collapsible current-balance fields.
 */
export function PlanBalances(props: {
  readonly members: MemberDraft[];
  readonly dispatch: Dispatch<PlanAction>;
}) {
  const { members, dispatch } = props;
  return (
    <details>
      <summary>Current balances (optional)</summary>
      <p className={`${styles.help} ${accountStyles.intro}`}>
        Manage linked balances in Accounts. Add manual balances for accounts you have not linked.
        These balances do not change monthly contributions or goal progress.
      </p>
      {members.map((draft, index) => (
        <MemberBalances key={draft.member.id} draft={draft} index={index} dispatch={dispatch} />
      ))}
    </details>
  );
}
