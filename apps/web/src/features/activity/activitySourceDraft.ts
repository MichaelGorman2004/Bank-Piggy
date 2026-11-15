import type {
  ActivityAccount,
  ActivityAccountSelection,
  ActivitySettingsRequest,
  ActivitySource,
} from '@bank-piggy/contracts';

/**
 * Preserves existing choices while excluding new accounts and removing revoked account IDs.
 * @param selections Account choices from the start of editing.
 * @param accounts Current provider account inventory.
 * @returns Canonical selections for current accounts; new accounts start excluded.
 */
export function reconcileActivityAccounts(
  selections: ReadonlyArray<ActivityAccountSelection>,
  accounts: ReadonlyArray<ActivityAccount>,
): ActivityAccountSelection[] {
  const saved = new Map(selections.map((selection) => [selection.accountId, selection]));
  return accounts.map((account) => {
    const selection = saved.get(account.accountId);
    return {
      accountId: account.accountId,
      enabled: selection?.enabled ?? false,
      ownership: selection?.ownership ?? account.ownership,
    };
  });
}

/**
 * Identifies inventory changes that do not change the source settings revision.
 * @param selections Original draft account choices.
 * @param accounts Current provider account inventory.
 * @returns Whether account identities changed during editing.
 */
export function activityAccountListChanged(
  selections: ReadonlyArray<ActivityAccountSelection>,
  accounts: ReadonlyArray<ActivityAccount>,
): boolean {
  const saved = new Map(selections.map((selection) => [selection.accountId, selection]));
  return (
    selections.length !== accounts.length ||
    accounts.some((account) => !saved.has(account.accountId))
  );
}

/**
 * Allows explicit opt-out when approval closes while keeping source activation fail closed.
 * @param source Current saved source permissions and revision.
 * @param input Explicit settings draft or no edit.
 * @returns Whether the current source draft can be saved.
 */
export function canSaveActivitySource(
  source: ActivitySource,
  input: ActivitySettingsRequest | null,
): boolean {
  return (
    source.canManage &&
    input !== null &&
    input.expectedRevision === source.revision &&
    (!input.enabled || source.allowed)
  );
}

/** Allows an unavailable saved choice to be switched off without permitting a new selection.
 * @param account Current provider account availability.
 * @param selection Current account-inclusion choice.
 * @returns Whether the inclusion checkbox can change.
 */
export function canToggleActivityAccount(
  account: ActivityAccount,
  selection: ActivityAccountSelection | undefined,
): boolean {
  return account.available || selection?.enabled === true;
}
