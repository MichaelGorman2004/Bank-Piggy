import { HOUSEHOLD_SCOPE } from '../plan/format.ts';
import type { ConnectedAccount, ConnectionList, ConnectionState } from '@bank-piggy/contracts';

/** Shared connection labels keep balance views and connection settings consistent. */
export const CONNECTION_STATE_LABELS: Readonly<Record<ConnectionState, string>> = {
  pending: 'Connection incomplete',
  connected: 'Connected',
  reconnect: 'Reconnect required',
  delayed: 'Update delayed · Last saved balances shown',
  disconnecting: 'Disconnect incomplete · Retry removal',
  disconnected: 'Disconnected',
};

/**
 * Shows a local synchronization time without treating an absent timestamp as fresh.
 * @param value Optional server timestamp.
 * @returns Local date and time, or an explicit unavailable label.
 */
export function syncTime(value: string | null | undefined): string {
  if (!value) return 'Not yet';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return 'Unavailable';
  return date.toLocaleString('en-US', { dateStyle: 'medium', timeStyle: 'short' });
}

/**
 * Keeps missing values and unsupported currencies distinct from a zero USD balance.
 * @param cents Optional provider balance in integer cents.
 * @param currency Optional provider currency code.
 * @returns An exact USD amount or a clear unavailable label.
 */
export function connectedMoney(
  cents: number | null | undefined,
  currency: string | null | undefined,
): string {
  if (!currency) return 'Currency unavailable';
  if (currency !== 'USD') return `${currency} · Not included`;
  if (cents === null || cents === undefined || !Number.isSafeInteger(cents))
    return 'Balance unavailable';
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(cents / 100);
}

const REMOVAL_UNVERIFIED = 'removal_unverified';
export const UNVERIFIED_REMOVAL_NOTICE =
  'Link closed locally. Bank access removal could not be confirmed. If you finished consent, remove access in Plaid or institution settings.';
const CONNECTION_NOTICES: Readonly<Partial<Record<string, string>>> = {
  removal_unverified: UNVERIFIED_REMOVAL_NOTICE,
  exchange_uncertain:
    'The connection result is uncertain. Do not create another connection. Reload status. An operator must check Plaid before another attempt.',
  new_accounts_available:
    'New accounts are available. The connecting member can use Reconnect to review shared accounts.',
  pending_disconnect:
    'Institution access will end soon. The connecting member can use Reconnect to renew consent.',
  pending_expiration:
    'Institution consent will expire soon. The connecting member can use Reconnect to renew it.',
  permission_revoked:
    'Institution consent was revoked. Imported balances are excluded. The connecting member can use Reconnect to restore consent.',
};

/**
 * Gives actionable provider recovery guidance without exposing raw provider errors.
 * @param code Safe server-owned connection state code.
 * @returns Specific recovery guidance, or null for an unrecognized code.
 */
export function connectionNotice(code: string | null): string | null {
  return code ? (CONNECTION_NOTICES[code] ?? null) : null;
}

/**
 * Distinguishes verified provider removal from closing a pending local Link attempt.
 * @param result Authoritative disconnect response.
 * @param id Server-owned connection identifier.
 * @returns Accurate removal feedback without claiming unverified revocation.
 */
export function disconnectFeedback(result: ConnectionList, id: string): string {
  return result.connections.find((connection) => connection.id === id)?.errorCode ===
    REMOVAL_UNVERIFIED
    ? UNVERIFIED_REMOVAL_NOTICE
    : 'Institution disconnected. Manual balances remain saved.';
}

/**
 * Converts Link failure codes to fixed messages without exposing provider error payloads.
 * @param code Safe provider error code, or null after a deliberate exit.
 * @returns Actionable retry guidance, or null for a normal exit.
 */
export function linkFailureMessage(code: string | null): string | null {
  if (!code) return null;
  return code === 'INVALID_LINK_TOKEN'
    ? 'This Plaid session is no longer valid. Choose Connect or Reconnect again to open a new session. Your saved account settings are kept.'
    : 'Plaid could not finish the connection. Your saved plan and account settings are kept. Try the connection again.';
}

/**
 * Detects selected current balances omitted from totals without calculating money.
 * @param connections Saved institution and account state for the current projection.
 * @param scope Household or member presentation scope.
 * @returns Whether an account in this scope needs its unavailable current balance reviewed.
 */
export function hasMissingCurrentBalance(
  connections: ConnectionList['connections'],
  scope: string,
): boolean {
  return connections.some((connection) => {
    if (connection.state === 'disconnecting' || connection.state === 'disconnected') return false;
    return connection.accounts.some(
      (account) =>
        selectedCurrentBalanceMissing(account) &&
        (scope === HOUSEHOLD_SCOPE ||
          account.mapping.ownership === scope ||
          account.mapping.ownership === 'shared'),
    );
  });
}

/**
 * Identifies an otherwise selected USD account with no usable current snapshot.
 * @param account Saved account and its explicit mapping.
 * @returns Whether only its missing current balance prevents a selected snapshot.
 */
function selectedCurrentBalanceMissing(account: ConnectedAccount): boolean {
  return (
    account.active &&
    account.mapping.enabled &&
    account.currency === 'USD' &&
    account.currentBalanceCents === null &&
    account.mapping.classification !== 'unclassified' &&
    (account.providerType === 'depository' || account.providerType === 'investment')
  );
}
