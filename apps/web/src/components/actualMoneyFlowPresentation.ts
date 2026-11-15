import type { ActivityResponse } from '@bank-piggy/contracts';
import { money } from '../features/plan/format.ts';

/**
 * Distinguishes missing history from a recorded month with no bank flow.
 * @param data Current authorized monthly Activity response.
 * @returns The visible chart state.
 */
export function actualMoneyFlowState(data: ActivityResponse): 'unavailable' | 'empty' | 'ready' {
  if (
    !data.comparison.bankAvailable ||
    data.coverage === 'disabled' ||
    data.coverage === 'not_connected'
  )
    return 'unavailable';
  return data.comparison.bankFlow.totalCents === 0 ? 'empty' : 'ready';
}

/**
 * Supplies exact recorded totals and the limits of the chart for screen readers.
 * @param data Current authorized monthly Activity response.
 * @returns Plain text with no inferred balances or investment deductions.
 */
export function actualMoneyFlowDescription(data: ActivityResponse): string {
  if (actualMoneyFlowState(data) === 'unavailable')
    return 'Recorded bank income and spending are unavailable for this month.';
  return [
    `Recorded income ${money(data.totals.incomeCents, 1, true)}.`,
    `Recorded spending ${money(data.totals.spendingCents, 1, true)}.`,
    'Each stream shows selected posted bank activity for this month.',
    'Its width shows its amount. Refunds and income reversals remain visible.',
    'A recorded surplus or spending gap is an activity difference.',
    'It is not an account balance or proof of borrowing.',
  ].join(' ');
}
