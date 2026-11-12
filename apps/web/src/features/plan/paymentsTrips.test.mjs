import assert from 'node:assert/strict';
import { test } from 'node:test';
import { paymentDraft, paymentValue } from './paymentDraft.ts';
import { termEndMonth, validMonth } from './month.ts';
import { goalDraft, goalValue } from './goalDraft.ts';
import { expenseDraft, validateExpense } from './expenseDraft.ts';

const HOUSEHOLD = { members: [{ id: 'member-1' }, { id: 'member-2' }] };

/** Creates a legacy cost without schedule metadata. */
function legacyExpense() {
  return {
    id: 'cost',
    name: 'Car',
    monthlyAmountCents: 80000,
    category: 'Transport',
    ownerId: 'shared',
    paidByMemberId: '',
    split: { method: 'custom', customFirstShareBps: 0 },
  };
}

/** Creates the minimum household needed to validate expense drafts. */
function householdFixture() {
  return {
    defaultSplitMethod: 'equal',
    members: [
      { id: 'member-1', annualBaseSalaryCents: 0 },
      { id: 'member-2', annualBaseSalaryCents: 0 },
    ],
  };
}

test('thirty-six payments end inclusively across year boundaries without Date year coercion', () => {
  assert.equal(termEndMonth('2026-01', '36'), '2028-12');
  assert.equal(termEndMonth('2026-12', '1'), '2026-12');
  assert.equal(termEndMonth('0001-12', '2'), '0002-01');
  for (const count of ['0', '-1', '1.5', '1201', 'Infinity'])
    assert.equal(termEndMonth('2026-01', count), null);
  assert.equal(termEndMonth('9999-12', '2'), null);
  for (const month of ['0000-01', '2026-13', '2026-1', '2026-01-01'])
    assert.equal(validMonth(month), false);
});

test('a finite payment saves only its canonical end month and keeps a zero custom share', () => {
  const household = householdFixture();
  const draft = {
    ...expenseDraft(household, legacyExpense()),
    kind: 'loan',
    startMonth: '2026-01',
    endMode: 'term',
    termMonths: '36',
  };
  const original = structuredClone(draft);
  const result = validateExpense(draft, household);
  assert.equal(result.error, undefined);
  assert.equal(result.expense.endMonth, '2028-12');
  assert.equal(result.expense.startMonth, '2026-01');
  assert.equal(result.expense.monthlyAmountCents, 80000);
  assert.equal(result.expense.split.customFirstShareBps, 0);
  assert.equal('termMonths' in result.expense, false);
  assert.deepEqual(draft, original);
});

test('legacy recurring costs stay undated, and changing to ongoing removes an old end date', () => {
  const draft = paymentDraft(legacyExpense());
  assert.equal(draft.startMonth, '');
  assert.equal('startMonth' in paymentValue(draft).schedule, false);
  const finite = { ...draft, startMonth: '2026-01', endMonth: '2028-12', endMode: 'month' };
  assert.equal(paymentValue(finite).schedule.endMonth, '2028-12');
  const ongoing = paymentValue({ ...finite, endMode: 'ongoing' });
  assert.equal('endMonth' in ongoing.schedule, false);
  assert.equal(ongoing.schedule.startMonth, '2026-01');
});

test('invalid payment terms and reversed end dates produce errors without mutating drafts', () => {
  const draft = paymentDraft();
  const invalid = [
    { ...draft, endMode: 'term', termMonths: '36' },
    { ...draft, startMonth: '2026-01', endMode: 'term', termMonths: '0' },
    { ...draft, startMonth: '2026-13' },
    { ...draft, startMonth: '2026-01', endMonth: '2025-12', endMode: 'month' },
  ];
  for (const item of invalid) {
    const original = structuredClone(item);
    assert.ok(paymentValue(item).error);
    assert.deepEqual(item, original);
  }
});

test('a travel goal keeps saved progress and costs separate from the chosen monthly allocation', () => {
  const draft = {
    ...goalDraft(HOUSEHOLD),
    name: 'Lisbon',
    amount: '0',
    kind: 'trip',
    saved: '300.01',
    targetMonth: '2027-04',
    costs: {
      flights: '600.01',
      lodging: '800',
      food: '120',
      transport: '25',
      activities: '50',
      other: '0',
    },
  };
  const result = goalValue(draft, 'trip', HOUSEHOLD);
  assert.equal(result.error, undefined);
  assert.equal(result.goal.monthlyAmountCents, 0);
  assert.equal(result.goal.trip.savedAmountCents, 30001);
  assert.equal(
    result.goal.trip.costs.reduce((sum, cost) => sum + cost.amountCents, 0),
    159501,
  );
  assert.equal('targetAmountCents' in result.goal.trip, false);
  assert.deepEqual(goalDraft(HOUSEHOLD, result.goal), {
    ...draft,
    amount: '0.00',
    costs: {
      flights: '600.01',
      lodging: '800.00',
      food: '120.00',
      transport: '25.00',
      activities: '50.00',
      other: '0.00',
    },
  });
});

test('ordinary goals omit trip data and reject malformed trip amounts or dates', () => {
  const ordinary = { ...goalDraft(HOUSEHOLD), name: 'Savings', amount: '20' };
  assert.equal('trip' in goalValue(ordinary, 'save', HOUSEHOLD).goal, false);
  const trip = { ...ordinary, kind: 'trip', targetMonth: '2027-04' };
  assert.ok(goalValue({ ...trip, targetMonth: '2027-13' }, 'trip', HOUSEHOLD).error);
  assert.ok(goalValue({ ...trip, saved: '-1' }, 'trip', HOUSEHOLD).error);
  assert.ok(
    goalValue({ ...trip, costs: { ...trip.costs, flights: '2.001' } }, 'trip', HOUSEHOLD).error,
  );
  assert.ok(
    goalValue(
      { ...trip, costs: { ...trip.costs, flights: '1000000000', lodging: '0.01' } },
      'trip',
      HOUSEHOLD,
    ).error,
  );
});
