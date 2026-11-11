import type { AccountType } from '@bank-piggy/contracts';
import { PiggyBank, TrendingUp } from 'lucide-react';
import type { AccountGroup } from '../features/accounts/accountTypes';
import { money } from '../features/plan/format';
import styles from './BalancePanel.module.css';
import rowStyles from './DashboardRows.module.css';

interface BalanceTypeGroup {
  readonly value: AccountType;
  readonly label: string;
  readonly count: number;
  readonly balanceCents: number;
}

/**
 * Places a server-owned balance total above its scoped account-type breakdown.
 * @param props Balance category, server total, and populated account types.
 * @param props.group Savings or investments label.
 * @param props.totalCents Server-calculated balance for the selected scope.
 * @param props.groups Populated account types belonging to this category.
 * @returns Category total and its manual account breakdown.
 */
export function BalanceCategory(props: {
  readonly group: AccountGroup;
  readonly totalCents: number;
  readonly groups: ReadonlyArray<BalanceTypeGroup>;
}) {
  const { group, totalCents, groups } = props;
  const Icon = group === 'Savings' ? PiggyBank : TrendingUp;
  return (
    <>
      <div className={rowStyles.balanceRow}>
        <span className={rowStyles.balanceLabel}>
          <Icon size={18} />
          {group}
        </span>
        <strong>{money(totalCents)}</strong>
      </div>
      <BalanceBreakdown groups={groups} />
    </>
  );
}

/**
 * Shows each populated account type once, including zero-balance accounts.
 * @param props Scoped account-type totals.
 * @param props.groups Populated account types within this balance category.
 * @returns Compact type, account-count, and balance rows.
 */
function BalanceBreakdown(props: { readonly groups: ReadonlyArray<BalanceTypeGroup> }) {
  const { groups } = props;
  if (!groups.length) return <p className={styles.empty}>No accounts entered.</p>;
  return (
    <dl className={styles.breakdown}>
      {groups.map((group) => (
        <div key={group.value}>
          <dt>
            {group.label}
            <span>
              {group.count} {group.count === 1 ? 'account' : 'accounts'}
            </span>
          </dt>
          <dd>{money(group.balanceCents)}</dd>
        </div>
      ))}
    </dl>
  );
}
