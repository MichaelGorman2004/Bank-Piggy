import type { Expense } from '@bank-piggy/contracts';
import { HOUSEHOLD_SCOPE } from '../plan/format.ts';

/**
 * Shows missing-due-day setup to the recorded payer before using financial ownership.
 * Shared unassigned bills retain household setup guidance; this does not assert an amount owed.
 * @param expenses Saved household expenses.
 * @param unscheduledIds Active manual expenses identified by the To-do endpoint.
 * @param scope Household or personal view identifier.
 * @returns Expenses whose reminder setup belongs in the selected view.
 */
export function unscheduledExpensesForScope(
  expenses: readonly Expense[],
  unscheduledIds: readonly string[],
  scope: string,
): Expense[] {
  return expenses.filter((expense) => {
    if (!unscheduledIds.includes(expense.id)) return false;
    if (scope === HOUSEHOLD_SCOPE) return true;
    if (expense.paidByMemberId) return expense.paidByMemberId === scope;
    return expense.ownerId === 'shared' || expense.ownerId === scope;
  });
}
