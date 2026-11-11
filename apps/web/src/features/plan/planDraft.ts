import type { Expense, Goal, Household, Member } from '@bank-piggy/contracts';
import { accountDraft, convertAccounts, type AccountDraft } from './accountDraft.ts';
import { moneyInput, parseMoney } from './format.ts';

export interface MemberDraft {
  readonly member: Member;
  readonly name: string;
  readonly salary: string;
  readonly takeHome: string;
  readonly accounts: AccountDraft[];
  readonly debt: string;
  readonly frequency: Member['payFrequency'];
}
interface ExpenseDraft {
  readonly expense: Expense;
  readonly amount: string;
}
interface GoalDraft {
  readonly goal: Goal;
  readonly amount: string;
}
export interface PlanDraft {
  readonly members: MemberDraft[];
  readonly expenses: ExpenseDraft[];
  readonly goals: GoalDraft[];
  readonly defaultSplit: Household['defaultSplitMethod'];
}
type PlanConversion =
  | { readonly household: Household; readonly error?: never; readonly step?: never }
  | { readonly error: string; readonly step: number; readonly household?: never };
type MemberConversion =
  | { readonly members: Member[]; readonly error?: never; readonly step?: never }
  | { readonly error: string; readonly step: number; readonly members?: never };

/**
 * Keeps editable money text separate from canonical integer-cent data.
 * @param member Server member record.
 * @returns Editable member draft.
 */
export function memberDraft(member: Member): MemberDraft {
  return {
    member,
    name: member.name,
    salary: moneyInput(member.annualBaseSalaryCents),
    takeHome: moneyInput(member.monthlyTakeHomeCents),
    accounts: member.accounts.map(accountDraft),
    debt: moneyInput(member.debtCents),
    frequency: member.payFrequency,
  };
}

/**
 * Validates member fields and identifies the step that contains an invalid value.
 * @param drafts Editable member text.
 * @returns Canonical members, or the invalid step and its message.
 */
function convertMembers(drafts: MemberDraft[]): MemberConversion {
  const members: Member[] = [];
  for (const draft of drafts) {
    const salary = parseMoney(draft.salary),
      takeHome = parseMoney(draft.takeHome);
    if (salary === null || takeHome === null || !draft.name.trim())
      return {
        error:
          'Check each member’s name, take-home pay, and salary. Use nonnegative amounts with up to two decimal places.',
        step: 0,
      };
    const debt = parseMoney(draft.debt);
    if (debt === null)
      return {
        error:
          'Check each member’s debt balances. Use nonnegative amounts with up to two decimal places.',
        step: 2,
      };
    const accountResult = convertAccounts(draft.accounts);
    if (!accountResult.accounts) return { error: accountResult.error, step: 2 };
    members.push({
      ...draft.member,
      name: draft.name.trim(),
      annualBaseSalaryCents: salary,
      monthlyTakeHomeCents: takeHome,
      accounts: accountResult.accounts,
      debtCents: debt,
      payFrequency: draft.frequency,
    });
  }
  return { members };
}

/**
 * Rejects unnamed or nonpositive expense drafts.
 * @param drafts Editable expense rows.
 * @returns Canonical expenses, or null when a row is invalid.
 */
function convertExpenses(drafts: ExpenseDraft[]): Expense[] | null {
  const expenses: Expense[] = [];
  for (const draft of drafts) {
    const amount = parseMoney(draft.amount);
    if (amount === null || amount <= 0 || !draft.expense.name.trim()) return null;
    expenses.push({
      ...draft.expense,
      name: draft.expense.name.trim(),
      monthlyAmountCents: amount,
    });
  }
  return expenses;
}

/**
 * Keeps zero-allocation targets valid while checking goal names and amounts.
 * @param drafts Editable goal rows.
 * @returns Canonical goals, or null when a row is invalid.
 */
function convertGoals(drafts: GoalDraft[]): Goal[] | null {
  const goals: Goal[] = [];
  for (const draft of drafts) {
    const amount = parseMoney(draft.amount);
    if (
      amount === null ||
      (!draft.goal.trip && !draft.goal.purchase && !draft.goal.savingsTarget && amount === 0) ||
      !draft.goal.name.trim()
    )
      return null;
    goals.push({ ...draft.goal, name: draft.goal.name.trim(), monthlyAmountCents: amount });
  }
  return goals;
}

/**
 * Converts valid drafts without mutating original records or editable text.
 * @param household Original server household.
 * @param draft Current editable plan.
 * @returns Canonical household, or the invalid step and its message.
 */
export function convertPlan(household: Household, draft: PlanDraft): PlanConversion {
  const memberResult = convertMembers(draft.members);
  if (memberResult.members === undefined) return memberResult;
  const expenses = convertExpenses(draft.expenses);
  if (!expenses)
    return { error: 'Each expense needs a name and a monthly amount greater than zero.', step: 1 };
  const goals = convertGoals(draft.goals);
  if (!goals)
    return {
      error:
        'Each goal needs a name and a valid monthly contribution. Goals with a savings target can have a zero contribution.',
      step: 2,
    };
  return {
    household: {
      ...household,
      members: memberResult.members,
      expenses,
      goals,
      defaultSplitMethod: draft.defaultSplit,
    },
  };
}
