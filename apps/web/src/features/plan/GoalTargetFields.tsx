import type { GoalDraft } from './goalDraft';
import { MonthPicker } from '../../components/MonthPicker';
import { TripFields } from './TripFields';
import { RothIraFields } from './RothIraFields';
import { SavingsTargetFields } from './SavingsTargetFields';
import styles from './Editors.module.css';

interface GoalTargetFieldsProps {
  readonly draft: GoalDraft;
  readonly onChange: (change: Partial<GoalDraft>) => void;
}

/**
 * Selects one goal type while retaining unfinished target fields in the draft.
 * @param props Current goal draft and field update action.
 * @returns Native goal type control.
 */
function GoalType(props: GoalTargetFieldsProps) {
  return (
    <label>
      Goal type
      <select
        value={props.draft.kind}
        onChange={(event) => {
          const kind = event.target.value;
          if (kind === 'roth_ira')
            props.onChange({ kind, name: props.draft.name || 'Roth IRA', trackTarget: true });
          else if (kind === 'goal' || kind === 'trip' || kind === 'purchase')
            props.onChange({ kind });
        }}
      >
        <option value="goal">Savings, investing, or debt goal</option>
        <option value="roth_ira">Roth IRA annual contribution</option>
        <option value="trip">Trip or vacation</option>
        <option value="purchase">Purchase to save for</option>
      </select>
    </label>
  );
}

/**
 * Collects manual saved progress and a target month for trips and purchases.
 * @param props Current goal draft and field update action.
 * @returns Deadline and saved balance controls.
 */
function TargetProgressFields(props: GoalTargetFieldsProps) {
  const { draft, onChange } = props;
  const kind = draft.kind === 'trip' ? 'trip' : 'purchase';
  return (
    <>
      <MonthPicker
        label={kind === 'trip' ? 'Trip target month' : 'Purchase target month'}
        required
        value={draft.targetMonth}
        onChange={(targetMonth) => {
          onChange({ targetMonth });
        }}
      />
      <label>
        Already saved for this {kind} ($)
        <input
          required
          inputMode="decimal"
          value={draft.saved}
          onChange={(event) => {
            onChange({ saved: event.target.value });
          }}
        />
      </label>
      <p className={styles.help}>
        The saved amount records progress. Only your monthly contribution is deducted from the
        budget. The suggestion includes the planning month and target month.
      </p>
    </>
  );
}

/**
 * Collects the total amount to save for a planned purchase.
 * @param props Current goal draft and field update action.
 * @returns Purchase target amount control.
 */
function PurchaseFields(props: GoalTargetFieldsProps) {
  return (
    <label>
      Purchase target ($)
      <input
        required
        inputMode="decimal"
        value={props.draft.targetAmount}
        onChange={(event) => {
          props.onChange({ targetAmount: event.target.value });
        }}
      />
    </label>
  );
}

/**
 * Adds optional target fields without combining trip and purchase details.
 * @param props Current goal draft and field update action.
 * @returns Goal type and its target controls.
 */
export function GoalTargetFields(props: GoalTargetFieldsProps) {
  return (
    <>
      <GoalType {...props} />
      {props.draft.kind === 'goal' && <SavingsTargetFields {...props} />}
      {props.draft.kind === 'roth_ira' && <RothIraFields {...props} />}
      {(props.draft.kind === 'trip' || props.draft.kind === 'purchase') && (
        <>
          <TargetProgressFields {...props} />
          {props.draft.kind === 'trip' ? <TripFields {...props} /> : <PurchaseFields {...props} />}
        </>
      )}
    </>
  );
}
