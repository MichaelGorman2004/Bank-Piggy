import type { Expense, Goal, HouseholdResponse } from '@bank-piggy/contracts';
import { HOUSEHOLD_SCOPE, expenseAmount } from './format.ts';
import type { GoalView } from './views.ts';

export type PaymentFilter = 'all' | NonNullable<Expense['kind']>;

/**
 * Filters all expense types and sorts canonical allocations without changing saved records.
 * @param data Complete plan and calculated allocations for the selected period.
 * @param scope Household or member identifier.
 * @param filter Selected payment type or all types.
 * @returns Largest allocations first, retaining saved order for ties and zero amounts.
 */
export function expensesForView(
  data: HouseholdResponse,
  scope: string,
  filter: PaymentFilter,
): Expense[] {
  return data.household.expenses
    .filter((expense) => {
      if (scope !== HOUSEHOLD_SCOPE && expense.ownerId !== 'shared' && expense.ownerId !== scope)
        return false;
      return filter === 'all' || (expense.kind ?? 'bill') === filter;
    })
    .sort(
      (first, second) => expenseAmount(second, data, scope) - expenseAmount(first, data, scope),
    );
}

/**
 * Separates long-term contributions from trip and purchase targets without changing records.
 * @param goals Complete saved goal list.
 * @param scope Household or member identifier.
 * @param view Focused contribution or target view.
 * @returns Saved records in their original order, including zero-contribution targets.
 */
export function goalsForView(goals: ReadonlyArray<Goal>, scope: string, view: GoalView): Goal[] {
  return goals.filter((goal) => {
    if (scope !== HOUSEHOLD_SCOPE && goal.ownerId !== 'shared' && goal.ownerId !== scope)
      return false;
    const target = Boolean(goal.trip || goal.purchase);
    return view === 'targets' ? target : !target;
  });
}
