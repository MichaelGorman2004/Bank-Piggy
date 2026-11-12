import type { Expense, Member } from '@bank-piggy/contracts';

export const HOUSEHOLD_MEMBER_IDS: ReadonlyArray<Member['id']> = ['member-1', 'member-2'];
export const MAX_HOUSEHOLD_MEMBERS = HOUSEHOLD_MEMBER_IDS.length;

/**
 * Requires the first person and permits one partner without changing saved identifiers.
 * @param members Saved people in canonical order.
 * @returns Whether the household contains one or two distinct canonical members.
 */
export function validMemberOrder(members: ReadonlyArray<Pick<Member, 'id'>>): boolean {
  return (
    members.length > 0 &&
    members.length <= MAX_HOUSEHOLD_MEMBERS &&
    members.every((member, index) => member.id === HOUSEHOLD_MEMBER_IDS.at(index))
  );
}

/**
 * Keeps a new personal record personal when a partner is added later.
 * @param members Current household people.
 * @returns The sole person's identifier, or shared ownership for two people.
 */
export function defaultOwner(members: ReadonlyArray<Pick<Member, 'id'>>): Expense['ownerId'] {
  const member = members.at(0);
  return members.length === 1 && member ? member.id : 'shared';
}

/**
 * Accepts household ownership or a person who is present in this plan.
 * @param ownerId Selected expense or goal owner.
 * @param members Current household people.
 * @returns Whether the owner is available.
 */
export function validOwner(ownerId: string, members: ReadonlyArray<Pick<Member, 'id'>>): boolean {
  return ownerId === 'shared' || members.some((member) => member.id === ownerId);
}

/**
 * Accepts an unrecorded payer or a person who is present in this plan.
 * @param payerId Selected payer, or an empty value when not recorded.
 * @param members Current household people.
 * @returns Whether the payer is available.
 */
export function validPayer(payerId: string, members: ReadonlyArray<Pick<Member, 'id'>>): boolean {
  return payerId === '' || members.some((member) => member.id === payerId);
}
