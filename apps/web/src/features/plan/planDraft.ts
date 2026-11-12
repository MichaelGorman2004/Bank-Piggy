import type { Expense, Goal, Household, Member } from '@bank-piggy/contracts';
import { memberIncomeValue } from './memberIncome.ts';
import { accountDraft, convertAccounts, type AccountDraft } from './accountDraft.ts';
import { payrollDraft, type PayrollDraft } from './payrollDraft.ts';
import { payScheduleDraft, type PayScheduleDraft } from './payScheduleDraft.ts';
import { moneyInput, parseMoney } from './format.ts';
import { validMemberOrder, validOwner, validPayer } from './householdMembers.ts';

export interface MemberDraft {
  readonly member: Member;
  readonly name: string;
  readonly salary: string;
  readonly takeHome: string;
  readonly accounts: AccountDraft[];
  readonly debt: string;
  readonly payroll: PayrollDraft;
  readonly paySchedule: PayScheduleDraft;
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
    payroll: payrollDraft(member.payroll),
    paySchedule: payScheduleDraft(member),
  };
}

/**
 * Validates member fields and identifies the step that contains an invalid value.
 * @param drafts Editable member text.
 * @returns Canonical members, or the invalid step and its message.
 */
function convertMembers(drafts: MemberDraft[]): MemberConversion {
  if (!validMemberOrder(drafts.map((draft) => draft.member)))
    return { error: 'Keep one person or add one partner to your household plan.', step: 0 };
  const members: Member[] = [];
  for (const draft of drafts) {
    const income = memberIncomeValue(draft);
    if (!income.fields) return { error: income.error, step: 0 };
    const debt = parseMoney(draft.debt);
    if (debt === null)
      return {
        error:
          'Check each member’s debt balances. Use nonnegative amounts with up to two decimal places.',
        step: 2,
      };
    const accountResult = convertAccounts(draft.accounts);
    if (!accountResult.accounts) return { error: accountResult.error, step: 2 };
    const member = { ...draft.member };
    delete member.payroll;
    delete member.paySchedule;
    members.push({
      ...member,
      ...income.fields,
      accounts: accountResult.accounts,
      debtCents: debt,
    });
  }
  return { members };
}

/**
 * Rejects unnamed or nonpositive expense drafts.
 * @param drafts Editable expense rows.
 * @param members Validated household people.
 * @returns Canonical expenses, or null when a row is invalid.
 */
function convertExpenses(drafts: ExpenseDraft[], members: Member[]): Expense[] | null {
  const expenses: Expense[] = [];
  for (const draft of drafts) {
    const amount = parseMoney(draft.amount);
    if (
      amount === null ||
      amount <= 0 ||
      !draft.expense.name.trim() ||
      !validOwner(draft.expense.ownerId, members) ||
      !validPayer(draft.expense.paidByMemberId, members)
    )
      return null;
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
 * @param members Validated household people.
 * @returns Canonical goals, or null when a row is invalid.
 */
function convertGoals(drafts: GoalDraft[], members: Member[]): Goal[] | null {
  const goals: Goal[] = [];
  for (const draft of drafts) {
    const amount = parseMoney(draft.amount);
    if (
      amount === null ||
      (!draft.goal.trip && !draft.goal.purchase && !draft.goal.savingsTarget && amount === 0) ||
      !draft.goal.name.trim() ||
      !validGoalOwner(draft.goal, members)
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
  const expenses = convertExpenses(draft.expenses, memberResult.members);
  if (!expenses)
    return {
      error:
        'Each expense needs a name, a positive monthly amount, and an available owner and payer.',
      step: 1,
    };
  const goals = convertGoals(draft.goals, memberResult.members);
  if (!goals)
    return {
      error:
        'Each goal needs a name, an available owner, and a valid monthly contribution. Goals with a savings target can have a zero contribution.',
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

/**
 * Retains personal ownership for annual IRA targets in the quick plan editor.
 * @param goal Goal whose owner is being edited.
 * @param members Current household people.
 * @returns Whether the goal owner is available and valid for its target type.
 */
function validGoalOwner(goal: Goal, members: Member[]): boolean {
  return (
    validOwner(goal.ownerId, members) &&
    (goal.contributionYear === undefined || goal.ownerId !== 'shared')
  );
}
