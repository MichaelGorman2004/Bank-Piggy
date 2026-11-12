import type { Expense, Household, Split } from '@bank-piggy/contracts';
import { moneyInput, parseMoney } from './format.ts';
import { paymentDraft, paymentValue, type PaymentDraft } from './paymentDraft.ts';
import { defaultOwner, validMemberOrder, validOwner, validPayer } from './householdMembers.ts';

export interface ExpenseDraft extends PaymentDraft {
  readonly id: string;
  readonly name: string;
  readonly amount: string;
  readonly category: string;
  readonly ownerId: Expense['ownerId'];
  readonly method: Split['method'];
  readonly share: string;
  readonly paidBy: Expense['paidByMemberId'];
}
type ExpenseResult =
  | { readonly expense: Expense; readonly error?: never }
  | { readonly error: string; readonly expense?: never };

/**
 * Creates editable strings without losing incomplete decimal input.
 * @param household Saved household plan.
 * @param expense Canonical expense record.
 * @param initialKind Default type for a new record; saved records retain their own type.
 * @returns The computed result.
 */
export function expenseDraft(
  household: Household,
  expense?: Expense,
  initialKind: Expense['kind'] = 'bill',
): ExpenseDraft {
  if (!expense)
    return {
      id: crypto.randomUUID(),
      name: '',
      amount: '',
      category: initialKind === 'subscription' ? 'Other' : 'Housing',
      ownerId: defaultOwner(household.members),
      method: household.defaultSplitMethod,
      share: '50',
      paidBy: '',
      ...paymentDraft(),
      kind: initialKind,
      paymentMode: initialKind === 'subscription' ? 'automatic' : 'manual',
    };
  return {
    id: expense.id,
    name: expense.name,
    amount: moneyInput(expense.monthlyAmountCents),
    category: expense.category,
    ownerId: expense.ownerId,
    method: expense.split.method,
    share: String(expense.split.customFirstShareBps / 100),
    paidBy: expense.paidByMemberId,
    ...paymentDraft(expense),
  };
}

/**
 * Rejects unavailable salary splits and invalid custom percentages before a save.
 * @param draft Current editable field values.
 * @param household Saved household plan.
 * @returns The computed result.
 */
function splitError(draft: ExpenseDraft, household: Household): string | null {
  if (draft.ownerId !== 'shared') return null;
  if (
    draft.method === 'salary' &&
    household.members.length > 1 &&
    household.members.every((member) => member.annualBaseSalaryCents === 0)
  )
    return 'Add annual base salaries to your plan before using a salary split.';
  const share = parseMoney(draft.share);
  if (draft.method === 'custom' && (share === null || share > 10000))
    return 'Enter a share from 0 to 100 percent.';
  return null;
}

/**
 * Rejects stale ownership and payer choices before producing a request.
 * @param draft Current owner and payer selections.
 * @param household Saved household people.
 * @returns An actionable membership error, or null when all selections exist.
 */
function membershipError(draft: ExpenseDraft, household: Household): string | null {
  if (
    !validMemberOrder(household.members) ||
    !validOwner(draft.ownerId, household.members) ||
    !validPayer(draft.paidBy, household.members)
  )
    return 'Choose an available household owner and payer for this expense.';
  return null;
}

/**
 * Validates entered amounts and produces the canonical expense request.
 * @param draft Current editable field values.
 * @param household Saved household plan.
 * @returns The computed result.
 */
export function validateExpense(draft: ExpenseDraft, household: Household): ExpenseResult {
  const membership = membershipError(draft, household);
  if (membership) return { error: membership };
  const amount = parseMoney(draft.amount);
  if (amount === null || amount === 0)
    return { error: 'Enter a monthly amount greater than zero, with up to two decimal places.' };
  if (!draft.name.trim()) return { error: 'Enter an expense name.' };
  const payment = paymentValue(draft);
  if (payment.error) return { error: payment.error };
  const error = splitError(draft, household);
  if (error) return { error };
  const split: Split =
    draft.method === 'custom'
      ? { method: draft.method, customFirstShareBps: parseMoney(draft.share) ?? 5000 }
      : { method: draft.method, customFirstShareBps: 0 };
  return {
    expense: {
      id: draft.id,
      name: draft.name.trim(),
      monthlyAmountCents: amount,
      category: draft.category,
      ownerId: draft.ownerId,
      paidByMemberId: draft.paidBy,
      split,
      ...payment.schedule,
    },
  };
}
