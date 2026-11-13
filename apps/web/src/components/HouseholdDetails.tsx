import { useState, type ReactNode } from 'react';
import { Info, TriangleAlert, Users } from 'lucide-react';
import type { Household, Member, PayrollSummary } from '@bank-piggy/contracts';
import { HOUSEHOLD_SCOPE, money } from '../features/plan/format';
import { BalanceAvailability } from '../features/connections/BalanceAvailability';
import { useConnectionData } from '../features/connections/ConnectionsContext';
import { hasMissingCurrentBalance } from '../features/connections/presentation';
import { Modal } from './Modal';
import styles from './PlanDetails.module.css';

interface HouseholdDetailsProps {
  readonly household: Household;
  readonly payrollSummaries: readonly PayrollSummary[];
  readonly summary: ReactNode;
}

/**
 * Keeps payroll estimates, split details, and balance notices outside the fixed summary card.
 * @param props Saved household and server-calculated monthly payroll estimates.
 * @returns A native dialog trigger that marks balance issues without hiding their explanation.
 */
export function HouseholdDetails(props: HouseholdDetailsProps) {
  const [open, setOpen] = useState(false);
  const state = useConnectionData();
  const needsReview =
    Boolean(state.error) ||
    (!state.loading && !state.balances) ||
    hasMissingCurrentBalance(state.data?.connections ?? [], HOUSEHOLD_SCOPE);
  const Icon = needsReview ? TriangleAlert : Info;
  return (
    <>
      <button
        className={styles.detailsButton}
        data-review={needsReview}
        aria-live="polite"
        onClick={() => {
          setOpen(true);
        }}
      >
        <Icon size={14} />
        {needsReview ? 'Review balances' : 'Household details'}
      </button>
      {open && (
        <Modal
          title="Household details"
          description="Payroll estimates, shared split, and balance notices."
          onClose={() => {
            setOpen(false);
          }}
        >
          <HouseholdDetailContent {...props} />
        </Modal>
      )}
    </>
  );
}

/**
 * Shows optional payroll investments without subtracting them from take-home pay again.
 * @param props Saved members, shared split, and monthly payroll amounts.
 * @returns Full detail content with the original amounts and actionable balance notices.
 */
function HouseholdDetailContent(props: HouseholdDetailsProps) {
  const { household, payrollSummaries } = props;
  const tracked = payrollSummaries.filter((summary) =>
    household.members.some((member) => member.id === summary.memberId && member.payroll),
  );
  const split = { salary: 'Base salary', equal: 'Equal shares', custom: 'Custom percentage' };
  return (
    <div className={styles.detailsContent}>
      <BalanceNotices />
      {props.summary}
      {household.members.length > 1 && (
        <p className={styles.sharedSplit}>
          <Users size={13} /> Shared split · {split[household.defaultSplitMethod]}
        </p>
      )}
      {tracked.length > 0 && <h3>Payroll investments · monthly estimates</h3>}
      {tracked.map((summary) => (
        <PayrollFacts key={summary.memberId} members={household.members} summary={summary} />
      ))}
      {tracked.length === 0 && <p>No payroll investments entered.</p>}
    </div>
  );
}

/**
 * Keeps failed refreshes actionable while missing-value and plan-revision notices stay explicit.
 * @returns Current balance notices and a read-only retry when a status request failed.
 */
function BalanceNotices() {
  const state = useConnectionData();
  return (
    <>
      {state.error && (
        <div>
          <p role="alert">{state.error}</p>
          <button
            className={styles.detailsButton}
            disabled={state.loading || state.busy}
            onClick={() => void state.reload()}
          >
            Reload balances
          </button>
        </div>
      )}
      <BalanceAvailability scope={HOUSEHOLD_SCOPE} />
    </>
  );
}

/**
 * Presents one member's unchanged server payroll amounts with distinct contribution labels.
 * @param props Saved member names and one monthly payroll estimate.
 * @param props.members Saved household members.
 * @param props.summary Canonical server-calculated payroll estimate.
 * @returns The member name and three monthly investment amounts.
 */
function PayrollFacts(props: {
  readonly members: readonly Member[];
  readonly summary: PayrollSummary;
}) {
  return (
    <div>
      <strong>{props.members.find((member) => member.id === props.summary.memberId)?.name}</strong>
      <dl>
        <div>
          <dt>Traditional 401(k)</dt>
          <dd>{money(props.summary.traditional401kMonthlyCents)}</dd>
        </div>
        <div>
          <dt>Roth 401(k)</dt>
          <dd>{money(props.summary.roth401kMonthlyCents)}</dd>
        </div>
        <div>
          <dt>Employer match</dt>
          <dd>{money(props.summary.employerMonthlyCents)}</dd>
        </div>
      </dl>
    </div>
  );
}
