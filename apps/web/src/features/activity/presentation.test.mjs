import assert from 'node:assert/strict';
import { test } from 'node:test';
import {
  hasActivityHistory,
  reviewCategories,
  canReviewActivity,
  activityAmount,
  fullActivityAmount,
} from './presentation.ts';
import { activityResponse, activityComparison, activityRecord } from './activityTestHelpers.mjs';

test('actual availability uses the authoritative comparison flags for each provider kind', () => {
  const data = activityResponse({ comparison: activityComparison({ bankAvailable: true }) });
  assert.equal(hasActivityHistory(data, 'depository'), true);
  assert.equal(hasActivityHistory(data, 'investment'), false);
  data.comparison = activityComparison({ investmentAvailable: true });
  assert.equal(hasActivityHistory(data, 'depository'), false);
  assert.equal(hasActivityHistory(data, 'investment'), true);
});

test('bank records cannot be recast as investment contributions and investment records cannot be payroll', () => {
  const bank = reviewCategories(activityRecord());
  const investment = reviewCategories(activityRecord({ kind: 'investment' }));
  assert.ok(bank.includes('paycheck'));
  assert.ok(!bank.includes('contribution'));
  assert.ok(!bank.includes('withdrawal'));
  assert.ok(investment.includes('contribution'));
  assert.ok(!investment.includes('paycheck'));
});

test('a personal view can have matched spending from another payer without owning the account', () => {
  const data = activityResponse({
    scope: 'member-2',
    comparison: activityComparison({ bankAvailable: true }),
    records: [
      activityRecord({ ownership: 'member-1', expenseId: 'rent', scopeAmountCents: 50_000 }),
    ],
  });
  assert.equal(hasActivityHistory(data, 'depository'), true);
  assert.equal(hasActivityHistory(activityResponse(), 'depository'), false);
});

test('pending, cancelled, unavailable amounts, and unsupported currencies cannot be manually counted', () => {
  assert.equal(canReviewActivity(activityRecord()), true);
  for (const patch of [
    { pending: true },
    { cancelled: true },
    { amountCents: null },
    { currency: null },
    { currency: 'EUR' },
  ]) {
    assert.equal(canReviewActivity(activityRecord(patch)), false);
  }
  assert.equal(activityAmount(activityRecord({ currency: 'EUR' })), 'EUR · Not included');
  assert.equal(activityAmount(activityRecord({ amountCents: null })), 'Amount unavailable');
  assert.equal(activityAmount(activityRecord({ amountCents: 0 })), '$0.00');
});

test('scoped amounts never substitute the full provider transaction for a missing share', () => {
  const record = activityRecord({ amountCents: 101, scopeAmountCents: 67 });
  assert.equal(activityAmount(record), '$0.67');
  assert.equal(fullActivityAmount(record), '$1.01');
  assert.equal(activityAmount({ ...record, scopeAmountCents: 0 }), '$0.00');
  assert.equal(activityAmount({ ...record, scopeAmountCents: null }), 'Share unavailable');
});
