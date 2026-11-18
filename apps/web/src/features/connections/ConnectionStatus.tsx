import { CircleCheck, LoaderCircle } from 'lucide-react';
import type { ConnectionState } from '@bank-piggy/contracts';
import { CONNECTION_STATE_LABELS } from './presentation';
import styles from './Connections.module.css';

/** Shows a check only for the server's connected state.
 * @param props Saved connection state.
 * @param props.state Current institution state.
 * @returns Visible status with its text label.
 */
export function ConnectionStatus({ state }: { readonly state: ConnectionState }) {
  return (
    <span className={styles.status}>
      {state === 'connected' && <CircleCheck className={styles.connected} size={14} aria-hidden />}
      {CONNECTION_STATE_LABELS[state]}
    </span>
  );
}

/** Announces connection progress while keeping surrounding content visible.
 * @param props Current loading message.
 * @param props.label Names the operation in progress.
 * @returns A spinner and accessible status text.
 */
export function ConnectionLoading({ label }: { readonly label: string }) {
  return (
    <p className={`${styles.status} ${styles.meta}`} role="status">
      <LoaderCircle className={styles.spinner} size={16} aria-hidden />
      {label}
    </p>
  );
}
