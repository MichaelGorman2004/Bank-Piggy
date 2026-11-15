import assert from 'node:assert/strict';
import { test } from 'node:test';
import {
  hasActivityHistory,
  isCurrentPaycheck,
  activityDate,
  sourceState,
} from './presentation.ts';
import { activityResponse, activityComparison } from './activityTestHelpers.mjs';

test('actual availability uses the authoritative comparison flags for each provider kind', () => {
  const data = activityResponse({ comparison: activityComparison({ bankAvailable: true }) });
  assert.equal(hasActivityHistory(data, 'depository'), true);
  assert.equal(hasActivityHistory(data, 'investment'), false);
  data.comparison = activityComparison({ investmentAvailable: true });
  assert.equal(hasActivityHistory(data, 'depository'), false);
  assert.equal(hasActivityHistory(data, 'investment'), true);
});

test('personal totals use server availability without requiring client-side records', () => {
  const data = activityResponse({
    scope: 'member-2',
    comparison: activityComparison({ bankAvailable: true }),
  });
  assert.equal(hasActivityHistory(data, 'depository'), true);
  assert.equal(hasActivityHistory(activityResponse(), 'depository'), false);
});

test('pay confirmation requires the suggestion from the selected month, scope, and saved plan', () => {
  const query = { month: '2026-11', scope: 'household' };
  const data = activityResponse({ suggestions: [{ id: 'pay-estimate' }] });
  assert.equal(isCurrentPaycheck(data, query, 3, 'pay-estimate'), true);
  assert.equal(isCurrentPaycheck(null, query, 3, 'pay-estimate'), false);
  assert.equal(isCurrentPaycheck(data, { ...query, month: '2026-12' }, 3, 'pay-estimate'), false);
  assert.equal(isCurrentPaycheck(data, { ...query, scope: 'member-1' }, 3, 'pay-estimate'), false);
  assert.equal(isCurrentPaycheck(data, query, 4, 'pay-estimate'), false);
  assert.equal(isCurrentPaycheck(data, query, 3, 'removed-estimate'), false);
});

test('source status and history dates use safe labels and calendar dates', () => {
  assert.equal(sourceState('consent_required'), 'Permission needed');
  assert.equal(sourceState('reconnect'), 'Reconnect needed');
  assert.equal(sourceState('unsupported'), 'History unavailable');
  assert.equal(activityDate('2026-11-01'), 'Nov 1, 2026');
  assert.equal(activityDate('2026-11-01T01:30:00Z'), 'Nov 1, 2026');
});
