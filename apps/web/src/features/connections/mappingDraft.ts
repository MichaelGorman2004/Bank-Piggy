import type {
  AccountMappingRequest,
  ConnectedAccount,
  ConnectedAccountType,
  Household,
} from '@bank-piggy/contracts';
import { ACCOUNT_TYPES } from '../accounts/accountTypes.ts';

export const CONNECTED_TYPES: ReadonlyArray<{
  readonly value: ConnectedAccountType;
  readonly label: string;
}> = [
  { value: 'unclassified', label: 'Choose an account type' },
  { value: 'checking', label: 'Checking' },
  { value: 'hsa', label: 'Health savings account (HSA)' },
  ...ACCOUNT_TYPES,
];

/**
 * Starts a mapping draft at the current server revision without selecting guessed classifications.
 * @param account Current provider account and explicit saved mapping.
 * @returns Revision-checked editable mapping.
 */
export function mappingDraft(account: ConnectedAccount): AccountMappingRequest {
  const { manualMemberId, manualAccountId, ...fields } = account.mapping;
  return {
    ...fields,
    revision: account.mappingRevision,
    ...(manualMemberId && manualAccountId ? { manualMemberId, manualAccountId } : {}),
  };
}

/**
 * Advances a saved draft's revision while retaining edits made during its request.
 * @param current Most recent local draft.
 * @param submitted Exact draft object sent to the server.
 * @param accepted Authoritative saved provider account.
 * @returns Accepted settings or newer local fields with the accepted revision.
 */
export function settleMappingDraft(
  current: AccountMappingRequest,
  submitted: AccountMappingRequest,
  accepted: ConnectedAccount,
): AccountMappingRequest {
  return current === submitted
    ? mappingDraft(accepted)
    : { ...current, revision: accepted.mappingRevision };
}

/**
 * Offers explicit manual replacements without using account names or masks as identity evidence.
 * @param household Current saved plan.
 * @returns Stable native selector options with both account and member identifiers.
 */
export function manualOptions(household: Household) {
  return household.members.flatMap((member) =>
    member.accounts.map((account) => ({
      value: JSON.stringify([member.id, account.id]),
      label: `${member.name} · ${account.name}`,
      memberId: member.id,
      accountId: account.id,
    })),
  );
}

/**
 * Explains snapshot exclusion without silently replacing missing values with zero.
 * @param account Authoritative provider snapshot.
 * @returns A useful exclusion reason, or null when the snapshot is usable.
 */
export function accountProblem(account: ConnectedAccount): string | null {
  if (!account.active) return 'This account is no longer shared by the institution.';
  if (account.providerType !== 'depository' && account.providerType !== 'investment')
    return 'This account type is shown in Accounts but cannot enter the current household balance totals.';
  if (account.currency !== 'USD') return 'Only USD balances can enter household totals.';
  if (account.currentBalanceCents === null)
    return 'A current balance in whole cents is not available.';
  return null;
}
