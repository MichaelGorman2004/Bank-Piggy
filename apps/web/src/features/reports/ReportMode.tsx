import type { ReportMode } from './reporting';
import styles from './Reports.module.css';

/** Chooses the source of reported amounts without editing a saved plan.
 * @param props Current report mode and the explicit selection action.
 * @param props.mode Planned forecast or recorded monthly activity.
 * @param props.onChange Report selection action.
 * @returns Native buttons with persistent keyboard focus and selected state.
 */
export function ReportModeControl(props: {
  readonly mode: ReportMode;
  readonly onChange: (mode: ReportMode) => void;
}) {
  return (
    <div className={styles.mode} role="group" aria-label="Report mode">
      <button
        type="button"
        aria-pressed={props.mode === 'planned'}
        onClick={() => {
          props.onChange('planned');
        }}
      >
        Planned
      </button>
      <button
        type="button"
        aria-pressed={props.mode === 'actual'}
        onClick={() => {
          props.onChange('actual');
        }}
      >
        Actual
      </button>
    </div>
  );
}
