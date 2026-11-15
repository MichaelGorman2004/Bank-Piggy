import assert from 'node:assert/strict';
import { test } from 'node:test';
import {
  reconcileActivityAccounts,
  activityAccountListChanged,
  canSaveActivitySource,
  canToggleActivityAccount,
} from './activitySourceDraft.ts';
import { activitySource } from './activityTestHelpers.mjs';

test('account inventory changes preserve current choices, remove revoked IDs, and add excluded choices', () => {
  const source = activitySource();
  const original = source.accounts[0];
  const selected = [
    { accountId: original.accountId, enabled: true, ownership: 'shared' },
    { accountId: 'revoked', enabled: true, ownership: 'member-1' },
  ];
  const inventory = [original, { ...original, accountId: 'new-account', enabled: true }];
  assert.equal(activityAccountListChanged(selected, inventory), true);
  const next = reconcileActivityAccounts(selected, inventory);
  assert.deepEqual(next, [
    selected[0],
    { accountId: 'new-account', enabled: false, ownership: original.ownership },
  ]);
  next[1].enabled = true;
  assert.equal(reconcileActivityAccounts(next, inventory)[1].enabled, true);
  assert.equal(activityAccountListChanged(next, inventory), false);
  assert.equal(activityAccountListChanged(selected, [{ ...original, available: false }]), true);
  assert.equal(
    reconcileActivityAccounts(selected, [{ ...original, available: false }])[0].enabled,
    true,
  );
});

test('a closed approval gate permits explicit opt-out and rejects all source activation', () => {
  const source = activitySource({ allowed: false });
  const disable = { expectedRevision: source.revision, enabled: false, accounts: [] };
  assert.equal(canSaveActivitySource(source, disable), true);
  assert.equal(canSaveActivitySource(source, { ...disable, enabled: true }), false);
  assert.equal(
    canSaveActivitySource({ ...source, enabled: false }, { ...disable, enabled: true }),
    false,
  );
  assert.equal(canSaveActivitySource({ ...source, canManage: false }, disable), false);
  assert.equal(canSaveActivitySource(source, { ...disable, expectedRevision: 0 }), false);
  assert.equal(canSaveActivitySource(source, null), false);
});

test('unavailable saved selections stay explicit and permit only an off action', () => {
  const source = activitySource();
  const unavailable = { ...source.accounts[0], available: false };
  const saved = { accountId: unavailable.accountId, enabled: true, ownership: 'member-1' };
  assert.equal(canToggleActivityAccount(unavailable, saved), true);
  assert.equal(canToggleActivityAccount(unavailable, { ...saved, enabled: false }), false);
  assert.equal(canToggleActivityAccount(unavailable, undefined), false);
  assert.equal(reconcileActivityAccounts([], [unavailable])[0].enabled, false);
  assert.equal(reconcileActivityAccounts([saved], [unavailable])[0].enabled, true);
});
