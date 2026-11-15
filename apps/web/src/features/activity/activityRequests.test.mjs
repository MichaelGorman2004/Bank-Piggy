import assert from 'node:assert/strict';
import { test } from 'node:test';
import { readActivity, mutateActivity } from './activityRequests.ts';
import {
  activityOwner,
  activityResponse,
  activityRecord,
  deferred,
} from './activityTestHelpers.mjs';

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

test('pagination uses latest full-month totals, updates duplicate revisions, and does not sum totals', async () => {
  const first = activityResponse({
    records: [activityRecord()],
    nextCursor: 'next',
    totalRecordCount: 2,
  });
  const second = activityResponse({
    records: [activityRecord({ revision: 2 }), activityRecord({ id: 'record-2' })],
    totals: { ...first.totals, incomeCents: 100_000 },
    totalRecordCount: 2,
  });
  const { owner, state } = activityOwner(async (query) => (query.cursor ? second : first));
  await readActivity(owner);
  await readActivity(owner, 'next');
  assert.deepEqual(
    state.data.records.map((record) => [record.id, record.revision]),
    [
      ['record-1', 2],
      ['record-2', 1],
    ],
  );
  assert.equal(state.data.totals.incomeCents, 100_000);
  assert.equal(state.data.nextCursor, null);
  assert.equal(await readActivity(owner, 'next'), null);
});

test('pagination cannot append an old scope page after a new selection', async () => {
  const page = deferred();
  const { owner, state } = activityOwner(async (query) =>
    query.cursor ? page.promise : activityResponse({ nextCursor: 'next' }),
  );
  await readActivity(owner);
  const pending = readActivity(owner, 'next');
  owner.query = { month: '2026-11', scope: 'member-1' };
  owner.generation += 1;
  owner.load = async () => activityResponse({ scope: 'member-1' });
  await readActivity(owner);
  page.resolve(activityResponse({ records: [activityRecord()] }));
  assert.equal(await pending, null);
  assert.equal(state.data.scope, 'member-1');
  assert.equal(state.data.records.length, 0);
});

test('writes are serialized and a selection changed during the write is reloaded afterward', async () => {
  const operation = deferred();
  const queries = [];
  const { owner, state } = activityOwner(async (query) => {
    queries.push(query);
    return activityResponse(query);
  });
  const saved = mutateActivity(owner, () => operation.promise, 'Review saved.');
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
  const next = activityResponse({ reviewCount: 0 });
  const { owner, state } = activityOwner(() => {
    reads += 1;
    return reads === 1 ? older.promise : Promise.resolve(next);
  });
  const pendingRead = readActivity(owner);
  const pendingWrite = mutateActivity(owner, () => operation.promise, 'Saved.');
  older.resolve(activityResponse({ reviewCount: 5 }));
  assert.equal(await pendingRead, null);
  operation.resolve({ revision: 2 });
  await pendingWrite;
  assert.equal(state.data, next);
  assert.equal(state.feedback, 'Saved.');
});

test('failed writes retain their original error across successful and failed reconciliation reads', async () => {
  const previous = activityResponse({ records: [activityRecord()] });
  const { owner, state } = activityOwner(async () => previous);
  await readActivity(owner);
  await mutateActivity(
    owner,
    async () => {
      throw new Error('Review changed in another window.');
    },
    'Saved.',
  );
  assert.equal(state.error, 'Review changed in another window.');
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

test('reload refreshes the previously visible page window instead of removing later-page forms', async () => {
  const queries = [];
  const first = activityResponse({ records: [activityRecord()], nextCursor: 'page-2' });
  const { owner, state } = activityOwner(async (query) => {
    queries.push(query);
    return query.cursor
      ? activityResponse({ records: [activityRecord({ id: 'record-2' })] })
      : first;
  });
  await readActivity(owner);
  await readActivity(owner, 'page-2');
  queries.length = 0;
  await readActivity(owner);
  assert.equal(state.data.records.length, 2);
  assert.deepEqual(queries, [
    { month: '2026-11', scope: 'household' },
    { month: '2026-11', scope: 'household', cursor: 'page-2' },
  ]);
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
