import { lazy, Suspense } from 'react';
import { Modal } from '../../components/Modal';
import type { useConnectionLink } from '../connections/useConnectionLink';
import { useConnectionData } from '../connections/ConnectionsContext';
import { ConnectionLoading } from '../connections/ConnectionStatus';
import styles from './Activity.module.css';
import formStyles from '../plan/Editors.module.css';

const PlaidLaunch = lazy(() =>
  import('../connections/PlaidLaunch').then((module) => ({ default: module.PlaidLaunch })),
);

/**
 * Retains one official Link permission flow and completion retry for the Activity dashboard.
 * @param props Current shared Link lifecycle.
 * @param props.link One dashboard-owned connection Link lifecycle.
 * @returns A native permission dialog suspended while official Plaid Link is active.
 */
export function ActivityLink(props: { readonly link: ReturnType<typeof useConnectionLink> }) {
  const { link } = props;
  const state = useConnectionData();
  return (
    <>
      {link.pendingSave && !link.open && (
        <button className={formStyles.secondary} onClick={link.show}>
          Finish saving activity permission
        </button>
      )}
      {link.open && (
        <Modal
          title="Institution access"
          description="Complete read-only institution permission in official Plaid Link. Your saved pay plan changes only after you confirm a pay suggestion."
          busy={state.busy || link.preparing}
          suspended={link.external}
          onClose={link.close}
        >
          <ActivityLinkFeedback link={link} />
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

/** Keeps permission failures and completion retries inside the bounded dialog.
 * @param props Dashboard-owned official Link lifecycle.
 * @param props.link Current permission and retry actions.
 * @returns Safe permission feedback and explicit retry controls.
 */
function ActivityLinkFeedback(props: { readonly link: ReturnType<typeof useConnectionLink> }) {
  const state = useConnectionData();
  const { link } = props;
  return (
    <div className={styles.form}>
      {state.error && (
        <p className={formStyles.error} role="alert">
          {state.error}
        </p>
      )}
      {state.feedback && (
        <p className={styles.note} role="status">
          {state.feedback}
        </p>
      )}
      {(link.preparing || link.link) && <ConnectionLoading label="Opening Plaid…" />}
      <ActivityLinkRetry link={link} />
      <ActivityLinkActions link={link} />
    </div>
  );
}

/** Retains an explicit completion retry after official permission succeeds.
 * @param props Current permission lifecycle.
 * @param props.link Failed completion and retry action.
 * @returns Completion retry guidance when a save is pending.
 */
function ActivityLinkRetry(props: { readonly link: ReturnType<typeof useConnectionLink> }) {
  const state = useConnectionData();
  const { link } = props;
  if (!link.pendingSave) return null;
  return (
    <>
      <p className={styles.notice}>
        Plaid permission finished, but saving did not finish. Retry saving or reload status before
        starting again.
      </p>
      <button
        className={formStyles.button}
        disabled={state.busy}
        onClick={() => {
          void link.retry();
        }}
      >
        Retry saving permission
      </button>
    </>
  );
}

/** Keeps permission status reload and close controls visible.
 * @param props Current permission lifecycle.
 * @param props.link Reload and close actions.
 * @returns Native permission actions.
 */
function ActivityLinkActions(props: { readonly link: ReturnType<typeof useConnectionLink> }) {
  const state = useConnectionData();
  const { link } = props;
  return (
    <div className={styles.actions}>
      <button
        className={formStyles.secondary}
        disabled={state.loading || state.busy || link.preparing}
        onClick={() => {
          void link.reload();
        }}
      >
        Reload status
      </button>
      <button
        className={formStyles.secondary}
        disabled={state.busy || link.preparing}
        onClick={link.close}
      >
        Done
      </button>
    </div>
  );
}
