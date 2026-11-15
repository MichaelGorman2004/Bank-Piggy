import { useMemo } from 'react';
import { money } from '../features/plan/format';
import { moneyFlowOption, type MoneyFlowProps } from './moneyFlowOption';
import { useMoneyFlowRenderer } from './useMoneyFlowRenderer';
import styles from './MoneyFlow.module.css';

/**
 * Offers the same financial totals as accessible text outside the visual chart.
 * @param props Component values and event handlers.
 * @returns Rendered interface.
 */
function FlowDescription(props: MoneyFlowProps) {
  const { summary, factor } = props;
  return (
    <p className={styles.accessible}>
      Planned income {money(summary.monthlyTakeHomeCents, factor)}. Expenses{' '}
      {money(summary.monthlyExpensesCents, factor)}. Goals{' '}
      {money(summary.monthlyGoalsCents, factor)}.{' '}
      {summary.monthlyRemainingCents < 0 ? 'Shortfall' : 'Left to plan'}{' '}
      {money(Math.abs(summary.monthlyRemainingCents), factor)}. Each stream's width shows its
      amount.
    </p>
  );
}

/**
 * Shows server-calculated income and allocations; shortfalls remain explicit.
 * @param props Component values and event handlers.
 * @returns Rendered interface.
 */
export function MoneyFlow(props: MoneyFlowProps) {
  const empty =
    props.summary.monthlyTakeHomeCents +
      props.summary.monthlyExpensesCents +
      props.summary.monthlyGoalsCents ===
    0;
  const { data, summary, scope, factor } = props;
  const input = useMemo(() => ({ data, summary, scope, factor }), [data, summary, scope, factor]);
  const ref = useMoneyFlowRenderer(input, moneyFlowOption, !empty);
  return (
    <div className={styles.wrap}>
      {empty ? (
        <div className={styles.empty}>
          Add your take-home pay and expenses to see your money flow.
        </div>
      ) : (
        <div ref={ref} className={styles.chart} aria-hidden="true" />
      )}
      <FlowDescription {...props} />
    </div>
  );
}
