import type { AccountBalance } from '@bank-piggy/contracts';
import {
  ACCOUNT_TYPES,
  MAX_ACCOUNT_BALANCE_CENTS,
  MAX_ACCOUNT_NAME_LENGTH,
  MAX_ACCOUNTS,
  accountTotals,
} from '../accounts/accountTypes.ts';
import { moneyInput, parseMoney } from './format.ts';

export interface AccountDraft extends Pick<AccountBalance, 'id' | 'name' | 'type'> {
  readonly balance: string;
}
export type AccountUpdate = Partial<Pick<AccountDraft, 'name' | 'type' | 'balance'>>;
type AccountConversion =
  | { readonly accounts: AccountBalance[]; readonly error?: never }
  | { readonly error: string; readonly accounts?: never };

/**
 * Copies a snapshot into text fields without changing its stable identifier.
 * @param account Canonical saved account.
 * @returns Editable account text.
 */
export function accountDraft(account: AccountBalance): AccountDraft {
  const { balanceCents, ...fields } = account;
  return { ...fields, balance: moneyInput(balanceCents) };
}

/**
 * Validates one account's identity, type, name, and exact-cent balance.
 * @param draft Editable account fields.
 * @returns Canonical account, or null when a field is invalid.
 */
function accountValue(draft: AccountDraft): AccountBalance | null {
  const balanceCents = parseMoney(draft.balance);
  const name = draft.name.trim();
  if (
    !/^[a-zA-Z0-9_-]{1,64}$/.test(draft.id) ||
    !name ||
    Array.from(name).length > MAX_ACCOUNT_NAME_LENGTH ||
    !ACCOUNT_TYPES.some((type) => type.value === draft.type) ||
    balanceCents === null ||
    balanceCents > MAX_ACCOUNT_BALANCE_CENTS
  )
    return null;
  return { id: draft.id, name, type: draft.type, balanceCents };
}

/**
 * Rejects invalid snapshots and group totals without changing editable text.
 * @param drafts One member's editable accounts.
 * @returns Canonical accounts or a validation message.
 */
export function convertAccounts(drafts: ReadonlyArray<AccountDraft>): AccountConversion {
  if (drafts.length > MAX_ACCOUNTS)
    return { error: `Each member can track up to ${String(MAX_ACCOUNTS)} accounts.` };
  const ids = new Set<string>();
  const accounts: AccountBalance[] = [];
  for (const draft of drafts) {
    const account = accountValue(draft);
    if (!account || ids.has(account.id))
      return {
        error:
          'Each account needs a unique ID, a name, a type, and a valid balance with up to two decimal places.',
      };
    ids.add(account.id);
    accounts.push(account);
  }
  const totals = accountTotals(accounts);
  if (
    totals.savingsCents > MAX_ACCOUNT_BALANCE_CENTS ||
    totals.investmentsCents > MAX_ACCOUNT_BALANCE_CENTS
  )
    return { error: 'A member’s total savings or investments is above the maximum amount.' };
  return { accounts };
}
