import type { ActivityResponse, Member } from '@bank-piggy/contracts';
import { money } from '../plan/format';
import { hasActivityHistory } from './presentation';
import styles from './ActivitySummary.module.css';
import layoutStyles from './Activity.module.css';

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

/**
 * Displays category spending and deposits less withdrawals for each investment account.
 * @param props Recorded monthly response and saved member names.
 * @param props.data Current monthly totals.
 * @param props.members Saved household members.
 * @returns Actual spending categories and contribution details.
 */
export function ActivitySummary(props: {
  readonly data: ActivityResponse;
  readonly members: ReadonlyArray<Member>;
}) {
  const { data } = props;
  return (
    <div className={styles.summary}>
      <section className={layoutStyles.section}>
        <h3>Spending by category</h3>
        {data.categories.length ? (
          <dl className={styles.amounts}>
            {data.categories.map((category) => (
              <div key={category.name}>
                <dt>{category.name}</dt>
                <dd>{money(category.amountCents, 1, true)}</dd>
              </div>
            ))}
          </dl>
        ) : (
          <p className={layoutStyles.note}>No recorded spending categories for this month.</p>
        )}
      </section>
      <section className={layoutStyles.section}>
        <h3>Investment contributions</h3>
        <p className={layoutStyles.note}>
          Deposits less withdrawals. Trades and market changes do not count as contributions.
        </p>
        <ContributionRows {...props} />
        {hasActivityHistory(data, 'investment') && (
          <p className={layoutStyles.note}>
            Investment income: {money(data.totals.investmentIncomeCents, 1, true)}. Fees:{' '}
            {money(data.totals.investmentFeesCents, 1, true)}.
          </p>
        )}
      </section>
    </div>
  );
}

/**
 * Retains account-level contribution facts and the ownership behind personal shares.
 * @param props Monthly investment contributions and household names.
 * @param props.data Monthly contribution rows.
 * @param props.members Saved household members.
 * @returns Account contribution rows or unavailable guidance.
 */
function ContributionRows(props: {
  readonly data: ActivityResponse;
  readonly members: ReadonlyArray<Member>;
}) {
  if (!props.data.contributions.length)
    return (
      <p className={layoutStyles.note}>No recorded investment contributions for this month.</p>
    );
  return (
    <div>
      {props.data.contributions.map((account) => (
        <div className={styles.contribution} key={account.accountId}>
          <div>
            <h4>{account.accountName}</h4>
            <p className={layoutStyles.note}>
              {account.ownership === 'shared'
                ? 'Shared account'
                : `Account owner: ${props.members.find((member) => member.id === account.ownership)?.name ?? 'Unavailable member'}`}
            </p>
          </div>
          <dl className={styles.amounts}>
            <div>
              <dt>Deposits</dt>
              <dd>{money(account.depositsCents, 1, true)}</dd>
            </div>
            <div>
              <dt>Withdrawals</dt>
              <dd>{money(account.withdrawalsCents, 1, true)}</dd>
            </div>
            <div>
              <dt>Net invested</dt>
              <dd>{money(account.netContributionsCents, 1, true)}</dd>
            </div>
          </dl>
        </div>
      ))}
    </div>
  );
}
