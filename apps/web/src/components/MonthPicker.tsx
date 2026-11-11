import { useId, type RefObject } from 'react';
import { CalendarDays, ChevronDown } from 'lucide-react';
import { MAX_MONTH, MIN_MONTH, monthLabel } from '../features/plan/month';
import { MonthCalendar } from './MonthCalendar';
import { useMonthPopover } from './useMonthPopover';
import styles from './MonthPicker.module.css';

interface MonthPickerProps {
  readonly label: string;
  readonly value: string;
  readonly onChange: (month: string) => void;
  readonly min?: string;
  readonly max?: string;
  readonly required?: boolean;
  readonly disabled?: boolean;
  readonly compact?: boolean;
  readonly className?: string;
}
interface MonthPopoverProps extends MonthPickerProps {
  readonly id: string;
  readonly popoverRef: RefObject<HTMLDivElement | null>;
  readonly open: boolean;
  readonly onOpen: (open: boolean) => void;
  readonly onClose: () => void;
}

/**
 * Keeps direct year edits inside the calendar until a month is selected.
 * @param props Native popover state and the owning field's selection action.
 * @returns A bounded month calendar and optional clear action.
 */
function MonthPopover(props: MonthPopoverProps) {
  const { id, popoverRef, open, onOpen, onClose, ...field } = props;
  /**
   * Updates the draft and restores focus to the opening control.
   * @param month Canonical month or an explicitly cleared optional value.
   */
  function select(month: string) {
    field.onChange(month);
    onClose();
  }
  return (
    <div
      ref={popoverRef}
      id={`${id}-calendar`}
      popover="auto"
      role="dialog"
      aria-label={field.label}
      className={styles.popover}
      style={{ visibility: open ? 'visible' : 'hidden' }}
      onToggle={(event) => {
        onOpen(event.newState === 'open');
      }}
    >
      <MonthCalendar
        key={String(open)}
        id={id}
        value={field.value}
        min={field.min ?? MIN_MONTH}
        max={field.max ?? MAX_MONTH}
        onSelect={select}
        disabled={field.disabled}
      />
      {!field.required && (
        <button
          type="button"
          className={styles.clear}
          disabled={field.disabled}
          onClick={() => {
            select('');
          }}
        >
          Clear month
        </button>
      )}
    </div>
  );
}

/**
 * Selects canonical months in a themed native popover; form owners validate required values.
 * @param props Calendar value, label, bounds, and draft update action.
 * @returns A labeled month trigger and a top-layer calendar.
 */
export function MonthPicker(props: MonthPickerProps) {
  const id = useId();
  const { trigger, popover, open, setOpen, close } = useMonthPopover();
  const label = props.value ? monthLabel(props.value) : 'Choose month';
  return (
    <div
      className={`${styles.field} ${props.compact ? styles.compact : ''} ${props.className ?? ''}`}
    >
      <label htmlFor={id}>{props.label}</label>
      <button
        ref={trigger}
        id={id}
        type="button"
        className={styles.trigger}
        popoverTarget={`${id}-calendar`}
        aria-haspopup="dialog"
        aria-expanded={open}
        aria-label={`${props.label}${props.required ? ', required' : ''}: ${label}`}
        disabled={props.disabled}
      >
        <CalendarDays size={17} aria-hidden="true" />
        <span>{label}</span>
        <ChevronDown size={15} aria-hidden="true" />
      </button>
      <MonthPopover
        {...props}
        id={id}
        popoverRef={popover}
        open={open}
        onOpen={setOpen}
        onClose={close}
      />
    </div>
  );
}
