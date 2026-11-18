import type { Household } from '@bank-piggy/contracts';
import { FinanceCard } from '../../components/FinanceCard';
import { ConnectedAccounts } from './ConnectedAccounts';
import { useConnectionData } from './ConnectionsContext';
import { AccountInstitution } from './AccountInstitution';
import { ConnectionLoading } from './ConnectionStatus';
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
      </header>
      <FinanceCard
        title="Linked accounts"
        description="Latest balances for your household"
        actions={null}
        controls={
          <div className={styles.controls}>
            <ConnectedAccounts household={props.household} showAdd />
          </div>
        }
        footer={
          <p className={styles.note}>Automatically checks for balance updates every 30 minutes.</p>
        }
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
          <p>Choose Add institution to link your first bank or brokerage.</p>
        </div>
      )}
    </>
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
        <div className={styles.error}>
          <p role="alert">{state.error}</p>
          <button
            className={styles.reload}
            disabled={state.busy || state.loading}
            onClick={() => {
              void state.reload();
            }}
          >
            Retry loading accounts
          </button>
        </div>
      )}
      {state.loading && !state.data && <ConnectionLoading label="Loading linked accounts…" />}
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
