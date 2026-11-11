import { ArrowDownLeft, ArrowUpRight, PiggyBank, Wallet } from 'lucide-react';
import type { Summary } from '@bank-piggy/contracts';
import { money } from '../features/plan/format';
import styles from './Dashboard.module.css';
import ownStyles from './SummaryCards.module.css';

interface SummaryCardsProps {
  readonly summary: Summary;
  readonly factor: number;
}
interface MetricProps {
  readonly label: string;
  readonly value: number;
  readonly note: string;
  readonly factor: number;
  readonly icon: typeof Wallet;
  readonly accent?: boolean;
}

/**
 * Displays one canonical financial total.
 * @param root0 Component values and event handlers.
 * @param root0.label Visible field label.
 * @param root0.value Current display value.
 * @param root0.note Explanation of the amount.
 * @param root0.factor Scale for the selected estimated period.
 * @param root0.icon Value used by this operation.
 * @param root0.accent Whether to emphasize the remaining amount.
 * @returns Rendered interface.
 */
function Metric({ label, value, note, factor, icon: Icon, accent = false }: MetricProps) {
  return (
    <article className={`${ownStyles.metric} ${accent ? ownStyles.metricAccent : ''}`}>
      <div className={ownStyles.metricTitle}>
        {label}
        <Icon size={16} />
      </div>
      <strong className={value < 0 ? styles.negative : undefined}>{money(value, factor)}</strong>
      <small>{note}</small>
    </article>
  );
}

/**
 * Uses one canonical server summary for every total in the selected scope.
 * @param root0 Component values and event handlers.
 * @param root0.summary Server-calculated totals.
 * @param root0.factor Scale for the selected estimated period.
 * @returns Rendered interface.
 */
export function SummaryCards({ summary, factor }: SummaryCardsProps) {
  return (
    <section className={ownStyles.summary} aria-label="Plan totals">
      <Metric
        label="Money in"
        value={summary.monthlyTakeHomeCents}
        note="Planned take-home pay"
        factor={factor}
        icon={ArrowDownLeft}
      />
      <Metric
        label="Living expenses"
        value={summary.monthlyExpensesCents}
        note="Bills, spending & minimum payments"
        factor={factor}
        icon={ArrowUpRight}
      />
      <Metric
        label="For your goals"
        value={summary.monthlyGoalsCents}
        note="Planned contributions"
        factor={factor}
        icon={PiggyBank}
      />
      <Metric
        label={summary.monthlyRemainingCents < 0 ? 'Plan shortfall' : 'Left to plan'}
        value={summary.monthlyRemainingCents}
        note="After expenses & goal contributions"
        factor={factor}
        icon={Wallet}
        accent
      />
    </section>
  );
}
