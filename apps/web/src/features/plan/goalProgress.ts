import type { Goal, GoalProjection } from '@bank-piggy/contracts';

/**
 * Reads saved progress independently of monthly allocations and selected display periods.
 * @param goal Saved goal with at most one target kind.
 * @param projection Server-calculated trip or purchase target, when applicable.
 * @returns Exact displayed amounts and bounded native progress values, or no target.
 */
export function goalProgress(goal: Goal, projection?: GoalProjection) {
  const details = goal.savingsTarget ?? goal.trip ?? goal.purchase;
  const target = goal.savingsTarget?.targetAmountCents ?? projection?.targetAmountCents;
  if (!details || target === undefined) return null;
  return {
    savedAmountCents: details.savedAmountCents,
    targetAmountCents: target,
    max: Math.max(target, 1),
    value: target === 0 ? 1 : Math.min(details.savedAmountCents, target),
  };
}
