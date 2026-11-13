import assert from 'node:assert/strict';
import { test } from 'node:test';
import { setImmediate } from 'node:timers/promises';
import { connectionMutation, connectionRead } from './connectionRequests.ts';
import { startAccountStatusUpdates } from './useAccountStatusUpdates.ts';
import { deferred, ownerFixture } from './connectionRequestTestHelpers.mjs';

test('Accounts entry polling joins an existing initial GET and receives its accepted result', async () => {
  const { state, options } = ownerFixture();
  const request = deferred();
  let loads = 0;
  const load = () => {
    loads += 1;
    return request.promise;
  };
  const initial = connectionRead(options, load);
  const page = Object.assign(new EventTarget(), { visibilityState: 'visible' });
  const browser = Object.assign(new EventTarget(), {
    setInterval: () => 1,
    clearInterval: () => {},
  });
  let polled;
  const stop = startAccountStatusUpdates(
    () => {
      polled = connectionRead(options, load);
      return polled;
    },
    page,
    browser,
  );
  await setImmediate();
  stop();
  assert.equal(loads, 1);
  assert.equal(polled, initial);
  const accepted = { planRevision: 1 };
  request.resolve(accepted);
  assert.equal(await initial, accepted);
  assert.equal(await polled, accepted);
  assert.equal(state.data, accepted);
});

test('revision changes coalesce one fresh GET and status callers receive the latest projection', async () => {
  const { state, options } = ownerFixture();
  const old = deferred();
  const latest = deferred();
  let loads = 0;
  let active = 0;
  let maximum = 0;
  const load = () => {
    loads += 1;
    active += 1;
    maximum = Math.max(maximum, active);
    return (loads === 1 ? old.promise : latest.promise).finally(() => {
      active -= 1;
    });
  };
  const initial = connectionRead(options, load);
  const revision = connectionRead(options, load, undefined, true);
  const poll = connectionRead(options, load);
  const newerRevision = connectionRead(options, load, undefined, true);
  assert.equal(revision, poll);
  assert.equal(revision, newerRevision);
  old.resolve({ planRevision: 1 });
  assert.equal(await initial, null);
  await setImmediate();
  assert.equal(loads, 2);
  assert.equal(maximum, 1);
  assert.equal(state.data, null);
  assert.equal(state.loading, true);
  const accepted = { planRevision: 3 };
  latest.resolve(accepted);
  assert.deepEqual(await Promise.all([revision, poll, newerRevision]), [
    accepted,
    accepted,
    accepted,
  ]);
  assert.equal(state.data, accepted);
  assert.equal(state.loading, false);
});

test('failed mutations reconcile after old GETs and preserve their error through revision changes', async () => {
  for (const queueBeforeMutation of [false, true]) {
    await failedMutationInterleaving(queueBeforeMutation);
  }
});

/** Proves both an existing queued revision and a read deferred by the mutation stay current. */
async function failedMutationInterleaving(queueBeforeMutation) {
  const { state, options } = ownerFixture();
  const old = deferred();
  const latest = deferred();
  const write = deferred();
  let loads = 0;
  const load = () => (++loads === 1 ? old.promise : latest.promise);
  options.read = (error, fresh) => connectionRead(options, load, error, fresh);
  const initial = connectionRead(options, load);
  if (queueBeforeMutation) void connectionRead(options, load, undefined, true);
  const mutation = connectionMutation(options, () => write.promise, 'Saved');
  if (!queueBeforeMutation) assert.equal(await connectionRead(options, load), null);
  write.reject(new Error('Provider removal failed.'));
  await setImmediate();
  const revised = connectionRead(options, load, undefined, true);
  const poll = connectionRead(options, load);
  let finished = false;
  void mutation.then(() => {
    finished = true;
  });
  assert.equal(loads, 1);
  old.resolve({ planRevision: 1 });
  assert.equal(await initial, null);
  await setImmediate();
  assert.equal(finished, false);
  assert.equal(loads, 2);
  assert.equal(state.data, null);
  const accepted = { planRevision: 2 };
  latest.resolve(accepted);
  assert.equal(await mutation, null);
  assert.equal(await revised, accepted);
  assert.equal(await poll, accepted);
  assert.equal(state.data, accepted);
  assert.equal(state.error, 'Provider removal failed.');
  assert.equal(options.pendingRead.current, false);
}

test('a newer accepted mutation clears an older queued mutation failure', async () => {
  const { state, options } = ownerFixture();
  const old = deferred();
  const latest = deferred();
  let loads = 0;
  const load = () => (++loads === 1 ? old.promise : latest.promise);
  options.read = (error, fresh) => connectionRead(options, load, error, fresh);
  const initial = connectionRead(options, load);
  const earlier = connectionRead(options, load, 'Earlier failure', true);
  const accepted = { planRevision: 2 };
  const mutation = connectionMutation(options, async () => accepted, 'Saved');
  await setImmediate();
  old.resolve({ planRevision: 1 });
  assert.equal(await initial, null);
  await setImmediate();
  latest.resolve(accepted);
  assert.equal(await mutation, accepted);
  assert.equal(await earlier, accepted);
  assert.equal(state.error, null);
  assert.equal(state.feedback, 'Saved');
});

test('provider cleanup prevents a queued fresh GET from dispatching after the active GET settles', async () => {
  const { state, options } = ownerFixture();
  const old = deferred();
  let loads = 0;
  const load = () => {
    loads += 1;
    return old.promise;
  };
  const initial = connectionRead(options, load);
  const revised = connectionRead(options, load, undefined, true);
  options.mounted.current = false;
  options.generation.current += 1;
  old.resolve({ planRevision: 1 });
  assert.equal(await initial, null);
  assert.equal(await revised, null);
  assert.equal(loads, 1);
  assert.equal(state.data, null);
  assert.equal(options.reads.current.active, null);
  assert.equal(options.reads.current.queued, null);
});

test('a synchronous loader failure releases GET ownership for a later read', async () => {
  const { state, options } = ownerFixture();
  assert.equal(
    await connectionRead(options, () => {
      throw new Error('Read failed.');
    }),
    null,
  );
  assert.equal(options.reads.current.active, null);
  assert.equal(state.error, 'Read failed.');
  const accepted = { planRevision: 2 };
  assert.equal(await connectionRead(options, async () => accepted), accepted);
  assert.equal(state.data, accepted);
  assert.equal(state.error, null);
});
