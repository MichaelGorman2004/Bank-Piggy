import { useConnectionData } from './ConnectionsContext';
import { hasMissingCurrentBalance } from './presentation';
import styles from './Connections.module.css';

/**
 * Warns about selected unavailable current balances only with a plan-matched projection.
 * @param props Presentation scope used for the displayed server totals.
 * @param props.scope Household or member identifier.
 * @returns A scoped review notice, or no notice during manual fallback.
 */
export function BalanceAvailability(props: { readonly scope: string }) {
  const state = useConnectionData();
  if (
    !state.balances ||
    !state.data ||
    !hasMissingCurrentBalance(state.data.connections, props.scope)
  )
    return null;
  return (
    <p className={styles.notice} role="status">
      Some selected accounts have no usable current balance. Totals exclude those connected
      balances. Open Connected accounts to review them.
    </p>
  );
}
