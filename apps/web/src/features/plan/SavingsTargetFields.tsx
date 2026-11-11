import type { GoalDraft } from './goalDraft';
import styles from './Editors.module.css';

interface SavingsTargetFieldsProps {
  readonly draft: GoalDraft;
  readonly onChange: (change: Partial<GoalDraft>) => void;
}

/**
 * Lets ordinary goals track manual savings without inventing contributions or changing balances.
 * @param props Current target fields and draft update action.
 * @returns Optional saved and target amount controls with their accounting boundary.
 */
export function SavingsTargetFields(props: SavingsTargetFieldsProps) {
  return (
    <>
      <label>
        Track goal progress
        <select
          value={props.draft.trackTarget ? 'target' : 'contribution'}
          onChange={(event) => {
            props.onChange({ trackTarget: event.target.value === 'target' });
          }}
        >
          <option value="contribution">Monthly contribution only</option>
          <option value="target">Saved amount and target</option>
        </select>
      </label>
      {props.draft.trackTarget && (
        <>
          <label>
            Savings target ($)
            <input
              required
              inputMode="decimal"
              value={props.draft.targetAmount}
              onChange={(event) => {
                props.onChange({ targetAmount: event.target.value });
              }}
            />
          </label>
          <label>
            Already saved for this goal ($)
            <input
              required
              inputMode="decimal"
              value={props.draft.saved}
              onChange={(event) => {
                props.onChange({ saved: event.target.value });
              }}
            />
          </label>
          <p className={styles.help}>
            Record your progress here. It does not change account balances or your chosen monthly
            contribution.
          </p>
        </>
      )}
    </>
  );
}
