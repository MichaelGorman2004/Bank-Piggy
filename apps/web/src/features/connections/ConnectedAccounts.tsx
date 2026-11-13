import { Link2 } from 'lucide-react';
import { lazy, Suspense } from 'react';
import type { Household } from '@bank-piggy/contracts';
import { Modal } from '../../components/Modal';
import { ConnectionRow } from './ConnectionRow';
import { ConnectAccountsFlow } from './ConnectAccountsFlow';
import { useConnectionData } from './ConnectionsContext';
import { useConnectionLink } from './useConnectionLink';
import styles from './Connections.module.css';
import formStyles from '../plan/Editors.module.css';

const PlaidLaunch = lazy(() =>
  import('./PlaidLaunch').then((module) => ({ default: module.PlaidLaunch })),
);

/**
 * Adds compact account management without changing workspace navigation or manual planning.
 * @param props Saved household used for ownership and explicit manual reconciliation.
 * @param props.household Saved household members and manual accounts.
 * @param props.label Optional action label for the current financial view.
 * @param props.connectOnOpen Opens at the account choice without starting a provider request.
 * @returns Account-management action and bounded native dialog with official Link consent.
 */
export function ConnectedAccounts(props: {
  readonly household: Household;
  readonly label?: string;
  readonly connectOnOpen?: boolean;
}) {
  const state = useConnectionData();
  const link = useConnectionLink(state.csrfToken, state);
  return (
    <>
      <button className={styles.balanceAction} onClick={link.show}>
        <Link2 size={14} />
        {props.label ?? 'Connected accounts'}
      </button>
      {link.open && (
        <Modal
          title="Connected accounts"
          description="Read-only balances for your shared household. Your budget and goal progress stay separate."
          onClose={link.close}
          busy={state.busy || link.preparing}
          suspended={link.external}
          wide
        >
          <ConnectionDialogContent
            household={props.household}
            link={link}
            initialChoosing={props.connectOnOpen ?? false}
          />
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
 * Keeps errors visible while retaining mounted mapping forms and provider retry actions.
 * @param props Saved household and transient official Link state.
 * @param props.household Saved members and manual accounts.
 * @param props.link Official Link lifecycle and dialog actions.
 * @param props.initialChoosing Whether a newly mounted choice starts open.
 * @returns Empty, unavailable, loading, or populated institution management content.
 */
function ConnectionDialogContent(props: {
  readonly household: Household;
  readonly link: ReturnType<typeof useConnectionLink>;
  readonly initialChoosing: boolean;
}) {
  const state = useConnectionData();
  return (
    <div className={styles.section}>
      <ConnectionFeedback link={props.link} />
      {props.link.link && !props.link.external && (
        <p className={formStyles.help} role="status">
          Loading official Plaid Link…
        </p>
      )}
      <ConnectionAvailability link={props.link} initialChoosing={props.initialChoosing} />
      {state.data?.connections.map((connection) => (
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
      <p className={formStyles.help}>
        Bank Piggy never asks for your bank password or verification codes. Complete consent in
        official Plaid Link or your institution’s page. Keep unsupported accounts as manual
        balances.
      </p>
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
      <div className={styles.toolbar}>
        <p className={formStyles.help}>Both members can see these account balances.</p>
        <button
          className={formStyles.secondary}
          disabled={state.busy || state.loading || props.link.preparing}
          onClick={() => {
            void props.link.reload();
          }}
        >
          {state.loading ? 'Loading…' : 'Reload status'}
        </button>
      </div>
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
            starting another connection. If the response was lost, Reload status can show the saved
            connection.
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
 * @param props.initialChoosing Whether the connect flow starts at its native account choice.
 * @returns Connection choices and accurate environment guidance.
 */
function ConnectionAvailability(props: {
  readonly link: ReturnType<typeof useConnectionLink>;
  readonly initialChoosing: boolean;
}) {
  const state = useConnectionData();
  return (
    <>
      {state.data?.enabled ? (
        <>
          {!props.link.link && (
            <ConnectAccountsFlow link={props.link} initialChoosing={props.initialChoosing} />
          )}
        </>
      ) : (
        state.data && (
          <p className={styles.notice}>
            Bank connections are not available in this environment. You can keep using manual
            accounts.
          </p>
        )
      )}
      {state.data?.environment === 'sandbox' && (
        <p className={styles.notice}>
          Plaid Sandbox · Test institutions and simulated balances. This is not live bank data.
        </p>
      )}
      {state.data && !state.data.connections.length && (
        <p className={formStyles.help}>
          No institutions connected. Link an institution, then review account ownership, type, and
          manual replacements before including balances.
        </p>
      )}
    </>
  );
}
