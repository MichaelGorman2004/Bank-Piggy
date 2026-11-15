import assert from 'node:assert/strict';
import { test } from 'node:test';
import { activityMatchKind, isInvestmentGoal } from './activityMatching.ts';
import {
  activityReviewDraft,
  activityReviewRequest,
  reviewCategoryChange,
  reviewMatchChange,
  staleActivityReview,
  validActivityMatch,
} from './activityReviewDraft.ts';
import { acceptActivityDraft } from './activityDrafts.ts';
import { activityRecord } from './activityTestHelpers.mjs';

test('only spending and investment contributions or withdrawals offer compatible plan matches', () => {
  const bank = activityRecord({ category: 'spending' });
  assert.equal(activityMatchKind(bank, null), 'expense');
  assert.equal(activityMatchKind({ ...bank, isReviewed: true }, null), null);
  assert.equal(activityMatchKind(bank, 'spending'), 'expense');
  for (const category of ['transfer', 'excluded', 'income', 'review'])
    assert.equal(activityMatchKind(bank, category), null);
  const investment = activityRecord({ kind: 'investment', category: 'contribution' });
  assert.equal(activityMatchKind(investment, null), 'goal');
  assert.equal(activityMatchKind(investment, 'withdrawal'), 'goal');
  assert.equal(activityMatchKind(investment, 'trade'), null);
  assert.equal(activityMatchKind(investment, 'spending'), null);
});

test('investment goal choices require an explicit purpose or IRA year, never a matching name', () => {
  assert.equal(isInvestmentGoal({ name: 'Investing' }), false);
  assert.equal(isInvestmentGoal({ purpose: 'investment' }), true);
  assert.equal(isInvestmentGoal({ contributionYear: 2026 }), true);
  assert.equal(isInvestmentGoal({ purpose: 'investment', trip: {} }), false);
  assert.equal(isInvestmentGoal({ contributionYear: 2026, purchase: {} }), false);
});

test('first edits capture both revisions and retain unavailable match IDs until a user decision', () => {
  const record = activityRecord({ expenseId: 'deleted-expense', category: 'spending' });
  const draft = activityReviewDraft(record, 7);
  assert.equal(draft.expectedRevision, 1);
  assert.equal(draft.expectedPlanRevision, 7);
  assert.equal(draft.expenseId, 'deleted-expense');
  assert.equal(staleActivityReview(draft, record, 7), false);
  assert.equal(staleActivityReview(draft, record, 8), true);
  assert.equal(staleActivityReview(draft, { ...record, revision: 2 }, 7), true);
  const cleared = { ...draft, expenseId: '' };
  assert.deepEqual(activityReviewRequest(cleared), {
    expectedRevision: 1,
    expectedPlanRevision: 7,
    category: null,
    transferPeerId: null,
    expectedPeerRevision: null,
    expenseId: null,
    goalId: null,
  });
});

test('classification changes remove incompatible matches and never pair a transfer with a plan target', () => {
  const record = activityRecord({ category: 'spending' });
  const draft = {
    ...activityReviewDraft(record, 3),
    expenseId: 'rent',
    goalId: 'invest',
    peerId: 'peer',
    peerRevision: 2,
  };
  const transfer = { ...draft, ...reviewCategoryChange(record, 'transfer') };
  assert.equal(transfer.expenseId, '');
  assert.equal(transfer.goalId, '');
  assert.equal(transfer.peerId, '');
  const automatic = { ...draft, ...reviewCategoryChange(record, null) };
  assert.equal(automatic.expenseId, 'rent');
  assert.equal(automatic.goalId, '');
  assert.equal(automatic.peerRevision, null);
  const restored = { ...draft, ...reviewCategoryChange({ ...record, isReviewed: true }, null) };
  assert.equal(restored.expenseId, '');
  assert.equal(restored.goalId, '');
});

test('matching automatic activity submits an explicit compatible reviewed classification', () => {
  for (const [kind, category, targetField] of [
    ['bank', 'spending', 'expenseId'],
    ['investment', 'contribution', 'goalId'],
    ['investment', 'withdrawal', 'goalId'],
  ]) {
    const record = activityRecord({ kind, category });
    const draft = activityReviewDraft(record, 3);
    const selected = { ...draft, ...reviewMatchChange(record, draft, 'target') };
    const request = activityReviewRequest(selected);
    assert.equal(request.category, category);
    assert.equal(request[targetField], 'target');
    assert.equal(request.transferPeerId, null);
    assert.equal(request.expectedPlanRevision, 3);
  }
});

test('accepted matches retain newer edits and the captured plan revision while advancing the record revision', () => {
  const submitted = { ...activityReviewDraft(activityRecord(), 4), edit: 1, expenseId: 'rent' };
  const newer = { ...submitted, edit: 2, expenseId: 'food' };
  const accepted = acceptActivityDraft(newer, submitted, 2);
  assert.equal(accepted.expenseId, 'food');
  assert.equal(accepted.expectedRevision, 2);
  assert.equal(accepted.expectedPlanRevision, 4);
  assert.equal(staleActivityReview(accepted, activityRecord({ revision: 2 }), 5), true);
  assert.equal(acceptActivityDraft(submitted, submitted, 2), null);
});

test('missing and retyped targets block saved matches until explicitly replaced or removed', () => {
  const record = activityRecord({ category: 'spending', expenseId: 'deleted' });
  const draft = activityReviewDraft(record, 3);
  const household = {
    expenses: [{ id: 'rent' }],
    goals: [{ id: 'invest', purpose: 'investment' }, { id: 'generic' }],
  };
  assert.equal(validActivityMatch(draft, record, household), false);
  assert.equal(validActivityMatch({ ...draft, expenseId: '' }, record, household), true);
  assert.equal(validActivityMatch({ ...draft, expenseId: 'rent' }, record, household), true);
  assert.equal(
    validActivityMatch({ ...draft, expenseId: 'rent', category: 'transfer' }, record, household),
    false,
  );
  const investment = activityRecord({ kind: 'investment', category: 'contribution' });
  const matched = { ...activityReviewDraft(investment, 3), goalId: 'invest' };
  assert.equal(validActivityMatch(matched, investment, household), true);
  assert.equal(validActivityMatch({ ...matched, goalId: 'generic' }, investment, household), false);
  assert.equal(validActivityMatch({ ...matched, expenseId: 'rent' }, investment, household), false);
});
