import type { ActivityResponse } from '@bank-piggy/contracts';
import { money } from '../plan/format';
import { hasActivityHistory } from './presentation';
import styles from './ActivitySummary.module.css';

/**
 * Shows full-month bank actuals and investment contributions with explicit coverage.
 * @param props Current authorized month.
 * @param props.data Server totals and source coverage.
 * @param props.unavailable Whether a saved change still needs current totals.
 * @returns Three actual totals, never a confirmed zero without recorded history.
 */
export function ActivityTotals(props: {
  readonly data: ActivityResponse;
  readonly unavailable: boolean;
}) {
  const { data } = props;
  const bank = !props.unavailable && hasActivityHistory(data, 'depository');
  const investment = !props.unavailable && hasActivityHistory(data, 'investment');
  const recorded = data.coverage === 'ready' ? '' : 'Recorded ';
  const totals = [
    { name: `${recorded}income`, amount: data.totals.incomeCents, available: bank },
    { name: `${recorded}spending`, amount: data.totals.spendingCents, available: bank },
    {
      name: `${recorded}net invested`,
      amount: data.totals.netInvestmentContributionsCents,
      available: investment,
    },
  ];
  return (
    <dl className={styles.totals}>
      {totals.map((total) => (
        <div key={total.name}>
          <dt>{total.name}</dt>
          <dd>{total.available ? money(total.amount, 1, true) : 'Unavailable'}</dd>
        </div>
      ))}
    </dl>
  );
}
