import type { Expense, Goal, Household } from '@bank-piggy/contracts';

interface PlanActionOptions {
  readonly household: Household;
  readonly onSave: (household: Household) => Promise<boolean>;
  readonly onClose: () => void;
}

/**
 * Applies each edit against one saved revision and closes only after persistence succeeds.
 * @param root0 Component values and event handlers.
 * @param root0.household Saved household plan.
 * @param root0.onSave Persists the next household revision.
 * @param root0.onClose Closes the current dialog or drawer.
 * @returns The computed result.
 */
export function planActions({ household, onSave, onClose }: PlanActionOptions) {
  /**
   * Closes the editor only after the server accepts its revision.
   * @param next Complete next household plan.
   */
  async function persist(next: Household) {
    if (await onSave(next)) onClose();
  }
  /**
   * Adds or replaces an expense in the complete household plan.
   * @param expense Canonical expense record.
   */
  async function saveExpense(expense: Expense) {
    const exists = household.expenses.some((item) => item.id === expense.id);
    const expenses = exists
      ? household.expenses.map((item) => (item.id === expense.id ? expense : item))
      : [...household.expenses, expense];
    await persist({ ...household, expenses });
  }
  /**
   * Removes an expense from the complete household plan.
   * @param id Record identifier.
   */
  async function deleteExpense(id: string) {
    await persist({ ...household, expenses: household.expenses.filter((item) => item.id !== id) });
  }
  /**
   * Adds or replaces a goal in the complete household plan.
   * @param goal Canonical goal record.
   */
  async function saveGoal(goal: Goal) {
    const exists = household.goals.some((item) => item.id === goal.id);
    const goals = exists
      ? household.goals.map((item) => (item.id === goal.id ? goal : item))
      : [...household.goals, goal];
    await persist({ ...household, goals });
  }
  /**
   * Removes a goal from the complete household plan.
   * @param id Record identifier.
   */
  async function deleteGoal(id: string) {
    await persist({ ...household, goals: household.goals.filter((item) => item.id !== id) });
  }
  return { persist, saveExpense, deleteExpense, saveGoal, deleteGoal };
}
