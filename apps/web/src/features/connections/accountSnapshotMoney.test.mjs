import assert from 'node:assert/strict';
import { test } from 'node:test';
import { accountSnapshotMoney } from './accountSnapshotMoney.ts';

test('missing and invalid snapshot amounts remain unavailable for every currency', () => {
  const missingOrInvalid = [
    null,
    undefined,
    1.5,
    -1.5,
    Number.NaN,
    Number.POSITIVE_INFINITY,
    Number.NEGATIVE_INFINITY,
    Number.MAX_SAFE_INTEGER + 1,
    Number.MIN_SAFE_INTEGER - 1,
  ];
  for (const cents of missingOrInvalid) {
    for (const currency of ['USD', 'CAD', null, undefined]) {
      assert.equal(accountSnapshotMoney(cents, currency), 'Unavailable');
    }
  }
  assert.equal(accountSnapshotMoney(), 'Unavailable');
});

test('USD snapshots distinguish zero, debt, and exact cent values', () => {
  assert.equal(accountSnapshotMoney(0, 'USD'), '$0.00');
  assert.equal(accountSnapshotMoney(-12345, 'USD'), '-$123.45');
  assert.equal(accountSnapshotMoney(12345, 'USD'), '$123.45');
  assert.equal(accountSnapshotMoney(1, 'USD'), '$0.01');
  assert.equal(accountSnapshotMoney(123456789012345, 'USD'), '$1,234,567,890,123.45');
});

test('non-USD snapshots retain their supplied currency and raw amount without conversion', () => {
  assert.equal(accountSnapshotMoney(12345, 'CAD'), '123.45 CAD');
  assert.equal(accountSnapshotMoney(-12345, 'EUR'), '-123.45 EUR');
  assert.equal(accountSnapshotMoney(0, 'GBP'), '0.00 GBP');
  assert.equal(accountSnapshotMoney(10001, 'JPY'), '100.01 JPY');
});

test('an absent currency keeps the number visible and states that its currency is unavailable', () => {
  for (const currency of [null, undefined, '']) {
    assert.equal(accountSnapshotMoney(12345, currency), '123.45 · Currency unavailable');
    assert.equal(accountSnapshotMoney(0, currency), '0.00 · Currency unavailable');
    assert.equal(accountSnapshotMoney(-101, currency), '-1.01 · Currency unavailable');
  }
});
