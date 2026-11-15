import assert from 'node:assert/strict';
import { test } from 'node:test';
import { sourceAuthorization } from './sourceAuthorization.ts';
import { activitySource } from './activityTestHelpers.mjs';

test('a bank reconnect restores institution access before asking for activity permission', () => {
  const source = activitySource({ state: 'reconnect', needsConsent: true });
  assert.deepEqual(sourceAuthorization(source), {
    input: { connectionId: source.connectionId, purpose: 'balances' },
    label: 'Reconnect institution',
  });
  assert.deepEqual(sourceAuthorization({ ...source, state: 'consent_required' }), {
    input: { connectionId: source.connectionId, purpose: 'activity' },
    label: 'Authorize history',
  });
  assert.equal(sourceAuthorization({ ...source, state: 'ready', needsConsent: false }), null);
});

test('investment reconnect uses only ordinary institution permission', () => {
  const source = activitySource({ kind: 'investment', state: 'reconnect', needsConsent: true });
  assert.equal(sourceAuthorization(source).input.purpose, 'balances');
  assert.equal(sourceAuthorization({ ...source, state: 'consent_required' }), null);
  assert.equal(sourceAuthorization({ ...source, state: 'ready', needsConsent: false }), null);
});

test('disabled history does not start another institution permission request', () => {
  assert.equal(
    sourceAuthorization(activitySource({ enabled: false, state: 'reconnect', needsConsent: true })),
    null,
  );
});
