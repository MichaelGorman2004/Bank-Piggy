import assert from 'node:assert/strict';
import { test } from 'node:test';
import { accountSnapshotBalanceLabel, accountSnapshotType } from './accountSnapshotLabels.ts';

/** Builds provider inputs with no confirmed account classification. */
function account(overrides = {}) {
  return {
    name: 'Roth Traditional IRA 401k',
    providerType: 'investment',
    providerSubtype: 'brokerage',
    mapping: { classification: 'unclassified' },
    ...overrides,
  };
}

test('account names never confirm retirement treatment or change provider classifications', () => {
  assert.equal(accountSnapshotType(account()), 'Brokerage');
  assert.equal(accountSnapshotType(account({ providerSubtype: '401k' })), '401(k)');
  assert.equal(accountSnapshotType(account({ providerSubtype: 'roth 401k' })), '401(k)');
  assert.equal(accountSnapshotType(account({ providerSubtype: 'roth' })), 'IRA');
  assert.equal(
    accountSnapshotType(account({ mapping: { classification: 'roth_ira' } })),
    'Roth IRA',
  );
  assert.equal(
    accountSnapshotType(account({ mapping: { classification: 'traditional_401k' } })),
    'Traditional 401(k)',
  );
});

test('unknown provider subtypes remain readable and missing subtypes use the provider type', () => {
  assert.equal(
    accountSnapshotType(account({ providerSubtype: 'new_account_kind' })),
    'New account kind',
  );
  assert.equal(accountSnapshotType(account({ providerSubtype: null })), 'Investment account');
  assert.equal(
    accountSnapshotType(account({ providerType: 'depository', providerSubtype: 'checking' })),
    'Checking',
  );
  assert.equal(
    accountSnapshotType(account({ providerType: 'credit', providerSubtype: 'credit card' })),
    'Credit card',
  );
});

test('credit and loan values describe money owed without presenting them as assets', () => {
  assert.equal(accountSnapshotBalanceLabel(account({ providerType: 'credit' })), 'Amount owed');
  assert.equal(accountSnapshotBalanceLabel(account({ providerType: 'loan' })), 'Amount owed');
  assert.equal(accountSnapshotBalanceLabel(account()), 'Balance');
});
