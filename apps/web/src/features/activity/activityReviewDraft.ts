import type {
  ActivityCategory,
  ActivityRecord,
  ActivityReviewRequest,
  Household,
} from '@bank-piggy/contracts';
import { activityMatchKind, isInvestmentGoal } from './activityMatching.ts';

export interface ReviewDraft {
  readonly expectedRevision: number;
  readonly expectedPlanRevision: number;
  readonly category: ActivityCategory | null;
  readonly peerId: string;
  readonly peerRevision: number | null;
  readonly expenseId: string;
  readonly goalId: string;
  readonly edit: number;
}

/** Captures record and plan revisions together when the first field is edited.
 * @param record Current saved activity and transfer evidence.
 * @param planRevision Current saved household plan revision.
 * @returns Editable values without guessing a plan match.
 */
export function activityReviewDraft(record: ActivityRecord, planRevision: number): ReviewDraft {
  return {
    expectedRevision: record.revision,
    expectedPlanRevision: planRevision,
    category: record.isReviewed ? record.category : null,
    peerId: record.transferPeerId ?? '',
    peerRevision:
      record.transferCandidates.find((candidate) => candidate.recordId === record.transferPeerId)
        ?.revision ?? null,
    expenseId: record.expenseId ?? '',
    goalId: record.goalId ?? '',
    edit: 0,
  };
}

/** Clears incompatible saved matches when the user changes classification.
 * @param record Current record kind and automatic classification.
 * @param category Chosen manual classification, or null for automatic rules.
 * @returns Classification changes with explicit transfer and match cleanup.
 */
export function reviewCategoryChange(
  record: ActivityRecord,
  category: ActivityCategory | null,
): Partial<ReviewDraft> {
  const kind = activityMatchKind(record, category);
  return {
    category,
    peerId: '',
    peerRevision: null,
    ...(kind !== 'expense' ? { expenseId: '' } : {}),
    ...(kind !== 'goal' ? { goalId: '' } : {}),
  };
}

/** Makes an automatic eligible classification explicit when a user chooses a plan target.
 * @param record Current provider kind and saved automatic classification.
 * @param draft Retained classification and match choices.
 * @param id Available target chosen by the user, or empty to remove the match.
 * @returns One compatible match with explicit classification and no transfer partner.
 */
export function reviewMatchChange(
  record: ActivityRecord,
  draft: ReviewDraft,
  id: string,
): Partial<ReviewDraft> {
  const kind = activityMatchKind(record, draft.category);
  if (!kind) return {};
  return {
    category: id ? (draft.category ?? record.category) : draft.category,
    expenseId: kind === 'expense' ? id : '',
    goalId: kind === 'goal' ? id : '',
    peerId: '',
    peerRevision: null,
  };
}

/** Serializes deliberate match choices and both captured revision guards.
 * @param draft Exact draft submitted by the user.
 * @returns Canonical review request with explicit null values for removed matches.
 */
export function activityReviewRequest(draft: ReviewDraft): ActivityReviewRequest {
  return {
    expectedRevision: draft.expectedRevision,
    expectedPlanRevision: draft.expectedPlanRevision,
    category: draft.category,
    transferPeerId: draft.peerId || null,
    expectedPeerRevision: draft.peerRevision,
    expenseId: draft.expenseId || null,
    goalId: draft.goalId || null,
  };
}

/** Rejects a draft when either its activity facts or plan targets changed.
 * @param draft Draft with revisions captured at the first edit.
 * @param record Current authoritative record.
 * @param planRevision Current saved household plan revision.
 * @returns Whether the user must reset and review current facts before saving.
 */
export function staleActivityReview(
  draft: ReviewDraft,
  record: ActivityRecord,
  planRevision: number,
): boolean {
  return draft.expectedRevision !== record.revision || draft.expectedPlanRevision !== planRevision;
}

/** Requires removed or retyped targets to be replaced or explicitly unmatched before saving.
 * @param draft Retained user match choices.
 * @param record Provider kind and current automatic classification.
 * @param household Current saved plan choices.
 * @returns Whether selected targets exist and suit this classification.
 */
export function validActivityMatch(
  draft: ReviewDraft,
  record: ActivityRecord,
  household: Pick<Household, 'expenses' | 'goals'>,
): boolean {
  if (draft.expenseId && draft.goalId) return false;
  const kind = activityMatchKind(record, draft.category);
  if (draft.expenseId)
    return kind === 'expense' && household.expenses.some((target) => target.id === draft.expenseId);
  if (draft.goalId)
    return (
      kind === 'goal' &&
      household.goals.some((target) => target.id === draft.goalId && isInvestmentGoal(target))
    );
  return true;
}
