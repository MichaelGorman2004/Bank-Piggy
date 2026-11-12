import type { Household } from '@bank-piggy/contracts';
import type { GoalDraft } from './goalDraft.ts';

/**
 * Selects a personal owner when applying the annual Roth IRA preset.
 * @param draft Current goal draft, including its retained owner.
 * @param change Fields edited by the goal-type or target controls.
 * @param household Current household people.
 * @returns The requested change with an available personal owner when needed.
 */
export function retirementGoalChange(
  draft: GoalDraft,
  change: Partial<GoalDraft>,
  household: Pick<Household, 'members'>,
): Partial<GoalDraft> {
  const first = household.members.at(0);
  if (change.kind === 'roth_ira' && draft.ownerId === 'shared' && first)
    return { ...change, ownerId: first.id };
  return change;
}

/**
 * Requires a personal annual target without guessing an IRA allowance.
 * @param draft Editable goal and contribution year.
 * @returns One actionable validation error, or null.
 */
export function retirementGoalError(draft: GoalDraft): string | null {
  if (draft.kind !== 'roth_ira') return null;
  if (draft.ownerId === 'shared') return 'Choose one person for this Roth IRA goal.';
  if (
    !/^\d{4}$/.test(draft.contributionYear) ||
    Number(draft.contributionYear) < 2000 ||
    Number(draft.contributionYear) > 2100
  )
    return 'Enter a contribution year from 2000 through 2100.';
  return null;
}
