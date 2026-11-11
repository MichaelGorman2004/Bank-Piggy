import { Pencil, Users } from 'lucide-react';
import type { Household, Member } from '@bank-piggy/contracts';
import { accountTotals } from '../features/accounts/accountTypes';
import { PAY_FREQUENCIES, money } from '../features/plan/format';
import styles from './Dashboard.module.css';
import ownStyles from './DashboardRows.module.css';
import { FinanceCard } from './FinanceCard';

interface PlanDetailsProps {
  readonly household: Household;
  readonly onEdit: () => void;
}

/**
 * Lists one member’s recorded plan inputs.
 * @param root0 Component values and event handlers.
 * @param root0.member Value used by this operation.
 * @returns Rendered interface.
 */
function MemberPlan({ member }: { readonly member: Member }) {
  const frequency =
    PAY_FREQUENCIES.find((item) => item.value === member.payFrequency)?.label ?? 'Monthly';
  return (
    <article className={ownStyles.planMember}>
      <h3>{member.name}</h3>
      <dl>
        <div>
          <dt>Monthly take-home</dt>
          <dd>{money(member.monthlyTakeHomeCents)}</dd>
        </div>
        <div>
          <dt>Annual base salary</dt>
          <dd>{money(member.annualBaseSalaryCents)}</dd>
        </div>
        <div>
          <dt>Pay frequency</dt>
          <dd>{frequency}</dd>
        </div>
        <div>
          <dt>Savings balance</dt>
          <dd>{money(accountTotals(member.accounts).savingsCents)}</dd>
        </div>
        <div>
          <dt>Investments</dt>
          <dd>{money(accountTotals(member.accounts).investmentsCents)}</dd>
        </div>
        <div>
          <dt>Outstanding debt</dt>
          <dd>{money(member.debtCents)}</dd>
        </div>
      </dl>
    </article>
  );
}

/**
 * Explains the inputs that produce the shared plan and its personal allocations.
 * @param root0 Component values and event handlers.
 * @param root0.household Saved household plan.
 * @param root0.onEdit Opens this record for editing.
 * @returns Rendered interface.
 */
export function PlanDetails({ household, onEdit }: PlanDetailsProps) {
  return (
    <FinanceCard
      title="Your household, at a glance"
      description="Manually entered estimates. Update them whenever life changes."
      actions={
        <button className={styles.button} onClick={onEdit}>
          <Pencil size={14} />
          Edit plan
        </button>
      }
    >
      <div className={ownStyles.planMembers}>
        {household.members.map((member) => (
          <MemberPlan member={member} key={member.id} />
        ))}
      </div>
      <div className={styles.goalNote}>
        <Users size={14} />
        <span>
          New shared expenses use{' '}
          {household.defaultSplitMethod === 'salary'
            ? 'annual base salary'
            : household.defaultSplitMethod === 'custom'
              ? 'a custom percentage'
              : 'an even split'}
          . Existing expenses keep their agreed split. The person who pays does not change each
          member’s share.
        </span>
      </div>
    </FinanceCard>
  );
}
