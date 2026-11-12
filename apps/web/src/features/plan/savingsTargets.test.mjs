import assert from 'node:assert/strict';
import { test } from 'node:test';
import { goalDraft, goalValue } from './goalDraft.ts';
import { goalProgress } from './goalProgress.ts';

const HOUSEHOLD = { members: [{ id: 'member-1' }, { id: 'member-2' }] };

/** Builds an ordinary target with saved progress separate from its contribution. */
function savingsDraft() {
  return {
    ...goalDraft(HOUSEHOLD),
    name: 'Emergency fund',
    amount: '100',
    saved: '650.01',
    targetAmount: '1000.02',
    trackTarget: true,
  };
}

test('ordinary progress round-trips exact cents independently of monthly allocation', () => {
  const draft = savingsDraft();
  const original = structuredClone(draft);
  const { goal, error } = goalValue(draft, 'emergency', HOUSEHOLD);
  assert.equal(error, undefined);
  assert.equal(goal.monthlyAmountCents, 10000);
  assert.deepEqual(goal.savingsTarget, { savedAmountCents: 65001, targetAmountCents: 100002 });
  assert.deepEqual(goalDraft(HOUSEHOLD, goal), { ...draft, amount: '100.00' });
  assert.deepEqual(draft, original);
  assert.deepEqual(goalProgress(goal), {
    savedAmountCents: 65001,
    targetAmountCents: 100002,
    max: 100002,
    value: 65001,
  });
});

test('progress validates positive target and exact nonnegative money without changing the draft', () => {
  for (const change of [
    { targetAmount: '0' },
    { targetAmount: '-1' },
    { targetAmount: '1.001' },
    { targetAmount: '1000000000.01' },
    { saved: '' },
    { saved: '-0.01' },
    { saved: '1.001' },
    { saved: '1000000000.01' },
  ]) {
    const draft = { ...savingsDraft(), ...change };
    const original = structuredClone(draft);
    assert.ok(goalValue(draft, 'emergency', HOUSEHOLD).error);
    assert.deepEqual(draft, original);
  }
});

test('targeted zero contributions and overfunding preserve exact saved amounts', () => {
  const { goal } = goalValue(
    { ...savingsDraft(), amount: '0', saved: '1200' },
    'emergency',
    HOUSEHOLD,
  );
  assert.equal(goal.monthlyAmountCents, 0);
  assert.equal(goalProgress(goal).savedAmountCents, 120000);
  assert.equal(goalProgress(goal).value, 100002);
  assert.equal(goalProgress(goal).max, 100002);
});

test('type and tracking switches retain text while saving only selected target details', () => {
  const draft = { ...savingsDraft(), targetMonth: '2027-06' };
  const off = goalValue({ ...draft, trackTarget: false }, 'emergency', HOUSEHOLD).goal;
  assert.equal(off.savingsTarget, undefined);
  assert.equal(goalProgress(off), null);
  for (const kind of ['trip', 'purchase']) {
    const selected = goalValue({ ...draft, kind }, 'target', HOUSEHOLD).goal;
    assert.equal(selected.savingsTarget, undefined);
    assert.equal(goalProgress(selected), null);
    const back = goalValue({ ...draft, kind: 'goal' }, 'target', HOUSEHOLD).goal;
    assert.equal(back.savingsTarget.savedAmountCents, 65001);
    assert.equal(back.trip, undefined);
    assert.equal(back.purchase, undefined);
  }
});

test('trip progress uses the canonical projection and handles a zero-cost target', () => {
  const goal = {
    ...goalValue({ ...savingsDraft(), kind: 'trip', targetMonth: '2027-06' }, 'trip', HOUSEHOLD)
      .goal,
  };
  assert.deepEqual(goalProgress(goal, { targetAmountCents: 0 }), {
    savedAmountCents: 65001,
    targetAmountCents: 0,
    max: 1,
    value: 1,
  });
});
