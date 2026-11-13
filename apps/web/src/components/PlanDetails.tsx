import { useState } from 'react';
import type { Household, Member, PayrollSummary } from '@bank-piggy/contracts';
import { accountTotals } from '../features/accounts/accountTypes';
import { money } from '../features/plan/format';
import { HouseholdDetails } from './HouseholdDetails';
import { FinanceCard } from './FinanceCard';
import styles from './PlanDetails.module.css';
import { useConnectionData } from '../features/connections/ConnectionsContext';

interface PlanFact {
  readonly label: string;
  readonly read: (member: Member) => number | string;
}
interface PlanDetailsProps {
  readonly household: Household;
  readonly payrollSummaries: readonly PayrollSummary[];
}
type FactSection = 'income' | 'balances';
const PAY_SCHEDULE: Record<Member['payFrequency'], string> = {
  weekly: 'Weekly',
  biweekly: 'Every 2 weeks',
  semimonthly: 'Twice a month',
  monthly: 'Monthly',
  irregular: 'Irregular',
};
const INCOME_FACTS: ReadonlyArray<PlanFact> = [
  { label: 'Take-home / mo', read: (member) => member.monthlyTakeHomeCents },
  { label: 'Base salary / yr', read: (member) => member.annualBaseSalaryCents },
  { label: 'Pay schedule', read: (member) => PAY_SCHEDULE[member.payFrequency] },
];
const BALANCE_FACTS: ReadonlyArray<PlanFact> = [
  { label: 'Savings', read: (member) => accountTotals(member.accounts).savingsCents },
  { label: 'Investments', read: (member) => accountTotals(member.accounts).investmentsCents },
  { label: 'Manual debt', read: (member) => member.debtCents },
];
const COMPACT_THRESHOLD_CENTS = 100_000_000;
const SMALL_CARD_COMPACT_THRESHOLD_CENTS = 10_000_000;

/**
 * Keeps large summary values readable while retaining their exact accessible amount.
 * @param props Canonical fact and member to describe.
 * @param props.fact Named fact and its canonical value reader.
 * @param props.member Member whose value is shown.
 * @param props.precise Shows every cent instead of a compact display label.
 * @returns A value with its complete amount available to assistive tools and hover.
 */
function FactValue(props: {
  readonly fact: PlanFact;
  readonly member: Member;
  readonly precise?: boolean;
}) {
  const value = props.fact.read(props.member);
  if (typeof value === 'string') return <span>{value}</span>;
  const exact = money(value);
  if (props.precise) return <span>{exact}</span>;
  const shortLabel =
    Math.abs(value) >= SMALL_CARD_COMPACT_THRESHOLD_CENTS
      ? new Intl.NumberFormat('en-US', {
          style: 'currency',
          currency: 'USD',
          notation: 'compact',
          maximumFractionDigits: 1,
        }).format(value / 100)
      : exact;
  return (
    <span title={exact} aria-label={exact}>
      <span className={styles.fullValue}>
        {Math.abs(value) >= COMPACT_THRESHOLD_CENTS ? shortLabel : exact}
      </span>
      <span className={styles.shortValue}>{shortLabel}</span>
    </span>
  );
}

/**
 * Shows the saved people in a compact table with native row and column headings.
 * @param props Members, fact readers, and the accessible table name.
 * @param props.members Saved household people.
 * @param props.facts Related facts to compare.
 * @param props.label Accessible table name and first-column heading.
 * @param props.precise Keeps full amount labels in the details dialog.
 * @returns An income or balance table that fits a short card.
 */
function FactTable(props: {
  readonly members: ReadonlyArray<Member>;
  readonly facts: ReadonlyArray<PlanFact>;
  readonly label: string;
  readonly precise?: boolean;
}) {
  return (
    <table className={styles.table} aria-label={props.label}>
      <thead>
        <tr>
          <th scope="col">{props.label}</th>
          {props.members.map((member) => (
            <th key={member.id} scope="col" title={member.name}>
              {member.name}
            </th>
          ))}
        </tr>
      </thead>
      <tbody>
        {props.facts.map((fact) => (
          <tr key={fact.label}>
            <th scope="row">{fact.label}</th>
            {props.members.map((member) => (
              <td key={member.id}>
                <FactValue fact={fact} member={member} precise={props.precise} />
              </td>
            ))}
          </tr>
        ))}
      </tbody>
    </table>
  );
}

/**
 * Fits the household's core facts in one view; detailed accounts stay in Balances.
 * @param props The saved household and its agreed default expense split.
 * @param props.household Canonical household with member account snapshots.
 * @param props.payrollSummaries Server-calculated monthly payroll estimates.
 * @returns Compact income and balance comparisons without a scrolling member list.
 */
export function PlanDetails(props: PlanDetailsProps) {
  const [section, setSection] = useState<FactSection>('income');
  return (
    <FinanceCard
      title="Your household at a glance"
      description="Recorded income and balances."
      actions={null}
      className={styles.card}
      scrollable={false}
      footer={<PlanDetailsControls {...props} section={section} onSection={setSection} />}
    >
      <div className={styles.tables}>
        <div data-panel="income" data-active={section === 'income'}>
          <FactTable members={props.household.members} facts={INCOME_FACTS} label="Income" />
        </div>
        <div data-panel="balances" data-active={section === 'balances'}>
          <CurrentBalanceFacts household={props.household} />
        </div>
      </div>
      <p className={styles.summaryFallback}>Open the full summary below.</p>
    </FinanceCard>
  );
}

/**
 * Keeps facts selectable in small cards and opens extended details outside the fixed summary.
 * @param props Saved facts, selected compact section, and its change handler.
 * @returns One fixed footer row without an additional scroll or control row.
 */
function PlanDetailsControls(
  props: PlanDetailsProps & {
    readonly section: FactSection;
    readonly onSection: (section: FactSection) => void;
  },
) {
  return (
    <div className={styles.footerActions}>
      <select
        className={styles.sectionSelect}
        aria-label="At a glance section"
        value={props.section}
        onChange={(event) => {
          const value = event.target.value;
          if (value === 'income' || value === 'balances') props.onSection(value);
        }}
      >
        <option value="income">Income</option>
        <option value="balances">Balances</option>
      </select>
      <HouseholdDetails
        household={props.household}
        payrollSummaries={props.payrollSummaries}
        summary={<FullSummary household={props.household} />}
      />
    </div>
  );
}

/**
 * Keeps exact amounts and full member headings reachable when the summary card is very small.
 * @param props Saved household for the same income and connected balance facts.
 * @param props.household Saved members and manual snapshots.
 * @returns Complete tables inside the details dialog, independent of the card's compact mode.
 */
function FullSummary(props: { readonly household: Household }) {
  return (
    <div className={styles.detailsFacts}>
      <FactTable members={props.household.members} facts={INCOME_FACTS} label="Income" precise />
      <CurrentBalanceFacts household={props.household} precise />
    </div>
  );
}

/**
 * Uses one plan-matched projection for current facts and withholds stale manual replacements.
 * @param props Saved household used for member labels and manual-only balances.
 * @param props.household Saved household with stable member identities.
 * @param props.precise Keeps full amount labels in the details dialog.
 * @returns Current per-member balances or an explicit pending notice.
 */
function CurrentBalanceFacts(props: { readonly household: Household; readonly precise?: boolean }) {
  const connections = useConnectionData();
  const balances = connections.data?.connections.length ? connections.balances?.members : undefined;
  const facts: ReadonlyArray<PlanFact> = balances
    ? [
        { label: 'Checking', read: (member) => balances[member.id].checkingCents },
        { label: 'Savings', read: (member) => balances[member.id].savingsCents },
        { label: 'Investments', read: (member) => balances[member.id].investmentsCents },
        { label: 'Health savings', read: (member) => balances[member.id].hsaCents },
        { label: 'Manual debt', read: (member) => member.debtCents },
      ]
    : BALANCE_FACTS;
  return (
    <div>
      {connections.balances ? (
        <FactTable
          members={props.household.members}
          facts={facts}
          label="Balances"
          precise={props.precise}
        />
      ) : (
        <p role="status">
          {connections.loading ? 'Loading balances…' : 'Current balances are not ready.'}
        </p>
      )}
    </div>
  );
}
