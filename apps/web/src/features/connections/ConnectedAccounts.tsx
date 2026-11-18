import { Link2, Plus } from 'lucide-react';
import { lazy, Suspense, useState } from 'react';
import type { Household } from '@bank-piggy/contracts';
import { Modal } from '../../components/Modal';
import { ConnectionRow } from './ConnectionRow';
import { ConnectAccountsFlow } from './ConnectAccountsFlow';
import { useConnectionData } from './ConnectionsContext';
import { useConnectionLink } from './useConnectionLink';
import { ConnectionLoading } from './ConnectionStatus';
import styles from './Connections.module.css';
import formStyles from '../plan/Editors.module.css';

const PlaidLaunch = lazy(() =>
  import('./PlaidLaunch').then((module) => ({ default: module.PlaidLaunch })),
);

/**
 * Adds compact account management without changing workspace navigation or manual planning.
 * @param props Saved household used for ownership and explicit manual reconciliation.
 * @param props.household Saved household members and manual accounts.
 * @param props.showAdd Shows a separate entry for adding an institution.
 * @returns Account-management action and bounded native dialog with official Link consent.
 */
export function ConnectedAccounts(props: {
  readonly household: Household;
  readonly showAdd?: boolean;
}) {
  const state = useConnectionData();
  const link = useConnectionLink(state.csrfToken, state);
  const [adding, setAdding] = useState(false);
  const entries = {
    showAdd: props.showAdd ?? false,
    onOpen: (add: boolean) => {
      setAdding(add);
      link.show();
    },
  };
  return (
    <>
      <ConnectionEntries {...entries} />
      {link.open && (
        <Modal
          title={
            link.link?.attempt.purpose === 'activity'
              ? 'Authorize activity history'
              : adding
                ? 'Add an institution'
                : 'Manage connections'
          }
          description={connectionDescription(link, adding)}
          onClose={link.close}
          busy={state.busy || link.preparing}
          suspended={link.external}
          wide={!adding}
        >
          <ConnectionDialogContent household={props.household} link={link} adding={adding} />
        </Modal>
      )}
      {link.link && (
        <Suspense fallback={null}>
          <PlaidLaunch
            session={link.link}
            active={link.external}
            onActivate={link.activate}
            onSuccess={link.finish}
            onExit={link.exit}
          />
        </Suspense>
      )}
    </>
  );
}

/**
 * Separates adding an institution from managing saved connections.
 * @param props Dialog entry actions.
 * @param props.showAdd Shows the primary action on the Accounts page.
 * @param props.onOpen Opens the selected dialog mode.
 * @returns The available connection entry buttons.
 */
function ConnectionEntries(props: {
  readonly showAdd: boolean;
  readonly onOpen: (add: boolean) => void;
}) {
  return (
    <>
      {props.showAdd && (
        <button
          className={formStyles.button}
          onClick={() => {
            props.onOpen(true);
          }}
        >
          <Plus size={14} aria-hidden />
          Add institution
        </button>
      )}
      <button
        className={styles.balanceAction}
        onClick={() => {
          props.onOpen(false);
        }}
      >
        <Link2 size={14} aria-hidden />
        Manage connections
      </button>
    </>
  );
}

/**
 * Keeps errors visible while retaining mounted mapping forms and provider retry actions.
 * @param props Saved household and transient official Link state.
 * @param props.household Saved members and manual accounts.
 * @param props.link Official Link lifecycle and dialog actions.
 * @param props.adding Whether the dialog is for adding an institution.
 * @returns Empty, unavailable, loading, or populated institution management content.
 */
function ConnectionDialogContent(props: {
  readonly household: Household;
  readonly link: ReturnType<typeof useConnectionLink>;
  readonly adding: boolean;
}) {
  const state = useConnectionData();
  return (
    <div className={styles.section}>
      <ConnectionFeedback link={props.link} />
      {props.link.preparing || (props.link.link && !props.link.external) ? (
        <ConnectionLoading label="Opening Plaid…" />
      ) : (
        state.loading && <ConnectionLoading label="Loading saved connections…" />
      )}
      <ConnectionAvailability link={props.link} adding={props.adding} />
      {!props.adding &&
        state.data?.connections.map((connection) => (
          <ConnectionRow
            key={connection.id}
            connection={connection}
            household={props.household}
            linkBlocked={
              props.link.preparing || Boolean(props.link.link) || Boolean(props.link.pendingSave)
            }
            onReconnect={() => {
              void props.link.start({ connectionId: connection.id });
            }}
          />
        ))}
    </div>
  );
}

/**
 * Presents provider request feedback and a retained exchange retry without removing account drafts.
 * @param props Current Link lifecycle.
 * @param props.link Current Link lifecycle.
 * @returns Status reload, mutation error, and explicit completion retry controls.
 */
function ConnectionFeedback(props: { readonly link: ReturnType<typeof useConnectionLink> }) {
  const state = useConnectionData();
  return (
    <>
      {(state.error || props.link.pendingSave) && (
        <button
          className={formStyles.secondary}
          disabled={state.busy || state.loading || props.link.preparing}
          onClick={() => {
            void props.link.reload();
          }}
        >
          Check saved status
        </button>
      )}
      {state.error && (
        <p className={formStyles.error} role="alert">
          {state.error}
        </p>
      )}
      {state.feedback && (
        <p className={formStyles.help} role="status">
          {state.feedback}
        </p>
      )}
      {props.link.pendingSave && (
        <div className={styles.section}>
          <p className={styles.notice}>
            Plaid consent finished, but saving the connection did not finish. Retry saving before
            starting another connection. Check saved status if you already finished in Plaid.
          </p>
          <button
            className={formStyles.secondary}
            disabled={state.busy}
            onClick={() => {
              void props.link.retry();
            }}
          >
            Retry saving connection
          </button>
        </div>
      )}
    </>
  );
}

/**
 * Distinguishes disabled, loading, empty, and simulated provider environments.
 * @param props Current Link lifecycle.
 * @param props.link Current Link lifecycle.
 * @param props.adding Whether the dialog is for adding an institution.
 * @returns Connection choices and accurate environment guidance.
 */
function ConnectionAvailability(props: {
  readonly link: ReturnType<typeof useConnectionLink>;
  readonly adding: boolean;
}) {
  const { data } = useConnectionData();
  if (!data) return null;
  if (!data.enabled) return <p className={styles.notice}>Bank connections are unavailable.</p>;
  return (
    <>
      {data.environment === 'sandbox' && (
        <p className={styles.notice}>Plaid Sandbox · Simulated accounts</p>
      )}
      {props.adding && !props.link.link && <ConnectAccountsFlow link={props.link} />}
      {!props.adding && data.connections.length === 0 && (
        <p className={formStyles.help}>No institutions linked yet.</p>
      )}
    </>
  );
}

/** Names the permission being recovered after an institution redirect.
 * @param link Current server-owned Link lifecycle.
 * @param adding Whether this dialog starts a new connection.
 * @returns The relevant read-only consent description.
 */
function connectionDescription(
  link: ReturnType<typeof useConnectionLink>,
  adding: boolean,
): string {
  return link.link?.attempt.purpose === 'activity'
    ? 'Read-only history for your consented accounts. Open Activity after permission is saved.'
    : adding
      ? 'Choose an account type to open Plaid. Both household members can see the balances you link.'
      : 'Review account settings, reconnect an institution, or disconnect it.';
}
