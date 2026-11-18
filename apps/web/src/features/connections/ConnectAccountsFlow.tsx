import type { useConnectionLink } from './useConnectionLink';
import { useConnectionData } from './ConnectionsContext';
import styles from './Connections.module.css';
import formStyles from '../plan/Editors.module.css';

/**
 * Opens Plaid directly from the account type the user selects.
 * @param props Current shared Link lifecycle.
 * @param props.link Session preparation and completion state.
 * @returns The two supported institution choices.
 */
export function ConnectAccountsFlow(props: {
  readonly link: ReturnType<typeof useConnectionLink>;
}) {
  const state = useConnectionData();
  const disabled = state.busy || props.link.preparing || Boolean(props.link.pendingSave);
  return (
    <div className={styles.actions}>
      <button
        className={formStyles.button}
        disabled={disabled}
        onClick={() => {
          void props.link.start({ kind: 'depository' });
        }}
      >
        Bank accounts
      </button>
      <button
        className={formStyles.button}
        disabled={disabled}
        onClick={() => {
          void props.link.start({ kind: 'investment' });
        }}
      >
        Investment accounts
      </button>
    </div>
  );
}
