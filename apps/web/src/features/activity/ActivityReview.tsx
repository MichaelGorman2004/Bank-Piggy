import { useState } from 'react';
import type { ActivityRecord, ActivityReviewRequest, Household } from '@bank-piggy/contracts';
import { acceptActivityDraft } from './activityDrafts';
import { ActivityMatchField, ActivityMatchFacts } from './ActivityMatchField';
import {
  activityReviewDraft,
  activityReviewRequest,
  reviewCategoryChange,
  staleActivityReview,
  validActivityMatch,
  type ReviewDraft,
} from './activityReviewDraft';
import { money } from '../plan/format';
import {
  ACTIVITY_CATEGORIES,
  activityAmount,
  activityDate,
  canReviewActivity,
  fullActivityAmount,
  reviewCategories,
} from './presentation';
import styles from './Activity.module.css';
import formStyles from '../plan/Editors.module.css';

interface ReviewProps {
  readonly record: ActivityRecord;
  readonly household: Household;
  readonly planRevision: number;
  readonly busy: boolean;
  readonly onReview: (
    record: ActivityRecord,
    input: ActivityReviewRequest,
  ) => Promise<ActivityRecord | null>;
}

/**
 * Keeps each review draft with its original record revision until explicitly saved or reset.
 * @param props Recorded activity and serialized review action.
 * @returns Expandable record facts and a deliberate classification form.
 */
export function ActivityReviewRow(props: ReviewProps) {
  const { record } = props;
  return (
    <details className={styles.record}>
      <summary>
        <span>
          <strong>{record.name || 'Activity record'}</strong>
          <small>
            {activityDate(record.date)} · {record.accountName} ·{' '}
            {ACTIVITY_CATEGORIES[record.category]}
          </small>
        </span>
        <strong>{activityAmount(record)}</strong>
      </summary>
      <div className={styles.recordBody}>
        <RecordFacts {...props} />
        {record.counterparty && <p className={styles.note}>{record.counterparty}</p>}
        {record.reviewReason && <p className={styles.notice}>{record.reviewReason}</p>}
        {record.transferPeerId && (
          <p className={styles.note}>
            Paired as a transfer. Choose a partner again to change this review.
          </p>
        )}
        {canReviewActivity(record) ? (
          <ReviewForm {...props} />
        ) : (
          <p className={styles.note}>
            {record.pending
              ? 'Pending activity is excluded until posted.'
              : record.cancelled
                ? 'Cancelled activity is excluded.'
                : 'This currency or amount is not supported for totals.'}
          </p>
        )}
      </div>
    </details>
  );
}

/** Labels the full provider amount separately from the selected view's allocated amount.
 * @param props Recorded facts and household member labels.
 * @returns Provider evidence and retained match details.
 */
function RecordFacts(props: ReviewProps) {
  const owner =
    props.record.ownership === 'shared'
      ? 'Shared account'
      : (props.household.members.find((member) => member.id === props.record.ownership)?.name ??
        'Unavailable member');
  return (
    <>
      <p className={styles.note}>
        Full transaction: {fullActivityAmount(props.record)} · Account owner: {owner}.
      </p>
      {props.record.scopeAmountCents === null && canReviewActivity(props.record) && (
        <p className={styles.note}>
          The amount for this view is unavailable. Reload activity to calculate the current share.
        </p>
      )}
      <ActivityMatchFacts record={props.record} household={props.household} />
    </>
  );
}

/**
 * Preserves newer edits during a save and requires a fresh decision after a stale revision.
 * @param props Record revision, current request state, and explicit save action.
 * @returns Native classification and transfer-partner controls.
 */
function ReviewForm(props: ReviewProps) {
  const { draft, value, change, save, reset } = useReviewDraft(props);
  const stale = staleActivityReview(value, props.record, props.planRevision);
  const validMatch = validActivityMatch(value, props.record, props.household);
  return (
    <form
      className={styles.form}
      onSubmit={(event) => {
        event.preventDefault();
        void save();
      }}
    >
      <ReviewFields record={props.record} value={value} onChange={change} />
      <ActivityMatchField
        record={props.record}
        value={value}
        household={props.household}
        onChange={change}
      />
      {draft && stale && (
        <p className={styles.notice}>
          This record or household plan changed. Your draft is kept. Reset the draft to review the
          latest facts and plan targets.
        </p>
      )}
      <div className={styles.actions}>
        <button
          className={formStyles.button}
          disabled={props.busy || !draft || stale || !validMatch}
        >
          Save review
        </button>
        <button
          className={formStyles.secondary}
          type="button"
          disabled={!draft || props.busy}
          onClick={() => {
            reset();
          }}
        >
          Reset draft
        </button>
      </div>
    </form>
  );
}

/**
 * Limits category choices to relevant record types and uses explicit transfer evidence.
 * @param props Record facts, current draft, and field-change action.
 * @param props.record Provider record with transfer candidates.
 * @param props.value Current review draft.
 * @param props.onChange Draft field change.
 * @returns Native review selectors.
 */
function ReviewFields(props: {
  readonly record: ActivityRecord;
  readonly value: ReviewDraft;
  readonly onChange: (patch: Partial<ReviewDraft>) => void;
}) {
  return (
    <>
      <label>
        Classification
        <select
          value={props.value.category ?? 'automatic'}
          onChange={(event) => {
            const category =
              reviewCategories(props.record).find((item) => item === event.target.value) ?? null;
            props.onChange(reviewCategoryChange(props.record, category));
          }}
        >
          <option value="automatic">Automatic classification</option>
          {reviewCategories(props.record).map((category) => (
            <option key={category} value={category}>
              {ACTIVITY_CATEGORIES[category]}
            </option>
          ))}
        </select>
      </label>
      {props.value.category === 'transfer' && <TransferField {...props} />}
      <p className={styles.note}>
        Transfers do not count as income or spending. Automatic classification restores the provider
        review rules.
      </p>
      {props.record.transferPeerId && (
        <p className={styles.note}>Saving without a transfer partner removes the saved pair.</p>
      )}
    </>
  );
}

/** Retains both record revisions and newer edits across review saves.
 * @param props Current record and explicit review action.
 * @returns Original-revision draft and deliberate save and reset actions.
 */
function useReviewDraft(props: ReviewProps) {
  const [draft, setDraft] = useState<ReviewDraft | null>(null);
  const initial = activityReviewDraft(props.record, props.planRevision);
  const value = draft ?? initial;
  /** Applies a field change against the revision at the start of editing.
   * @param patch Explicit review change.
   */
  function change(patch: Partial<ReviewDraft>) {
    setDraft((previous) => ({
      ...(previous ?? initial),
      ...patch,
      edit: (previous?.edit ?? 0) + 1,
    }));
  }
  /** Saves this exact draft and retains any newer field changes. */
  async function save() {
    if (
      !draft ||
      props.busy ||
      staleActivityReview(value, props.record, props.planRevision) ||
      !validActivityMatch(value, props.record, props.household)
    )
      return;
    const saved = await props.onReview(props.record, activityReviewRequest(value));
    if (saved) setDraft((current) => acceptActivityDraft(current, value, saved.revision));
  }
  return {
    draft,
    value,
    change,
    save,
    reset: () => {
      setDraft(null);
    },
  };
}

/** Selects a transfer partner and captures that partner revision at the time of choice.
 * @param props Current record, draft, and field change action.
 * @returns Native partner selector with recorded evidence.
 */
function TransferField(props: Parameters<typeof ReviewFields>[0]) {
  return (
    <label>
      Transfer partner
      <select
        value={props.value.peerId}
        onChange={(event) => {
          const peer = props.record.transferCandidates.find(
            (candidate) => candidate.recordId === event.target.value,
          );
          props.onChange({ peerId: peer?.recordId ?? '', peerRevision: peer?.revision ?? null });
        }}
      >
        <option value="">No paired record</option>
        {props.record.transferCandidates.map((candidate) => (
          <option key={candidate.recordId} value={candidate.recordId}>
            {activityDate(candidate.date)} · {candidate.accountName} ·{' '}
            {money(candidate.amountCents, 1, true)}
          </option>
        ))}
      </select>
    </label>
  );
}
