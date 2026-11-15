import type { ActivityRecord, Household } from '@bank-piggy/contracts';
import { reviewMatchChange, type ReviewDraft } from './activityReviewDraft';
import { activityMatchKind, isInvestmentGoal } from './activityMatching';
import styles from './Activity.module.css';

interface MatchProps {
  readonly record: ActivityRecord;
  readonly value: ReviewDraft;
  readonly household: Household;
  readonly onChange: (patch: Partial<ReviewDraft>) => void;
}

/** Shows explicit plan targets without matching names or changing manual progress.
 * @param props Current classification, plan choices, and retained review draft.
 * @returns One compatible native match selector, or no control for other classifications.
 */
export function ActivityMatchField(props: MatchProps) {
  const kind = activityMatchKind(props.record, props.value.category);
  if (!kind) return null;
  const targets =
    kind === 'expense' ? props.household.expenses : props.household.goals.filter(isInvestmentGoal);
  const selected = kind === 'expense' ? props.value.expenseId : props.value.goalId;
  const unavailable = selected && !targets.some((target) => target.id === selected);
  return (
    <>
      <label>
        {kind === 'expense' ? 'Match expense budget' : 'Match investment goal'}
        <select
          value={selected}
          onChange={(event) => {
            const id = targets.find((target) => target.id === event.target.value)?.id ?? '';
            props.onChange(reviewMatchChange(props.record, props.value, id));
          }}
        >
          <option value="">Unmatched</option>
          {unavailable && <option value={selected}>Unavailable target ({selected})</option>}
          {targets.map((target) => (
            <option key={target.id} value={target.id}>
              {target.name}
            </option>
          ))}
        </select>
      </label>
      {unavailable && (
        <p className={styles.notice}>
          This target is no longer available. Choose a current target or Unmatched before saving.
        </p>
      )}
      <p className={styles.note}>
        A match records the agreed plan share. The account owner does not set the agreed share.
        {kind === 'goal' && ' Matching does not change saved progress or annual contributions.'}
      </p>
    </>
  );
}

/** Keeps historical match identity and allocation visible when a target changes or disappears.
 * @param props Provider record and current saved household plan.
 * @param props.record Record with its retained match and agreed allocation.
 * @param props.household Current saved target names.
 * @returns Saved match facts and the fixed-share explanation.
 */
export function ActivityMatchFacts(props: Pick<MatchProps, 'record' | 'household'>) {
  const expense = props.household.expenses.find((target) => target.id === props.record.expenseId);
  const goal = props.household.goals.find(
    (target) => target.id === props.record.goalId && isInvestmentGoal(target),
  );
  const id = props.record.expenseId ?? props.record.goalId;
  const target = expense ?? goal;
  return (
    <>
      {id && (
        <p className={styles.note}>
          Saved match:{' '}
          {target?.name ??
            `Unavailable target (${id}). This activity is unmatched in plan comparisons.`}
        </p>
      )}
      {props.record.allocation && (
        <p className={styles.note}>
          Matched shares stay fixed. Save Unmatched, then match again to use the current plan share.
        </p>
      )}
    </>
  );
}
