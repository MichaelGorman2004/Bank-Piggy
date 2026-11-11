import { TRIP_CATEGORIES, type GoalDraft } from './goalDraft';
import styles from './Editors.module.css';

interface TripFieldsProps {
  readonly draft: GoalDraft;
  readonly onChange: (change: Partial<GoalDraft>) => void;
}

/**
 * Collects the category costs that make up a trip savings target.
 * @param props Current goal draft and field update action.
 * @returns Itemized travel cost controls.
 */
export function TripFields(props: TripFieldsProps) {
  const { draft, onChange } = props;
  return (
    <>
      <div className={styles.grid}>
        {TRIP_CATEGORIES.map(({ category, label }) => (
          <label key={category}>
            {label} ($)
            <input
              required
              inputMode="decimal"
              value={draft.costs[category]}
              onChange={(event) => {
                onChange({ costs: { ...draft.costs, [category]: event.target.value } });
              }}
            />
          </label>
        ))}
      </div>
      <p className={styles.help}>
        Your trip target is the sum of these costs. Save to see the target and suggested monthly
        savings. You choose the actual monthly contribution.
      </p>
    </>
  );
}
