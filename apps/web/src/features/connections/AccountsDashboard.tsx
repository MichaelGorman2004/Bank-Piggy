import { RefreshCw } from 'lucide-react';
import type { Household } from '@bank-piggy/contracts';
import { FinanceCard } from '../../components/FinanceCard';
import { ConnectedAccounts } from './ConnectedAccounts';
import { useConnectionData } from './ConnectionsContext';
import { AccountInstitution } from './AccountInstitution';
import styles from './AccountsDashboard.module.css';

/**
 * Lists every provider account, including excluded and unsupported account types.
 * Account values are full snapshots; the view does not calculate financial totals.
 * @param props Saved household members for connection and account ownership labels.
 * @param props.household Saved household members and manual accounts.
 * @returns One bounded card with shared connection management and saved balances.
 */
export function AccountsDashboard(props: { readonly household: Household }) {
  return (
    <div className={styles.page}>
      <header className={styles.pageHeader}>
        <h1>Accounts</h1>
        <p>Latest balance snapshots from your linked institutions.</p>
      </header>
      <FinanceCard
        title="Latest linked balances"
        description="All household accounts · Full account balances"
        actions={null}
        controls={<AccountsActions household={props.household} />}
        footer={<p className={styles.note}>Checks for updated balances every 30 minutes.</p>}
      >
        <AccountsContent household={props.household} />
      </FinanceCard>
    </div>
  );
}

/**
 * Keeps all account rows mounted through status reloads and provider failures.
 * @param props Saved household names for account and institution labels.
 * @param props.household Saved household members.
 * @returns Authorized institution snapshots or explicit empty guidance.
 */
function AccountsContent(props: { readonly household: Household }) {
  const state = useConnectionData();
  const connections = state.data?.connections.filter(
    (connection) => connection.state !== 'disconnected' || connection.accounts.length > 0,
  );
  return (
    <>
      <AccountsFeedback />
      {connections?.map((connection) => (
        <AccountInstitution
          key={connection.id}
          connection={connection}
          household={props.household}
        />
      ))}
      {connections?.length === 0 && (
        <div className={styles.empty}>
          <h3>No accounts linked yet</h3>
          <p>Use Connect accounts to link an institution or review an incomplete connection.</p>
          <p>Your manual account balances remain in Overview.</p>
        </div>
      )}
    </>
  );
}

/**
 * Reloads saved status without requesting a new provider read or changing mappings.
 * @param props Saved household for the existing connection manager.
 * @param props.household Saved household members and manual accounts.
 * @returns Fixed account connection and saved-status actions.
 */
function AccountsActions(props: { readonly household: Household }) {
  const state = useConnectionData();
  return (
    <div className={styles.controls}>
      <div className={styles.actions}>
        <ConnectedAccounts household={props.household} label="Connect accounts" connectOnOpen />
        <button
          className={styles.reload}
          disabled={state.loading || state.busy}
          onClick={() => {
            void state.reload();
          }}
        >
          <RefreshCw size={14} />
          {state.loading ? 'Loading…' : 'Reload status'}
        </button>
      </div>
    </div>
  );
}

/**
 * Retains loaded snapshots through failed status reads and labels simulated data.
 * @returns Current loading, failure, provider, and operation feedback.
 */
function AccountsFeedback() {
  const state = useConnectionData();
  return (
    <div className={styles.feedback}>
      {state.error && (
        <p className={styles.error} role="alert">
          {state.error} Reload status to try again.
        </p>
      )}
      {state.loading && (
        <p className={styles.note} role="status">
          {state.data ? 'Reloading saved connection status…' : 'Loading linked account balances…'}
        </p>
      )}
      {state.feedback && (
        <p className={styles.note} role="status">
          {state.feedback}
        </p>
      )}
      {state.data?.environment === 'sandbox' && (
        <p className={styles.notice}>Plaid Sandbox · Simulated institutions and balances.</p>
      )}
      {state.data && !state.data.enabled && (
        <p className={styles.notice}>
          Bank connections are not available in this environment. Manual balances remain in
          Overview.
        </p>
      )}
    </div>
  );
}
