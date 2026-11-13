import { useState } from 'react';
import type { Connection, Household } from '@bank-piggy/contracts';
import { disconnectConnection, refreshConnection } from './api';
import { useConnectionData } from './ConnectionsContext';
import { ConnectedAccountRow } from './ConnectedAccountRow';
import {
  CONNECTION_STATE_LABELS,
  connectionNotice,
  disconnectFeedback,
  syncTime,
} from './presentation';
import styles from './Connections.module.css';
import formStyles from '../plan/Editors.module.css';

interface ConnectionRowProps {
  readonly connection: Connection;
  readonly household: Household;
  readonly onReconnect: () => void;
  readonly linkBlocked: boolean;
}

/**
 * Shows connection ownership, freshness, and explicit provider recovery actions.
 * @param props Authorized connection and saved household members.
 * @returns One institution with its account mapping disclosures.
 */
export function ConnectionRow(props: ConnectionRowProps) {
  const { connection, household } = props;
  const state = useConnectionData();
  const memberName =
    household.members.find((item) => item.id === connection.connectedByMemberId)?.name ??
    'the connecting member';
  return (
    <section className={styles.connection} aria-label={connection.institutionName}>
      <h3>{connection.institutionName}</h3>
      <ConnectionMetadata connection={connection} memberName={memberName} />
      <ConnectionControls {...props} memberName={memberName} />
      <div className={styles.accounts}>
        {connection.accounts.map((account) => (
          <ConnectedAccountRow
            key={account.id}
            account={account}
            connectionId={connection.id}
            household={household}
            csrfToken={state.csrfToken}
          />
        ))}
      </div>
    </section>
  );
}

/**
 * Labels connection state and provider freshness without promising real-time data.
 * @param props Authorized connection and connecting member name.
 * @param props.connection Current connection snapshot.
 * @param props.memberName Saved connecting member name.
 * @returns Connection state and scheduled-update timing.
 */
function ConnectionMetadata(props: {
  readonly connection: Connection;
  readonly memberName: string;
}) {
  const connection = props.connection;
  const notice = connectionNotice(connection.errorCode);
  return (
    <p className={styles.meta}>
      {CONNECTION_STATE_LABELS[connection.state]} · Linked by {props.memberName}
      <br />
      Last successful update: {syncTime(connection.lastSyncedAt)}
      <br />
      {connection.nextSyncAt && (
        <>
          Next scheduled check: {syncTime(connection.nextSyncAt)}
          <br />
        </>
      )}
      Updates are scheduled snapshots. Institution data can be older.
      {notice && (
        <>
          <br />
          {notice}
        </>
      )}
    </p>
  );
}

/**
 * Retains disconnect confirmation after a failed provider removal.
 * @param props Current connection, connecting member, and Link reconnect action.
 * @returns Authorized connection actions or removal confirmation.
 */
function ConnectionControls(props: ConnectionRowProps & { readonly memberName: string }) {
  const [confirm, setConfirm] = useState(false);
  if (props.connection.state === 'disconnected') return null;
  return (
    <>
      <ConnectionActions
        connection={props.connection}
        onReconnect={props.onReconnect}
        linkBlocked={props.linkBlocked}
        onDisconnect={() => {
          setConfirm(true);
        }}
      />
      {!props.connection.canManage && (
        <p className={styles.meta}>
          Only {props.memberName} can reconnect or disconnect this institution.
        </p>
      )}
      {confirm && (
        <DisconnectConfirmation
          connection={props.connection}
          linkBlocked={props.linkBlocked}
          onCancel={() => {
            setConfirm(false);
          }}
        />
      )}
    </>
  );
}

/**
 * Uses server capabilities for deliberate refresh, reconnect, and disconnect actions.
 * @param props Authorized connection and explicit action callbacks.
 * @param props.connection Authorized connection snapshot.
 * @param props.onReconnect Opens official Link in update mode.
 * @param props.onDisconnect Opens removal confirmation.
 * @param props.linkBlocked Whether unresolved Link work owns connection changes.
 * @returns Native provider action buttons.
 */
function ConnectionActions(props: {
  readonly connection: Connection;
  readonly onReconnect: () => void;
  readonly onDisconnect: () => void;
  readonly linkBlocked: boolean;
}) {
  const state = useConnectionData();
  const canRefresh = props.connection.state === 'connected' || props.connection.state === 'delayed';
  return (
    <div className={styles.actions}>
      {canRefresh && (
        <button
          className={formStyles.secondary}
          disabled={state.busy || props.linkBlocked}
          onClick={() => {
            void state.run(
              () => refreshConnection(props.connection.id, state.csrfToken),
              'Update requested. Balances change after a successful scheduled check.',
            );
          }}
        >
          Request update
        </button>
      )}
      {props.connection.canReconnect && (
        <button
          className={formStyles.secondary}
          disabled={state.busy || props.linkBlocked}
          onClick={props.onReconnect}
        >
          Reconnect with Plaid
        </button>
      )}
      {props.connection.canManage && (
        <button
          className={formStyles.secondary}
          disabled={state.busy || props.linkBlocked}
          onClick={props.onDisconnect}
        >
          Disconnect
        </button>
      )}
    </div>
  );
}

/**
 * Keeps removal intent available when provider revocation fails.
 * @param props Connection being removed and cancellation action.
 * @param props.connection Authorized connection being removed.
 * @param props.onCancel Returns to the institution actions.
 * @param props.linkBlocked Whether unresolved Link work owns connection changes.
 * @returns Explicit removal confirmation and a retryable provider operation.
 */
function DisconnectConfirmation(props: {
  readonly connection: Connection;
  readonly onCancel: () => void;
  readonly linkBlocked: boolean;
}) {
  const state = useConnectionData();
  return (
    <div className={styles.confirm}>
      <p className={formStyles.help}>
        Disconnect {props.connection.institutionName}? Bank Piggy will revoke provider access and
        remove its connected balances. Saved manual balances remain. A later new connection can use
        another production allowance slot.
      </p>
      <div className={styles.actions}>
        <button
          className={formStyles.danger}
          disabled={state.busy || props.linkBlocked}
          onClick={() => {
            void state.run(
              () => disconnectConnection(props.connection.id, state.csrfToken),
              (result) => disconnectFeedback(result, props.connection.id),
            );
          }}
        >
          {state.busy ? 'Disconnecting…' : 'Confirm disconnect'}
        </button>
        <button
          className={formStyles.secondary}
          disabled={state.busy || props.linkBlocked}
          onClick={props.onCancel}
        >
          Keep connection
        </button>
      </div>
    </div>
  );
}
