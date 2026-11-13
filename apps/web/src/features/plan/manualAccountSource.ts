import type { ConnectionList, EffectiveBalances } from '@bank-piggy/contracts';

export type ManualAccountSource =
  | { readonly state: 'linked'; readonly balanceCents: number }
  | { readonly state: 'fallback' | 'pending' };
interface ConnectionSnapshot {
  readonly data: ConnectionList | null;
  readonly balances: EffectiveBalances | null;
}

/**
 * Resolves an explicit manual match from the server projection without changing the saved draft.
 * Account names, masks, and unsupported raw balances cannot establish an authoritative match.
 * @param snapshot Shared connection data and its plan-matched effective projection.
 * @param memberID Stable owner of the saved manual account.
 * @param accountID Stable saved manual account identifier.
 * @returns Linked current balance, labeled fallback state, or no enabled manual match.
 */
export function manualAccountSource(
  snapshot: ConnectionSnapshot,
  memberID: string,
  accountID: string,
): ManualAccountSource | undefined {
  const matches = snapshot.data?.connections
    .flatMap((connection) => connection.accounts)
    .filter(
      ({ mapping }) =>
        mapping.enabled &&
        mapping.manualMemberId === memberID &&
        mapping.manualAccountId === accountID,
    );
  if (!matches?.length) return undefined;
  if (!snapshot.balances) return { state: 'pending' };
  const matchedIDs = new Set(matches.map((account) => account.id));
  const effective = snapshot.balances.accounts.find(
    (row) => matchedIDs.has(row.id) && row.source === 'plaid' && row.included,
  );
  return effective?.balanceCents !== null && effective?.balanceCents !== undefined
    ? { state: 'linked', balanceCents: effective.balanceCents }
    : { state: 'fallback' };
}
