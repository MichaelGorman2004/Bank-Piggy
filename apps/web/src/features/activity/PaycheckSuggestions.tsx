import { useState } from 'react';
import type { Member, PaycheckSuggestion } from '@bank-piggy/contracts';
import { PAY_FREQUENCIES, money } from '../plan/format';
import { activityDate } from './presentation';
import styles from './Activity.module.css';
import summaryStyles from './ActivitySummary.module.css';
import formStyles from '../plan/Editors.module.css';

interface PaycheckProps {
  readonly suggestion: PaycheckSuggestion;
  readonly members: ReadonlyArray<Member>;
  readonly busy: boolean;
  readonly onApply: (
    suggestion: PaycheckSuggestion,
    memberId: 'member-1' | 'member-2',
  ) => Promise<boolean>;
}

/**
 * Requires a named member and explicit confirmation before updating saved net pay.
 * @param props Aggregate pay estimates, saved members, and plan update action.
 * @returns Pay estimates and an explicit plan confirmation action.
 */
export function PaycheckSuggestionPanel(props: PaycheckProps) {
  const [memberId, setMemberId] = useState('');
  const [appliedTo, setAppliedTo] = useState<string | null>(null);
  const member = props.members.find((item) => item.id === memberId);
  const { suggestion } = props;
  /** Applies only the explicitly selected member and reports the saved outcome. */
  async function apply() {
    if (memberId !== 'member-1' && memberId !== 'member-2') return;
    if (await props.onApply(suggestion, memberId)) setAppliedTo(memberId);
  }
  return (
    <details className={styles.suggestion}>
      <summary>
        <strong>Pay estimate</strong>
        <span>{suggestion.depositCount} deposits · Review pay suggestion</span>
      </summary>
      <div className={styles.form}>
        <p className={styles.note}>
          This suggestion uses bank net deposits. The employer and payroll deductions are not
          verified.{suggestion.variableAmount ? ' Deposit amounts vary.' : ''}
        </p>
        <PaycheckMember members={props.members} memberId={memberId} onMember={setMemberId} />
        <PaycheckFields suggestion={suggestion} member={member} />
        <PaycheckNotes />
        <button
          className={formStyles.button}
          type="button"
          disabled={props.busy || !member || appliedTo === memberId}
          onClick={() => {
            void apply();
          }}
        >
          {appliedTo === memberId
            ? 'Pay suggestion applied'
            : `Confirm and update ${member?.name ?? 'member'}’s pay plan`}
        </button>
      </div>
    </details>
  );
}

/**
 * Shows the exact proposed fields before the user confirms a saved-plan change.
 * @param props Suggested pay facts and the explicitly selected member.
 * @param props.suggestion Suggested net pay and calendar schedule.
 * @param props.member Selected saved member, if any.
 * @returns Saved monthly take-home and proposed pay fields.
 */
function PaycheckFields(props: {
  readonly suggestion: PaycheckSuggestion;
  readonly member?: Member;
}) {
  const { suggestion, member } = props;
  const frequency = PAY_FREQUENCIES.find((item) => item.value === suggestion.frequency)?.label;
  return (
    <dl className={summaryStyles.amounts}>
      {member && (
        <div>
          <dt>Saved monthly take-home</dt>
          <dd>{money(member.monthlyTakeHomeCents, 1, true)}</dd>
        </div>
      )}
      <div>
        <dt>Proposed monthly take-home</dt>
        <dd>{money(suggestion.monthlyTakeHomeCents, 1, true)}</dd>
      </div>
      <div>
        <dt>Typical net paycheck</dt>
        <dd>{money(suggestion.typicalPaycheckCents, 1, true)}</dd>
      </div>
      <div>
        <dt>Pay frequency</dt>
        <dd>{frequency}</dd>
      </div>
      <div>
        <dt>Payday anchor</dt>
        <dd>{activityDate(suggestion.paySchedule.anchorDate)}</dd>
      </div>
      <div>
        <dt>Next expected payday</dt>
        <dd>{suggestion.nextPayDate ? activityDate(suggestion.nextPayDate) : 'Unavailable'}</dd>
      </div>
      <div>
        <dt>Payday schedule</dt>
        <dd>{scheduleLabel(suggestion)}</dd>
      </div>
    </dl>
  );
}

/**
 * Explains the proposed calendar days without inferring holiday adjustments.
 * @param suggestion Suggested payday schedule.
 * @returns Plain calendar schedule text.
 */
function scheduleLabel(suggestion: PaycheckSuggestion): string {
  const schedule = suggestion.paySchedule;
  if (suggestion.frequency === 'weekly') return 'Every seven days from the payday anchor';
  if (suggestion.frequency === 'biweekly') return 'Every fourteen days from the payday anchor';
  if (suggestion.frequency === 'semimonthly')
    return `Days ${schedule.semiMonthlyDays.join(' and ')} each month`;
  return schedule.monthEnd
    ? 'Last day of each month'
    : `Day ${schedule.anchorDate.slice(8, 10)} each month`;
}

/** Explains estimate limits and saved plan changes before confirmation.
 * @returns Aggregate-pay coverage and the exact plan fields that will change.
 */
function PaycheckNotes() {
  return (
    <>
      <p className={styles.notice}>
        This suggestion covers deposits from one payer into one account. Split direct deposits or
        multiple jobs can mean this is only part of the member’s income. Review the member’s full
        income before replacing monthly take-home pay.
      </p>
      <p className={styles.note}>
        Apply changes monthly take-home pay, pay frequency, and the payday schedule. Annual salary,
        payroll investment rates, and goals stay as saved.
      </p>
    </>
  );
}

/** Requires a member choice for every saved pay-plan update.
 * @param props Saved member choices and current explicit selection.
 * @param props.members Saved household members.
 * @param props.memberId Explicitly selected member identity.
 * @param props.onMember Member selection action.
 * @returns Required native member selector.
 */
function PaycheckMember(props: {
  readonly members: ReadonlyArray<Member>;
  readonly memberId: string;
  readonly onMember: (id: string) => void;
}) {
  const { members, memberId } = props;
  return (
    <label>
      Apply to member
      <select
        required
        value={memberId}
        onChange={(event) => {
          props.onMember(event.target.value);
        }}
      >
        <option value="">Choose a member</option>
        {members.map((item) => (
          <option key={item.id} value={item.id}>
            {item.name}
          </option>
        ))}
      </select>
    </label>
  );
}
