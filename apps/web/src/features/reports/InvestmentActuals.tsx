import type { ActivityResponse, Goal, Household } from '@bank-piggy/contracts';
import { money } from '../plan/format';
import { ReportAmounts } from './ReportAmounts';
import styles from './Reports.module.css';

interface InvestmentActualsProps {
  readonly data: ActivityResponse;
  readonly household: Household;
  readonly onGoal: (goal: Goal) => void;
  readonly onReview: () => void;
}

/** Compares investment plans and net contributions without deducting payroll funds twice.
 * @param props Monthly recorded investment activity, goals, and explicit edit actions.
 * @returns Separate payroll estimates, goal matches, and account-level net contributions.
 */
export function InvestmentActuals(props: InvestmentActualsProps) {
  const { comparison } = props.data;
  return (
    <>
      <InvestmentPlanSummary data={props.data} />
      <section className={styles.section}>
        <h3>Investment goals</h3>
        <p className={styles.note}>
          Goal comparisons use explicit transaction matches. Saved goal progress and
          contribution-year records stay as entered.
        </p>
        {comparison.investmentGoals.map((row) => (
          <InvestmentGoalRow key={row.goalId} row={row} {...props} />
        ))}
        {!comparison.investmentGoals.length && (
          <p className={styles.empty}>
            Choose Investment goal or Roth IRA in a goal's settings to compare its contributions.
          </p>
        )}
        <p className={styles.notice}>
          Unmatched net contributions:{' '}
          {comparison.investmentAvailable
            ? money(comparison.unmatchedInvestmentNetCents, 1, true)
            : 'Unavailable'}
          . Deposits and withdrawals can offset each other.
        </p>
        <button type="button" className={styles.action} onClick={props.onReview}>
          Review contribution matches
        </button>
      </section>
      <InvestmentAccountRows data={props.data} />
    </>
  );
}

/** Keeps payroll estimates distinct from recorded funds and cash-funded investment goals.
 * @param props Current canonical monthly comparison.
 * @param props.data Selected monthly plan and recorded contribution totals.
 * @returns Planned components, net contributions, and the signed difference.
 */
function InvestmentPlanSummary(props: { readonly data: ActivityResponse }) {
  const { comparison, totals } = props.data;
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
        {money(comparison.investmentGoalBudgetCents, 1, true)} in investment goals. Recorded
        contributions can include employee and employer funds. They stay separate from take-home
        cash flow.
      </p>
    </section>
  );
}

/** Shows one goal's frozen matched allocation beside its current monthly plan.
 * @param props Comparison row and saved goal edit action.
 * @param props.row Server-calculated goal comparison.
 * @returns Monthly goal facts without writing saved target progress.
 */
function InvestmentGoalRow(
  props: InvestmentActualsProps & {
    readonly row: ActivityResponse['comparison']['investmentGoals'][number];
  },
) {
  const { row } = props;
  const goal = props.household.goals.find((item) => item.id === row.goalId);
  return (
    <article className={styles.row}>
      <div className={styles.rowHeading}>
        <div>
          <h4>{row.name}</h4>
          <p className={styles.note}>{row.recordCount} matched transactions</p>
        </div>
        {goal && (
          <button
            type="button"
            className={styles.action}
            aria-label={`Edit ${row.name} goal`}
            onClick={() => {
              props.onGoal(goal);
            }}
          >
            Edit goal
          </button>
        )}
      </div>
      <ReportAmounts
        items={[
          { label: 'Planned', cents: row.budgetCents, available: true },
          {
            label: 'Matched net contributions',
            cents: row.netContributionsCents,
            available: props.data.comparison.investmentAvailable,
          },
          {
            label: 'Difference to plan',
            cents: row.remainingCents,
            available: props.data.comparison.investmentAvailable,
          },
        ]}
      />
    </article>
  );
}

/** Keeps gross deposits, withdrawals, and signed net amounts visible for every recorded account.
 * @param props Monthly response with full-ledger account contributions.
 * @param props.data Canonical selected month and scope.
 * @returns Per-account contribution facts or explicit missing-history guidance.
 */
function InvestmentAccountRows(props: { readonly data: ActivityResponse }) {
  return (
    <section className={styles.section}>
      <h3>Recorded by account</h3>
      {!props.data.comparison.investmentAvailable && (
        <p className={styles.empty}>Investment history is unavailable for this month.</p>
      )}
      {props.data.contributions.map((account) => (
        <article className={styles.row} key={account.accountId}>
          <div className={styles.rowHeading}>
            <h4>{account.accountName}</h4>
          </div>
          <ReportAmounts
            items={[
              {
                label: 'Deposits',
                cents: account.depositsCents,
                available: props.data.comparison.investmentAvailable,
              },
              {
                label: 'Withdrawals',
                cents: account.withdrawalsCents,
                available: props.data.comparison.investmentAvailable,
              },
              {
                label: 'Net contributions',
                cents: account.netContributionsCents,
                available: props.data.comparison.investmentAvailable,
              },
            ]}
          />
        </article>
      ))}
    </section>
  );
}
