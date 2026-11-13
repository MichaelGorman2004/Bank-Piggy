import { Users } from 'lucide-react';
import type { Household, Member, PayrollSummary } from '@bank-piggy/contracts';
import { accountTotals } from '../features/accounts/accountTypes';
import { HOUSEHOLD_SCOPE, money } from '../features/plan/format';
import { PayrollOverview } from './PayrollOverview';
import { FinanceCard } from './FinanceCard';
import styles from './PlanDetails.module.css';
import { BalanceAvailability } from '../features/connections/BalanceAvailability';
import { useConnectionData } from '../features/connections/ConnectionsContext';

interface PlanFact {
  readonly label: string;
  readonly read: (member: Member) => number | string;
}
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
  { label: 'Debt', read: (member) => member.debtCents },
];
const COMPACT_THRESHOLD_CENTS = 100_000_000;

/**
 * Keeps large summary values readable while retaining their exact accessible amount.
 * @param props Canonical fact and member to describe.
 * @param props.fact Named fact and its canonical value reader.
 * @param props.member Member whose value is shown.
 * @returns A value with its complete amount available to assistive tools and hover.
 */
function FactValue(props: { readonly fact: PlanFact; readonly member: Member }) {
  const value = props.fact.read(props.member);
  if (typeof value === 'string') return <span>{value}</span>;
  const exact = money(value);
  const label =
    value >= COMPACT_THRESHOLD_CENTS
      ? new Intl.NumberFormat('en-US', {
          style: 'currency',
          currency: 'USD',
          notation: 'compact',
          maximumFractionDigits: 1,
        }).format(value / 100)
      : exact;
  return (
    <span title={exact} aria-label={exact}>
      {label}
    </span>
  );
}

/**
 * Shows the saved people in three compact rows with native row and column headings.
 * @param props Members, fact readers, and the accessible table name.
 * @param props.members Saved household people.
 * @param props.facts Three related facts to compare.
 * @param props.label Accessible table name and first-column heading.
 * @returns An income or balance table that fits a short card.
 */
function FactTable(props: {
  readonly members: ReadonlyArray<Member>;
  readonly facts: ReadonlyArray<PlanFact>;
  readonly label: string;
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
                <FactValue fact={fact} member={member} />
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
export function PlanDetails(props: {
  readonly household: Household;
  readonly payrollSummaries: readonly PayrollSummary[];
}) {
  const connections = useConnectionData();
  const balances = connections.data?.connections.length ? connections.balances?.members : undefined;
  const balanceFacts: ReadonlyArray<PlanFact> = balances
    ? [
        {
          label: 'Checking',
          read: (member) => balances[member.id].checkingCents,
        },
        { label: 'Savings', read: (member) => balances[member.id].savingsCents },
        {
          label: 'Investments',
          read: (member) => balances[member.id].investmentsCents,
        },
        {
          label: 'Health savings',
          read: (member) => balances[member.id].hsaCents,
        },
        { label: 'Debt (manual)', read: (member) => member.debtCents },
      ]
    : BALANCE_FACTS;
  const split = { salary: 'Base salary', equal: 'Equal shares', custom: 'Custom percentage' };
  return (
    <FinanceCard
      title="Your household at a glance"
      description="Recorded income and balances."
      actions={null}
      footer={
        props.household.members.length > 1 ? (
          <p className={styles.split}>
            <Users size={13} />
            Shared split · {split[props.household.defaultSplitMethod]}
          </p>
        ) : null
      }
    >
      <div className={styles.tables}>
        <FactTable members={props.household.members} facts={INCOME_FACTS} label="Income" />
        <div>
          <BalanceAvailability scope={HOUSEHOLD_SCOPE} />
          <FactTable members={props.household.members} facts={balanceFacts} label="Balances" />
        </div>
      </div>
      <PayrollOverview members={props.household.members} summaries={props.payrollSummaries} />
    </FinanceCard>
  );
}
