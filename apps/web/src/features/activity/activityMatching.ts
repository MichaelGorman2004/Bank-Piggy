import type { ActivityCategory, ActivityRecord, Goal } from '@bank-piggy/contracts';

/** Selects an explicit plan-match type from the current classification.
 * @param record Posted provider facts and saved classification.
 * @param category Manual choice, or null to retain the automatic classification.
 * @returns The compatible target type, or null for records outside plan comparisons.
 */
export function activityMatchKind(
  record: ActivityRecord,
  category: ActivityCategory | null,
): 'expense' | 'goal' | null {
  if (category === null && record.isReviewed) return null;
  const selected = category ?? record.category;
  if (record.kind === 'bank' && selected === 'spending') return 'expense';
  if (record.kind === 'investment' && (selected === 'contribution' || selected === 'withdrawal'))
    return 'goal';
  return null;
}

/** Requires an explicit investment purpose or annual IRA contribution year.
 * @param goal Saved plan goal; editable names do not establish investment intent.
 * @returns Whether investment activity can be matched to this goal.
 */
export function isInvestmentGoal(goal: Goal): boolean {
  return !goal.trip && !goal.purchase && (goal.purpose === 'investment' || !!goal.contributionYear);
}
