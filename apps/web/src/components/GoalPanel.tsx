import { ArrowUpRight, Check, Plus, Sprout, TriangleAlert } from 'lucide-react';
import type { Goal, HouseholdResponse, Summary } from '@bank-piggy/contracts';
import type { CSSProperties } from 'react';
import { HOUSEHOLD_SCOPE, money } from '../features/plan/format';
import { goalsForView } from '../features/plan/viewRecords';
import type { GoalView } from '../features/plan/views';
import styles from './Dashboard.module.css';
import ownStyles from './GoalPanel.module.css';
import { GoalRow } from './GoalRow';
import { FinanceCard } from './FinanceCard';

interface GoalPanelProps {
  readonly data: HouseholdResponse;
  readonly summary: Summary;
  readonly scope: string;
  readonly factor: number;
  readonly view: GoalView;
  readonly onEdit: (goal: Goal) => void;
  readonly onAdd: () => void;
}
interface ProgressStyle extends CSSProperties {
  readonly '--goal-fill': string;
}

/**
 * Gives each empty contribution view a matching action.
 * @param props Focused goal view and editor action.
 * @returns A short prompt to create the first goal or target plan.
 */
function EmptyGoals(props: Pick<GoalPanelProps, 'view' | 'onAdd'>) {
  const targets = props.view === 'targets';
  return (
    <div className={styles.empty}>
      <p>
        {targets
          ? 'Give your next trip or purchase a plan.'
          : 'Big plans start with a little each month.'}
      </p>
      <button className={styles.textButton} onClick={props.onAdd}>
        {targets ? 'Add your first plan' : 'Add your first goal'}
        <ArrowUpRight size={14} />
      </button>
    </div>
  );
}

/**
 * Builds the goal amount view.
 * @param goal Canonical goal record.
 * @param data Validated household response.
 * @param scope Selected household or member identifier.
 * @returns The computed result.
 */
function goalAmount(goal: Goal, data: HouseholdResponse, scope: string): number {
  const share = data.summary.goalShares.find((item) => item.goalId === goal.id);
  if (!share) return 0;
  if (scope === HOUSEHOLD_SCOPE) return share.firstMemberCents + share.secondMemberCents;
  return scope === data.household.members[0]?.id ? share.firstMemberCents : share.secondMemberCents;
}

/**
 * Explains whether planned contributions fit available income.
 * @param root0 Component values and event handlers.
 * @param root0.summary Server-calculated totals.
 * @param root0.factor Scale for the selected estimated period.
 * @returns Rendered interface.
 */
function GoalFit({ summary, factor }: { readonly summary: Summary; readonly factor: number }) {
  const shortfall = summary.monthlyRemainingCents < 0;
  const Icon = shortfall ? TriangleAlert : Check;
  return (
    <p className={`${styles.goalNote} ${shortfall ? styles.negative : ''}`}>
      <Icon size={13} />
      {shortfall
        ? `Your plan is ${money(-summary.monthlyRemainingCents, factor)} over available income. Adjust expenses or contributions.`
        : `All contributions fit, with ${money(summary.monthlyRemainingCents, factor)} still left to plan.`}
    </p>
  );
}

/**
 * Explains how planned goals fit inside income after expenses.
 * @param root0 Component values and event handlers.
 * @param root0.summary Server-calculated totals.
 * @param root0.factor Scale for the selected estimated period.
 * @param root0.amountCents Contributions in this focused view.
 * @returns Rendered interface.
 */
function GoalIntro({
  summary,
  factor,
  amountCents,
}: Pick<GoalPanelProps, 'summary' | 'factor'> & { readonly amountCents: number }) {
  const percentage =
    summary.monthlyAvailableCents > 0
      ? Math.min(100, (amountCents / summary.monthlyAvailableCents) * 100)
      : 0;
  const style: ProgressStyle = { '--goal-fill': `${String(percentage)}%` };
  return (
    <div className={ownStyles.goalIntro}>
      <div className={ownStyles.goalRing} style={style}>
        <div className={ownStyles.goalRingInner}>
          <Sprout size={24} />
        </div>
      </div>
      <div>
        <strong>{money(amountCents, factor)}</strong>
        <p>of {money(summary.monthlyAvailableCents, factor)} available after expenses</p>
      </div>
    </div>
  );
}

/**
 * Rows pair canonical planned contributions with optional manual saved-to-target progress.
 * @param root0 Component values and event handlers.
 * @param root0.data Validated household response.
 * @param root0.summary Server-calculated totals.
 * @param root0.scope Selected household or member identifier.
 * @param root0.factor Scale for the selected estimated period.
 * @param root0.view Selected long-term goal or target view.
 * @param root0.onEdit Opens this record for editing.
 * @param root0.onAdd Opens a new record editor.
 * @returns Rendered interface.
 */
export function GoalPanel({ data, summary, scope, factor, view, onEdit, onAdd }: GoalPanelProps) {
  const goals = goalsForView(data.household.goals, scope, view);
  const amountCents = goals.reduce((total, goal) => total + goalAmount(goal, data, scope), 0);
  const targets = view === 'targets';
  const addLabel = targets ? 'Add plan' : 'Add goal';
  return (
    <FinanceCard
      title={targets ? 'Trips & purchases' : 'Savings & goals'}
      description={
        targets
          ? 'Save for a target, at your pace.'
          : 'Planned contributions, not account balances.'
      }
      actions={
        <button className={styles.iconButton} aria-label={addLabel} onClick={onAdd}>
          <Plus size={16} />
        </button>
      }
      footer={<GoalFit summary={summary} factor={factor} />}
    >
      <GoalIntro summary={summary} factor={factor} amountCents={amountCents} />
      {goals.length ? (
        goals.map((goal) => (
          <GoalRow
            key={goal.id}
            goal={goal}
            memberCount={data.household.members.length}
            projection={(goal.purchase ? data.purchaseProjections : data.tripProjections).find(
              (item) => item.goalId === goal.id,
            )}
            amount={goalAmount(goal, data, scope)}
            factor={factor}
            onEdit={onEdit}
          />
        ))
      ) : (
        <EmptyGoals view={view} onAdd={onAdd} />
      )}
    </FinanceCard>
  );
}
