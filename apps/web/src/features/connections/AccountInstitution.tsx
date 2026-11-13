import type { Connection, Household } from '@bank-piggy/contracts';
import { AccountSnapshotRow } from './AccountSnapshotRow';
import { CONNECTION_STATE_LABELS, connectionNotice, syncTime } from './presentation';
import styles from './AccountsDashboard.module.css';

/**
 * Groups raw account snapshots by institution login and exposes update freshness.
 * @param props Authorized institution and household member names.
 * @param props.connection Authorized institution state and all its accounts.
 * @param props.household Saved household members.
 * @returns Institution state, saved update times, and every linked account.
 */
export function AccountInstitution(props: {
  readonly connection: Connection;
  readonly household: Household;
}) {
  const { connection, household } = props;
  const member = household.members.find((item) => item.id === connection.connectedByMemberId);
  const notice = connectionNotice(connection.errorCode);
  return (
    <section className={styles.institution} aria-label={connection.institutionName}>
      <div className={styles.institutionHeading}>
        <div>
          <h3>{connection.institutionName}</h3>
          <p className={styles.note}>Plaid · Linked by {member?.name ?? 'household member'}</p>
        </div>
        <span className={styles.state}>{CONNECTION_STATE_LABELS[connection.state]}</span>
      </div>
      <p className={styles.freshness}>
        Last successful sync: {syncTime(connection.lastSyncedAt)}
        {connection.nextSyncAt && ` · Next check: ${syncTime(connection.nextSyncAt)}`}
      </p>
      {notice && <p className={styles.notice}>{notice}</p>}
      {connection.accounts.length === 0 && (
        <p className={styles.message}>
          No account balances received yet. Use Connect accounts to review this connection.
        </p>
      )}
      <div className={styles.accounts}>
        {connection.accounts.map((account) => (
          <AccountSnapshotRow key={account.id} account={account} household={household} />
        ))}
      </div>
    </section>
  );
}
