import type { ConnectedAccount, Household } from '@bank-piggy/contracts';
import { useConnectionData } from './ConnectionsContext';
import { syncTime } from './presentation';
import { accountSnapshotMoney } from './accountSnapshotMoney';
import { accountSnapshotBalanceLabels, accountSnapshotType } from './accountSnapshotLabels';
import styles from './AccountsDashboard.module.css';

interface SnapshotProps {
  readonly account: ConnectedAccount;
  readonly household: Household;
}

/**
 * Shows current and available provider values separately without deriving totals.
 * @param props One raw account and saved household ownership labels.
 * @returns An account row that remains visible when inactive or excluded.
 */
export function AccountSnapshotRow(props: SnapshotProps) {
  const { account } = props;
  const labels = accountSnapshotBalanceLabels(account);
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
          <dt>{labels.current}</dt>
          <dd>{accountSnapshotMoney(account.currentBalanceCents, account.currency)}</dd>
        </div>
        <div>
          <dt>{labels.available}</dt>
          <dd>{accountSnapshotMoney(account.availableBalanceCents, account.currency)}</dd>
        </div>
      </dl>
      <AccountSnapshotDetails {...props} />
    </article>
  );
}

/**
 * Keeps ownership and update details available without obscuring account values.
 * @param props Raw account and saved household names.
 * @returns Explicit owner, classification, and household-total inclusion labels.
 */
function AccountSnapshotDetails(props: SnapshotProps) {
  const { account } = props;
  const state = useConnectionData();
  const included = state.balances?.accounts.find(
    (item) => item.source === 'plaid' && item.id === account.id,
  )?.included;
  return (
    <details className={styles.details}>
      <summary aria-label={`Account details for ${account.name}`}>Account details</summary>
      <div className={styles.detailContent}>
        <p>Owner: {accountOwnerLabel(props)}</p>
        <p>
          {!state.balances
            ? 'Household totals status updating'
            : included
              ? 'Included in household totals'
              : 'Excluded from household totals'}
        </p>
        <p>
          Institution data time:{' '}
          {account.providerUpdatedAt ? syncTime(account.providerUpdatedAt) : 'Not provided'}
        </p>
        <p>
          Provider account type: {account.providerType}
          {account.providerSubtype ? ` / ${account.providerSubtype}` : ''}
        </p>
        {account.mapping.manualAccountId && (
          <p>This connected balance replaces a matching manual balance when included.</p>
        )}
        <p>Use Connect accounts to manage the owner, type, and balance settings.</p>
      </div>
    </details>
  );
}

/**
 * Labels ownership only after an account classification has been saved.
 * @param props Raw account and household names.
 * @returns Saved account owner or an explicit unconfirmed state.
 */
function accountOwnerLabel(props: SnapshotProps): string {
  const { account, household } = props;
  if (account.mapping.classification === 'unclassified') return 'Not confirmed';
  if (account.mapping.ownership === 'shared') return 'Joint / shared';
  return (
    household.members.find((member) => member.id === account.mapping.ownership)?.name ??
    'Unavailable'
  );
}
