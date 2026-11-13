import assert from 'node:assert/strict';
import { test } from 'node:test';
import { manualAccountSource } from './manualAccountSource.ts';

/** Builds an explicit manual match and a separate authoritative server projection. */
function snapshotFixture() {
  return {
    data: {
      connections: [
        {
          state: 'connected',
          accounts: [
            {
              id: 'linked-account',
              name: 'Retirement',
              mask: '1234',
              currentBalanceCents: 999,
              mapping: {
                enabled: true,
                manualMemberId: 'member-1',
                manualAccountId: 'manual-account',
              },
            },
          ],
        },
      ],
    },
    balances: {
      accounts: [
        { id: 'linked-account', source: 'plaid', included: true, balanceCents: 12345 },
        { id: 'member-1/manual-account', source: 'manual', included: false, balanceCents: 500 },
      ],
    },
  };
}

test('a matched manual editor uses the projected current balance, including zero and negative cents', () => {
  for (const balanceCents of [12345, 0, -101]) {
    const snapshot = snapshotFixture();
    snapshot.balances.accounts[0].balanceCents = balanceCents;
    const original = structuredClone(snapshot);
    assert.deepEqual(manualAccountSource(snapshot, 'member-1', 'manual-account'), {
      state: 'linked',
      balanceCents,
    });
    assert.deepEqual(snapshot, original);
  }
});

test('account names, masks, another member, and disabled matches cannot override a manual balance', () => {
  const snapshot = snapshotFixture();
  assert.equal(manualAccountSource(snapshot, 'member-2', 'manual-account'), undefined);
  assert.equal(manualAccountSource(snapshot, 'member-1', 'another-account'), undefined);
  snapshot.data.connections[0].accounts[0].mapping.enabled = false;
  assert.equal(manualAccountSource(snapshot, 'member-1', 'manual-account'), undefined);
  assert.equal(
    manualAccountSource({ data: null, balances: null }, 'member-1', 'manual-account'),
    undefined,
  );
});

test('missing, unsupported, inactive, or disconnected projections preserve the manual fallback', () => {
  for (const change of [
    (row) => ({ ...row, included: false }),
    (row) => ({ ...row, balanceCents: null }),
    (row) => ({ ...row, source: 'manual' }),
    (row) => ({ ...row, id: 'another-linked-account' }),
  ]) {
    const snapshot = snapshotFixture();
    snapshot.balances.accounts[0] = change(snapshot.balances.accounts[0]);
    assert.deepEqual(manualAccountSource(snapshot, 'member-1', 'manual-account'), {
      state: 'fallback',
    });
    assert.equal(snapshot.balances.accounts[1].balanceCents, 500);
  }
  const snapshot = snapshotFixture();
  snapshot.data.connections[0].state = 'disconnected';
  snapshot.balances.accounts.shift();
  assert.deepEqual(manualAccountSource(snapshot, 'member-1', 'manual-account'), {
    state: 'fallback',
  });
});

test('an unmatched plan revision cannot promote a raw provider balance to current editor data', () => {
  const snapshot = snapshotFixture();
  snapshot.balances = null;
  assert.deepEqual(manualAccountSource(snapshot, 'member-1', 'manual-account'), {
    state: 'pending',
  });
  assert.equal(snapshot.data.connections[0].accounts[0].currentBalanceCents, 999);
});

test('an older disconnected mapping cannot hide the current replacement for that manual account', () => {
  const snapshot = snapshotFixture();
  const oldConnection = structuredClone(snapshot.data.connections[0]);
  oldConnection.state = 'disconnected';
  oldConnection.accounts[0].id = 'old-linked-account';
  snapshot.data.connections.unshift(oldConnection);
  assert.deepEqual(manualAccountSource(snapshot, 'member-1', 'manual-account'), {
    state: 'linked',
    balanceCents: 12345,
  });
});
