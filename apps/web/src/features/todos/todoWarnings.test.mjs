import assert from 'node:assert/strict';
import { test } from 'node:test';
import { unscheduledExpensesForScope } from './todoWarnings.ts';

const BILL = {
  id: 'bill',
  name: 'Water',
  category: 'Utilities',
  monthlyAmountCents: 6000,
  ownerId: 'member-1',
  paidByMemberId: 'member-2',
  split: { method: 'equal', customFirstShareBps: 0 },
  kind: 'bill',
};

test('a cross-owner recorded payer receives the missing-due warning and its owner does not', () => {
  assert.deepEqual(unscheduledExpensesForScope([BILL], ['bill'], 'household'), [BILL]);
  assert.deepEqual(unscheduledExpensesForScope([BILL], ['bill'], 'member-2'), [BILL]);
  assert.deepEqual(unscheduledExpensesForScope([BILL], ['bill'], 'member-1'), []);
  assert.deepEqual(unscheduledExpensesForScope([BILL], [], 'member-2'), []);
});

test('recorded shared-bill payers take priority while unassigned shared bills keep setup guidance', () => {
  const recorded = { ...BILL, ownerId: 'shared' };
  assert.deepEqual(unscheduledExpensesForScope([recorded], ['bill'], 'member-1'), []);
  assert.deepEqual(unscheduledExpensesForScope([recorded], ['bill'], 'member-2'), [recorded]);
  const unassigned = {
    ...recorded,
    paidByMemberId: '',
    split: { method: 'custom', customFirstShareBps: 0 },
  };
  assert.deepEqual(unscheduledExpensesForScope([unassigned], ['bill'], 'member-1'), [unassigned]);
  assert.deepEqual(unscheduledExpensesForScope([unassigned], ['bill'], 'member-2'), [unassigned]);
  const personal = { ...BILL, paidByMemberId: '' };
  assert.deepEqual(unscheduledExpensesForScope([personal], ['bill'], 'member-1'), [personal]);
  assert.deepEqual(unscheduledExpensesForScope([personal], ['bill'], 'member-2'), []);
});
