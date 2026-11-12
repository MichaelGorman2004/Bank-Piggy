import { useReducer, useState, type SyntheticEvent } from 'react';
import type { Household } from '@bank-piggy/contracts';
import { ArrowLeft, ArrowRight } from 'lucide-react';
import { Modal } from '../../components/Modal';
import { convertPlan, type PlanDraft } from './planDraft';
import { initialPlan, planReducer } from './planReducer';
import { PlanIncomeStep } from './PlanIncomeStep';
import { PlanExpensesStep } from './PlanExpensesStep';
import { PlanGoalsStep } from './PlanGoalsStep';
import styles from './Editors.module.css';

interface PlanEditorProps {
  readonly household: Household;
  readonly busy: boolean;
  readonly error: string | null;
  readonly onClose: () => void;
  readonly onSave: (household: Household) => Promise<void>;
}
const STEPS = ['Income', 'Expenses', 'Goals & balances'];
const LAST_STEP = STEPS.length - 1;

/**
 * Shows the active setup step.
 * @param props Current step index.
 * @param props.step Zero-based active step.
 * @returns Setup progress labels.
 */
function PlanProgress(props: { readonly step: number }) {
  const { step } = props;
  return (
    <div className={styles.steps} aria-label="Plan setup progress">
      {STEPS.map((label, index) => (
        <div
          key={label}
          className={`${styles.step} ${index === step ? styles.current : ''}`}
          aria-current={index === step ? 'step' : undefined}
        >
          {index + 1}. {label}
        </div>
      ))}
    </div>
  );
}

interface PlanActionsProps {
  readonly step: number;
  readonly busy: boolean;
  readonly onBack: () => void;
  readonly onClose: () => void;
}
/**
 * Provides back, cancel, and submit controls.
 * @param props Current step and navigation callbacks.
 * @returns Setup action buttons.
 */
function PlanActions(props: PlanActionsProps) {
  const { step, busy, onBack, onClose } = props;
  return (
    <div className={styles.actions}>
      {step > 0 && (
        <button type="button" className={styles.secondary} disabled={busy} onClick={onBack}>
          <ArrowLeft size={15} />
          Back
        </button>
      )}
      <button type="button" className={styles.secondary} disabled={busy} onClick={onClose}>
        Cancel
      </button>
      <button className={styles.button} disabled={busy}>
        {busy ? (
          'Saving…'
        ) : step < LAST_STEP ? (
          <>
            Continue
            <ArrowRight size={15} />
          </>
        ) : (
          'Save household plan'
        )}
      </button>
    </div>
  );
}

/**
 * Collects a complete plan and preserves drafts when validation or saving fails.
 * @param props Server household and save state.
 * @returns Household setup modal.
 */
export function PlanEditor(props: PlanEditorProps) {
  const { household, busy, error, onClose, onSave } = props;
  const [draft, dispatch] = useReducer(planReducer, household, initialPlan);
  const { step, localError, submit, back } = usePlanNavigation(household, draft, onSave);
  return (
    <Modal
      title="Your household plan"
      description="Start with estimates. Add known costs now, then refine your plan."
      onClose={onClose}
      busy={busy}
      wide
    >
      <PlanProgress step={step} />
      <form
        className={styles.form}
        onSubmit={(event) => {
          void submit(event);
        }}
      >
        <fieldset disabled={busy} className={styles.fields}>
          {step === 0 && <PlanIncomeStep members={draft.members} dispatch={dispatch} />}
          {step === 1 && <PlanExpensesStep state={draft} dispatch={dispatch} />}
          {step === 2 && <PlanGoalsStep state={draft} dispatch={dispatch} />}
          {(localError || error) && (
            <p className={styles.error} role="alert">
              {localError ?? error}
            </p>
          )}
          <PlanActions step={step} busy={busy} onBack={back} onClose={onClose} />
        </fieldset>
      </form>
    </Modal>
  );
}

/**
 * Validates final submission and returns to the step with invalid data.
 * @param household Original server household.
 * @param draft Current editable values.
 * @param onSave Asynchronous server save.
 * @returns Navigation state and handlers.
 */
function usePlanNavigation(
  household: Household,
  draft: PlanDraft,
  onSave: PlanEditorProps['onSave'],
) {
  const [step, setStep] = useState(0);
  const [localError, setLocalError] = useState<string | null>(null);
  /**
   * Advances setup or saves the validated plan.
   * @param event Form submission event.
   * @returns Completion of the save request.
   */
  async function submit(event: SyntheticEvent<HTMLFormElement>) {
    event.preventDefault();
    setLocalError(null);
    if (step < LAST_STEP) {
      setStep(step + 1);
      return;
    }
    const result = convertPlan(household, draft);
    if (!result.household) {
      setLocalError(result.error);
      setStep(result.step);
      return;
    }
    await onSave(result.household);
  }
  /** Returns to the previous step and clears local validation feedback. */
  function back() {
    setStep(step - 1);
    setLocalError(null);
  }
  return { step, localError, submit, back };
}
