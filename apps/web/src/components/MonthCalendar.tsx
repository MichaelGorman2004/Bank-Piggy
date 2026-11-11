import { useState, type KeyboardEvent } from 'react';
import { ChevronLeft, ChevronRight } from 'lucide-react';
import { validMonth } from '../features/plan/month';
import styles from './MonthPicker.module.css';

interface MonthCalendarProps {
  readonly id: string;
  readonly value: string;
  readonly min: string;
  readonly max: string;
  readonly disabled?: boolean;
  readonly onSelect: (month: string) => void;
}
interface YearControlsProps extends MonthCalendarProps {
  readonly year: string;
  readonly onYear: (year: string) => void;
}
const MONTH_NAMES = [
  'January',
  'February',
  'March',
  'April',
  'May',
  'June',
  'July',
  'August',
  'September',
  'October',
  'November',
  'December',
];
const MONTH_COLUMNS = 3;

/**
 * Moves between enabled month buttons without submitting an enclosing form.
 * @param event Keyboard event from the month group.
 */
function moveMonthFocus(event: KeyboardEvent<HTMLDivElement>) {
  if (!(event.target instanceof HTMLButtonElement)) return;
  const buttons = Array.from(
    event.currentTarget.querySelectorAll<HTMLButtonElement>('button:not(:disabled)'),
  );
  const current = buttons.indexOf(event.target);
  const movement: Readonly<Partial<Record<string, number>>> = {
    ArrowLeft: -1,
    ArrowRight: 1,
    ArrowUp: -MONTH_COLUMNS,
    ArrowDown: MONTH_COLUMNS,
    Home: -buttons.length,
    End: buttons.length,
  };
  const offset = movement[event.key];
  if (offset === undefined) return;
  event.preventDefault();
  buttons[Math.max(0, Math.min(current + offset, buttons.length - 1))]?.focus();
}

/**
 * Allows direct year entry while keeping incomplete text separate from the selected month.
 * @param props Calendar range, displayed year, and year update action.
 * @returns Themed year input and previous/next year buttons.
 */
function YearControls(props: YearControlsProps) {
  const year = Number(props.year);
  return (
    <div className={styles.yearControls}>
      <button
        type="button"
        aria-label="Previous year"
        disabled={props.disabled || year <= Number(props.min.slice(0, 4))}
        onClick={() => {
          props.onYear(String(year - 1));
        }}
      >
        <ChevronLeft size={19} />
      </button>
      <label htmlFor={`${props.id}-year`}>
        Year
        <input
          id={`${props.id}-year`}
          inputMode="numeric"
          maxLength={4}
          value={props.year}
          disabled={props.disabled}
          autoComplete="off"
          onChange={(event) => {
            if (/^\d*$/.test(event.target.value)) props.onYear(event.target.value);
          }}
          onKeyDown={(event) => {
            if (event.key === 'Enter') event.preventDefault();
          }}
        />
      </label>
      <button
        type="button"
        aria-label="Next year"
        disabled={props.disabled || year >= Number(props.max.slice(0, 4))}
        onClick={() => {
          props.onYear(String(year + 1));
        }}
      >
        <ChevronRight size={19} />
      </button>
    </div>
  );
}

/**
 * Starts at the chosen year, or the current year within the allowed range.
 * @param props Selected month and inclusive calendar bounds.
 * @returns A year suitable for direct editing.
 */
function initialYear(props: MonthCalendarProps): string {
  const current = validMonth(props.value) ? props.value : new Date().toISOString().slice(0, 7);
  const bounded = current < props.min ? props.min : current > props.max ? props.max : current;
  return String(Number(bounded.slice(0, 4)));
}

/**
 * Renders the twelve calendar months with canonical validation and inclusive bounds.
 * @param props Current month, displayed year, allowed range, and selection action.
 * @returns Keyboard-accessible year and month controls.
 */
export function MonthCalendar(props: MonthCalendarProps) {
  const [yearText, setYear] = useState(() => initialYear(props));
  const year = yearText.padStart(4, '0');
  return (
    <>
      <YearControls {...props} year={yearText} onYear={setYear} />
      <div className={styles.months} role="group" aria-label="Months" onKeyDown={moveMonthFocus}>
        {MONTH_NAMES.map((name, index) => {
          const month = `${year}-${String(index + 1).padStart(2, '0')}`;
          return (
            <button
              key={name}
              type="button"
              data-month={month}
              aria-label={`${name} ${year}`}
              aria-pressed={props.value === month}
              disabled={
                props.disabled || !validMonth(month) || month < props.min || month > props.max
              }
              onClick={() => {
                props.onSelect(month);
              }}
            >
              {name.slice(0, 3)}
            </button>
          );
        })}
      </div>
    </>
  );
}
