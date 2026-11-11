import { useEffect, useRef, useState } from 'react';
import { Info } from 'lucide-react';
import { MAX_PLANNING_MONTH, monthLabel, termEndMonth, validMonth } from '../features/plan/month';
import type { Period } from '../features/plan/format';
import { MonthPicker } from './MonthPicker';
import styles from './PlanningMonth.module.css';

interface PlanningMonthProps {
  readonly month: string;
  readonly period: Period;
  readonly disabled: boolean;
  readonly loading: boolean;
  readonly onMonth: (month: string) => Promise<boolean>;
}

/**
 * Applies valid selections and restores failed reads without overwriting newer input.
 * @param props Accepted month and the callback that reports whether the next read succeeded.
 * @returns The editable selection and a month-change handler.
 */
function useMonthSelection(props: PlanningMonthProps) {
  const generation = useRef(0);
  const [selection, setSelection] = useState<string | null>(null);
  useEffect(
    () => () => {
      generation.current += 1;
    },
    [],
  );
  /**
   * Awaits the selected projection and restores a failed latest choice to the accepted month.
   * @param month Canonical calendar selection.
   */
  async function change(month: string) {
    const version = ++generation.current;
    if (!validMonth(month) || month > MAX_PLANNING_MONTH) return;
    setSelection(month);
    await props.onMonth(month);
    if (version === generation.current) setSelection(null);
  }
  return { value: selection ?? props.month, change };
}

/**
 * Keeps forecast assumptions available without a permanent chart caption.
 * @param props Accepted month and reporting period.
 * @returns A native disclosure with the exact projection range.
 */
function ProjectionDetails(props: Pick<PlanningMonthProps, 'month' | 'period'>) {
  const last = termEndMonth(props.month, '12') ?? props.month;
  const description =
    props.period === 'yearly'
      ? `12-month projection: ${monthLabel(props.month)} through ${monthLabel(last)}. Payment start and end dates are included. Balances stay unchanged.`
      : `Planning ${monthLabel(props.month)}. Only payments active in this month count toward totals.`;
  return (
    <details className={styles.details}>
      <summary aria-label="Projection details">
        <Info size={16} />
      </summary>
      <p>{description}</p>
    </details>
  );
}

/**
 * Applies a valid month immediately while keeping the accepted plan visible during the read.
 * @param props Accepted calendar month, read progress, and authorized projection action.
 * @returns A themed month selector with no separate submit step.
 */
export function PlanningMonth(props: PlanningMonthProps) {
  const { value, change } = useMonthSelection(props);
  return (
    <div className={styles.bar} aria-busy={props.loading}>
      <MonthPicker
        label="Planning month"
        value={value}
        required
        compact
        max={MAX_PLANNING_MONTH}
        disabled={props.disabled}
        onChange={(month) => {
          void change(month);
        }}
      />
      <span className={styles.labelText} role="status">
        {props.loading ? 'Updating month…' : `Planning ${monthLabel(props.month)}`}
      </span>
      <ProjectionDetails month={props.month} period={props.period} />
    </div>
  );
}
