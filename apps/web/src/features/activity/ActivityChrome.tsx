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
    { id: 'sources', label: 'History settings' },
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
      {(props.state.error || props.state.totalsStale) && (
        <button
          className={styles.reload}
          disabled={props.state.loading || props.blocked}
          onClick={() => {
            void props.state.reload();
          }}
        >
          {props.state.loading ? 'Loading…' : 'Retry activity'}
        </button>
      )}
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
          {state.error} Your unsaved settings and member choices are kept.
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
 * Shows the provider update cadence outside the scroll body.
 * @returns Plain timing guidance for recorded monthly totals.
 */
export function ActivityFooter() {
  return (
    <div className={styles.footer}>
      <p className={styles.note}>
        Activity updates automatically. Institution checks are at least 30 minutes apart.
      </p>
    </div>
  );
}

/** Shows available-history limits and simulated-source status.
 * @param props Monthly coverage facts.
 * @param props.data Current monthly response.
 * @returns Coverage and environment notices.
 */
function CoverageFeedback({ data }: { readonly data: NonNullable<ActivityState['data']> }) {
  const notice =
    data.coverage === 'disabled' ? 'Open History settings to enable monthly totals.' : data.notice;
  return (
    <>
      {notice && <p className={styles.notice}>{notice}</p>}
      {data.coverage === 'incomplete' && (
        <p className={styles.notice}>
          Recorded amounts cover selected accounts and available history only. Missing history is
          not a confirmed zero.
        </p>
      )}
      {data.environment === 'sandbox' && (
        <p className={styles.notice}>
          Plaid Sandbox · Simulated activity. This is not live bank data.
        </p>
      )}
    </>
  );
}
