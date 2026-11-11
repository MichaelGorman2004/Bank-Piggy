import assert from 'node:assert/strict';
import { test } from 'node:test';
import {
  ACCOUNT_TYPES,
  MAX_ACCOUNT_BALANCE_CENTS,
  MAX_ACCOUNT_NAME_LENGTH,
  MAX_ACCOUNTS,
} from '../accounts/accountTypes.ts';
import { accountDraft, convertAccounts } from './accountDraft.ts';
import { convertPlan } from './planDraft.ts';
import { initialPlan, planReducer } from './planReducer.ts';

/** Creates one canonical account with exact cents. */
function accountFixture(update = {}) {
  return {
    id: 'retirement_1',
    name: 'Work retirement',
    type: 'roth_401k',
    balanceCents: 12345,
    ...update,
  };
}

/** Creates a complete plan for draft edit and retention checks. */
function householdFixture() {
  return {
    revision: 3,
    members: ['member-1', 'member-2'].map((id) => ({
      id,
      name: id,
      annualBaseSalaryCents: 5000000,
      monthlyTakeHomeCents: 300000,
      payFrequency: 'monthly',
      accounts: [accountFixture()],
      debtCents: 0,
    })),
    defaultSplitMethod: 'equal',
    expenses: [],
    goals: [],
  };
}

test('every account type round trips with zero and exact-cent balances', () => {
  for (const { value: type } of ACCOUNT_TYPES) {
    for (const balanceCents of [0, 12345, MAX_ACCOUNT_BALANCE_CENTS]) {
      const account = accountFixture({ type, balanceCents });
      const draft = accountDraft(account);
      const original = structuredClone(draft);
      assert.deepEqual(convertAccounts([draft]), { accounts: [account] });
      assert.deepEqual(draft, original);
    }
  }
});

test('invalid balances do not coerce or change incomplete money text', () => {
  for (const balance of ['', '.', '-1', '1.001', '1e3', '1000000000.01', '9007199254740992']) {
    const draft = { ...accountDraft(accountFixture()), balance };
    const original = structuredClone(draft);
    assert.equal(convertAccounts([draft]).accounts, undefined, balance);
    assert.deepEqual(draft, original);
  }
});

test('account identity, type, name, and duplicate IDs are validated', () => {
  const draft = accountDraft(accountFixture());
  for (const update of [
    { id: '' },
    { id: 'not an ID' },
    { id: 'a'.repeat(65) },
    { name: ' ' },
    { name: 'a'.repeat(MAX_ACCOUNT_NAME_LENGTH + 1) },
    { type: 'checking' },
  ]) {
    assert.equal(convertAccounts([{ ...draft, ...update }]).accounts, undefined);
  }
  assert.equal(convertAccounts([draft, { ...draft, name: 'Another name' }]).accounts, undefined);
  const boundary = {
    ...draft,
    id: 'a'.repeat(64),
    name: ` ${'n'.repeat(MAX_ACCOUNT_NAME_LENGTH)} `,
  };
  assert.equal(convertAccounts([boundary]).accounts[0].name.length, MAX_ACCOUNT_NAME_LENGTH);
});

test('account counts and savings or investment aggregate caps hold at their boundaries', () => {
  const draft = accountDraft(accountFixture({ balanceCents: 0 }));
  const accounts = Array.from({ length: MAX_ACCOUNTS }, (_, index) => ({
    ...draft,
    id: `account-${String(index)}`,
  }));
  assert.equal(convertAccounts(accounts).accounts.length, MAX_ACCOUNTS);
  assert.equal(convertAccounts([...accounts, { ...draft, id: 'too-many' }]).accounts, undefined);
  for (const types of [
    ['savings', 'high_yield_savings'],
    ['roth_ira', 'brokerage'],
  ]) {
    const atLimit = { ...draft, id: 'limit', type: types[0], balance: '1000000000.00' };
    const remainder = { ...draft, id: 'extra', type: types[1], balance: '0.00' };
    assert.equal(convertAccounts([atLimit, remainder]).accounts.length, 2);
    assert.equal(convertAccounts([atLimit, { ...remainder, balance: '0.01' }]).accounts, undefined);
  }
});

test('queued account edits preserve incomplete text and all newer member edits', () => {
  const household = householdFixture();
  const initial = initialPlan(household);
  let draft = planReducer(initial, { type: 'addAccount', memberIndex: 0, id: 'new-savings' });
  draft = planReducer(draft, {
    type: 'account',
    memberIndex: 0,
    id: 'new-savings',
    update: { name: 'Buffer', balance: '1.' },
  });
  draft = planReducer(draft, { type: 'member', index: 0, update: { name: 'New name' } });
  draft = planReducer(draft, {
    type: 'account',
    memberIndex: 0,
    id: 'new-savings',
    update: { type: 'high_yield_savings' },
  });
  draft = planReducer(draft, { type: 'removeAccount', memberIndex: 0, id: 'retirement_1' });
  assert.deepEqual(draft.members[0].accounts, [
    { id: 'new-savings', name: 'Buffer', type: 'high_yield_savings', balance: '1.' },
  ]);
  assert.equal(draft.members[0].name, 'New name');
  assert.deepEqual(draft.members[1], initial.members[1]);
  assert.equal(initial.members[0].accounts.length, 1);
  const result = convertPlan(household, draft);
  assert.equal(result.household.members[0].accounts[0].balanceCents, 100);
  assert.equal(draft.members[0].accounts[0].balance, '1.');
});

test('account inserts stop at the limit and duplicate IDs do not replace saved rows', () => {
  let state = initialPlan(householdFixture());
  const first = state.members[0].accounts[0];
  state = planReducer(state, { type: 'addAccount', memberIndex: 0, id: first.id });
  assert.deepEqual(state.members[0].accounts, [first]);
  for (let index = 1; index <= MAX_ACCOUNTS; index += 1) {
    state = planReducer(state, { type: 'addAccount', memberIndex: 0, id: `new-${String(index)}` });
  }
  assert.equal(state.members[0].accounts.length, MAX_ACCOUNTS);
  assert.deepEqual(state.members[0].accounts[0], first);
});

test('account names use the canonical Unicode character limit', () => {
  const account = { id: 'unicode', name: '🐷'.repeat(100), type: 'savings', balance: '0' };
  assert.equal(convertAccounts([account]).error, undefined);
  assert.ok(convertAccounts([{ ...account, name: '🐷'.repeat(101) }]).error);
});
