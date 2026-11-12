import type { GoalDraft } from './goalDraft';
import { PlanMoneyField } from './PlanFields';
import styles from './Editors.module.css';

interface RothIraFieldsProps {
  readonly draft: GoalDraft;
  readonly onChange: (change: Partial<GoalDraft>) => void;
}

/**
 * Tracks a user-chosen Roth IRA annual amount independently of account balances.
 * @param props Editable annual target and manual contribution progress.
 * @returns Contribution year, target, and already-contributed fields.
 */
export function RothIraFields(props: RothIraFieldsProps) {
  const { draft, onChange } = props;
  return (
    <>
      <label>
        Contribution year
        <input
          type="number"
          min="2000"
          max="2100"
          step="1"
          required
          value={draft.contributionYear}
          onChange={(event) => {
            onChange({ contributionYear: event.target.value });
          }}
        />
      </label>
      <PlanMoneyField
        label="Annual contribution target ($)"
        required
        value={draft.targetAmount}
        onChange={(targetAmount) => {
          onChange({ targetAmount });
        }}
      />
      <PlanMoneyField
        label="Already contributed for this year ($)"
        required
        value={draft.saved}
        onChange={(saved) => {
          onChange({ saved });
        }}
      />
      <p className={styles.help}>
        Enter the annual amount you can contribute. Bank Piggy does not determine your IRA limit or
        eligibility. Only the chosen monthly contribution reduces the plan. Recorded contributions
        do not change account balances.
      </p>
    </>
  );
}
