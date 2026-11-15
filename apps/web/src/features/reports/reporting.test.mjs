import assert from 'node:assert/strict';
import { test } from 'node:test';
import { activityResponse } from '../activity/activityTestHelpers.mjs';
import { currentReport, reportingPeriod } from './reporting.ts';

test('recorded reports use calendar months while preserving the selected forecast period', () => {
  const forecast = 'yearly';
  for (const view of ['flow', 'expenses', 'goals']) {
    assert.equal(reportingPeriod(view, 'actual', forecast), 'monthly');
    assert.equal(reportingPeriod(view, 'planned', forecast), forecast);
  }
  assert.equal(reportingPeriod('overview', 'actual', forecast), forecast);
  assert.equal(reportingPeriod('targets', 'actual', forecast), forecast);
});

test('a changed month, share, plan, or accepted mutation withholds old comparisons', () => {
  const data = activityResponse({ month: '2026-11', scope: 'member-1', planRevision: 7 });
  const target = { month: '2026-11', scope: 'member-1', revision: 7 };
  assert.equal(currentReport(data, target, false), data);
  for (const patch of [{ month: '2026-12' }, { scope: 'household' }, { revision: 8 }]) {
    assert.equal(currentReport(data, { ...target, ...patch }, false), null);
  }
  assert.equal(currentReport(data, target, true), null);
  assert.equal(currentReport(null, target, false), null);
});

test('coherent incomplete history retains its server availability and recorded values', () => {
  const data = activityResponse({ coverage: 'incomplete' });
  data.comparison.bankAvailable = false;
  data.totals.spendingCents = 0;
  const current = currentReport(
    data,
    { month: data.month, scope: data.scope, revision: data.planRevision },
    false,
  );
  assert.equal(current, data);
  assert.equal(current.comparison.bankAvailable, false);
  assert.equal(current.coverage, 'incomplete');
});
