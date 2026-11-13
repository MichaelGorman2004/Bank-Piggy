import assert from 'node:assert/strict';
import { test } from 'node:test';
import { connectionMutation, connectionRead, hasCompletedAttempt } from './connectionRequests.ts';
import { deferred, ownerFixture } from './connectionRequestTestHelpers.mjs';

test('only one mutation runs and a completed request cannot replace a newer request generation', async () => {
  const { state, options } = ownerFixture();
  const first = deferred();
  let secondRan = false;
  const pending = connectionMutation(options, () => first.promise, 'Saved');
  assert.equal(state.busy, true);
  assert.equal(
    await connectionMutation(
      options,
      async () => {
        secondRan = true;
        return { id: 'second' };
      },
      'Saved',
    ),
    null,
  );
  assert.equal(secondRan, false);
  options.generation.current += 1;
  state.data = { id: 'newer' };
  first.resolve({ id: 'older' });
  assert.equal(await pending, null);
  assert.deepEqual(state.data, { id: 'newer' });
  assert.equal(state.busy, false);
  assert.equal(options.lock.current, false);
});

test('a stale background read cannot replace an accepted provider mutation', async () => {
  const { state, options } = ownerFixture();
  const snapshot = deferred();
  const reading = connectionRead(options, () => snapshot.promise);
  const accepted = { id: 'accepted-mutation' };
  assert.equal(await connectionMutation(options, async () => accepted, 'Saved'), accepted);
  snapshot.resolve({ id: 'stale-read' });
  assert.equal(await reading, null);
  assert.equal(state.data, accepted);
  assert.equal(state.feedback, 'Saved');
});

test('lost new or reconnect responses require proof of the exact completed attempt', () => {
  const connection = {
    id: 'connection',
    state: 'reconnect',
    lastCompletedAttemptId: 'older-attempt',
  };
  assert.equal(
    hasCompletedAttempt({ connections: [connection] }, 'connection', 'new-attempt'),
    false,
  );
  assert.equal(
    hasCompletedAttempt({ connections: [connection] }, 'connection', 'older-attempt'),
    true,
  );
  assert.equal(
    hasCompletedAttempt({ connections: [connection] }, 'foreign', 'older-attempt'),
    false,
  );
  assert.equal(hasCompletedAttempt(null, 'connection', 'older-attempt'), false);
  for (const state of ['pending', 'disconnecting', 'disconnected']) {
    assert.equal(
      hasCompletedAttempt(
        { connections: [{ ...connection, state }] },
        'connection',
        'older-attempt',
      ),
      false,
    );
  }
  assert.equal(
    hasCompletedAttempt(
      { connections: [{ ...connection, lastCompletedAttemptId: null }] },
      'connection',
      'new-attempt',
    ),
    false,
  );
  assert.equal(
    hasCompletedAttempt(
      { connections: [{ ...connection, lastCompletedAttemptId: 'new-attempt' }] },
      'connection',
      'new-attempt',
    ),
    true,
  );
});

test('a deferred plan read preserves the original mutation failure and is awaited', async () => {
  const { state, options } = ownerFixture();
  const failure = deferred();
  const read = deferred();
  let receivedError;
  options.read = async (preservedError) => {
    receivedError = preservedError;
    await read.promise;
    state.data = { id: 'updated-manual-plan' };
    state.error = preservedError;
  };
  const pending = connectionMutation(options, () => failure.promise, 'Saved');
  options.pendingRead.current = true;
  failure.reject(new Error('Provider removal failed.'));
  await Promise.resolve();
  assert.equal(receivedError, 'Provider removal failed.');
  assert.equal(state.error, 'Provider removal failed.');
  let finished = false;
  void pending.then(() => {
    finished = true;
  });
  await Promise.resolve();
  assert.equal(finished, false);
  read.resolve();
  assert.equal(await pending, null);
  assert.deepEqual(state.data, { id: 'updated-manual-plan' });
  assert.equal(state.error, 'Provider removal failed.');
  assert.equal(options.pendingRead.current, false);
});

test('unmounted connection operations release their lock without exposing results to another session', async () => {
  const { state, options } = ownerFixture();
  const request = deferred();
  const pending = connectionMutation(options, () => request.promise, 'Saved');
  options.mounted.current = false;
  options.generation.current += 1;
  request.resolve({ id: 'old-session' });
  assert.equal(await pending, null);
  assert.equal(state.data, null);
  assert.equal(state.feedback, null);
  assert.equal(options.lock.current, false);
});
