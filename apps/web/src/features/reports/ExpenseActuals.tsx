import type { ActivityResponse } from '@bank-piggy/contracts';
import { ReportAmounts } from './ReportAmounts';
import styles from './Reports.module.css';

/** Compares the total saved monthly budget with recorded spending across selected accounts.
 * @param props Canonical monthly totals and aggregate budget comparison.
 * @param props.data Current month, scope, source coverage, and recorded totals.
 * @returns Budget, spending, and signed remaining amounts without purchase detail.
 */
export function ExpenseActuals({ data }: { readonly data: ActivityResponse }) {
  const { comparison } = data;
  return (
    <section className={styles.section}>
      <ReportAmounts
        items={[
          { label: 'Budget', cents: comparison.expenseBudgetCents, available: true },
          {
            label: 'Spent',
            cents: data.totals.spendingCents,
            available: comparison.bankAvailable,
          },
          {
            label: 'Remaining',
            cents: comparison.expenseRemainingCents,
            available: comparison.bankAvailable,
          },
        ]}
      />
      <p className={styles.notice}>
        Spending is purchases less refunds across selected accounts. Transfers and credit-card
        payments are excluded. Your saved budget stays separate.
      </p>
    </section>
  );
}
