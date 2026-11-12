import { useCallback, useEffect, useState } from 'react';
import type { InvitationPreview, Session } from '@bank-piggy/contracts';
import { loadSession } from '../../api';
import { errorMessage } from '../session/errors';
import { prepareJoin } from './joinInvitation';
import styles from '../../components/Welcome.module.css';

interface JoinConfirmationProps {
  readonly preview: InvitationPreview;
  readonly session: Session | null;
}

/**
 * Prepares a scrubbed invitation and loads sign-in availability without loading finances.
 * @returns Nonfinancial preview, feedback, and a deliberate retry action.
 */
function useJoinPreview() {
  const [preview, setPreview] = useState<InvitationPreview | null>(null);
  const [session, setSession] = useState<Session | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    let active = true;
    void Promise.all([prepareJoin(), loadSession()])
      .then(([nextPreview, nextSession]) => {
        if (active) {
          setPreview(nextPreview);
          setSession(nextSession);
        }
      })
      .catch((failure: unknown) => {
        if (active) setError(errorMessage(failure));
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [attempt]);
  const retry = useCallback(() => {
    setError(null);
    setLoading(true);
    setAttempt((current) => current + 1);
  }, []);
  return { preview, session, error, loading, retry };
}

/**
 * Confirms the partner profile before Google sign-in; no financial plan is requested.
 * @returns A nonfinancial join page after the invitation token has been scrubbed.
 */
export function JoinPage() {
  const state = useJoinPreview();
  return (
    <main className={styles.page}>
      <section className={styles.card}>
        <div className={styles.story}>
          <div className={styles.brand}>
            Bank Piggy<span>Your money, in one place.</span>
          </div>
          <img
            className={styles.pig}
            src="/assets/banker-piggy.png"
            alt="Bank Piggy, a pink pig in a brown suit and orange tie"
          />
          <div>
            <h1>A shared household plan.</h1>
            <p>Join with your own Google account.</p>
          </div>
        </div>
        <div className={styles.entry}>
          <h2>Join your household.</h2>
          {state.loading && (
            <p className={styles.status} role="status">
              Checking invitation…
            </p>
          )}
          {state.preview && <JoinConfirmation preview={state.preview} session={state.session} />}
          {state.error && (
            <div className={styles.error} role="alert">
              <p>{state.error}</p>
              <button disabled={state.loading} onClick={state.retry}>
                Try again
              </button>
            </div>
          )}
          <p className={styles.note}>
            <a href="/">Back to Bank Piggy</a>
          </p>
        </div>
      </section>
    </main>
  );
}

/**
 * Names the invitation and gives the user the explicit Google join action.
 * @param props Authorized invitation preview and configured sign-in availability.
 * @returns Confirmation text and a Google join link when private sign-in is ready.
 */
function JoinConfirmation(props: JoinConfirmationProps) {
  const configured = props.session?.mode === 'private' && props.session.googleConfigured;
  return (
    <>
      <p>
        Join {props.preview.householdName} as {props.preview.memberName}.
      </p>
      <p className={styles.note}>
        Google sign-in links your account to this household. Both members can see the shared plan.
      </p>
      <a
        className={`${styles.google} ${configured ? '' : styles.disabled}`}
        href={configured ? '/auth/google?join=1' : undefined}
        aria-disabled={!configured}
      >
        <span className={styles.googleLetter} aria-hidden="true">
          G
        </span>
        Join household with Google
      </a>
      {!configured && (
        <p className={styles.note}>Private Google sign-in is not configured on this instance.</p>
      )}
      <p className={styles.note}>
        Invitation expires {new Date(props.preview.expiresAt).toLocaleString()}.
      </p>
    </>
  );
}
