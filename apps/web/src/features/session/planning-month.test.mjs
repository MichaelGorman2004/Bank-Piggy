import assert from 'node:assert/strict';
import { test } from 'node:test';
import { locationMonth } from './planning-month.ts';

test('invalid month links fall back to the server current month instead of trapping sign-in', () => {
  for (const month of ['bogus', '9999-01', '0000-01', '2026-13', '2026-1', '']) {
    assert.equal(locationMonth(`http://localhost/?month=${encodeURIComponent(month)}`), undefined);
  }
  assert.equal(locationMonth('http://localhost/'), undefined);
  assert.equal(locationMonth('http://localhost/?month=0001-01'), '0001-01');
  assert.equal(locationMonth('http://localhost/?month=9998-12'), '9998-12');
  assert.equal(locationMonth('http://localhost/?month=2029-01'), '2029-01');
});
