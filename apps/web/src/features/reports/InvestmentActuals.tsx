import type { ActivityResponse } from '@bank-piggy/contracts';
import { money } from '../plan/format';
import { ReportAmounts } from './ReportAmounts';
import styles from './Reports.module.css';

/** Compares the monthly investment plan with aggregate net contributions.
 * @param props Current canonical monthly comparison.
 * @param props.data Selected monthly plan, recorded totals, and investment coverage.
 * @returns Planned contributions, net contributions, and their signed difference.
 */
export function InvestmentActuals({ data }: { readonly data: ActivityResponse }) {
  const { comparison, totals } = data;
  return (
    <section className={styles.section}>
      <ReportAmounts
        items={[
          {
            label: 'Planned contributions',
            cents: comparison.investmentBudgetCents,
            available: true,
          },
          {
            label: 'Recorded net contributions',
            cents: totals.netInvestmentContributionsCents,
            available: comparison.investmentAvailable,
          },
          {
            label: 'Difference to plan',
            cents: comparison.investmentRemainingCents,
            available: comparison.investmentAvailable,
          },
        ]}
      />
      <p className={styles.notice}>
        The plan includes {money(comparison.payrollBudgetCents, 1, true)} in payroll estimates and{' '}
        {money(comparison.investmentGoalBudgetCents, 1, true)} in investment goals. Net
        contributions are deposits less withdrawals. Trades and market changes are excluded.
      </p>
      <p className={styles.note}>
        Contributions stay separate from take-home cash flow and saved goal progress.
      </p>
    </section>
  );
}
