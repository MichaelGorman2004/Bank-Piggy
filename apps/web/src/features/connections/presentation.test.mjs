import assert from 'node:assert/strict';
import { test } from 'node:test';
import {
  connectedMoney,
  hasMissingCurrentBalance,
  connectionNotice,
  disconnectFeedback,
  UNVERIFIED_REMOVAL_NOTICE,
} from './presentation.ts';

test('connected balances distinguish missing data, zero, negative balances, and other currencies', () => {
  assert.equal(connectedMoney(undefined, 'USD'), 'Balance unavailable');
  assert.equal(connectedMoney(0, 'USD'), '$0.00');
  assert.equal(connectedMoney(-10001, 'USD'), '-$100.01');
  assert.equal(connectedMoney(12345, 'CAD'), 'CAD · Not included');
  assert.equal(connectedMoney(12345, undefined), 'Currency unavailable');
  assert.equal(connectedMoney(1.5, 'USD'), 'Balance unavailable');
  assert.equal(connectedMoney(Number.MAX_SAFE_INTEGER + 1, 'USD'), 'Balance unavailable');
});

test('local pending closure never claims verified provider revocation', () => {
  assert.equal(
    disconnectFeedback(
      { connections: [{ id: 'pending', errorCode: 'removal_unverified' }] },
      'pending',
    ),
    UNVERIFIED_REMOVAL_NOTICE,
  );
  assert.match(
    disconnectFeedback({ connections: [{ id: 'connected', errorCode: null }] }, 'connected'),
    /^Institution disconnected/,
  );
  assert.match(connectionNotice('exchange_uncertain'), /Do not create another/);
  assert.match(connectionNotice('permission_revoked'), /balances are excluded/);
  assert.equal(connectionNotice('unknown_provider_error'), null);
  assert.equal(connectionNotice(null), null);
});

test('server consent and account notice codes remain actionable after a balance update', () => {
  assert.match(connectionNotice('new_accounts_available'), /Reconnect to review/);
  assert.match(connectionNotice('pending_disconnect'), /Reconnect to renew/);
  assert.match(connectionNotice('pending_expiration'), /Reconnect to renew/);
});

/** Builds account inputs with a previously enabled USD mapping. */
function selectedAccount(overrides = {}) {
  return {
    active: true,
    providerType: 'depository',
    currency: 'USD',
    currentBalanceCents: null,
    availableBalanceCents: 100,
    mapping: { enabled: true, ownership: 'member-1', classification: 'checking' },
    ...overrides,
  };
}

test('review notice depends on current balance, not available balance or a zero amount', () => {
  const needsReview = (account) =>
    hasMissingCurrentBalance([{ state: 'connected', accounts: [account] }], 'household');
  assert.equal(needsReview(selectedAccount()), true);
  assert.equal(needsReview(selectedAccount({ currentBalanceCents: 0 })), false);
  assert.equal(
    needsReview(selectedAccount({ currentBalanceCents: 123, availableBalanceCents: null })),
    false,
  );
  assert.equal(needsReview(selectedAccount({ availableBalanceCents: null })), true);
});

test('review notice follows household, personal, and shared account ownership', () => {
  const personal = [{ state: 'connected', accounts: [selectedAccount()] }];
  assert.equal(hasMissingCurrentBalance(personal, 'household'), true);
  assert.equal(hasMissingCurrentBalance(personal, 'member-1'), true);
  assert.equal(hasMissingCurrentBalance(personal, 'member-2'), false);
  const shared = [
    {
      state: 'delayed',
      accounts: [
        selectedAccount({
          providerType: 'investment',
          mapping: { enabled: true, ownership: 'shared', classification: 'brokerage' },
        }),
      ],
    },
  ];
  for (const scope of ['household', 'member-1', 'member-2']) {
    assert.equal(hasMissingCurrentBalance(shared, scope), true);
  }
});

test('review notice excludes accounts that cannot enter the displayed USD totals', () => {
  const mapping = selectedAccount().mapping;
  const excluded = [
    selectedAccount({ active: false }),
    selectedAccount({ currency: 'CAD' }),
    selectedAccount({ currency: null }),
    selectedAccount({ providerType: 'credit' }),
    selectedAccount({ mapping: { ...mapping, enabled: false } }),
    selectedAccount({ mapping: { ...mapping, classification: 'unclassified' } }),
  ];
  for (const account of excluded) {
    assert.equal(
      hasMissingCurrentBalance([{ state: 'connected', accounts: [account] }], 'household'),
      false,
    );
  }
  for (const state of ['disconnecting', 'disconnected']) {
    assert.equal(
      hasMissingCurrentBalance([{ state, accounts: [selectedAccount()] }], 'household'),
      false,
    );
  }
  assert.equal(hasMissingCurrentBalance([], 'household'), false);
  assert.equal(
    hasMissingCurrentBalance([{ state: 'connected', accounts: [] }], 'household'),
    false,
  );
});
