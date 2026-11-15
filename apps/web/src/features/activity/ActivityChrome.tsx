import type { ActivityState, ActivitySection } from './activityTypes';
import styles from './Activity.module.css';
import formStyles from '../plan/Editors.module.css';
type Section = ActivitySection;

/**
 * Keeps section choice and reload actions outside the scroll body.
 * @param props Active section and activity request state.
 * @param props.section Active content section.
 * @param props.onSection Explicit section action.
 * @param props.state Activity request owner.
 * @param props.blocked Whether a mutation or permission flow owns writes.
 * @returns Native section buttons and stored-data reload.
 */
export function ActivityControls(props: {
  readonly section: Section;
  readonly onSection: (section: Section) => void;
  readonly state: ActivityState;
  readonly blocked: boolean;
}) {
  const sections: ReadonlyArray<{ id: Section; label: string }> = [
    { id: 'summary', label: 'Summary' },
    { id: 'review', label: `Review (${String(props.state.data?.reviewCount ?? 0)})` },
    { id: 'sources', label: 'Sources' },
  ];
  return (
    <div className={styles.controls}>
      <div className={styles.sections} aria-label="Activity sections">
        {sections.map((section) => (
          <button
            key={section.id}
            aria-pressed={props.section === section.id}
            onClick={() => {
              props.onSection(section.id);
            }}
          >
            {section.label}
          </button>
        ))}
      </div>
      <button
        className={styles.reload}
        disabled={props.state.loading || props.blocked}
        onClick={() => {
          void props.state.reload();
        }}
      >
        {props.state.loading ? 'Loading…' : 'Reload activity'}
      </button>
    </div>
  );
}

/**
 * Retains loaded activity through failures and identifies incomplete and simulated coverage.
 * @param props Current activity data and safe request feedback.
 * @param props.state Activity request state.
 * @returns Failure, loading, coverage, and mutation feedback.
 */
export function ActivityFeedback(props: { readonly state: ActivityState }) {
  const { state } = props;
  return (
    <div className={styles.feedback}>
      {state.error && (
        <p className={formStyles.error} role="alert">
          {state.error} Your unsaved review and settings choices are kept.
        </p>
      )}
      {state.loading && (
        <p className={styles.note} role="status">
          {state.data ? 'Reading saved activity…' : 'Loading monthly activity…'}
        </p>
      )}
      {state.feedback && (
        <p className={styles.note} role="status">
          {state.feedback}
        </p>
      )}
      {state.data && <CoverageFeedback data={state.data} />}
    </div>
  );
}

/**
 * Keeps pagination reachable while full-month totals stay independent of loaded pages.
 * @param props Monthly data and current section.
 * @param props.state Activity request state.
 * @param props.section Visible activity section.
 * @returns Recorded-data cadence and explicit next-page action.
 */
export function ActivityFooter(props: {
  readonly state: ActivityState;
  readonly section: Section;
}) {
  const { data } = props.state;
  return (
    <div className={styles.footer}>
      <p className={styles.note}>
        {props.section === 'review' && data
          ? `${String(data.records.length)} of ${String(data.totalRecordCount)} records. `
          : ''}
        Checks saved activity every 30 minutes.
      </p>
      {props.section === 'review' && data?.nextCursor && (
        <button
          className={formStyles.secondary}
          disabled={props.state.loading || props.state.busy || props.state.totalsStale}
          onClick={() => {
            void props.state.more();
          }}
        >
          Load more activity
        </button>
      )}
    </div>
  );
}

/** Shows available-history limits and simulated-source status.
 * @param props Monthly coverage facts.
 * @param props.data Current monthly response.
 * @returns Coverage and environment notices.
 */
function CoverageFeedback(props: { readonly data: NonNullable<ActivityState['data']> }) {
  const state = { data: props.data };
  return (
    <>
      {state.data.notice && <p className={styles.notice}>{state.data.notice}</p>}
      {state.data.coverage === 'incomplete' && (
        <p className={styles.notice}>
          Recorded amounts cover selected accounts and available history only. Missing history is
          not a confirmed zero.
        </p>
      )}
      {state.data.environment === 'sandbox' && (
        <p className={styles.notice}>
          Plaid Sandbox · Simulated activity. This is not live bank data.
        </p>
      )}
    </>
  );
}
