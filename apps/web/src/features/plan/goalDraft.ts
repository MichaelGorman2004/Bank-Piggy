import type { Goal, PurchaseTarget, TripBudget, TripCost } from '@bank-piggy/contracts';
import { moneyInput, parseMoney } from './format.ts';
import { validMonth } from './month.ts';

export const TRIP_CATEGORIES: ReadonlyArray<{
  readonly category: TripCost['category'];
  readonly label: string;
}> = [
  { category: 'flights', label: 'Flights' },
  { category: 'lodging', label: 'Lodging' },
  { category: 'food', label: 'Food' },
  { category: 'transport', label: 'Local transport' },
  { category: 'activities', label: 'Activities' },
  { category: 'other', label: 'Other costs' },
];
export type GoalKind = 'goal' | 'trip' | 'purchase';
export interface GoalDraft {
  readonly name: string;
  readonly amount: string;
  readonly ownerId: Goal['ownerId'];
  readonly kind: GoalKind;
  readonly saved: string;
  readonly targetAmount: string;
  readonly targetMonth: string;
  readonly trackTarget: boolean;
  readonly costs: Readonly<Record<TripCost['category'], string>>;
}
type GoalResult =
  | { readonly goal: Goal; readonly error?: never }
  | { readonly error: string; readonly goal?: never };
type TargetResult =
  | { readonly details: Pick<Goal, 'trip' | 'purchase' | 'savingsTarget'>; readonly error?: never }
  | { readonly error: string; readonly details?: never };
type ProgressResult =
  | {
      readonly progress: Pick<PurchaseTarget, 'savedAmountCents' | 'targetMonth'>;
      readonly error?: never;
    }
  | { readonly error: string; readonly progress?: never };
type TripCostResult =
  | { readonly costs: TripCost[]; readonly error?: never }
  | { readonly error: string; readonly costs?: never };
const MAX_MONEY_CENTS = 100_000_000_000;

/**
 * Builds editable target fields without changing the saved goal.
 * @param goal Existing goal, when editing.
 * @param initialKind Initial target type for a new goal.
 * @returns A goal draft with optional trip or purchase details.
 */
export function goalDraft(goal?: Goal, initialKind: GoalKind = 'goal'): GoalDraft {
  const value: Partial<Goal> = goal ?? {};
  return {
    name: value.name ?? '',
    amount: goal ? moneyInput(goal.monthlyAmountCents) : '',
    ownerId: value.ownerId ?? 'shared',
    kind: goal ? (goal.trip ? 'trip' : goal.purchase ? 'purchase' : 'goal') : initialKind,
    trackTarget: Boolean(value.savingsTarget),
    ...targetDraft(value),
    costs: tripCostDraft(value.trip),
  };
}

/**
 * Copies saved progress from either target type into editable field text.
 * @param goal Saved optional target details.
 * @returns Editable saved amount, optional target amount, and deadline month.
 */
function targetDraft(goal: Pick<Goal, 'trip' | 'purchase' | 'savingsTarget'>) {
  const dated = goal.trip ?? goal.purchase;
  const target = dated ?? goal.savingsTarget;
  const amount = goal.purchase ?? goal.savingsTarget;
  return {
    saved: moneyInput(target?.savedAmountCents ?? 0),
    targetAmount: moneyInput(amount?.targetAmountCents ?? 0),
    targetMonth: dated?.targetMonth ?? '',
  };
}

/**
 * Reads one saved category without inventing a cost.
 * @param trip Saved travel budget, when present.
 * @param category Travel category to read.
 * @returns Editable money text.
 */
function savedCost(trip: TripBudget | undefined, category: TripCost['category']): string {
  return moneyInput(trip?.costs.find((item) => item.category === category)?.amountCents ?? 0);
}

/**
 * Copies all six travel categories into the form.
 * @param trip Saved travel budget, when present.
 * @returns Editable costs by category.
 */
function tripCostDraft(trip?: TripBudget): GoalDraft['costs'] {
  return {
    flights: savedCost(trip, 'flights'),
    lodging: savedCost(trip, 'lodging'),
    food: savedCost(trip, 'food'),
    transport: savedCost(trip, 'transport'),
    activities: savedCost(trip, 'activities'),
    other: savedCost(trip, 'other'),
  };
}

/**
 * Validates the progress inputs shared by trip and purchase targets.
 * @param draft Current editable goal fields.
 * @returns Saved progress and target month, or an actionable error.
 */
function targetProgress(draft: GoalDraft): ProgressResult {
  if (!validMonth(draft.targetMonth)) return { error: 'Choose a valid target month.' };
  const savedAmountCents = parseMoney(draft.saved);
  if (savedAmountCents === null || savedAmountCents > MAX_MONEY_CENTS)
    return { error: 'Enter a valid amount already saved.' };
  return { progress: { savedAmountCents, targetMonth: draft.targetMonth } };
}

/**
 * Validates travel costs without adding them to monthly allocations.
 * @param draft Current editable goal fields.
 * @returns Canonical category costs, or an actionable error.
 */
function tripCosts(draft: GoalDraft): TripCostResult {
  const costs: TripCost[] = [];
  for (const { category } of TRIP_CATEGORIES) {
    const amountCents = parseMoney(draft.costs[category]);
    if (amountCents === null || amountCents > MAX_MONEY_CENTS)
      return { error: 'Enter valid trip costs with up to two decimal places.' };
    costs.push({ category, amountCents });
  }
  if (costs.reduce((sum, cost) => sum + cost.amountCents, 0) > MAX_MONEY_CENTS)
    return { error: 'The trip budget is above the maximum amount.' };
  return { costs };
}

/**
 * Selects exactly one target type and keeps target progress out of budget totals.
 * @param draft Editable target inputs.
 * @returns The selected target details, or an actionable error.
 */
function targetValue(draft: GoalDraft): TargetResult {
  if (draft.kind === 'goal') return savingsTargetValue(draft);
  const { progress, error } = targetProgress(draft);
  if (!progress) return { error };
  if (draft.kind === 'purchase') {
    const targetAmountCents = parseMoney(draft.targetAmount);
    if (targetAmountCents === null || targetAmountCents > MAX_MONEY_CENTS)
      return { error: 'Enter a valid purchase target with up to two decimal places.' };
    return { details: { purchase: { ...progress, targetAmountCents } } };
  }
  const travel = tripCosts(draft);
  if (!travel.costs) return { error: travel.error };
  return { details: { trip: { ...progress, costs: travel.costs } } };
}

/**
 * Records an optional ordinary savings target independently of the monthly contribution.
 * @param draft Editable target and saved amount, retained across type changes.
 * @returns Canonical progress or an actionable field error.
 */
function savingsTargetValue(draft: GoalDraft): TargetResult {
  if (!draft.trackTarget) return { details: {} };
  const targetAmountCents = parseMoney(draft.targetAmount);
  const savedAmountCents = parseMoney(draft.saved);
  if (targetAmountCents === null || targetAmountCents <= 0 || targetAmountCents > MAX_MONEY_CENTS)
    return { error: 'Enter a savings target greater than zero, with up to two decimal places.' };
  if (savedAmountCents === null || savedAmountCents > MAX_MONEY_CENTS)
    return { error: 'Enter a valid amount already saved.' };
  return { details: { savingsTarget: { targetAmountCents, savedAmountCents } } };
}

/**
 * Produces one goal record; its entered allocation remains the only monthly deduction.
 * @param draft Editable monthly contribution and optional target inputs.
 * @param id Stable identifier for this goal.
 * @returns Validated goal or one field error.
 */
export function goalValue(draft: GoalDraft, id: string): GoalResult {
  const monthlyAmountCents = parseMoney(draft.amount);
  if (
    monthlyAmountCents === null ||
    monthlyAmountCents > MAX_MONEY_CENTS ||
    (draft.kind === 'goal' && !draft.trackTarget && monthlyAmountCents === 0)
  )
    return {
      error:
        'Enter a valid monthly contribution. Goals with a target can have a zero contribution.',
    };
  if (!draft.name.trim()) return { error: 'Enter a goal name.' };
  const target = targetValue(draft);
  if (!target.details) return { error: target.error };
  return {
    goal: {
      id,
      name: draft.name.trim(),
      monthlyAmountCents,
      ownerId: draft.ownerId,
      ...target.details,
    },
  };
}
