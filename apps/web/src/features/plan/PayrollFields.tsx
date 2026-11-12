import { MAX_MATCH_TIERS, type PayrollDraft, type MatchTierDraft } from './payrollDraft';
import styles from './Editors.module.css';
import ownStyles from './PayrollFields.module.css';

interface MatchTierProps {
  readonly tier: MatchTierDraft;
  readonly index: number;
  readonly onChange: (tier: MatchTierDraft) => void;
  readonly onRemove: () => void;
}
interface PayrollFieldsProps {
  readonly draft: PayrollDraft;
  readonly onChange: (draft: PayrollDraft) => void;
}

/**
 * Keeps payroll investments separate from goals funded with take-home pay.
 * @param props Editable payroll settings and their update action.
 * @returns Optional 401(k) and match fields.
 */
export function PayrollFields(props: PayrollFieldsProps) {
  const { draft, onChange } = props;
  return (
    <details className={ownStyles.disclosure}>
      <summary>401(k) contributions & employer match</summary>
      <div className={styles.section}>
        <label className={ownStyles.toggle}>
          <input
            type="checkbox"
            checked={draft.enabled}
            onChange={(event) => {
              onChange({ ...draft, enabled: event.target.checked });
            }}
          />
          Show payroll investment estimates
        </label>
        {draft.enabled && <PayrollContributions {...props} />}
        <p className={styles.help}>
          These investments already come out of your paycheck. They do not reduce take-home pay
          again. Estimates use annual base salary and do not apply contribution limits or vesting
          rules.
        </p>
      </div>
    </details>
  );
}

/**
 * Collects employee percentages and an explicit employer-match method.
 * @param props Editable payroll settings and their update action.
 * @returns Salary percentages and match controls.
 */
function PayrollContributions(props: PayrollFieldsProps) {
  const { draft, onChange } = props;
  return (
    <>
      <EmployeeContributions {...props} />
      <label>
        Employer match
        <select
          value={draft.method}
          onChange={(event) => {
            const method = event.target.value;
            if (method === 'none' || method === 'tiers' || method === 'fixed')
              onChange({ ...draft, method });
          }}
        >
          <option value="none">No employer match</option>
          <option value="tiers">Percentage match tiers</option>
          <option value="fixed">Entered monthly amount</option>
        </select>
      </label>
      {draft.method === 'tiers' && <MatchTiers {...props} />}
      {draft.method === 'fixed' && (
        <label>
          Employer match per month ($)
          <input
            required
            inputMode="decimal"
            value={draft.employerMonthly}
            onChange={(event) => {
              onChange({ ...draft, employerMonthly: event.target.value });
            }}
          />
        </label>
      )}
      {draft.method === 'fixed' && (
        <p className={styles.help}>
          Use the expected monthly amount for an unusual match or a match you already know.
        </p>
      )}
    </>
  );
}

/**
 * Edits one salary band while retaining incomplete text in other bands.
 * @param props Current tier and its update and removal actions.
 * @returns Two percentage fields with a named remove button.
 */
function MatchTier(props: MatchTierProps) {
  return (
    <div className={ownStyles.tier}>
      <label>
        Match rate (%)
        <input
          aria-label={`Tier ${String(props.index + 1)} match rate (%)`}
          required
          inputMode="decimal"
          value={props.tier.rate}
          onChange={(event) => {
            props.onChange({ ...props.tier, rate: event.target.value });
          }}
        />
      </label>
      <label>
        Up to salary contribution (%)
        <input
          aria-label={`Tier ${String(props.index + 1)} salary cap (%)`}
          required
          inputMode="decimal"
          value={props.tier.upTo}
          onChange={(event) => {
            props.onChange({ ...props.tier, upTo: event.target.value });
          }}
        />
      </label>
      <button
        type="button"
        className={styles.secondary}
        aria-label={`Remove match tier ${String(props.index + 1)}`}
        onClick={props.onRemove}
      >
        Remove
      </button>
    </div>
  );
}

/**
 * Uses cumulative caps to describe the next band of employee contributions.
 * @param props Editable payroll settings and their update action.
 * @returns Bounded match-tier editor and a concrete example.
 */
function MatchTiers(props: PayrollFieldsProps) {
  const { draft, onChange } = props;
  return (
    <div className={styles.section}>
      <p className={styles.help}>
        Example: 100% of contributions up to 3% of salary, then 50% of the next contributions up to
        5% of salary. Enter caps in increasing order.
      </p>
      {draft.tiers.map((tier, index) => (
        <MatchTier
          key={index}
          tier={tier}
          index={index}
          onChange={(next) => {
            onChange({
              ...draft,
              tiers: draft.tiers.map((current, position) => (position === index ? next : current)),
            });
          }}
          onRemove={() => {
            onChange({ ...draft, tiers: draft.tiers.filter((_, position) => position !== index) });
          }}
        />
      ))}
      {draft.tiers.length < MAX_MATCH_TIERS && (
        <button
          type="button"
          className={styles.add}
          onClick={() => {
            onChange({ ...draft, tiers: [...draft.tiers, { upTo: '', rate: '' }] });
          }}
        >
          Add match tier
        </button>
      )}
    </div>
  );
}

/**
 * Collects the two distinct employee 401(k) salary percentages.
 * @param props Editable payroll contribution settings.
 * @returns Traditional and Roth 401(k) inputs.
 */
function EmployeeContributions(props: PayrollFieldsProps) {
  const { draft, onChange } = props;
  return (
    <div className={styles.grid}>
      <label>
        Traditional 401(k) (% of salary)
        <input
          inputMode="decimal"
          required
          value={draft.traditional}
          onChange={(event) => {
            onChange({ ...draft, traditional: event.target.value });
          }}
        />
      </label>
      <label>
        Roth 401(k) (% of salary)
        <input
          inputMode="decimal"
          required
          value={draft.roth}
          onChange={(event) => {
            onChange({ ...draft, roth: event.target.value });
          }}
        />
      </label>
    </div>
  );
}
