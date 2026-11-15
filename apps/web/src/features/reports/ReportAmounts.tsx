import { money } from '../plan/format';
import styles from './Reports.module.css';

export interface ReportAmount {
  readonly label: string;
  readonly cents: number;
  readonly available: boolean;
}

/** Presents canonical amounts without replacing unavailable history with zero.
 * @param props Named server-calculated amounts and their availability.
 * @param props.items Parallel financial facts for this comparison.
 * @returns An accessible description list with exact currency values.
 */
export function ReportAmounts(props: { readonly items: ReadonlyArray<ReportAmount> }) {
  return (
    <dl className={styles.amounts}>
      {props.items.map((item) => (
        <div key={item.label}>
          <dt>{item.label}</dt>
          <dd className={item.available && item.cents < 0 ? styles.negative : undefined}>
            {item.available ? money(item.cents, 1, true) : 'Unavailable'}
          </dd>
        </div>
      ))}
    </dl>
  );
}
