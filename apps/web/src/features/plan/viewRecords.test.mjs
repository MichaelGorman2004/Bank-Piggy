import assert from 'node:assert/strict';
import { test } from 'node:test';
import { expensesForView, goalsForView } from './viewRecords.ts';

/** Creates saved costs with legacy, inactive, tied, and zero-share records. */
function expenseFixture() {
  const evenSplit = { method: 'equal', customFirstShareBps: 0 };
  const base = {
    name: 'Saved cost',
    monthlyAmountCents: 200,
    category: 'Other',
    ownerId: 'shared',
    paidByMemberId: '',
    split: { method: 'custom', customFirstShareBps: 0 },
  };
  return [
    { ...base, id: 'legacy', split: evenSplit },
    { ...base, id: 'service', kind: 'subscription', monthlyAmountCents: 500 },
    { ...base, id: 'ended-loan', kind: 'loan', endMonth: '2020-01', monthlyAmountCents: 90000 },
    {
      ...base,
      id: 'upcoming-bill',
      kind: 'bill',
      startMonth: '2027-01',
      monthlyAmountCents: 2000,
      split: evenSplit,
    },
    { ...base, id: 'mine', ownerId: 'member-1', kind: 'bill', monthlyAmountCents: 300 },
    {
      ...base,
      id: 'other-service',
      ownerId: 'member-2',
      kind: 'subscription',
      monthlyAmountCents: 700,
    },
    { ...base, id: 'tie-bill', kind: 'bill', monthlyAmountCents: 500 },
    { ...base, id: 'rounding', kind: 'bill', monthlyAmountCents: 101, split: evenSplit },
    { ...base, id: 'zero-share', kind: 'loan', monthlyAmountCents: 600 },
  ];
}

/** Supplies server allocations independently of saved amounts and schedule metadata. */
function responseFixture() {
  const shares = [
    ['legacy', 100, 100],
    ['service', 0, 500],
    ['ended-loan', 0, 0],
    ['upcoming-bill', 0, 0],
    ['mine', 300, 0],
    ['other-service', 0, 700],
    ['tie-bill', 0, 500],
    ['rounding', 51, 50],
    ['zero-share', 0, 600],
  ];
  return {
    planningMonth: '2026-11',
    household: {
      expenses: expenseFixture(),
      members: [{ id: 'member-1' }, { id: 'member-2' }],
    },
    summary: {
      expenseShares: shares.map(([expenseId, firstMemberCents, secondMemberCents]) => ({
        expenseId,
        firstMemberCents,
        secondMemberCents,
      })),
    },
  };
}

/** Creates ordinary, zero-contribution, and fully funded target plans. */
function goalFixture() {
  const base = { name: 'Saved goal', monthlyAmountCents: 100, ownerId: 'shared' };
  return [
    { ...base, id: 'savings' },
    {
      ...base,
      id: 'trip',
      monthlyAmountCents: 0,
      trip: { savedAmountCents: 100, targetMonth: '2020-01', costs: [] },
    },
    {
      ...base,
      id: 'purchase',
      purchase: { savedAmountCents: 100, targetAmountCents: 100, targetMonth: '2027-01' },
    },
    { ...base, id: 'mine', ownerId: 'first' },
    { ...base, id: 'other', ownerId: 'second' },
  ];
}

test('all expense types stay reachable in descending allocations without changing the plan', () => {
  const data = responseFixture();
  const original = structuredClone(data);
  const expenses = expensesForView(data, 'household', 'all');
  assert.deepEqual(
    expenses.map((item) => item.id),
    [
      'other-service',
      'zero-share',
      'service',
      'tie-bill',
      'mine',
      'legacy',
      'rounding',
      'ended-loan',
      'upcoming-bill',
    ],
  );
  assert.equal(expenses.length, data.household.expenses.length);
  for (const expense of expenses)
    assert.equal(
      expense,
      data.household.expenses.find((item) => item.id === expense.id),
    );
  assert.deepEqual(data, original);
});

test('type filters partition the unified list and treat legacy records as bills', () => {
  const data = responseFixture();
  const subscriptions = expensesForView(data, 'household', 'subscription');
  const bills = expensesForView(data, 'household', 'bill');
  const loans = expensesForView(data, 'household', 'loan');
  assert.deepEqual(
    subscriptions.map((item) => item.id),
    ['other-service', 'service'],
  );
  assert.deepEqual(
    bills.map((item) => item.id),
    ['tie-bill', 'mine', 'legacy', 'rounding', 'upcoming-bill'],
  );
  assert.deepEqual(
    loans.map((item) => item.id),
    ['zero-share', 'ended-loan'],
  );
  assert.equal(subscriptions.length + bills.length + loans.length, data.household.expenses.length);
});

test('personal sorting uses allocated cents while keeping shared inactive and zero shares', () => {
  const data = responseFixture();
  assert.deepEqual(
    expensesForView(data, 'member-1', 'all').map((item) => item.id),
    [
      'mine',
      'legacy',
      'rounding',
      'service',
      'ended-loan',
      'upcoming-bill',
      'tie-bill',
      'zero-share',
    ],
  );
  assert.deepEqual(
    expensesForView(data, 'member-2', 'bill').map((item) => item.id),
    ['tie-bill', 'legacy', 'rounding', 'upcoming-bill'],
  );
  assert.deepEqual(
    expensesForView(data, 'member-1', 'subscription').map((item) => item.id),
    ['service'],
  );
});

test('selected annual allocations can reorder upcoming payments and preserve equal totals', () => {
  const data = responseFixture();
  data.summary.expenseShares = data.summary.expenseShares.map((share) => ({
    ...share,
    firstMemberCents: share.expenseId === 'upcoming-bill' ? 10000 : share.firstMemberCents * 12,
    secondMemberCents: share.expenseId === 'upcoming-bill' ? 10000 : share.secondMemberCents * 12,
  }));
  const original = structuredClone(data);
  assert.deepEqual(
    expensesForView(data, 'household', 'all').map((item) => item.id),
    [
      'upcoming-bill',
      'other-service',
      'zero-share',
      'service',
      'tie-bill',
      'mine',
      'legacy',
      'rounding',
      'ended-loan',
    ],
  );
  assert.deepEqual(
    expensesForView(data, 'member-1', 'bill').map((item) => item.id),
    ['upcoming-bill', 'mine', 'legacy', 'rounding', 'tie-bill'],
  );
  assert.deepEqual(data, original);
});

test('goal views retain overdue, zero-contribution, and funded targets with original ownership', () => {
  const goals = goalFixture();
  const original = structuredClone(goals);
  const ordinary = goalsForView(goals, 'household', 'goals');
  const targets = goalsForView(goals, 'household', 'targets');
  assert.deepEqual(
    ordinary.map((item) => item.id),
    ['savings', 'mine', 'other'],
  );
  assert.deepEqual(
    targets.map((item) => item.id),
    ['trip', 'purchase'],
  );
  assert.equal(ordinary.length + targets.length, goals.length);
  assert.equal(targets[0], goals[1]);
  assert.deepEqual(
    goalsForView(goals, 'first', 'goals').map((item) => item.id),
    ['savings', 'mine'],
  );
  assert.deepEqual(goalsForView(goals, 'first', 'targets'), targets);
  assert.deepEqual(goals, original);
});

test('focused views support an empty plan', () => {
  const data = responseFixture();
  data.household.expenses = [];
  for (const filter of ['all', 'subscription', 'bill', 'loan'])
    assert.deepEqual(expensesForView(data, 'household', filter), []);
  assert.deepEqual(goalsForView([], 'household', 'goals'), []);
  assert.deepEqual(goalsForView([], 'household', 'targets'), []);
});
