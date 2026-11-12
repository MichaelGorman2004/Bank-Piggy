import assert from 'node:assert/strict';
import { test } from 'node:test';
import { goalDraft, goalValue } from './goalDraft.ts';
import { retirementGoalChange } from './retirementGoal.ts';
import { expenseDraft } from './expenseDraft.ts';
import { paymentValue } from './paymentDraft.ts';

const HOUSEHOLD = {
  members: [{ id: 'member-1' }, { id: 'member-2' }],
  defaultSplitMethod: 'equal',
};

test('Roth IRA preset selects a personal owner and records a chosen annual target once', () => {
  const draft = goalDraft(HOUSEHOLD);
  const change = retirementGoalChange(
    draft,
    { kind: 'roth_ira', name: 'Roth IRA', trackTarget: true },
    HOUSEHOLD,
  );
  assert.equal(change.ownerId, 'member-1');
  const current = {
    ...draft,
    ...change,
    amount: '0',
    targetAmount: '1234.56',
    saved: '100.01',
    contributionYear: '2027',
  };
  const original = structuredClone(current);
  const result = goalValue(current, 'ira', HOUSEHOLD);
  assert.equal(result.error, undefined);
  assert.equal(result.goal.monthlyAmountCents, 0);
  assert.equal(result.goal.contributionYear, 2027);
  assert.deepEqual(result.goal.savingsTarget, {
    targetAmountCents: 123456,
    savedAmountCents: 10001,
  });
  assert.equal(result.goal.purchase, undefined);
  assert.equal(result.goal.trip, undefined);
  assert.equal(goalDraft(HOUSEHOLD, result.goal).kind, 'roth_ira');
  assert.deepEqual(current, original);
});

test('annual IRA targets reject shared ownership, invalid years, and zero targets', () => {
  const draft = {
    ...goalDraft(HOUSEHOLD),
    name: 'Roth IRA',
    kind: 'roth_ira',
    ownerId: 'member-2',
    amount: '50',
    targetAmount: '2000',
    saved: '0',
    contributionYear: '2026',
  };
  for (const update of [
    { ownerId: 'shared' },
    { contributionYear: '2101' },
    { contributionYear: '2026.5' },
    { targetAmount: '0' },
  ]) {
    const current = { ...draft, ...update };
    const original = structuredClone(current);
    assert.ok(goalValue(current, 'ira', HOUSEHOLD).error);
    assert.deepEqual(current, original);
  }
  const ordinary = goalValue({ ...draft, kind: 'goal' }, 'ordinary', HOUSEHOLD).goal;
  assert.equal(ordinary.contributionYear, undefined);
});

test('automatic subscriptions stay budgeted without an invented manual task due day', () => {
  const automatic = expenseDraft(HOUSEHOLD, undefined, 'subscription');
  assert.equal(automatic.paymentMode, 'automatic');
  assert.equal(paymentValue(automatic).schedule.paymentMode, 'automatic');
  assert.equal(paymentValue(automatic).schedule.dueDay, undefined);
  const manual = { ...expenseDraft(HOUSEHOLD), dueDay: '31' };
  assert.equal(paymentValue(manual).schedule.dueDay, 31);
  assert.equal(paymentValue({ ...manual, dueDay: '' }).schedule.dueDay, undefined);
  assert.equal(paymentValue({ ...manual, paymentMode: 'automatic' }).schedule.dueDay, undefined);
  assert.equal(
    paymentValue({ ...manual, paymentMode: 'automatic', dueDay: 'not-a-day' }).schedule.dueDay,
    undefined,
  );
  for (const dueDay of ['0', '32', '1.5', 'not-a-day'])
    assert.ok(paymentValue({ ...manual, dueDay }).error);
});
