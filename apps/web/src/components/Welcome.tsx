import { ArrowRight, LockKeyhole } from 'lucide-react';
import type { Session } from '@bank-piggy/contracts';
import styles from './Welcome.module.css';

interface WelcomeProps {
  readonly session: Session | null;
  readonly error: string | null;
  readonly loading: boolean;
  readonly onDemo: () => void;
  readonly onRetry: () => void;
}

/**
 * Introduces the shared household plan with its banker pig mascot.
 * @returns Rendered interface.
 */
function WelcomeStory() {
  return (
    <div className={styles.story}>
      <div className={styles.brand}>
        Bank Piggy<span>Your money, together.</span>
      </div>
      <img
        className={styles.pig}
        src="/assets/banker-piggy.png"
        alt="Bank Piggy, a pink pig in a brown suit and orange tie"
      />
      <div>
        <h1>
          Your household money,
          <br />
          in one place.
        </h1>
        <p>See where your money goes. Make a plan that works for both of you.</p>
      </div>
    </div>
  );
}

/**
 * Separates private sign-in from the fictional sample session.
 * @param props Component values and event handlers.
 * @param props.session Current cookie-backed session.
 * @param props.onDemo Opens the sample household.
 * @returns Rendered interface.
 */
function EntryActions(props: { readonly session: Session; readonly onDemo: () => void }) {
  const configured = props.session.googleConfigured;
  return (
    <>
      <a
        className={`${styles.google} ${configured ? '' : styles.disabled}`}
        href={configured ? '/auth/google' : undefined}
        aria-disabled={!configured}
      >
        <span className={styles.googleLetter} aria-hidden="true">
          G
        </span>
        Continue with Google
      </a>
      {!configured && (
        <p className={styles.note}>Private Google sign-in is not configured on this instance.</p>
      )}
      {props.session.mode === 'demo' && <DemoEntry onDemo={props.onDemo} />}
    </>
  );
}

/**
 * Opens the sample only when the server has enabled demo mode.
 * @param props Sample session action.
 * @param props.onDemo Opens the sample household.
 * @returns Sample entry controls.
 */
function DemoEntry(props: { readonly onDemo: () => void }) {
  return (
    <>
      <div className={styles.divider}>
        <span>Your household plan</span>
      </div>
      <button className={styles.demo} onClick={props.onDemo}>
        Open household plan
        <ArrowRight size={18} />
      </button>
      <p className={styles.note}>No bank connection. Changes are saved to this household plan.</p>
    </>
  );
}

/**
 * Gives loading failures a direct retry action.
 * @param props Component values and event handlers.
 * @returns Rendered interface.
 */
function WelcomeEntry(props: WelcomeProps) {
  return (
    <div className={styles.entry}>
      <div className={styles.lock}>
        <LockKeyhole size={21} />
      </div>
      <h2>Welcome to Bank Piggy.</h2>
      <p>Manage your household’s monthly plan.</p>
      {props.loading ? (
        <p className={styles.status} role="status">
          Opening Bank Piggy…
        </p>
      ) : (
        props.session && <EntryActions session={props.session} onDemo={props.onDemo} />
      )}
      {props.error && (
        <div className={styles.error} role="alert">
          <p>{props.error}</p>
          <button onClick={props.onRetry} disabled={props.loading}>
            Try again
          </button>
        </div>
      )}
    </div>
  );
}

/**
 * Offers configured Google sign-in and the loopback-only sample household.
 * @param props Component values and event handlers.
 * @returns Rendered interface.
 */
export function Welcome(props: WelcomeProps) {
  return (
    <main className={styles.page}>
      <section className={styles.card}>
        <WelcomeStory />
        <WelcomeEntry {...props} />
      </section>
    </main>
  );
}
