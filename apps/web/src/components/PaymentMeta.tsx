import { Users } from 'lucide-react';
import type { Expense, HouseholdResponse } from '@bank-piggy/contracts';
import { splitLabel } from '../features/plan/format';
import { monthLabel } from '../features/plan/month';
import { paymentStatus } from '../features/plan/paymentDraft';
import styles from './DashboardRows.module.css';

interface PaymentMetaProps {
  readonly expense: Expense;
  readonly data: HouseholdResponse;
}

/**
 * Shows the saved split and keeps ended or upcoming records understandable.
 * @param props Expense metadata and selected planning month.
 * @returns Allocation and inclusive schedule labels.
 */
export function PaymentMeta(props: PaymentMetaProps) {
  const { expense, data } = props;
  const status = paymentStatus(expense, data.planningMonth);
  return (
    <>
      <p className={styles.rowMeta}>
        <Users />
        {splitLabel(expense, data.household.members)}
        <span>·</span>
        {expense.category}
      </p>
      {(expense.startMonth || expense.endMonth || expense.kind) && (
        <p className={styles.rowNote}>
          {status}
          {expense.startMonth ? ` · From ${monthLabel(expense.startMonth)}` : ''}
          {expense.endMonth ? ` · Last payment ${monthLabel(expense.endMonth)}` : ' · Ongoing'}
        </p>
      )}
    </>
  );
}
