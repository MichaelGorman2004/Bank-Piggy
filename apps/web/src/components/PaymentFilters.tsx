import type { PaymentFilter } from '../features/plan/viewRecords';
import styles from './PaymentFilters.module.css';

interface PaymentFiltersProps {
  readonly value: PaymentFilter;
  readonly onChange: (value: PaymentFilter) => void;
}

/**
 * Filters manual recurring payments without changing their saved records.
 * @param props Current type filter and change action.
 * @returns A native, keyboard-accessible payment filter.
 */
export function PaymentFilters(props: PaymentFiltersProps) {
  return (
    <label className={styles.filter}>
      Payment type
      <select
        value={props.value}
        onChange={(event) => {
          const value = event.target.value;
          if (value === 'all' || value === 'subscription' || value === 'bill' || value === 'loan')
            props.onChange(value);
        }}
      >
        <option value="all">All types</option>
        <option value="subscription">Subscriptions</option>
        <option value="bill">Bills & costs</option>
        <option value="loan">Loans</option>
      </select>
    </label>
  );
}
