import { Pencil, WalletCards } from 'lucide-react';
import type { Household, Summary } from '@bank-piggy/contracts';
import { scopedAccountGroups } from '../features/accounts/accountTypes';
import { money } from '../features/plan/format';
import styles from './Dashboard.module.css';
import ownStyles from './DashboardRows.module.css';
import balanceStyles from './BalancePanel.module.css';
import { FinanceCard } from './FinanceCard';
import { BalanceCategory } from './BalanceBreakdown';
import { ConnectedAccounts } from '../features/connections/ConnectedAccounts';
import { useConnectionData } from '../features/connections/ConnectionsContext';
import { EffectiveBalances } from '../features/connections/EffectiveBalances';
import connectionStyles from '../features/connections/Connections.module.css';

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
  const connections = useConnectionData();
  const connected = Boolean(connections.data?.connections.length);
  const balances = connections.balances;
  return (
    <FinanceCard
      title="Where you stand"
      description={
        connected && balances
          ? 'Current balance snapshots · USD'
          : 'Current balances · Manually tracked'
      }
      actions={
        <div className={connectionStyles.balanceActions}>
          <ConnectedAccounts household={household} />
          <button className={styles.iconButton} onClick={onEdit} aria-label="Edit balances">
            <Pencil size={13} />
          </button>
        </div>
      }
      footer={
        <p className={ownStyles.balanceFooter}>
          {connected && !balances
            ? 'Connected balances are updating for the saved plan. Showing manual snapshots.'
            : balanceHelp(Boolean(connections.error), connected)}
        </p>
      }
    >
      {connected && balances ? (
        <EffectiveBalances balances={balances} household={household} scope={scope} />
      ) : (
        <ManualBalanceCategories household={household} scope={scope} summary={summary} />
      )}
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

/**
 * Preserves the existing typed manual breakdown when no connection snapshot is available.
 * @param props Saved manual accounts and server totals.
 * @returns The savings and investment categories from the manual plan.
 */
function ManualBalanceCategories(props: Omit<BalancePanelProps, 'onEdit'>) {
  const groups = scopedAccountGroups(props.household, props.scope);
  return (
    <>
      <BalanceCategory
        group="Savings"
        totalCents={props.summary.savingsCents}
        groups={groups.filter((group) => group.group === 'Savings')}
      />
      <BalanceCategory
        group="Investments"
        totalCents={props.summary.investmentsCents}
        groups={groups.filter((group) => group.group === 'Investments')}
      />
    </>
  );
}

/**
 * Describes the visible snapshot source and makes connection failure actionable.
 * @param failed Whether a connection request failed.
 * @param connected Whether the household has a connected balance source.
 * @returns Honest snapshot guidance.
 */
function balanceHelp(failed: boolean, connected: boolean): string {
  if (failed)
    return 'Connected balances could not load. Showing the last saved view; open Connected accounts to retry.';
  return connected
    ? 'Connected snapshots update on a schedule. Manual balances stay separate from your monthly plan.'
    : 'Manual balances do not update automatically. Connect an institution for scheduled balance snapshots.';
}
