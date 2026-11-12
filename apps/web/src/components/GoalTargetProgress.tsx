import { Plane, ShoppingBag } from 'lucide-react';
import type { Goal, GoalProjection, TripBudget } from '@bank-piggy/contracts';
import { money } from '../features/plan/format';
import { monthLabel } from '../features/plan/month';
import { TRIP_CATEGORIES } from '../features/plan/goalDraft';
import styles from './GoalTargetProgress.module.css';

interface GoalTargetProgressProps {
  readonly goal: Goal;
  readonly memberCount: number;
  readonly projection?: GoalProjection;
}

/**
 * Explains deadline behavior with the exact server-calculated saving periods.
 * @param projection Server-owned target and monthly saving suggestion.
 * @returns A transparent saving instruction or funded/deadline status.
 */
function savingSuggestion(projection: GoalProjection): string {
  if (projection.status === 'funded')
    return 'Target funded. Your monthly contribution stays under your control.';
  if (projection.status === 'overdue')
    return 'The target month has passed. Update the date or saved amount; no future monthly suggestion is calculated.';
  return `Suggested: ${money(projection.suggestedMonthlyCents)} per month across ${String(projection.monthsRemaining)} monthly saving periods, including the planning month and target month.`;
}

/**
 * Keeps the itemized travel budget available alongside trip progress.
 * @param props Saved trip category costs.
 * @param props.trip Validated trip budget with category costs.
 * @returns Expandable travel cost breakdown.
 */
function TripCostBreakdown(props: { readonly trip: TripBudget }) {
  return (
    <details>
      <summary>Trip cost breakdown</summary>
      <dl>
        {props.trip.costs.map((cost) => (
          <div key={cost.category}>
            <dt>
              {TRIP_CATEGORIES.find((item) => item.category === cost.category)?.label ??
                cost.category}
            </dt>
            <dd>{money(cost.amountCents)}</dd>
          </div>
        ))}
      </dl>
    </details>
  );
}

/**
 * Presents manual target progress independently from monthly goal allocations.
 * @param props Goal with saved progress and a server-calculated target projection.
 * @returns Target, remaining savings, deadline, and optional travel costs.
 */
export function GoalTargetProgress(props: GoalTargetProgressProps) {
  const { goal, projection } = props;
  const details = goal.trip ?? goal.purchase;
  if (!details || !projection) return null;
  const target = projection.targetAmountCents;
  const label = goal.trip ? 'Trip budget' : 'Purchase target';
  return (
    <div className={styles.target}>
      <p className={styles.title}>
        {goal.trip ? <Plane size={14} /> : <ShoppingBag size={14} />}
        {props.memberCount > 1 && goal.ownerId === 'shared'
          ? `Shared ${label.toLowerCase()}`
          : label}{' '}
        · {monthLabel(details.targetMonth)}
      </p>
      <p>
        {money(details.savedAmountCents)} saved of {money(target)} ·{' '}
        {money(projection.remainingAmountCents)} left
      </p>
      <p>{savingSuggestion(projection)}</p>
      {goal.trip && <TripCostBreakdown trip={goal.trip} />}
    </div>
  );
}
