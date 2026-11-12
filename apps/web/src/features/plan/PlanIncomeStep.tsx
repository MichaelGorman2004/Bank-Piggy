import { useState, type Dispatch } from 'react';
import { Plus } from 'lucide-react';
import { PAY_FREQUENCIES } from './format';
import type { MemberDraft } from './planDraft';
import type { PlanAction } from './planReducer';
import { PlanMoneyField } from './PlanFields';
import { PayrollFields } from './PayrollFields';
import { PayScheduleFields } from './PayScheduleFields';
import { MAX_HOUSEHOLD_MEMBERS } from './householdMembers';
import styles from './Editors.module.css';

interface MemberIncomeProps {
  readonly draft: MemberDraft;
  readonly index: number;
  readonly dispatch: Dispatch<PlanAction>;
  readonly autoFocus: boolean;
}
interface MemberStepProps {
  readonly members: MemberDraft[];
  readonly dispatch: Dispatch<PlanAction>;
}

/**
 * Edits one member's name and income without changing its identifier.
 * @param props Member draft, position, and draft dispatch.
 * @returns Member income fields.
 */
function MemberIncome(props: MemberIncomeProps) {
  const { draft, index, dispatch } = props;
  return (
    <section className={styles.member}>
      <h3>{draft.name || `Member ${String(index + 1)}`}</h3>
      <div className={styles.grid}>
        <label>
          Name
          <input
            autoFocus={props.autoFocus}
            required
            maxLength={60}
            value={draft.name}
            onChange={(event) => {
              dispatch({ type: 'member', index, update: { name: event.target.value } });
            }}
          />
        </label>
        <PlanPayFrequency {...props} />
        <IncomeMoneyFields {...props} />
      </div>
      <PayScheduleFields
        frequency={draft.frequency}
        draft={draft.paySchedule}
        onChange={(paySchedule) => {
          dispatch({ type: 'member', index, update: { paySchedule } });
        }}
      />
      <PayrollFields
        draft={draft.payroll}
        onChange={(payroll) => {
          dispatch({ type: 'member', index, update: { payroll } });
        }}
      />
    </section>
  );
}

/**
 * Starts with the saved people and lets the user add one partner.
 * @param props Member drafts and draft dispatch.
 * @returns Income setup step.
 */
export function PlanIncomeStep(props: MemberStepProps) {
  const { members, dispatch } = props;
  const [focusNewMember, setFocusNewMember] = useState(false);
  return (
    <div className={styles.section}>
      <div>
        {members.map((draft, index) => (
          <MemberIncome
            key={draft.member.id}
            draft={draft}
            index={index}
            dispatch={dispatch}
            autoFocus={index === 0 || focusNewMember}
          />
        ))}
      </div>
      {members.length < MAX_HOUSEHOLD_MEMBERS && (
        <button
          type="button"
          className={styles.add}
          onClick={() => {
            setFocusNewMember(true);
            dispatch({ type: 'addMember' });
          }}
        >
          <Plus size={15} aria-hidden="true" />
          Add person
        </button>
      )}
    </div>
  );
}

/**
 * Selects a canonical pay frequency without an unsafe string cast.
 * @param props Member draft, position, and draft dispatch.
 * @returns Pay frequency selection.
 */
function PlanPayFrequency(props: MemberIncomeProps) {
  const { draft, index, dispatch } = props;
  return (
    <label>
      Pay frequency
      <select
        value={draft.frequency}
        onChange={(event) => {
          const frequency = PAY_FREQUENCIES.find(
            (item) => item.value === event.target.value,
          )?.value;
          if (frequency) dispatch({ type: 'member', index, update: { frequency } });
        }}
      >
        {PAY_FREQUENCIES.map((frequency) => (
          <option key={frequency.value} value={frequency.value}>
            {frequency.label}
          </option>
        ))}
      </select>
    </label>
  );
}

/**
 * Keeps net pay distinct from the base salary used for splits and payroll estimates.
 * @param props Member draft and draft update action.
 * @returns Monthly take-home and annual salary fields.
 */
function IncomeMoneyFields(props: MemberIncomeProps) {
  const { draft, index, dispatch } = props;
  return (
    <>
      <PlanMoneyField
        label="Monthly take-home after tax & automatic investments ($)"
        required
        pattern="[0-9]+(\.[0-9]{1,2})?"
        value={draft.takeHome}
        onChange={(takeHome) => {
          dispatch({ type: 'member', index, update: { takeHome } });
        }}
        help="Use the pay that reaches your bank after tax, 401(k), and other automatic payroll deductions. For biweekly pay, multiply a paycheck by 26 ÷ 12."
      />
      <PlanMoneyField
        label="Annual base salary ($)"
        required
        pattern="[0-9]+(\.[0-9]{1,2})?"
        value={draft.salary}
        onChange={(salary) => {
          dispatch({ type: 'member', index, update: { salary } });
        }}
        help="Used for salary splits and payroll investment estimates. Enter 0 if you will not use them."
      />
    </>
  );
}
