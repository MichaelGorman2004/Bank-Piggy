import type {
  ActivityCategory,
  ActivityRecord,
  ActivityResponse,
  ConnectionKind,
} from '@bank-piggy/contracts';
import { money } from '../plan/format.ts';

export const ACTIVITY_CATEGORIES: Readonly<Record<ActivityCategory, string>> = {
  paycheck: 'Paycheck',
  income: 'Other income',
  spending: 'Spending',
  contribution: 'Investment contribution',
  withdrawal: 'Investment withdrawal',
  transfer: 'Transfer',
  trade: 'Investment trade',
  investment_income: 'Investment income',
  excluded: 'Excluded',
  review: 'Needs review',
};

/**
 * Uses the server's view-specific availability after source permission and matched-share checks.
 * @param data Authorized monthly response.
 * @param kind Source kind required by this total.
 * @returns Whether this source kind has available recorded amounts for the selected view.
 */
export function hasActivityHistory(data: ActivityResponse, kind: ConnectionKind): boolean {
  return kind === 'depository'
    ? data.comparison.bankAvailable
    : data.comparison.investmentAvailable;
}

/**
 * Restricts review choices to record types that the server can classify.
 * @param record Current provider record.
 * @returns Relevant explicit review categories.
 */
export function reviewCategories(record: ActivityRecord): ReadonlyArray<ActivityCategory> {
  return record.kind === 'bank'
    ? ['paycheck', 'income', 'spending', 'transfer', 'excluded', 'review']
    : [
        'contribution',
        'withdrawal',
        'transfer',
        'trade',
        'investment_income',
        'excluded',
        'review',
      ];
}

/**
 * Keeps pending, cancelled, and unsupported amounts outside manual totals review.
 * @param record Provider activity record.
 * @returns Whether a classification can be edited.
 */
export function canReviewActivity(record: ActivityRecord): boolean {
  return (
    !record.pending && !record.cancelled && record.currency === 'USD' && record.amountCents !== null
  );
}

/**
 * Shows the server-calculated share without substituting the full transaction after a save.
 * @param record Provider activity with an optional share for the selected view.
 * @returns Exact scoped amount or explicit share, currency, and availability guidance.
 */
export function activityAmount(record: ActivityRecord): string {
  if (record.currency !== 'USD' || record.amountCents === null) return fullActivityAmount(record);
  return record.scopeAmountCents === null
    ? 'Share unavailable'
    : money(record.scopeAmountCents, 1, true);
}

/** Shows the full provider transaction as evidence, without treating it as a personal share.
 * @param record Provider activity with the original full amount and currency.
 * @returns Exact supported full amount, or explicit currency and availability guidance.
 */
export function fullActivityAmount(record: ActivityRecord): string {
  if (record.currency !== 'USD')
    return `${record.currency ?? 'Currency unavailable'} · Not included`;
  return record.amountCents === null ? 'Amount unavailable' : money(record.amountCents, 1, true);
}

/**
 * Names the saved history state without exposing provider error codes.
 * @param state Source history state.
 * @returns Short user-facing status.
 */
export function sourceState(state: ActivityResponse['sources'][number]['state']): string {
  const labels = {
    disabled: 'History off',
    consent_required: 'Permission needed',
    initializing: 'History loading',
    ready: 'History available',
    delayed: 'Update delayed',
    reconnect: 'Reconnect needed',
    unsupported: 'History unavailable',
  };
  return labels[state];
}

/**
 * Formats a calendar date without a timezone shift.
 * @param date Canonical calendar date or timestamp.
 * @returns Readable date using local calendar fields.
 */
export function activityDate(date: string): string {
  return new Intl.DateTimeFormat('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  }).format(new Date(`${date.slice(0, 10)}T12:00:00`));
}
