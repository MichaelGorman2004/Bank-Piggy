declare global {
  interface Window {
    consumeBankPiggyPlaidReturn?: () => string | null;
  }
}

let redirect = window.consumeBankPiggyPlaidReturn?.() ?? null;
delete window.consumeBankPiggyPlaidReturn;

/**
 * Reads the institution OAuth callback captured before document assets loaded.
 * @returns The original complete redirect URI, held only in transient memory.
 */
export function plaidReturn(): string | null {
  return redirect;
}

/** Removes OAuth callback parameters from application memory after Link finishes. */
export function clearPlaidReturn() {
  redirect = null;
}
