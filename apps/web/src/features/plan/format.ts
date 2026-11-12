import type { Expense, HouseholdResponse, Member } from '@bank-piggy/contracts';

export type Period = 'monthly' | 'yearly' | 'paycheck';
export const HOUSEHOLD_SCOPE = 'household';
export const PAY_FREQUENCIES: ReadonlyArray<{
  readonly value: Member['payFrequency'];
  readonly label: string;
  readonly perYear: number;
}> = [
  { value: 'monthly', label: 'Monthly', perYear: 12 },
  { value: 'semimonthly', label: 'Twice a month', perYear: 24 },
  { value: 'biweekly', label: 'Every two weeks', perYear: 26 },
  { value: 'weekly', label: 'Weekly', perYear: 52 },
  { value: 'irregular', label: 'Irregular', perYear: 12 },
];
export const EXPENSE_CATEGORIES = [
  'Housing',
  'Groceries',
  'Utilities',
  'Transport',
  'Dining',
  'Health',
  'Debt payments',
  'Other',
];

/**
 * Formats integer cents as US dollars.
 * @param cents Canonical amount in cents.
 * @param factor Optional multiplier for the selected plan period, default one.
 * @param precise Whether to always show cents, default false. Fractional dollars show cents.
 * @returns A currency label.
 */
export function money(cents: number, factor = 1, precise = false): string {
  const digits = precise || Math.round(cents * factor) % 100 !== 0 ? 2 : 0;
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    maximumFractionDigits: digits,
    minimumFractionDigits: digits,
  }).format((cents * factor) / 100);
}

/**
 * Parses decimal money text into integer cents without rounding the entered value.
 * @param value Nonnegative decimal text with at most two decimal places.
 * @returns Integer cents, or null when the input is invalid.
 */
export function parseMoney(value: string): number | null {
  if (!/^\d+(\.\d{0,2})?$/.test(value.trim())) return null;
  const [whole = '0', decimal = ''] = value.trim().split('.');
  const result = Number(whole) * 100 + Number(decimal.padEnd(2, '0'));
  return Number.isSafeInteger(result) ? result : null;
}

/**
 * Formats a saved amount for an editable decimal field.
 * @param cents Canonical amount in cents.
 * @returns Decimal text with two fractional digits.
 */
export function moneyInput(cents: number): string {
  return (cents / 100).toFixed(2);
}

/**
 * Keeps server annual totals intact and scales monthly rates for an average paycheck.
 * @param period Selected reporting period.
 * @param member Selected member, absent for the household view.
 * @returns The multiplier applied to monthly amounts.
 */
export function periodFactor(period: Period, member: Member | undefined): number {
  if (period === 'yearly') return 1;
  if (period !== 'paycheck' || !member) return 1;
  const frequency = PAY_FREQUENCIES.find((item) => item.value === member.payFrequency);
  return 12 / (frequency?.perYear ?? 12);
}

/**
 * Reads the selected expense allocation from the server calculation.
 * @param expense Expense to display.
 * @param data Saved household and calculated shares.
 * @param scope Household or member identifier.
 * @returns The selected amount in cents.
 */
export function expenseAmount(expense: Expense, data: HouseholdResponse, scope: string): number {
  const share = data.summary.expenseShares.find((item) => item.expenseId === expense.id);
  if (!share) return 0;
  if (scope === HOUSEHOLD_SCOPE) return share.firstMemberCents + share.secondMemberCents;
  return data.household.members[0]?.id === scope ? share.firstMemberCents : share.secondMemberCents;
}

/**
 * Describes the rule used to allocate an expense.
 * @param expense Expense and saved split rule.
 * @param members Members available for personal ownership labels.
 * @returns The split or owner label.
 */
export function splitLabel(expense: Expense, members: ReadonlyArray<Member>): string {
  if (expense.ownerId !== 'shared')
    return members.find((member) => member.id === expense.ownerId)?.name ?? 'Personal';
  if (members.length === 1) return 'Full amount';
  if (expense.split.method === 'salary') return 'Salary split';
  if (expense.split.method === 'custom') {
    const firstPercent = expense.split.customFirstShareBps / 100;
    return `${String(firstPercent)}% / ${String(100 - firstPercent)}%`;
  }
  return '50 / 50 split';
}
