import type { ActivityResponse, Expense, Household } from '@bank-piggy/contracts';
import { money } from '../plan/format';
import { ReportAmounts } from './ReportAmounts';
import styles from './Reports.module.css';

interface ExpenseActualsProps {
  readonly data: ActivityResponse;
  readonly household: Household;
  readonly onExpense: (expense: Expense) => void;
  readonly onReview: () => void;
}

/** Compares saved monthly budgets with explicitly matched spending from the full ledger.
 * @param props Canonical monthly comparisons and saved expense edit actions.
 * @returns Exact budget, matched spending, and signed remaining amounts.
 */
export function ExpenseActuals(props: ExpenseActualsProps) {
  const { comparison } = props.data;
  return (
    <>
      <p className={styles.note}>
        Only transactions you match count toward a bill or budget item. Check unmatched spending
        before using the remaining amounts.
      </p>
      {comparison.bankAvailable ? (
        <p className={styles.notice}>
          Unmatched spending: <strong>{money(comparison.unmatchedSpendingCents, 1, true)}</strong>{' '}
          across {comparison.unmatchedSpendingCount} records.
        </p>
      ) : (
        <p className={styles.notice}>
          Bank history is unavailable for this month. Open Activity to review source permissions and
          coverage.
        </p>
      )}
      {comparison.expenses.map((row) => (
        <ExpenseComparisonRow
          key={row.expenseId}
          row={row}
          available={comparison.bankAvailable}
          {...props}
        />
      ))}
      {!comparison.expenses.length && (
        <p className={styles.empty}>
          No expense budgets or matched spending are available for this month and share.
        </p>
      )}
      <button type="button" className={styles.action} onClick={props.onReview}>
        Review transaction matches
      </button>
    </>
  );
}

/** Keeps editing attached to the saved expense, including rows with a zero current budget.
 * @param props Current comparison row and saved household actions.
 * @param props.row Server-calculated expense comparison.
 * @param props.available Whether selected bank history is available.
 * @returns One monthly expense comparison with its explicit match count.
 */
function ExpenseComparisonRow(
  props: ExpenseActualsProps & {
    readonly row: ActivityResponse['comparison']['expenses'][number];
    readonly available: boolean;
  },
) {
  const { row } = props;
  const expense = props.household.expenses.find((item) => item.id === row.expenseId);
  return (
    <article className={styles.row}>
      <div className={styles.rowHeading}>
        <div>
          <h3>{row.name}</h3>
          <p className={styles.note}>
            {row.category} · {row.recordCount} matched transactions
          </p>
        </div>
        {expense && (
          <button
            type="button"
            className={styles.action}
            aria-label={`Edit ${row.name} budget`}
            onClick={() => {
              props.onExpense(expense);
            }}
          >
            Edit budget
          </button>
        )}
      </div>
      <ReportAmounts
        items={[
          { label: 'Budget', cents: row.budgetCents, available: true },
          { label: 'Matched spent', cents: row.spentCents, available: props.available },
          {
            label: row.remainingCents < 0 ? 'Over budget' : 'Remaining after matches',
            cents: row.remainingCents < 0 ? -row.remainingCents : row.remainingCents,
            available: props.available,
          },
        ]}
      />
    </article>
  );
}
