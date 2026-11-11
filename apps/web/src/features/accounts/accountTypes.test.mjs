import assert from 'node:assert/strict';
import { test } from 'node:test';
import { ACCOUNT_TYPES, accountTotals, scopedAccountGroups } from './accountTypes.ts';

/** Creates account snapshots with distinct balances across members and types. */
function householdFixture() {
  return {
    members: [
      {
        id: 'member-1',
        accounts: [
          { id: 'savings', type: 'savings', balanceCents: 10001 },
          { id: 'yield', type: 'high_yield_savings', balanceCents: 20002 },
          { id: 'roth', type: 'roth_ira', balanceCents: 30003 },
        ],
      },
      {
        id: 'member-2',
        accounts: [
          { id: 'savings', type: 'savings', balanceCents: 40004 },
          { id: 'brokerage', type: 'brokerage', balanceCents: 0 },
        ],
      },
    ],
  };
}

test('balance totals count savings and high-yield savings once and retain exact cents', () => {
  const { members } = householdFixture();
  assert.deepEqual(accountTotals(members[0].accounts), {
    savingsCents: 30003,
    investmentsCents: 30003,
  });
  assert.deepEqual(accountTotals(members.flatMap((member) => member.accounts)), {
    savingsCents: 70007,
    investmentsCents: 30003,
  });
});

test('all retirement and brokerage types count as investments while both savings types stay separate', () => {
  const accounts = ACCOUNT_TYPES.map((type) => ({
    id: type.value,
    name: type.label,
    type: type.value,
    balanceCents: 101,
  }));
  assert.deepEqual(accountTotals(accounts), { savingsCents: 202, investmentsCents: 606 });
});

test('the balance breakdown scopes ownership and retains zero-balance account types', () => {
  const household = householdFixture();
  const groups = scopedAccountGroups(household, 'household');
  assert.equal(groups.find((group) => group.value === 'savings').balanceCents, 50005);
  assert.equal(groups.find((group) => group.value === 'savings').count, 2);
  assert.equal(groups.find((group) => group.value === 'brokerage').balanceCents, 0);
  const personal = scopedAccountGroups(household, 'member-2');
  assert.deepEqual(
    personal.map((group) => group.value),
    ['brokerage', 'savings'],
  );
  assert.equal(personal.find((group) => group.value === 'savings').balanceCents, 40004);
  assert.deepEqual(scopedAccountGroups(household, 'unknown-member'), []);
  assert.deepEqual(scopedAccountGroups({ members: [] }, 'household'), []);
});
