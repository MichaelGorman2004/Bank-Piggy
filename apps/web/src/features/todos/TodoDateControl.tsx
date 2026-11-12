import { localDate, validDate } from '../plan/payScheduleDraft';
import styles from './Todos.module.css';

interface TodoDateControlProps {
  readonly date: string;
  readonly onDate: (date: string) => void;
}

/**
 * Makes earlier pay periods reachable without changing the cash-flow planning month.
 * @param props Selected local date and its update action.
 * @returns Native date control and current-date shortcut.
 */
export function TodoDateControl(props: TodoDateControlProps) {
  return (
    <div className={styles.dateControl}>
      <label>
        As-of date
        <input
          type="date"
          required
          min="2000-01-01"
          max="2100-12-31"
          value={props.date}
          onChange={(event) => {
            if (validDate(event.target.value)) props.onDate(event.target.value);
          }}
        />
      </label>
      <button
        type="button"
        onClick={() => {
          props.onDate(localDate(new Date()));
        }}
      >
        Today
      </button>
      <p>Each person’s pay period follows their saved pay dates.</p>
    </div>
  );
}
