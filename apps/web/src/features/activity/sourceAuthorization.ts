import type { ActivitySource, LinkTokenRequest } from '@bank-piggy/contracts';

/**
 * Restores institution access before requesting separate bank-history permission.
 * @param source Saved source opt-in, connection state, and permission requirement.
 * @returns The deliberate next Link action, or null when no permission action is needed.
 */
export function sourceAuthorization(
  source: ActivitySource,
): { readonly input: LinkTokenRequest; readonly label: string } | null {
  if (!source.enabled || !source.allowed) return null;
  if (source.state === 'reconnect')
    return {
      input: { connectionId: source.connectionId, purpose: 'balances' },
      label: 'Reconnect institution',
    };
  if (source.kind === 'depository' && source.needsConsent)
    return {
      input: { connectionId: source.connectionId, purpose: 'activity' },
      label: 'Authorize history',
    };
  return null;
}
