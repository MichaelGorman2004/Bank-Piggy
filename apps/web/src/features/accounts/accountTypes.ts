import type { AccountBalance, AccountType, Household } from '@bank-piggy/contracts';
import { HOUSEHOLD_SCOPE } from '../plan/format.ts';

export type AccountGroup = 'Savings' | 'Investments';
export const ACCOUNT_TYPES: ReadonlyArray<{
  readonly value: AccountType;
  readonly label: string;
  readonly group: AccountGroup;
}> = [
  { value: 'traditional_401k', label: 'Traditional 401(k)', group: 'Investments' },
  { value: 'roth_401k', label: 'Roth 401(k)', group: 'Investments' },
  { value: 'traditional_ira', label: 'Traditional IRA', group: 'Investments' },
  { value: 'roth_ira', label: 'Roth IRA', group: 'Investments' },
  { value: 'brokerage', label: 'Brokerage', group: 'Investments' },
  { value: 'high_yield_savings', label: 'High-yield savings', group: 'Savings' },
  { value: 'savings', label: 'Savings', group: 'Savings' },
  { value: 'other_investment', label: 'Other investment', group: 'Investments' },
];
export const MAX_ACCOUNTS = 20;
export const MAX_ACCOUNT_NAME_LENGTH = 100;
export const MAX_ACCOUNT_BALANCE_CENTS = 100_000_000_000;

/**
 * Classifies savings snapshots separately from investment snapshots.
 * @param type Canonical account type.
 * @returns The balance group for this type.
 */
function accountGroup(type: AccountType): AccountGroup {
  return type === 'savings' || type === 'high_yield_savings' ? 'Savings' : 'Investments';
}

/**
 * Adds each canonical account once without changing any monthly allocation.
 * @param accounts Saved accounts owned by one or more members.
 * @returns Separate savings and investment balances in integer cents.
 */
export function accountTotals(accounts: ReadonlyArray<AccountBalance>) {
  return accounts.reduce(
    (totals, account) => ({
      savingsCents:
        totals.savingsCents + (accountGroup(account.type) === 'Savings' ? account.balanceCents : 0),
      investmentsCents:
        totals.investmentsCents +
        (accountGroup(account.type) === 'Investments' ? account.balanceCents : 0),
    }),
    { savingsCents: 0, investmentsCents: 0 },
  );
}

/**
 * Groups the selected members' snapshots by canonical type in a stable order.
 * @param household Saved household and member-owned accounts.
 * @param scope Household scope or a member identifier.
 * @returns Populated account types with their counts and summed balances.
 */
export function scopedAccountGroups(household: Household, scope: string) {
  const accounts = household.members
    .filter((member) => scope === HOUSEHOLD_SCOPE || member.id === scope)
    .flatMap((member) => member.accounts);
  return ACCOUNT_TYPES.flatMap((type) => {
    const matches = accounts.filter((account) => account.type === type.value);
    return matches.length
      ? [
          {
            ...type,
            count: matches.length,
            balanceCents: matches.reduce((total, account) => total + account.balanceCents, 0),
          },
        ]
      : [];
  });
}
