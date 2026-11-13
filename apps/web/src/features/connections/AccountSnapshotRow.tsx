import type { ConnectedAccount } from '@bank-piggy/contracts';
import { accountSnapshotMoney } from './accountSnapshotMoney';
import { accountSnapshotBalanceLabel, accountSnapshotType } from './accountSnapshotLabels';
import styles from './AccountsDashboard.module.css';

/**
 * Shows the primary provider balance without deriving totals or replacing missing values.
 * @param props One raw account snapshot.
 * @param props.account Raw provider account and saved classification.
 * @returns An account row that remains visible when inactive or excluded.
 */
export function AccountSnapshotRow(props: { readonly account: ConnectedAccount }) {
  const { account } = props;
  const label = accountSnapshotBalanceLabel(account);
  const balance = accountSnapshotMoney(account.currentBalanceCents, account.currency);
  return (
    <article className={styles.account}>
      <div className={styles.accountIdentity}>
        <h4>
          {account.name}
          {account.mask ? ` · ${account.mask}` : ''}
        </h4>
        <p className={styles.note}>
          {accountSnapshotType(account)} · {account.active ? 'Active' : 'Inactive'}
        </p>
      </div>
      <dl className={styles.balances}>
        <div>
          <dt>{label}</dt>
          <dd>{balance === 'Unavailable' ? `${label} not provided` : balance}</dd>
        </div>
      </dl>
    </article>
  );
}
