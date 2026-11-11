import type { Expense, Household } from '@bank-piggy/contracts';
import { moneyInput } from './format.ts';
import { memberDraft, type MemberDraft, type PlanDraft } from './planDraft.ts';
import { accountRows, type PlanAccountAction } from './planAccountRows.ts';

export type PlanRowUpdate = Partial<Pick<Expense, 'name' | 'ownerId'> & { amount: string }>;
const EQUAL_SHARE_BPS = 5000;
type MemberUpdate = Partial<Omit<MemberDraft, 'member' | 'accounts'>>;
export type PlanAction =
  | { type: 'member'; index: number; update: MemberUpdate }
  | PlanAccountAction
  | { type: 'expense'; index: number; update: PlanRowUpdate }
  | { type: 'goal'; index: number; update: PlanRowUpdate }
  | { type: 'split'; method: Household['defaultSplitMethod'] }
  | { type: 'addExpense'; id: string }
  | { type: 'addGoal'; id: string }
  | { type: 'removeExpense'; index: number }
  | { type: 'removeGoal'; index: number };

/**
 * Copies server records into text drafts when the editor opens.
 * @param household Server household.
 * @returns Initial editable plan.
 */
export function initialPlan(household: Household): PlanDraft {
  return {
    members: household.members.map(memberDraft),
    expenses: household.expenses.map((expense) => ({
      expense,
      amount: moneyInput(expense.monthlyAmountCents),
    })),
    goals: household.goals.map((goal) => ({ goal, amount: moneyInput(goal.monthlyAmountCents) })),
    defaultSplit: household.defaultSplitMethod,
  };
}

/**
 * Merges only edited expense fields against current draft state.
 * @param state Current plan draft.
 * @param action Expense row edit.
 * @returns Updated expense drafts.
 */
function expenseRows(state: PlanDraft, action: Extract<PlanAction, { type: 'expense' }>) {
  const { amount, ...fields } = action.update;
  return state.expenses.map((draft, index) =>
    index === action.index
      ? { amount: amount ?? draft.amount, expense: { ...draft.expense, ...fields } }
      : draft,
  );
}

/**
 * Merges only edited goal fields against current draft state.
 * @param state Current plan draft.
 * @param action Goal row edit.
 * @returns Updated goal drafts.
 */
function goalRows(state: PlanDraft, action: Extract<PlanAction, { type: 'goal' }>) {
  const { amount, ...fields } = action.update;
  return state.goals.map((draft, index) =>
    index === action.index
      ? { amount: amount ?? draft.amount, goal: { ...draft.goal, ...fields } }
      : draft,
  );
}

/**
 * Creates an empty expense with the selected split default.
 * @param method Split method for this new row.
 * @param id Identifier created by the add event.
 * @returns New expense draft.
 */
function newExpense(
  method: Household['defaultSplitMethod'],
  id: string,
): PlanDraft['expenses'][number] {
  const split =
    method === 'custom'
      ? { method, customFirstShareBps: EQUAL_SHARE_BPS }
      : { method, customFirstShareBps: 0 };
  return {
    expense: {
      id,
      name: '',
      category: 'Other',
      monthlyAmountCents: 0,
      ownerId: 'shared',
      paidByMemberId: '',
      split,
    },
    amount: '',
  };
}

/**
 * Applies field edits to current state without discarding queued edits.
 * @param state Current editable plan.
 * @param action Requested edit, insertion, or deletion.
 * @returns Next editable plan.
 */
export function planReducer(state: PlanDraft, action: PlanAction): PlanDraft {
  if ('memberIndex' in action) return { ...state, members: accountRows(state.members, action) };
  switch (action.type) {
    case 'member':
      return {
        ...state,
        members: state.members.map((member, index) =>
          index === action.index ? { ...member, ...action.update } : member,
        ),
      };
    case 'expense':
      return { ...state, expenses: expenseRows(state, action) };
    case 'goal':
      return { ...state, goals: goalRows(state, action) };
    case 'split':
      return { ...state, defaultSplit: action.method };
    case 'addExpense':
      return { ...state, expenses: [...state.expenses, newExpense(state.defaultSplit, action.id)] };
    case 'addGoal':
      return {
        ...state,
        goals: [
          ...state.goals,
          {
            goal: { id: action.id, name: '', monthlyAmountCents: 0, ownerId: 'shared' },
            amount: '',
          },
        ],
      };
    case 'removeExpense':
      return { ...state, expenses: state.expenses.filter((_, index) => index !== action.index) };
    case 'removeGoal':
      return { ...state, goals: state.goals.filter((_, index) => index !== action.index) };
  }
}
