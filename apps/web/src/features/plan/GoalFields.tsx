import type { Household } from '@bank-piggy/contracts';
import type { GoalDraft } from './goalDraft';
import { retirementGoalChange } from './retirementGoal';
import { GoalTargetFields } from './GoalTargetFields';

interface GoalFieldsProps {
  readonly draft: GoalDraft;
  readonly household: Household;
  readonly onChange: (change: Partial<GoalDraft>) => void;
}

/**
 * Collects contributions from take-home pay and identifies the goal's owner.
 * @param props Component values and event handlers.
 * @returns Rendered interface.
 */
export function GoalFields(props: GoalFieldsProps) {
  const { draft, onChange } = props;
  return (
    <>
      <label>
        Goal name
        <input
          autoFocus
          required
          maxLength={80}
          value={draft.name}
          onChange={(event) => {
            onChange({ name: event.target.value });
          }}
          placeholder="e.g. Emergency fund"
        />
      </label>
      <label>
        Monthly contribution ($)
        <input
          required
          inputMode="decimal"
          value={draft.amount}
          onChange={(event) => {
            onChange({ amount: event.target.value });
          }}
          placeholder="0.00"
        />
      </label>
      <GoalOwner {...props} />
      <GoalTargetFields
        draft={draft}
        onChange={(change) => {
          onChange(retirementGoalChange(draft, change, props.household));
        }}
      />
    </>
  );
}

/**
 * Narrows the selected owner to a member of this household.
 * @param props Goal draft and household membership.
 * @returns Owner selection field.
 */
function GoalOwner(props: GoalFieldsProps) {
  const { draft, onChange } = props;
  return (
    <label>
      Who is this goal for?
      <select
        value={draft.ownerId}
        onChange={(event) => {
          const ownerId =
            event.target.value === 'shared'
              ? 'shared'
              : props.household.members.find((member) => member.id === event.target.value)?.id;
          if (ownerId) onChange({ ownerId });
        }}
      >
        <option value="shared" disabled={draft.kind === 'roth_ira'}>
          {props.household.members.length === 1 ? 'Household' : 'Both of us (50 / 50)'}
        </option>
        {props.household.members.map((member) => (
          <option key={member.id} value={member.id}>
            {member.name}
          </option>
        ))}
      </select>
    </label>
  );
}
