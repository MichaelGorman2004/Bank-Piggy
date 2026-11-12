import type { Member } from '@bank-piggy/contracts';
import type { PayScheduleDraft } from './payScheduleDraft';
import styles from './Editors.module.css';
import ownStyles from './PayrollFields.module.css';

interface PayScheduleFieldsProps {
  readonly frequency: Member['payFrequency'];
  readonly draft: PayScheduleDraft;
  readonly onChange: (draft: PayScheduleDraft) => void;
}
interface PayDayProps {
  readonly label: string;
  readonly value: string;
  readonly onChange: (value: string) => void;
}

/**
 * Anchors tasks to entered pay dates without guessing a payroll schedule.
 * @param props Pay frequency, editable dates, and their update action.
 * @returns Optional pay-date anchor and twice-monthly day controls.
 */
export function PayScheduleFields(props: PayScheduleFieldsProps) {
  if (props.frequency === 'irregular')
    return (
      <p className={styles.help}>To-do pay periods need a regular pay frequency and a pay date.</p>
    );
  return (
    <div className={styles.section}>
      <label>
        Pay date (optional)
        <input
          type="date"
          min="2000-01-01"
          max="2100-12-31"
          value={props.draft.anchorDate}
          onChange={(event) => {
            props.onChange({ ...props.draft, anchorDate: event.target.value });
          }}
        />
        <small className={styles.help}>
          Use a known payday. This sets the repeating pay periods for your To-do list.
        </small>
      </label>
      <MonthlyPayDays {...props} />
      <p className={styles.help}>
        Dates follow the entered schedule. Weekends and holidays are not adjusted.
      </p>
    </div>
  );
}

/**
 * Distinguishes a fixed monthly date from the last day of each month.
 * @param props Pay frequency and calendar-day draft settings.
 * @returns Monthly calendar-day controls when applicable.
 */
function MonthlyPayDays(props: PayScheduleFieldsProps) {
  if (props.frequency === 'monthly')
    return (
      <label className={ownStyles.toggle}>
        <input
          type="checkbox"
          checked={props.draft.monthEnd}
          onChange={(event) => {
            props.onChange({ ...props.draft, monthEnd: event.target.checked });
          }}
        />
        Payday is the last day of the month
      </label>
    );
  if (props.frequency !== 'semimonthly') return null;
  return (
    <>
      <div className={styles.grid}>
        <PayDay
          label="First monthly pay day"
          value={props.draft.firstDay}
          onChange={(firstDay) => {
            props.onChange({ ...props.draft, firstDay });
          }}
        />
        <PayDay
          label="Second monthly pay day"
          value={props.draft.secondDay}
          onChange={(secondDay) => {
            props.onChange({ ...props.draft, secondDay });
          }}
        />
      </div>
      <p className={styles.help}>
        Short months use the last day for a pay day after month end. Choose a first day before the
        28th so the two dates stay separate.
      </p>
    </>
  );
}

/**
 * Collects a calendar day for a twice-monthly pay schedule.
 * @param props Named day field and its update action.
 * @returns Native bounded number input.
 */
function PayDay(props: PayDayProps) {
  return (
    <label>
      {props.label}
      <input
        type="number"
        min="1"
        max="31"
        step="1"
        required
        value={props.value}
        onChange={(event) => {
          props.onChange(event.target.value);
        }}
      />
    </label>
  );
}
