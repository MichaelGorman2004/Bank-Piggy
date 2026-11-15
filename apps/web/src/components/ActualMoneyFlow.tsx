import type { ActivityResponse } from '@bank-piggy/contracts';
import { actualMoneyFlowOption } from './actualMoneyFlowOption';
import { actualMoneyFlowDescription, actualMoneyFlowState } from './actualMoneyFlowPresentation';
import { useMoneyFlowRenderer } from './useMoneyFlowRenderer';
import styles from './MoneyFlow.module.css';

/**
 * Shows canonical recorded bank flow without treating it as a balance or complete ledger.
 * @param props Current authorized monthly response.
 * @param props.data Saved bank flow, totals, and source availability.
 * @returns Recorded bank flow with exact accessible totals and explicit exclusions.
 */
export function ActualMoneyFlow({ data }: { readonly data: ActivityResponse }) {
  const state = actualMoneyFlowState(data);
  const ref = useMoneyFlowRenderer(
    data.comparison.bankFlow,
    actualMoneyFlowOption,
    state === 'ready',
  );
  return (
    <div className={styles.wrap}>
      {state === 'ready' ? (
        <div ref={ref} className={styles.chart} aria-hidden="true" />
      ) : (
        <div className={styles.empty}>
          {state === 'unavailable'
            ? 'Unavailable. No selected bank history is available for this month.'
            : 'No net recorded bank flow for this month.'}
        </div>
      )}
      <p className={styles.flowNote}>
        {data.environment === 'sandbox' && 'Sample data. '}
        Selected posted bank activity only. Purchases and refunds are combined. A recorded surplus
        or spending gap does not show an account balance or prove borrowing.
      </p>
      <p className={styles.accessible}>{actualMoneyFlowDescription(data)}</p>
    </div>
  );
}
