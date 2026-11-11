import type { Dispatch } from 'react';
import { PAY_FREQUENCIES } from './format';
import type { MemberDraft } from './planDraft';
import type { PlanAction } from './planReducer';
import { PlanMoneyField } from './PlanFields';
import styles from './Editors.module.css';

interface MemberIncomeProps {
  readonly draft: MemberDraft;
  readonly index: number;
  readonly dispatch: Dispatch<PlanAction>;
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
            autoFocus={index === 0}
            required
            maxLength={60}
            value={draft.name}
            onChange={(event) => {
              dispatch({ type: 'member', index, update: { name: event.target.value } });
            }}
          />
        </label>
        <PlanPayFrequency {...props} />
        <PlanMoneyField
          label="Monthly take-home ($)"
          required
          pattern="[0-9]+(\.[0-9]{1,2})?"
          value={draft.takeHome}
          onChange={(takeHome) => {
            dispatch({ type: 'member', index, update: { takeHome } });
          }}
          help="After taxes and payroll deductions. For biweekly pay, multiply a paycheck by 26 ÷ 12."
        />
        <PlanMoneyField
          label="Annual base salary ($)"
          required
          pattern="[0-9]+(\.[0-9]{1,2})?"
          value={draft.salary}
          onChange={(salary) => {
            dispatch({ type: 'member', index, update: { salary } });
          }}
          help="Used only for salary splits. Enter 0 if you will not use them."
        />
      </div>
    </section>
  );
}

/**
 * Shows both household members' income drafts.
 * @param props Member drafts and draft dispatch.
 * @returns Income setup step.
 */
export function PlanIncomeStep(props: MemberStepProps) {
  const { members, dispatch } = props;
  return (
    <div>
      {members.map((draft, index) => (
        <MemberIncome key={draft.member.id} draft={draft} index={index} dispatch={dispatch} />
      ))}
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
