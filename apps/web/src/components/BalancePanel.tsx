import { Pencil, WalletCards } from 'lucide-react';
import type { Household, Summary } from '@bank-piggy/contracts';
import { scopedAccountGroups } from '../features/accounts/accountTypes';
import { money } from '../features/plan/format';
import styles from './Dashboard.module.css';
import ownStyles from './DashboardRows.module.css';
import balanceStyles from './BalancePanel.module.css';
import { FinanceCard } from './FinanceCard';
import { BalanceCategory } from './BalanceBreakdown';

interface BalancePanelProps {
  readonly summary: Summary;
  readonly household: Household;
  readonly scope: string;
  readonly onEdit: () => void;
}

/**
 * Balance snapshots are independent of the selected period's estimated cash flow.
 * @param root0 Component values and event handlers.
 * @param root0.summary Server-calculated totals.
 * @param root0.household Saved member-owned account snapshots.
 * @param root0.scope Household scope or a member identifier.
 * @param root0.onEdit Opens this record for editing.
 * @returns Rendered interface.
 */
export function BalancePanel({ summary, household, scope, onEdit }: BalancePanelProps) {
  const groups = scopedAccountGroups(household, scope);
  return (
    <FinanceCard
      title="Where you stand"
      description="Current balances · Manually tracked"
      actions={
        <button className={styles.iconButton} onClick={onEdit} aria-label="Edit balances">
          <Pencil size={13} />
        </button>
      }
      footer={
        <p className={ownStyles.balanceFooter}>
          Balances are entered manually and do not update automatically.
        </p>
      }
    >
      <BalanceCategory
        group="Savings"
        totalCents={summary.savingsCents}
        groups={groups.filter((group) => group.group === 'Savings')}
      />
      <BalanceCategory
        group="Investments"
        totalCents={summary.investmentsCents}
        groups={groups.filter((group) => group.group === 'Investments')}
      />
      <div className={`${ownStyles.balanceRow} ${balanceStyles.debt}`}>
        <span className={ownStyles.balanceLabel}>
          <WalletCards size={18} />
          Outstanding debt
        </span>
        <strong>{money(summary.debtCents)}</strong>
      </div>
    </FinanceCard>
  );
}
