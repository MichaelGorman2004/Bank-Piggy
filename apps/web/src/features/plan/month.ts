const MONTH_PATTERN = /^\d{4}-(0[1-9]|1[0-2])$/;
export const MIN_MONTH = '0001-01';
export const MAX_MONTH = '9999-12';
export const MAX_PLANNING_MONTH = '9998-12';
export const MAX_PAYMENT_MONTHS = 1200;

/**
 * Checks exact calendar months before arithmetic or API requests.
 * @param value Calendar month text.
 * @returns Whether the month has a positive four-digit year.
 */
export function validMonth(value: string): boolean {
  return MONTH_PATTERN.test(value) && value.slice(0, 4) !== '0000';
}

/**
 * Computes an inclusive final payment month from a finite number of payments.
 * @param start First payment month in YYYY-MM format.
 * @param count Number of monthly payments, including the first.
 * @returns Final payment month, or null for an invalid or out-of-range term.
 */
export function termEndMonth(start: string, count: string): string | null {
  if (!validMonth(start) || !/^\d+$/.test(count)) return null;
  const periods = Number(count);
  if (!Number.isInteger(periods) || periods < 1 || periods > MAX_PAYMENT_MONTHS) return null;
  const number = Number(start.slice(0, 4)) * 12 + Number(start.slice(5)) - 1 + periods - 1;
  const year = Math.floor(number / 12);
  if (year > 9999) return null;
  return `${String(year).padStart(4, '0')}-${String((number % 12) + 1).padStart(2, '0')}`;
}

/**
 * Formats a calendar month without converting it through a timezone.
 * @param value Month in YYYY-MM format.
 * @returns A readable month and year, or the original text when invalid.
 */
export function monthLabel(value: string): string {
  if (!validMonth(value)) return value;
  const date = new Date(`${value}-01T12:00:00Z`);
  return new Intl.DateTimeFormat('en-US', {
    month: 'short',
    year: 'numeric',
    timeZone: 'UTC',
  }).format(date);
}
