import type { BalanceTotals, EffectiveBalances as BalanceProjection } from '@bank-piggy/contracts';
import { PiggyBank, TrendingUp, Wallet, HeartPulse } from 'lucide-react';
import { HOUSEHOLD_SCOPE, money } from '../plan/format';
import { BalanceAvailability } from './BalanceAvailability';
import rowStyles from '../../components/DashboardRows.module.css';

const TOTAL_ROWS = [
  { field: 'checkingCents', label: 'Checking', Icon: Wallet },
  { field: 'savingsCents', label: 'Savings', Icon: PiggyBank },
  { field: 'investmentsCents', label: 'Investments', Icon: TrendingUp },
  { field: 'hsaCents', label: 'Health savings', Icon: HeartPulse },
] satisfies ReadonlyArray<{
  readonly field: keyof BalanceTotals;
  readonly label: string;
  readonly Icon: typeof Wallet;
}>;

/**
 * Shows the server's reconciled USD totals for the selected presentation scope.
 * @param props Authorized projection and selected presentation scope.
 * @param props.balances Server-owned totals and account inclusion decisions.
 * @param props.scope Household or personal presentation scope.
 * @returns Balance category totals and balance-availability guidance.
 */
export function EffectiveBalances(props: {
  readonly balances: BalanceProjection;
  readonly scope: string;
}) {
  const totals =
    props.scope === HOUSEHOLD_SCOPE
      ? props.balances.household
      : props.balances.members[props.scope];
  return (
    <>
      <BalanceAvailability scope={props.scope} />
      {TOTAL_ROWS.map(({ field, label, Icon }) => (
        <div className={rowStyles.balanceRow} key={field}>
          <span className={rowStyles.balanceLabel}>
            <Icon size={18} />
            {label}
          </span>
          <strong>{money(totals[field])}</strong>
        </div>
      ))}
    </>
  );
}
