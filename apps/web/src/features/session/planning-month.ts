import { validMonth } from '../plan/month.ts';

/**
 * Reads the explicit month from the page URL before the initial API request.
 * @param url Page URL, defaulting to the current browser location.
 * @returns A supported month, or undefined to use the server's current month.
 */
export function locationMonth(url = window.location.href): string | undefined {
  const month = new URL(url).searchParams.get('month');
  return month && validMonth(month) && month < '9999-01' ? month : undefined;
}

/**
 * Preserves the accepted preview month across a full browser reload.
 * @param month Server-accepted calendar month.
 */
export function rememberMonth(month: string | undefined): void {
  if (!month) return;
  const url = new URL(window.location.href);
  url.searchParams.set('month', month);
  window.history.replaceState(null, '', url);
}
