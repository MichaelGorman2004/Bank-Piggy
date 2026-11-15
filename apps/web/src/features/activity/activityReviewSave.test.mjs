import assert from 'node:assert/strict';
import { test } from 'node:test';
import { acceptActivityRecord, mutateActivity, readActivity } from './activityRequests.ts';
import { activityOwner, activityResponse, activityRecord } from './activityTestHelpers.mjs';
import { activityAmount } from './presentation.ts';

test('an accepted review remains visible when its follow-up read fails and stale totals are withheld', async () => {
  const original = activityRecord();
  const saved = {
    ...original,
    revision: 2,
    category: 'transfer',
    isReviewed: true,
    scopeAmountCents: null,
  };
  const { owner, state } = activityOwner(async () => activityResponse({ records: [original] }));
  await readActivity(owner);
  owner.load = async () => {
    throw new Error('Reload failed.');
  };
  const result = await mutateActivity(
    owner,
    async () => {
      acceptActivityRecord(owner, saved);
      return saved;
    },
    'Review saved.',
  );
  assert.equal(result, saved);
  assert.equal(state.data.records[0], saved);
  assert.equal(activityAmount(state.data.records[0]), 'Share unavailable');
  assert.equal(state.error, 'Reload failed.');
  assert.equal(state.totalsStale, true);
});
