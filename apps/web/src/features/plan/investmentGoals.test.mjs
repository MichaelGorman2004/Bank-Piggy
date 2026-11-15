import assert from 'node:assert/strict';
import { test } from 'node:test';
import { goalDraft, goalValue } from './goalDraft.ts';

const HOUSEHOLD = { members: [{ id: 'member-1' }, { id: 'member-2' }] };

/** Creates a manual investment goal with independent saved progress. */
function investmentDraft() {
  return {
    ...goalDraft(HOUSEHOLD, undefined, 'investment'),
    name: 'Long-term fund',
    amount: '125.01',
    trackTarget: true,
    targetAmount: '1000.03',
    saved: '400.02',
  };
}

test('an investment goal persists explicit purpose and manual progress independently of its budget', () => {
  const draft = investmentDraft();
  const original = structuredClone(draft);
  const { goal, error } = goalValue(draft, 'invest', HOUSEHOLD);
  assert.equal(error, undefined);
  assert.equal(goal.purpose, 'investment');
  assert.equal(goal.monthlyAmountCents, 12501);
  assert.deepEqual(goal.savingsTarget, { targetAmountCents: 100003, savedAmountCents: 40002 });
  assert.equal(goal.contributionYear, undefined);
  assert.deepEqual(goalDraft(HOUSEHOLD, goal), draft);
  assert.deepEqual(draft, original);
});

test('names never convert an ordinary goal to an investment goal', () => {
  for (const name of ['Investing', 'Roth IRA', 'Brokerage retirement']) {
    const goal = goalValue({ ...investmentDraft(), kind: 'goal', name }, 'generic', HOUSEHOLD).goal;
    assert.equal(goal.purpose, undefined);
    assert.equal(goalDraft(HOUSEHOLD, goal).kind, 'goal');
  }
});

test('investment type changes save only the selected type and retain draft progress', () => {
  const draft = {
    ...investmentDraft(),
    targetMonth: '2027-06',
    contributionYear: '2027',
    ownerId: 'member-1',
  };
  const original = structuredClone(draft);
  for (const kind of ['goal', 'trip', 'purchase', 'roth_ira']) {
    const { goal, error } = goalValue({ ...draft, kind }, 'target', HOUSEHOLD);
    assert.equal(error, undefined);
    assert.equal(goal.purpose, undefined);
    assert.equal(goalDraft(HOUSEHOLD, goal).kind, kind);
    assert.equal(goal.monthlyAmountCents, 12501);
  }
  const { goal } = goalValue({ ...draft, trackTarget: false }, 'invest', HOUSEHOLD);
  assert.equal(goal.purpose, 'investment');
  assert.equal(goal.savingsTarget, undefined);
  assert.deepEqual(draft, original);
});

test('investment allocations use ordinary goal validation for zero and exact cents', () => {
  assert.equal(
    goalValue({ ...investmentDraft(), amount: '0' }, 'invest', HOUSEHOLD).goal.monthlyAmountCents,
    0,
  );
  for (const patch of [{ amount: '0', trackTarget: false }, { amount: '1.001' }, { saved: '-1' }])
    assert.ok(goalValue({ ...investmentDraft(), ...patch }, 'invest', HOUSEHOLD).error);
});
