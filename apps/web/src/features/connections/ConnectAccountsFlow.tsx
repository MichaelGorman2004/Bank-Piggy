import { useEffect, useId, useRef, useState } from 'react';
import type { LinkTokenRequest } from '@bank-piggy/contracts';
import { useConnectionData } from './ConnectionsContext';
import type { useConnectionLink } from './useConnectionLink';
import styles from './Connections.module.css';
import formStyles from '../plan/Editors.module.css';

type LinkState = ReturnType<typeof useConnectionLink>;
type ConnectionKind = NonNullable<LinkTokenRequest['kind']>;

/**
 * Opens an inline product choice before each separate institution consent.
 * @param props Official Link lifecycle and request locks.
 * @param props.link Official Link lifecycle.
 * @param props.initialChoosing Starts at the account choice on mount without requesting consent.
 * @returns One entry action, an optional choice, and consent guidance.
 */
export function ConnectAccountsFlow(props: {
  readonly link: LinkState;
  readonly initialChoosing?: boolean;
}) {
  const state = useConnectionData();
  const [choosing, setChoosing] = useState(props.initialChoosing ?? false);
  const entry = useRef<HTMLButtonElement>(null);
  const previousChoice = useRef(false);
  const choiceId = useId();
  const busy = state.busy || props.link.preparing || Boolean(props.link.pendingSave);
  useEffect(() => {
    if (!choosing && previousChoice.current) entry.current?.focus();
    previousChoice.current = choosing;
  }, [choosing]);
  return (
    <div className={styles.linkPanel}>
      {!choosing && (
        <div className={styles.actions}>
          <button
            ref={entry}
            className={formStyles.button}
            disabled={busy}
            aria-expanded={choosing}
            aria-controls={choiceId}
            onClick={() => {
              setChoosing(true);
            }}
          >
            Connect accounts
          </button>
        </div>
      )}
      {choosing && (
        <ConnectionChoice
          id={choiceId}
          link={props.link}
          busy={busy}
          onCancel={() => {
            setChoosing(false);
          }}
        />
      )}
      <ConnectionGuidance />
    </div>
  );
}

/**
 * Retains the selected product through blocked requests and preparation failures.
 * @param props Official Link start action, request locks, and cancel action.
 * @param props.id Identifier referenced by the entry action.
 * @param props.link Official Link lifecycle.
 * @param props.busy Whether another connection operation holds the request lock.
 * @param props.onCancel Returns to the single entry action.
 * @returns Native product selection and explicit Plaid launch controls.
 */
function ConnectionChoice(props: {
  readonly id: string;
  readonly link: LinkState;
  readonly busy: boolean;
  readonly onCancel: () => void;
}) {
  const [kind, setKind] = useState<ConnectionKind>('depository');
  const selection = useRef<HTMLSelectElement>(null);
  const starting = useRef(false);
  useEffect(() => {
    selection.current?.focus();
  }, []);
  /** Starts at most one request while preserving the choice on a blocked start. */
  async function continueToPlaid() {
    if (props.busy || starting.current) return;
    starting.current = true;
    try {
      await props.link.start({ kind });
    } finally {
      starting.current = false;
    }
  }
  return (
    <div id={props.id} className={formStyles.section}>
      <label>
        Which accounts do you want to connect?
        <select
          ref={selection}
          value={kind}
          disabled={props.busy}
          onChange={(event) => {
            setKind(event.target.value === 'investment' ? 'investment' : 'depository');
          }}
        >
          <option value="depository">Bank accounts</option>
          <option value="investment">Investments</option>
        </select>
      </label>
      <ChoiceActions busy={props.busy} onContinue={continueToPlaid} onCancel={props.onCancel} />
    </div>
  );
}

/**
 * Keeps explicit consent and cancel actions together with the shared request lock.
 * @param props Request state and deliberate user actions.
 * @param props.busy Whether a connection operation is active.
 * @param props.onContinue Starts official Plaid Link for the selected product.
 * @param props.onCancel Returns to the entry action.
 * @returns Accessible native action buttons.
 */
function ChoiceActions(props: {
  readonly busy: boolean;
  readonly onContinue: () => Promise<void>;
  readonly onCancel: () => void;
}) {
  return (
    <div className={styles.actions}>
      <button
        className={formStyles.button}
        disabled={props.busy}
        onClick={() => void props.onContinue()}
      >
        Continue to Plaid
      </button>
      <button className={formStyles.secondary} disabled={props.busy} onClick={props.onCancel}>
        Cancel
      </button>
    </div>
  );
}

/**
 * Explains separate institution consent and limited provider availability.
 * @returns Institution consent and provider allowance guidance.
 */
function ConnectionGuidance() {
  return (
    <p className={formStyles.help}>
      Add institutions one at a time. Approve each connection in official Plaid Link. To add another
      institution, choose Connect accounts again. Institution and account availability vary. Use
      Reconnect for existing connections. Production linking has a limited allowance; removing a
      connection does not restore it.
    </p>
  );
}
