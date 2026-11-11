import { MAX_ACCOUNTS } from '../accounts/accountTypes.ts';
import type { AccountDraft, AccountUpdate } from './accountDraft';
import type { MemberDraft } from './planDraft';

export type PlanAccountAction =
  | { type: 'account'; memberIndex: number; id: string; update: AccountUpdate }
  | { type: 'addAccount'; memberIndex: number; id: string }
  | { type: 'removeAccount'; memberIndex: number; id: string };

/**
 * Changes one member's accounts without discarding other field edits.
 * @param accounts Current account drafts.
 * @param action Account edit, insertion, or deletion.
 * @returns Next account drafts; inserts stop at the contract limit.
 */
function nextAccounts(accounts: AccountDraft[], action: PlanAccountAction): AccountDraft[] {
  if (action.type === 'removeAccount')
    return accounts.filter((account) => account.id !== action.id);
  if (action.type === 'account')
    return accounts.map((account) =>
      account.id === action.id ? { ...account, ...action.update } : account,
    );
  if (accounts.length >= MAX_ACCOUNTS || accounts.some((account) => account.id === action.id))
    return accounts;
  return [...accounts, { id: action.id, name: '', type: 'savings', balance: '' }];
}

/**
 * Applies an account event to the current member draft so queued edits survive.
 * @param members Current member drafts.
 * @param action Account action with its member position and stable account identifier.
 * @returns Members with only the selected account collection changed.
 */
export function accountRows(members: MemberDraft[], action: PlanAccountAction): MemberDraft[] {
  return members.map((member, index) =>
    index === action.memberIndex
      ? { ...member, accounts: nextAccounts(member.accounts, action) }
      : member,
  );
}
