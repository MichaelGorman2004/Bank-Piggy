import { useState } from 'react';
import type { Household, Summary, PayrollSummary } from '@bank-piggy/contracts';
import { BalancePanel } from './BalancePanel';
import { FinanceCard } from './FinanceCard';
import { PlanDetails } from './PlanDetails';
import { SummaryCards } from './SummaryCards';
import styles from './HouseholdOverview.module.css';

interface HouseholdOverviewProps {
  readonly household: Household;
  readonly payrollSummaries: readonly PayrollSummary[];
  readonly summary: Summary;
  readonly factor: number;
  readonly scope: string;
  readonly onEdit: () => void;
}
const SECTIONS = [
  { id: 'details', label: 'At a glance' },
  { id: 'balances', label: 'Balances' },
  { id: 'totals', label: 'Plan totals' },
] satisfies ReadonlyArray<{ readonly id: string; readonly label: string }>;

/**
 * Allocates bounded cards on desktop and selects one card on limited screens.
 * @param props Canonical plan, scoped balances, reporting totals, and edit action.
 * @returns Household cards whose bodies scroll independently of the page.
 */
export function HouseholdOverview(props: HouseholdOverviewProps) {
  const [active, setActive] = useState('details');
  return (
    <div className={styles.overview}>
      <div className={styles.sections} role="group" aria-label="Household plan section">
        {SECTIONS.map((section) => (
          <button
            key={section.id}
            aria-pressed={active === section.id}
            onClick={() => {
              setActive(section.id);
            }}
          >
            {section.label}
          </button>
        ))}
      </div>
      <div className={styles.panels}>
        <div className={styles.totals} data-active={active === 'totals'}>
          <FinanceCard
            title="Plan totals"
            description="Estimates for the selected scope and period."
            actions={null}
          >
            <SummaryCards summary={props.summary} factor={props.factor} />
          </FinanceCard>
        </div>
        <div className={styles.details} data-active={active === 'details'}>
          <PlanDetails household={props.household} payrollSummaries={props.payrollSummaries} />
        </div>
        <div className={styles.balances} data-active={active === 'balances'}>
          <BalancePanel
            household={props.household}
            scope={props.scope}
            summary={props.summary}
            onEdit={props.onEdit}
          />
        </div>
      </div>
    </div>
  );
}
