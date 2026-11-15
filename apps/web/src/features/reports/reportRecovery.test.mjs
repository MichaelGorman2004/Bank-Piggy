import assert from 'node:assert/strict';
import { test } from 'node:test';
import { mutateActivity, readActivity } from '../activity/activityRequests.ts';
import { activityOwner, activityResponse } from '../activity/activityTestHelpers.mjs';
import { currentReport } from './reporting.ts';

test('an accepted pay update with failed reconciliation cannot expose old comparisons', async () => {
  const { owner, state } = activityOwner(async () => {
    throw new Error('Activity reload failed.');
  });
  owner.data = activityResponse({ planRevision: 3 });
  const saved = await mutateActivity(
    owner,
    async () => ({ household: { revision: 4 } }),
    'Pay saved.',
  );
  assert.equal(saved.household.revision, 4);
  assert.equal(state.totalsStale, true);
  assert.equal(
    currentReport(
      owner.data,
      { month: owner.query.month, scope: owner.query.scope, revision: 3 },
      state.totalsStale,
    ),
    null,
  );
  owner.load = async () => activityResponse({ planRevision: 4 });
  await readActivity(owner);
  assert.equal(state.totalsStale, false);
  assert.equal(
    currentReport(
      owner.data,
      { month: owner.query.month, scope: owner.query.scope, revision: 4 },
      state.totalsStale,
    ),
    owner.data,
  );
});

test('an uncertain mutation and failed read keep the original error and withhold old actuals', async () => {
  const { owner, state } = activityOwner(async () => {
    throw new Error('Read failed.');
  });
  owner.data = activityResponse();
  const saved = await mutateActivity(
    owner,
    async () => {
      throw new Error('Write response lost.');
    },
    'Saved.',
  );
  assert.equal(saved, null);
  assert.equal(state.error, 'Write response lost.');
  assert.equal(state.totalsStale, true);
  assert.equal(state.busy, false);
});
