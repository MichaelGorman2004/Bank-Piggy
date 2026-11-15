import assert from 'node:assert/strict';
import { test } from 'node:test';
import { acceptActivitySource, mutateActivity, readActivity } from './activityRequests.ts';
import { acceptActivityDraft } from './activityDrafts.ts';
import {
  activityOwner,
  activityResponse,
  activitySource,
  deferred,
} from './activityTestHelpers.mjs';

test('accepted source settings stay visible when the following monthly read fails and newer edits remain', async () => {
  const original = activitySource({ enabled: false, revision: 2, state: 'disabled' });
  const saved = activitySource({ enabled: true, revision: 3, state: 'initializing' });
  const { owner, state } = activityOwner(async () => activityResponse({ sources: [original] }));
  await readActivity(owner);
  const put = deferred();
  const submitted = { expectedRevision: 2, enabled: true, edit: 1, accounts: [] };
  let current = submitted;
  owner.load = async () => {
    throw new Error('Monthly reload failed.');
  };
  const save = mutateActivity(
    owner,
    async () => {
      const response = await put.promise;
      acceptActivitySource(owner, response);
      return response;
    },
    'History settings saved.',
  );
  current = { ...current, edit: 2, enabled: false };
  put.resolve(saved);
  const response = await save;
  current = acceptActivityDraft(current, submitted, response.revision);
  assert.equal(state.data.sources[0].enabled, true);
  assert.equal(owner.data.sources[0].revision, 3);
  assert.equal(state.error, 'Monthly reload failed.');
  assert.equal(state.totalsStale, true);
  assert.deepEqual(current, { ...submitted, expectedRevision: 3, edit: 2, enabled: false });
  owner.load = async () => activityResponse({ sources: [saved] });
  await readActivity(owner);
  assert.equal(state.totalsStale, false);
});

test('accepted source publishing cannot replace a newer source revision or update an unmounted view', async () => {
  const source = activitySource({ revision: 5 });
  const { owner, state } = activityOwner(async () => activityResponse({ sources: [source] }));
  await readActivity(owner);
  acceptActivitySource(owner, { ...source, revision: 4, enabled: false });
  assert.equal(state.data.sources[0], source);
  owner.mounted = false;
  const changes = state.changes;
  acceptActivitySource(owner, { ...source, revision: 6, enabled: false });
  assert.equal(state.changes, changes);
});
