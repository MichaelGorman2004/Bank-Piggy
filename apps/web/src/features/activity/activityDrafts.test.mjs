import assert from 'node:assert/strict';
import { test } from 'node:test';
import { acceptActivityDraft } from './activityDrafts.ts';

test('accepted source saves retain newer account edits and advance only the submitted revision', () => {
  const submitted = {
    expectedRevision: 4,
    edit: 1,
    enabled: true,
    accounts: [{ accountId: 'bank-account', enabled: true, ownership: 'member-1' }],
  };
  const editedDuringSave = {
    ...submitted,
    edit: 2,
    accounts: [{ ...submitted.accounts[0], ownership: 'shared' }],
  };
  const accepted = acceptActivityDraft(editedDuringSave, submitted, 5);
  assert.equal(accepted.expectedRevision, 5);
  assert.equal(accepted.edit, 2);
  assert.equal(accepted.accounts[0].ownership, 'shared');
  assert.equal(acceptActivityDraft(submitted, submitted, 5), null);
});

test('accepted source saves preserve newer consent edits without changing a newer revision owner', () => {
  const submitted = {
    expectedRevision: 2,
    edit: 1,
    enabled: true,
    accounts: [{ accountId: 'bank-account', enabled: true, ownership: 'member-1' }],
  };
  const editedDuringSave = {
    ...submitted,
    edit: 2,
    enabled: false,
    accounts: [{ accountId: 'bank-account', enabled: false, ownership: 'shared' }],
  };
  assert.deepEqual(acceptActivityDraft(editedDuringSave, submitted, 3), {
    ...editedDuringSave,
    expectedRevision: 3,
  });
  const newOwner = { ...editedDuringSave, expectedRevision: 9 };
  assert.equal(acceptActivityDraft(newOwner, submitted, 3), newOwner);
  assert.equal(acceptActivityDraft(null, submitted, 3), null);
});
