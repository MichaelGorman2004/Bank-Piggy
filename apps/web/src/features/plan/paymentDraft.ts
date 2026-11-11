import type { Expense } from '@bank-piggy/contracts';
import { termEndMonth, validMonth } from './month.ts';

export interface PaymentDraft {
  readonly kind: NonNullable<Expense['kind']>;
  readonly providerId: string;
  readonly startMonth: string;
  readonly endMonth: string;
  readonly endMode: 'ongoing' | 'month' | 'term';
  readonly termMonths: string;
}
interface PaymentResult {
  readonly schedule?: Pick<Expense, 'kind' | 'providerId' | 'startMonth' | 'endMonth'>;
  readonly error?: string;
}

/**
 * Keeps old unscheduled costs active while making their metadata editable.
 * @param expense Existing recurring cost, when editing.
 * @returns Editable schedule fields.
 */
export function paymentDraft(expense?: Expense): PaymentDraft {
  const payment: Partial<Expense> = expense ?? {};
  return {
    kind: payment.kind ?? 'bill',
    providerId: payment.providerId ?? '',
    startMonth: payment.startMonth ?? '',
    endMonth: payment.endMonth ?? '',
    endMode: payment.endMonth ? 'month' : 'ongoing',
    termMonths: '',
  };
}

/**
 * Converts a finite term to one canonical inclusive end month.
 * @param draft Editable payment metadata.
 * @returns Validated API metadata or one actionable error.
 */
export function paymentValue(draft: PaymentDraft): PaymentResult {
  if (draft.startMonth && !validMonth(draft.startMonth))
    return { error: 'Enter a valid start month.' };
  const result = paymentEnd(draft);
  if (result.error) return result;
  return {
    schedule: {
      kind: draft.kind,
      ...(draft.providerId ? { providerId: draft.providerId } : {}),
      ...(draft.startMonth ? { startMonth: draft.startMonth } : {}),
      ...(result.month ? { endMonth: result.month } : {}),
    },
  };
}

/**
 * Validates the one canonical inclusive end month.
 * @param draft Editable payment metadata.
 * @returns A final month when finite, or a field error.
 */
function paymentEnd(draft: PaymentDraft): { readonly month?: string; readonly error?: string } {
  if (draft.endMode === 'ongoing') return {};
  const month =
    draft.endMode === 'term' ? termEndMonth(draft.startMonth, draft.termMonths) : draft.endMonth;
  if (!month || !validMonth(month))
    return {
      error: 'Add a valid final month, or a start month and a term from 1 to 1,200 months.',
    };
  if (draft.startMonth && month < draft.startMonth)
    return { error: 'The final payment must be on or after the start month.' };
  return { month };
}

/**
 * Describes whether a saved payment contributes to the selected month.
 * @param expense Saved cost and its optional schedule.
 * @param month Selected projection month.
 * @returns A clear payment status for management lists.
 */
export function paymentStatus(expense: Expense, month: string): string {
  if (expense.startMonth && expense.startMonth > month) return 'Upcoming';
  if (expense.endMonth && expense.endMonth < month) return 'Ended';
  return 'Active';
}
