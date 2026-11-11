import assert from 'node:assert/strict';
import { test } from 'node:test';
import Ajv from 'ajv';
import schema from '@bank-piggy/contracts/schema.json' with { type: 'json' };
import { goalDraft, goalValue } from './goalDraft.ts';

/** Creates editable purchase inputs with nonzero saved progress and exact cents. */
function purchaseDraft() {
  return {
    ...goalDraft(undefined, 'purchase'),
    name: 'Laptop',
    amount: '25.01',
    ownerId: 'member-1',
    targetAmount: '1200.01',
    saved: '300.02',
    targetMonth: '2027-04',
  };
}

test('a purchase saves progress separately from its chosen monthly allocation', () => {
  const draft = purchaseDraft();
  const original = structuredClone(draft);
  const result = goalValue(draft, 'laptop');
  assert.equal(result.error, undefined);
  assert.deepEqual(result.goal, {
    id: 'laptop',
    name: 'Laptop',
    monthlyAmountCents: 2501,
    ownerId: 'member-1',
    purchase: { targetAmountCents: 120001, savedAmountCents: 30002, targetMonth: '2027-04' },
  });
  assert.deepEqual(goalDraft(result.goal), draft);
  assert.deepEqual(draft, original);
});

test('purchase inputs reject invalid cents, limits, and target months without changing drafts', () => {
  const changes = [
    { targetAmount: '-1' },
    { targetAmount: '1.001' },
    { targetAmount: '1000000000.01' },
    { saved: '-0.01' },
    { saved: '1.001' },
    { saved: '1000000000.01' },
    { targetMonth: '2027-13' },
    { targetMonth: '0000-01' },
    { amount: '1000000000.01' },
  ];
  for (const change of changes) {
    const draft = { ...purchaseDraft(), ...change };
    const original = structuredClone(draft);
    assert.ok(goalValue(draft, 'laptop').error);
    assert.deepEqual(draft, original);
  }
});

test('purchase targets accept zero allocations, zero targets, and bounded saved amounts', () => {
  const result = goalValue(
    {
      ...purchaseDraft(),
      amount: '0',
      targetAmount: '0',
      saved: '1000000000',
      targetMonth: '9999-12',
    },
    'laptop',
  );
  assert.equal(result.error, undefined);
  assert.equal(result.goal.monthlyAmountCents, 0);
  assert.equal(result.goal.purchase.targetAmountCents, 0);
  assert.equal(result.goal.purchase.savedAmountCents, 100_000_000_000);
  assert.equal(
    goalValue({ ...purchaseDraft(), kind: 'goal', amount: '0' }, 'saving').goal,
    undefined,
  );
});

test('goal type changes retain editable text and save only the selected target details', () => {
  const draft = purchaseDraft();
  const ordinary = goalValue({ ...draft, kind: 'goal' }, 'saving').goal;
  assert.equal('purchase' in ordinary, false);
  assert.equal('trip' in ordinary, false);
  const trip = goalValue({ ...draft, kind: 'trip' }, 'vacation').goal;
  assert.equal('purchase' in trip, false);
  assert.equal(trip.trip.savedAmountCents, 30002);
  const purchase = goalValue({ ...draft, kind: 'purchase' }, 'laptop').goal;
  assert.equal('trip' in purchase, false);
  assert.equal(purchase.purchase.targetAmountCents, 120001);
  assert.equal(draft.targetAmount, '1200.01');
  assert.equal(goalDraft(ordinary, 'trip').kind, 'goal');
  assert.equal(goalDraft(trip, 'purchase').kind, 'trip');
  assert.equal(goalDraft(purchase, 'trip').kind, 'purchase');
});

test('the canonical schema accepts legacy goals and rejects incomplete or conflicting targets', () => {
  const validator = new Ajv({ allErrors: true });
  validator.addSchema(schema, 'bank-piggy');
  const validGoal = validator.compile({ $ref: 'bank-piggy#/$defs/Goal' });
  const purchase = goalValue(purchaseDraft(), 'laptop').goal;
  assert.equal(validGoal(purchase), true);
  const ordinary = goalValue({ ...purchaseDraft(), kind: 'goal' }, 'saving').goal;
  assert.equal(validGoal(ordinary), true);
  assert.equal('purchase' in ordinary, false);
  const trip = goalValue({ ...purchaseDraft(), kind: 'trip' }, 'vacation').goal;
  assert.equal(validGoal(trip), true);
  assert.equal(validGoal({ ...trip, purchase: purchase.purchase }), false);
  for (const target of [null, {}, { ...purchase.purchase, targetAmountCents: 1.5 }]) {
    assert.equal(validGoal({ ...purchase, purchase: target }), false);
  }
});

test('editing an ordinary contribution retains existing saved-target metadata', () => {
  const original = {
    id: 'fund',
    name: 'Emergency fund',
    monthlyAmountCents: 0,
    ownerId: 'shared',
    savingsTarget: { targetAmountCents: 100000, savedAmountCents: 65000 },
  };
  const draft = goalDraft(original);
  const { goal } = goalValue({ ...draft, name: 'Renamed fund' }, original.id);
  assert.deepEqual(goal.savingsTarget, original.savingsTarget);
  assert.equal(goal.monthlyAmountCents, 0);
  assert.equal(original.name, 'Emergency fund');
});
