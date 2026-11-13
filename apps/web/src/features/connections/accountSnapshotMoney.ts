import { connectedMoney } from './presentation.ts';

/**
 * Formats a raw provider snapshot, including currencies that do not enter USD totals.
 * Missing balances remain unavailable; a missing currency is never treated as USD.
 * @param cents Optional provider balance in integer cents.
 * @param currency Optional provider currency code.
 * @returns A full account value with currency or an explicit unavailable label.
 */
export function accountSnapshotMoney(
  cents: number | null | undefined,
  currency: string | null | undefined,
): string {
  if (cents === null || cents === undefined || !Number.isSafeInteger(cents)) return 'Unavailable';
  if (currency === 'USD') return connectedMoney(cents, currency);
  const value = new Intl.NumberFormat('en-US', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(cents / 100);
  return `${value} ${currency || '· Currency unavailable'}`;
}
