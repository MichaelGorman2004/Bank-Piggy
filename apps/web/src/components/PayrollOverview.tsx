import type { Member, PayrollSummary } from '@bank-piggy/contracts';
import { money } from '../features/plan/format';
import styles from './PlanDetails.module.css';

interface PayrollOverviewProps {
  readonly members: readonly Member[];
  readonly summaries: readonly PayrollSummary[];
}

/**
 * Shows payroll investments separately because take-home pay already excludes them.
 * @param props Saved people and server-calculated monthly payroll amounts.
 * @returns An optional compact payroll disclosure within the bounded overview card.
 */
export function PayrollOverview(props: PayrollOverviewProps) {
  const tracked = props.summaries.filter((summary) =>
    props.members.some((member) => member.id === summary.memberId && member.payroll),
  );
  if (tracked.length === 0) return null;
  return (
    <details className={styles.payroll}>
      <summary>Payroll investments · monthly estimates</summary>
      {tracked.map((summary) => (
        <div key={summary.memberId}>
          <strong>{props.members.find((member) => member.id === summary.memberId)?.name}</strong>
          <dl>
            <div>
              <dt>Traditional 401(k)</dt>
              <dd>{money(summary.traditional401kMonthlyCents)}</dd>
            </div>
            <div>
              <dt>Roth 401(k)</dt>
              <dd>{money(summary.roth401kMonthlyCents)}</dd>
            </div>
            <div>
              <dt>Employer match</dt>
              <dd>{money(summary.employerMonthlyCents)}</dd>
            </div>
          </dl>
        </div>
      ))}
      <p>
        These amounts do not reduce take-home pay again. Estimates use base salary and entered match
        rules; limits and vesting are not applied.
      </p>
    </details>
  );
}
