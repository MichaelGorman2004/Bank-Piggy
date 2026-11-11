import assert from 'node:assert/strict';
import { test } from 'node:test';
import { convertPlan, memberDraft } from './planDraft.ts';

/** Creates a complete plan with distinct money text for both members. */
function planFixture() {
  const members = ['member-1', 'member-2'].map((id) => ({
    id,
    name: id,
    annualBaseSalaryCents: 10000000,
    monthlyTakeHomeCents: 400000,
    accounts: [
      { id: 'savings', name: 'Rainy day', type: 'savings', balanceCents: 12345 },
      { id: 'retirement', name: 'Work retirement', type: 'traditional_401k', balanceCents: 200000 },
    ],
    debtCents: 30000,
    payFrequency: 'monthly',
  }));
  const household = {
    revision: 1,
    members,
    expenses: [],
    goals: [],
    defaultSplitMethod: 'equal',
  };
  return {
    household,
    draft: { members: members.map(memberDraft), expenses: [], goals: [], defaultSplit: 'equal' },
  };
}

test('invalid income and balance fields return their own step without changing the draft', () => {
  const fields = [
    ['name', ' ', 0],
    ['salary', '-1', 0],
    ['takeHome', '1.001', 0],
    ['debt', '1.001', 2],
  ];
  for (const index of [0, 1]) {
    for (const [field, invalid, step] of fields) {
      const { household, draft } = planFixture();
      draft.members[index][field] = invalid;
      const original = structuredClone(draft);
      const result = convertPlan(household, draft);
      assert.equal(result.step, step, `${field} for member ${String(index + 1)}`);
      assert.equal(result.household, undefined);
      assert.ok(result.error.includes(step === 0 ? 'take-home' : 'balances'));
      assert.deepEqual(draft, original);
    }
  }
});

test('a corrected balance saves exact cents and leaves the draft unchanged', () => {
  const { household, draft } = planFixture();
  draft.members[0].accounts[0].balance = '19.99';
  draft.members[1].debt = '0';
  const original = structuredClone(draft);
  const result = convertPlan(household, draft);
  assert.equal(result.error, undefined);
  assert.equal(result.household.members[0].accounts[0].balanceCents, 1999);
  assert.equal(result.household.members[1].debtCents, 0);
  assert.equal(result.household.members[0].monthlyTakeHomeCents, 400000);
  assert.deepEqual(draft, original);
  assert.equal(household.members[0].accounts[0].balanceCents, 12345);
});

test('account validation returns to balances and retains all text after a rejected conversion', () => {
  const { household, draft } = planFixture();
  draft.members[1].accounts[0].balance = '1.001';
  draft.members[0].name = 'A newer name';
  const original = structuredClone(draft);
  const result = convertPlan(household, draft);
  assert.equal(result.household, undefined);
  assert.equal(result.step, 2);
  assert.ok(result.error.includes('account'));
  assert.deepEqual(draft, original);
});

test('account changes retain ordinary target progress and zero monthly allocations', () => {
  const { household, draft } = planFixture();
  const goal = {
    id: 'buffer',
    name: 'Emergency buffer',
    monthlyAmountCents: 0,
    ownerId: 'shared',
    savingsTarget: { targetAmountCents: 500000, savedAmountCents: 200001 },
  };
  household.goals = [goal];
  draft.goals = [{ goal, amount: '0.00' }];
  draft.members[0].accounts[0].balance = '999.99';
  const result = convertPlan(household, draft);
  assert.equal(result.error, undefined);
  assert.deepEqual(result.household.goals, household.goals);
  assert.equal(result.household.members[0].monthlyTakeHomeCents, 400000);
  assert.equal(result.household.members[0].accounts[0].balanceCents, 99999);
});

test('the household editor preserves a zero-allocation trip and its target metadata', () => {
  const { household, draft } = planFixture();
  const trip = {
    savedAmountCents: 1234,
    targetMonth: '2027-06',
    costs: [{ category: 'flights', amountCents: 50000 }],
  };
  const goal = { id: 'vacation', name: 'Vacation', monthlyAmountCents: 0, ownerId: 'shared', trip };
  household.goals = [goal];
  draft.goals = [{ goal, amount: '0.00' }];
  const original = structuredClone(draft);
  const result = convertPlan(household, draft);
  assert.equal(result.error, undefined);
  assert.equal(result.household.goals[0].monthlyAmountCents, 0);
  assert.deepEqual(result.household.goals[0].trip, trip);
  assert.deepEqual(draft, original);
});

test('the household editor preserves purchase metadata and permits a zero monthly allocation', () => {
  const { household, draft } = planFixture();
  const purchase = { targetAmountCents: 125001, savedAmountCents: 30002, targetMonth: '2027-06' };
  const goal = {
    id: 'laptop',
    name: 'Laptop',
    monthlyAmountCents: 1,
    ownerId: 'member-2',
    purchase,
  };
  household.goals = [goal];
  draft.goals = [{ goal, amount: '0.00' }];
  const original = structuredClone(draft);
  const result = convertPlan(household, draft);
  assert.equal(result.error, undefined);
  assert.deepEqual(result.household.goals[0], { ...goal, monthlyAmountCents: 0 });
  assert.deepEqual(draft, original);
  assert.equal(household.goals[0].monthlyAmountCents, 1);
});
