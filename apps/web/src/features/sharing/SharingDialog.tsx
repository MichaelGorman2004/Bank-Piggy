import { Modal } from '../../components/Modal';
import type { useSharing } from './useSharing';
import styles from '../plan/Editors.module.css';
import ownStyles from './Sharing.module.css';

type SharingState = ReturnType<typeof useSharing>;
interface SharingProps {
  readonly state: SharingState;
}

/**
 * Formats invitation expiry without including the secret invitation URL.
 * @param expiry Server-issued timestamp.
 * @returns Local expiry label.
 */
function expiryLabel(expiry: string): string {
  return new Date(expiry).toLocaleString('en-US', { dateStyle: 'medium', timeStyle: 'short' });
}

/**
 * Offers a partner link and keeps invitation errors separate from plan saves.
 * @param props Current sharing state and authorized actions.
 * @returns A bounded dialog with copy, revoke, and replacement controls.
 */
export function SharingDialog(props: SharingProps) {
  const state = props.state;
  if (!state.open) return null;
  return (
    <Modal
      title="Share your household"
      description="Invite your partner to join this household with their own Google account."
      onClose={state.close}
      busy={state.busy}
    >
      <div className={styles.section}>
        {state.planSaved && <p className={ownStyles.saved}>Your household plan is saved.</p>}
        {!state.available ? (
          <p className={styles.help}>
            Household sharing needs private mode with Google sign-in configured. Local demo access
            cannot create a usable invitation.
          </p>
        ) : (
          <ShareContent state={state} />
        )}
        <ShareFailure state={state} />
        {state.feedback && (
          <p className={styles.help} role="status">
            {state.feedback}
          </p>
        )}
        <div className={styles.actions}>
          <button className={styles.secondary} disabled={state.busy} onClick={state.close}>
            Done
          </button>
        </div>
      </div>
    </Modal>
  );
}

/**
 * Retries only the invitation after an accepted financial save.
 * @param props Current sharing failure and request actions.
 * @returns Visible error feedback with a deliberate retry action.
 */
function ShareFailure(props: SharingProps) {
  const state = props.state;
  if (!state.error) return null;
  return (
    <div className={styles.error} role="alert">
      <p>
        {state.planSaved
          ? state.failedAction === 'revoke'
            ? 'Plan saved; link revocation failed. '
            : 'Plan saved; invitation failed. '
          : ''}
        {state.error}
      </p>
      <button
        type="button"
        className={styles.secondary}
        disabled={state.busy}
        onClick={() => {
          void state.retry();
        }}
      >
        {state.failedAction === 'revoke' ? 'Retry revocation' : 'Retry invitation'}
      </button>
    </div>
  );
}

/**
 * Shows linked-account status before exposing explicit link actions.
 * @param props Current invitation state and actions.
 * @returns Availability, expiry, and deliberate invitation controls.
 */
function ShareContent(props: SharingProps) {
  const state = props.state;
  if (state.busy && !state.status)
    return (
      <p className={styles.help} role="status">
        Opening invitation…
      </p>
    );
  if (state.status?.linked)
    return (
      <p className={styles.help}>
        Both household members have linked sign-ins. This household has no open invitation slot.
      </p>
    );
  if (!state.status?.canInvite)
    return <p className={styles.help}>Only the first household member can invite a partner.</p>;
  return (
    <>
      <p className={styles.help}>
        Send this link to {state.status.memberName ?? 'your partner'}. The first person who uses the
        link and completes Google sign-in joins your household. Share it only with your partner.
      </p>
      <ShareLink state={state} />
      <ShareActions state={state} />
    </>
  );
}

/**
 * Shows the in-memory link and its expiry without retaining it between dialogs.
 * @param props Current raw URL and nonfinancial invitation status.
 * @returns Selectable link, expiry, and replacement-link explanation.
 */
function ShareLink(props: SharingProps) {
  const state = props.state;
  const expiry = state.created?.expiresAt ?? state.status?.expiresAt;
  return (
    <>
      {state.created && (
        <label>
          Invitation link
          <input
            className={ownStyles.link}
            readOnly
            value={state.created.url}
            onFocus={(event) => {
              event.target.select();
            }}
          />
        </label>
      )}
      {expiry && (
        <p className={styles.help}>
          Link expiry: {expiryLabel(expiry)}. An expired link cannot be used.
        </p>
      )}
      {state.status?.pending && !state.created && (
        <p className={styles.help}>
          A link was already created. Create a new link if you no longer have it. The earlier link
          will stop working.
        </p>
      )}
    </>
  );
}

/**
 * Creates, replaces, copies, or revokes a link only after a button action.
 * @param props Current link and authorized invitation actions.
 * @returns Explicit sharing actions with serialized request controls.
 */
function ShareActions(props: SharingProps) {
  const state = props.state;
  return (
    <div className={ownStyles.actions}>
      {state.created && (
        <button
          className={styles.button}
          disabled={state.busy}
          onClick={() => {
            void state.copy();
          }}
        >
          Copy link
        </button>
      )}
      <button
        className={styles.secondary}
        disabled={state.busy}
        onClick={() => {
          void state.create();
        }}
      >
        {state.busy
          ? 'Creating link…'
          : state.status?.pending
            ? 'Create new link'
            : 'Create invitation link'}
      </button>
      {state.status?.pending && (
        <button
          className={styles.secondary}
          disabled={state.busy}
          onClick={() => {
            void state.revoke();
          }}
        >
          Revoke link
        </button>
      )}
    </div>
  );
}
