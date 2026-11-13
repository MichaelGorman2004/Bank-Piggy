import { useConnectionData } from './ConnectionsContext';
import { hasMissingCurrentBalance } from './presentation';
import styles from './Connections.module.css';

/**
 * Explains pending totals and missing snapshots without presenting manual values as linked facts.
 * @param props Presentation scope used for the displayed server totals.
 * @param props.scope Household or member identifier.
 * @returns Pending, failed, or scoped missing-balance guidance.
 */
export function BalanceAvailability(props: { readonly scope: string }) {
  const state = useConnectionData();
  if (!state.balances) {
    return (
      <p className={styles.notice} role={state.error ? 'alert' : 'status'}>
        {state.error
          ? 'Balances could not load. Open Connected accounts to retry.'
          : state.data && !state.loading
            ? 'The saved plan changed. Reload this page to load current balances.'
            : 'Loading current balances…'}
      </p>
    );
  }
  if (!state.data || !hasMissingCurrentBalance(state.data.connections, props.scope)) return null;
  return (
    <p className={styles.notice} role="status">
      Some selected accounts have no usable current balance. Totals exclude those connected
      balances. Open Connected accounts to review them.
    </p>
  );
}
