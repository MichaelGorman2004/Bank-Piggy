import assert from 'node:assert/strict';
import { test } from 'node:test';
import { accountProblem, manualOptions, mappingDraft, settleMappingDraft } from './mappingDraft.ts';

/** Builds an isolated account snapshot with no inferred classification. */
function accountFixture() {
  return {
    id: 'connected-account',
    name: 'Roth high-yield example',
    providerType: 'investment',
    active: true,
    currency: 'USD',
    currentBalanceCents: 10001,
    mappingRevision: 2,
    mapping: {
      enabled: false,
      ownership: 'member-1',
      classification: 'unclassified',
      manualMemberId: null,
      manualAccountId: null,
    },
  };
}

test('mapping drafts do not infer Roth or high-yield type from account names', () => {
  const draft = mappingDraft(accountFixture());
  assert.equal(draft.classification, 'unclassified');
  assert.equal(draft.enabled, false);
  assert.equal(draft.revision, 2);
  assert.equal(Object.hasOwn(draft, 'manualAccountId'), false);
  assert.equal(Object.hasOwn(draft, 'manualMemberId'), false);
});

test('an accepted mapping advances the revision while preserving edits made during its request', async () => {
  const account = accountFixture();
  const submitted = { ...mappingDraft(account), enabled: true, classification: 'brokerage' };
  let complete;
  const response = new Promise((resolve) => {
    complete = resolve;
  });
  const newer = {
    ...submitted,
    ownership: 'shared',
    manualMemberId: 'member-2',
    manualAccountId: 'manual-2',
  };
  const accepted = {
    ...account,
    mappingRevision: 3,
    mapping: { ...account.mapping, ...submitted },
  };
  complete(accepted);
  const saved = await response;
  assert.deepEqual(settleMappingDraft(newer, submitted, saved), { ...newer, revision: 3 });
  assert.deepEqual(settleMappingDraft(submitted, submitted, saved), mappingDraft(saved));
  assert.equal(submitted.revision, 2);
});

test('manual replacement options use both stable IDs even when names and account IDs match', () => {
  const options = manualOptions({
    members: [
      { id: 'member-1', name: 'First', accounts: [{ id: 'same-id', name: 'Same name' }] },
      { id: 'member-2', name: 'Second', accounts: [{ id: 'same-id', name: 'Same name' }] },
    ],
  });
  assert.equal(new Set(options.map((option) => option.value)).size, 2);
  assert.equal(options[1].memberId, 'member-2');
  assert.equal(options[1].accountId, 'same-id');
});

test('unsupported snapshots stay explicit while zero and negative USD balances remain available', () => {
  const account = accountFixture();
  assert.equal(
    accountProblem({ ...account, active: false }),
    'This account is no longer shared by the institution.',
  );
  assert.equal(
    accountProblem({ ...account, providerType: 'credit' }),
    'This account type is shown in Accounts but cannot enter the current household balance totals.',
  );
  assert.match(accountProblem({ ...account, currency: null }), /Only USD/);
  assert.match(accountProblem({ ...account, currency: 'CAD' }), /Only USD/);
  assert.match(
    accountProblem({ ...account, currentBalanceCents: null }),
    /whole cents.*not available/,
  );
  assert.equal(accountProblem({ ...account, currentBalanceCents: 0 }), null);
  assert.equal(accountProblem({ ...account, currentBalanceCents: -1 }), null);
});
