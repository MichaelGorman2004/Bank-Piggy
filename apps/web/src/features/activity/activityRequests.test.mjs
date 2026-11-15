import assert from 'node:assert/strict';
import { test } from 'node:test';
import { readActivity, mutateActivity } from './activityRequests.ts';
import { activityOwner, activityResponse, deferred } from './activityTestHelpers.mjs';

test('an older month cannot replace a newer month after controlled out-of-order reads', async () => {
  const first = deferred();
  const second = deferred();
  const { owner, state } = activityOwner((query) =>
    query.month === '2026-11' ? first.promise : second.promise,
  );
  const older = readActivity(owner);
  owner.query = { month: '2026-12', scope: 'member-1' };
  owner.generation += 1;
  const newer = readActivity(owner);
  const accepted = activityResponse({ month: '2026-12', scope: 'member-1' });
  second.resolve(accepted);
  assert.equal(await newer, accepted);
  first.resolve(activityResponse());
  assert.equal(await older, null);
  assert.equal(state.data, accepted);
  assert.equal(state.loading, false);
});

test('a response from the previous scope cannot replace the current monthly selection', async () => {
  const household = deferred();
  const personal = deferred();
  const { owner, state } = activityOwner((query) =>
    query.scope === 'household' ? household.promise : personal.promise,
  );
  const older = readActivity(owner);
  owner.query = { month: '2026-11', scope: 'member-1' };
  owner.generation += 1;
  const newer = readActivity(owner);
  const accepted = activityResponse({ scope: 'member-1' });
  personal.resolve(accepted);
  assert.equal(await newer, accepted);
  household.resolve(activityResponse());
  assert.equal(await older, null);
  assert.equal(state.data, accepted);
  assert.equal(state.loading, false);
});

test('writes are serialized and a selection changed during the write is reloaded afterward', async () => {
  const operation = deferred();
  const queries = [];
  const { owner, state } = activityOwner(async (query) => {
    queries.push(query);
    return activityResponse(query);
  });
  const saved = mutateActivity(owner, () => operation.promise, 'History settings saved.');
  let competingWrites = 0;
  assert.equal(
    await mutateActivity(
      owner,
      async () => {
        competingWrites += 1;
        return {};
      },
      'Other write',
    ),
    null,
  );
  owner.query = { month: '2026-12', scope: 'member-2' };
  assert.equal(await readActivity(owner), null);
  operation.resolve({ revision: 2 });
  assert.deepEqual(await saved, { revision: 2 });
  assert.equal(competingWrites, 0);
  assert.deepEqual(queries, [{ month: '2026-12', scope: 'member-2' }]);
  assert.equal(state.data.month, '2026-12');
  assert.equal(state.busy, false);
});

test('a write invalidates a previously dispatched read before recording the updated snapshot', async () => {
  const older = deferred();
  const operation = deferred();
  let reads = 0;
  const next = activityResponse({ totals: { ...activityResponse().totals, incomeCents: 25_000 } });
  const { owner, state } = activityOwner(() => {
    reads += 1;
    return reads === 1 ? older.promise : Promise.resolve(next);
  });
  const pendingRead = readActivity(owner);
  const pendingWrite = mutateActivity(owner, () => operation.promise, 'Saved.');
  older.resolve(activityResponse({ totals: { ...next.totals, incomeCents: 99_000 } }));
  assert.equal(await pendingRead, null);
  operation.resolve({ revision: 2 });
  await pendingWrite;
  assert.equal(state.data, next);
  assert.equal(state.feedback, 'Saved.');
});

test('failed writes retain their original error across successful and failed reconciliation reads', async () => {
  const previous = activityResponse({ notice: 'Previously recorded monthly totals.' });
  const { owner, state } = activityOwner(async () => previous);
  await readActivity(owner);
  await mutateActivity(
    owner,
    async () => {
      throw new Error('Settings changed in another window.');
    },
    'Saved.',
  );
  assert.equal(state.error, 'Settings changed in another window.');
  assert.equal(state.data, previous);
  owner.load = async () => {
    throw new Error('Reload failed.');
  };
  await mutateActivity(
    owner,
    async () => {
      throw new Error('Original save failure.');
    },
    'Saved.',
  );
  assert.equal(state.error, 'Original save failure.');
  assert.equal(state.data, previous);
});

test('reload failure retains recorded aggregates and a later successful read replaces them', async () => {
  const previous = activityResponse({ notice: 'Previously recorded monthly totals.' });
  const { owner, state } = activityOwner(async () => previous);
  await readActivity(owner);
  owner.load = async () => {
    throw new Error('Monthly reload failed.');
  };
  assert.equal(await readActivity(owner), null);
  assert.equal(state.data, previous);
  assert.equal(state.error, 'Monthly reload failed.');
  assert.equal(state.loading, false);
  const next = activityResponse({ planRevision: 4 });
  owner.load = async () => next;
  assert.equal(await readActivity(owner), next);
  assert.equal(state.data, next);
  assert.equal(state.error, null);
});

test('unmount prevents pending reads and writes from publishing or starting reconciliation', async () => {
  const pending = deferred();
  let reads = 0;
  const { owner, state } = activityOwner(async () => {
    reads += 1;
    return activityResponse();
  });
  const write = mutateActivity(owner, () => pending.promise, 'Saved.');
  owner.mounted = false;
  owner.generation += 1;
  const changes = state.changes;
  pending.resolve({ revision: 2 });
  assert.equal(await write, null);
  assert.equal(await readActivity(owner), null);
  assert.equal(state.changes, changes);
  assert.equal(reads, 0);
});

test('unmount prevents a pending aggregate read from publishing state', async () => {
  const pending = deferred();
  const { owner, state } = activityOwner(() => pending.promise);
  const read = readActivity(owner);
  owner.mounted = false;
  owner.generation += 1;
  const changes = state.changes;
  pending.resolve(activityResponse());
  assert.equal(await read, null);
  assert.equal(state.data, null);
  assert.equal(state.changes, changes);
});
