import assert from 'node:assert/strict';
import { test } from 'node:test';
import { expenseDraft, validateExpense } from './expenseDraft.ts';
import { expenseAmount, splitLabel } from './format.ts';
import { goalDraft, goalValue } from './goalDraft.ts';
import { convertPlan } from './planDraft.ts';
import { initialPlan, planReducer } from './planReducer.ts';

/** Creates one complete person with independent income and account balances. */
function memberFixture(id) {
  return {
    id,
    name: id === 'member-1' ? 'Alex' : 'Sam',
    annualBaseSalaryCents: 0,
    monthlyTakeHomeCents: 400001,
    accounts: [{ id: 'buffer', name: 'Buffer', type: 'savings', balanceCents: 12345 }],
    debtCents: 4567,
    payFrequency: 'biweekly',
  };
}

/** Creates personal records whose ownership must survive adding a partner. */
function householdFixture(count = 1) {
  return {
    revision: 7,
    members: ['member-1', 'member-2'].slice(0, count).map(memberFixture),
    defaultSplitMethod: 'custom',
    expenses: [
      {
        id: 'rent',
        name: 'Rent',
        monthlyAmountCents: 100001,
        category: 'Housing',
        ownerId: 'member-1',
        paidByMemberId: 'member-1',
        split: { method: 'custom', customFirstShareBps: 3333 },
        startMonth: '2026-11',
        endMonth: '2027-10',
      },
    ],
    goals: [
      {
        id: 'saving',
        name: 'Buffer',
        monthlyAmountCents: 2501,
        ownerId: 'member-1',
        savingsTarget: { targetAmountCents: 100003, savedAmountCents: 20001 },
      },
    ],
  };
}

test('one person round-trips with no second record and preserves all plan details', () => {
  const household = householdFixture();
  const original = structuredClone(household);
  const draft = initialPlan(household);
  assert.equal(draft.members.length, 1);
  assert.deepEqual(convertPlan(household, draft), { household });
  assert.deepEqual(household, original);
  assert.equal(draft.members[0].takeHome, '4000.01');
});

test('adding a person preserves incomplete drafts and stops after one partner', () => {
  const household = householdFixture();
  let draft = initialPlan(household);
  draft = planReducer(draft, {
    type: 'member',
    index: 0,
    update: { name: 'New name', takeHome: '1.' },
  });
  draft = planReducer(draft, {
    type: 'account',
    memberIndex: 0,
    id: 'buffer',
    update: { balance: '2.' },
  });
  draft = planReducer(draft, { type: 'expense', index: 0, update: { amount: '3.' } });
  draft = planReducer(draft, { type: 'goal', index: 0, update: { amount: '4.' } });
  const original = structuredClone(draft);
  const next = planReducer(draft, { type: 'addMember' });
  assert.equal(next.members.length, 2);
  assert.equal(next.members[1].member.id, 'member-2');
  assert.equal(next.members[1].name, '');
  assert.deepEqual(next.members[1].accounts, []);
  assert.strictEqual(next.members[0], draft.members[0]);
  assert.strictEqual(next.expenses, draft.expenses);
  assert.strictEqual(next.goals, draft.goals);
  assert.deepEqual(draft, original);
  assert.strictEqual(planReducer(next, { type: 'addMember' }), next);
  assert.equal(convertPlan(household, next).step, 0);
  const named = planReducer(next, { type: 'member', index: 1, update: { name: 'Partner' } });
  const result = convertPlan(household, named).household;
  assert.equal(result.members[0].monthlyTakeHomeCents, 100);
  assert.equal(result.members[0].accounts[0].balanceCents, 200);
  assert.equal(result.expenses[0].ownerId, 'member-1');
  assert.equal(result.goals[0].ownerId, 'member-1');
  assert.equal(next.members[0].takeHome, '1.');
});

test('a stored second person remains present with zero income and empty balances', () => {
  const household = householdFixture(2);
  household.members[1] = {
    ...household.members[1],
    monthlyTakeHomeCents: 0,
    accounts: [],
    debtCents: 0,
  };
  const draft = initialPlan(household);
  assert.equal(draft.members.length, 2);
  assert.deepEqual(convertPlan(household, draft), { household });
  assert.strictEqual(planReducer(draft, { type: 'addMember' }), draft);
  household.members[1].name = '';
  assert.equal(initialPlan(household).members.length, 2);
});

test('new one-person records are personal while two-person records start shared', () => {
  for (const count of [1, 2]) {
    const household = householdFixture(count);
    const ownerId = count === 1 ? 'member-1' : 'shared';
    let draft = initialPlan(household);
    draft = planReducer(draft, { type: 'addExpense', id: 'new-expense' });
    draft = planReducer(draft, { type: 'addGoal', id: 'new-goal' });
    assert.equal(draft.expenses.at(-1).expense.ownerId, ownerId);
    assert.equal(draft.goals.at(-1).goal.ownerId, ownerId);
    assert.equal(expenseDraft(household).ownerId, ownerId);
    for (const kind of ['goal', 'trip', 'purchase'])
      assert.equal(goalDraft(household, undefined, kind).ownerId, ownerId);
  }
});

test('adding a partner does not change personal ownership or existing split rules', () => {
  const household = householdFixture();
  const original = structuredClone(household);
  let draft = planReducer(initialPlan(household), { type: 'addMember' });
  draft = planReducer(draft, { type: 'member', index: 1, update: { name: 'Partner' } });
  const saved = convertPlan(household, draft).household;
  assert.deepEqual(saved.expenses, original.expenses);
  assert.deepEqual(saved.goals, original.goals);
  assert.deepEqual(saved.members[0], original.members[0]);
  assert.deepEqual(household, original);
});

test('saved shared records retain their rules and describe the full one-person amount', () => {
  const household = householdFixture();
  const goal = { ...household.goals[0], ownerId: 'shared' };
  for (const method of ['equal', 'salary', 'custom']) {
    const expense = {
      ...household.expenses[0],
      ownerId: 'shared',
      split: { method, customFirstShareBps: 0 },
    };
    const draft = expenseDraft(household, expense);
    assert.deepEqual(validateExpense(draft, household).expense, { ...expense, kind: 'bill' });
    assert.equal(splitLabel(expense, household.members), 'Full amount');
    const data = {
      household,
      summary: {
        expenseShares: [{ expenseId: expense.id, firstMemberCents: 100001, secondMemberCents: 0 }],
      },
    };
    assert.equal(expenseAmount(expense, data, 'household'), 100001);
    assert.equal(expenseAmount(expense, data, 'member-1'), 100001);
  }
  const draft = goalDraft(household, goal);
  assert.equal(draft.ownerId, 'shared');
  assert.deepEqual(goalValue(draft, goal.id, household), { goal });
});

test('plan and individual editors reject owners and payers absent from the household', () => {
  const household = householdFixture();
  for (const field of ['ownerId', 'paidByMemberId']) {
    const draft = initialPlan(household);
    draft.expenses[0].expense = { ...draft.expenses[0].expense, [field]: 'member-2' };
    const original = structuredClone(draft);
    assert.equal(convertPlan(household, draft).step, 1);
    assert.deepEqual(draft, original);
  }
  const draft = initialPlan(household);
  draft.goals[0].goal = { ...draft.goals[0].goal, ownerId: 'member-2' };
  assert.equal(convertPlan(household, draft).step, 2);
  const expense = expenseDraft(household, household.expenses[0]);
  for (const update of [{ ownerId: 'member-2' }, { paidBy: 'member-2' }])
    assert.ok(validateExpense({ ...expense, ...update }, household).error);
  const goal = { ...goalDraft(household, household.goals[0]), ownerId: 'member-2' };
  const original = structuredClone(goal);
  assert.ok(goalValue(goal, 'saving', household).error);
  assert.deepEqual(goal, original);
});

test('invalid member counts and identifiers cannot pass any editor validation', () => {
  const household = householdFixture();
  const invalid = [
    [],
    ['member-2'],
    ['member-1', 'member-1'],
    ['member-2', 'member-1'],
    ['member-1', 'member-2', 'member-3'],
  ];
  for (const ids of invalid) {
    const invalidHousehold = { ...household, members: ids.map(memberFixture) };
    const draft = initialPlan(invalidHousehold);
    assert.equal(convertPlan(invalidHousehold, draft).step, 0);
    assert.ok(
      validateExpense(expenseDraft(household, household.expenses[0]), invalidHousehold).error,
    );
    assert.ok(
      goalValue(goalDraft(household, household.goals[0]), 'saving', invalidHousehold).error,
    );
    assert.strictEqual(planReducer(draft, { type: 'addMember' }), draft);
  }
});
