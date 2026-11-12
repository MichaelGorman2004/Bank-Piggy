import { useState, type SubmitEvent } from 'react';
import type { Goal, Household } from '@bank-piggy/contracts';
import { Modal } from '../../components/Modal';
import { EditorActions } from './EditorActions';
import { GoalFields } from './GoalFields';
import { goalDraft, goalValue, type GoalDraft, type GoalKind } from './goalDraft';
import styles from './Editors.module.css';

interface GoalEditorProps {
  readonly household: Household;
  readonly goal?: Goal;
  readonly initialKind?: GoalKind;
  readonly busy: boolean;
  readonly error: string | null;
  readonly onClose: () => void;
  readonly onSave: (goal: Goal) => Promise<void>;
  readonly onDelete: (id: string) => Promise<void>;
}

/**
 * Retains incomplete field text and validates the complete request before saving.
 * @param props Component values and event handlers.
 * @returns Current draft state and event handlers.
 */
function useGoalForm(props: GoalEditorProps) {
  const [draft, setDraft] = useState(() =>
    goalDraft(props.household, props.goal, props.initialKind),
  );
  const [localError, setLocalError] = useState<string | null>(null);
  /**
   * Merges edited fields into the latest local draft.
   * @param change Fields to merge into the current draft.
   */
  function update(change: Partial<GoalDraft>) {
    setDraft((current) => ({ ...current, ...change }));
  }
  /**
   * Validates the draft and awaits the requested save.
   * @param event Browser form or keyboard event.
   */
  async function submit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    const result = goalValue(draft, props.goal?.id ?? crypto.randomUUID(), props.household);
    if (result.error) {
      setLocalError(result.error);
      return;
    }
    setLocalError(null);
    if (result.goal) await props.onSave(result.goal);
  }
  return { draft, localError, submit, update };
}

/**
 * Edits a monthly goal without confusing planned contributions with saved balances.
 * @param props Component values and event handlers.
 * @returns Rendered interface.
 */
export function GoalEditor(props: GoalEditorProps) {
  const { draft, localError, submit, update } = useGoalForm(props);
  return (
    <Modal
      title={props.goal ? 'Edit goal' : 'Make room for a goal'}
      description="Plan contributions from take-home pay. Add a savings target or an annual Roth IRA goal."
      onClose={props.onClose}
      busy={props.busy}
    >
      <form
        className={styles.form}
        onSubmit={(event) => {
          void submit(event);
        }}
      >
        <fieldset disabled={props.busy} className={styles.fields}>
          <GoalFields draft={draft} household={props.household} onChange={update} />
          {(localError || props.error) && (
            <p className={styles.error} role="alert">
              {localError ?? props.error}
            </p>
          )}
          <EditorActions
            kind="goal"
            busy={props.busy}
            onClose={props.onClose}
            onDelete={
              props.goal
                ? () => {
                    void props.onDelete(props.goal?.id ?? '');
                  }
                : undefined
            }
          />
        </fieldset>
      </form>
    </Modal>
  );
}
