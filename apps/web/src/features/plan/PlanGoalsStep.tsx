import type { Dispatch } from 'react';
import { Plus } from 'lucide-react';
import type { PlanDraft } from './planDraft';
import type { PlanAction } from './planReducer';
import { PlanItemRow } from './PlanFields';
import { PlanBalances } from './PlanBalancesStep';
import styles from './Editors.module.css';

interface GoalsStepProps {
  readonly state: Pick<PlanDraft, 'members' | 'goals'>;
  readonly dispatch: Dispatch<PlanAction>;
}

/**
 * Edits optional goal contributions and current balances.
 * @param props Goal drafts, member drafts, and draft dispatch.
 * @returns Goals and balances setup step.
 */
export function PlanGoalsStep(props: GoalsStepProps) {
  const { state, dispatch } = props;
  return (
    <div className={styles.section}>
      <h3>Make room for what matters</h3>
      <p className={styles.help}>
        Goals are optional. Shared goals split evenly. Add only contributions paid from take-home
        pay.
      </p>
      {state.goals.map((draft, index) => (
        <PlanItemRow
          key={draft.goal.id}
          kind="Goal"
          item={draft.goal}
          amount={draft.amount}
          members={state.members}
          first={index === 0}
          onChange={(update) => {
            dispatch({ type: 'goal', index, update });
          }}
          onRemove={() => {
            dispatch({ type: 'removeGoal', index });
          }}
        />
      ))}
      <button
        type="button"
        className={styles.add}
        onClick={() => {
          dispatch({ type: 'addGoal', id: crypto.randomUUID() });
        }}
      >
        <Plus size={15} />
        Add goal
      </button>
      <PlanBalances members={state.members} dispatch={dispatch} />
    </div>
  );
}
