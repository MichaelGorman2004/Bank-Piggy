import { useState } from 'react';
import {
  Car,
  CreditCard,
  Home,
  MoreHorizontal,
  Plus,
  ShoppingBasket,
  Utensils,
  Zap,
} from 'lucide-react';
import type { Expense, HouseholdResponse } from '@bank-piggy/contracts';
import { HOUSEHOLD_SCOPE, expenseAmount, money } from '../features/plan/format';
import { expensesForView, type PaymentFilter } from '../features/plan/viewRecords';
import styles from './Dashboard.module.css';
import ownStyles from './DashboardRows.module.css';
import { ProviderIcon } from '../features/providers/ProviderIcon';
import { PaymentMeta } from './PaymentMeta';
import { PaymentFilters } from './PaymentFilters';
import { FinanceCard } from './FinanceCard';

interface ExpenseListProps {
  readonly data: HouseholdResponse;
  readonly scope: string;
  readonly factor: number;
  readonly onEdit: (expense: Expense) => void;
  readonly onAdd: (kind?: Expense['kind']) => void;
}
const CATEGORY_ICONS: Record<string, typeof Home> = {
  Housing: Home,
  Groceries: ShoppingBasket,
  Food: ShoppingBasket,
  Transport: Car,
  Dining: Utensils,
  Lifestyle: Utensils,
  Bills: Zap,
  Utilities: Zap,
  Debt: CreditCard,
  'Debt payments': CreditCard,
};

/**
 * Shows one expense and its agreed allocation.
 * @param root0 Component values and event handlers.
 * @param root0.expense Canonical expense record.
 * @param root0.data Validated household response.
 * @param root0.scope Selected household or member identifier.
 * @param root0.factor Scale for the selected estimated period.
 * @param root0.onEdit Opens this record for editing.
 * @returns Rendered interface.
 */
function ExpenseRow({
  expense,
  data,
  scope,
  factor,
  onEdit,
}: Omit<ExpenseListProps, 'onAdd'> & { readonly expense: Expense }) {
  const Icon = CATEGORY_ICONS[expense.category] ?? CreditCard;
  const amount = expenseAmount(expense, data, scope);
  return (
    <div className={ownStyles.expenseRow}>
      <div className={ownStyles.categoryIcon}>
        {expense.providerId || expense.kind ? (
          <ProviderIcon providerId={expense.providerId} kind={expense.kind} size={17} />
        ) : (
          <Icon size={17} />
        )}
      </div>
      <div>
        <p className={ownStyles.rowName}>{expense.name}</p>
        <PaymentMeta expense={expense} data={data} />
      </div>
      <div>
        <p className={ownStyles.rowAmount}>{money(amount, factor)}</p>
        {scope !== HOUSEHOLD_SCOPE && <small className={ownStyles.rowNote}>Your share</small>}
      </div>
      <button
        className={styles.iconButton}
        aria-label={`Edit ${expense.name}`}
        onClick={() => {
          onEdit(expense);
        }}
      >
        <MoreHorizontal size={15} />
      </button>
    </div>
  );
}

/**
 * Gives an empty expense plan one direct next action.
 * @param root0 Component values and event handlers.
 * @param root0.filter Selected payment type or all types.
 * @param root0.onAdd Opens a new record editor.
 * @returns Rendered interface.
 */
function EmptyExpenses({
  filter,
  onAdd,
}: {
  readonly filter: PaymentFilter;
  readonly onAdd: () => void;
}) {
  const subscription = filter === 'subscription';
  return (
    <div className={styles.empty}>
      <ShoppingBasket size={27} />
      <p>
        {subscription
          ? 'Add the services you pay for each month.'
          : 'Add regular payments and everyday spending to build a useful plan.'}
      </p>
      <button className={styles.button} onClick={onAdd}>
        <Plus size={14} />
        {subscription ? 'Add subscription' : 'Add expense'}
      </button>
    </div>
  );
}

/**
 * Shows recurring estimates with their agreed split; edits always target the whole expense.
 * @param root0 Component values and event handlers.
 * @param root0.data Validated household response.
 * @param root0.scope Selected household or member identifier.
 * @param root0.factor Scale for the selected estimated period.
 * @param root0.onEdit Opens this record for editing.
 * @param root0.onAdd Opens a new record editor.
 * @returns Rendered interface.
 */
export function ExpenseList({ data, scope, factor, onEdit, onAdd }: ExpenseListProps) {
  const [filter, setFilter] = useState<PaymentFilter>('all');
  const expenses = expensesForView(data, scope, filter);
  const subscription = filter === 'subscription';
  const addLabel = subscription ? 'Add subscription' : 'Add expense';
  /** Opens a draft with the selected payment type, without changing saved records. */
  function addExpense() {
    onAdd(filter === 'all' ? 'bill' : filter);
  }
  return (
    <FinanceCard
      title="Expenses"
      description={
        scope === HOUSEHOLD_SCOPE
          ? 'The household total, counted once.'
          : 'Your agreed share of the household plan.'
      }
      actions={
        <button className={styles.iconButton} aria-label={addLabel} onClick={addExpense}>
          <Plus size={16} />
        </button>
      }
      controls={<PaymentFilters value={filter} onChange={setFilter} />}
    >
      {expenses.length === 0 ? (
        <EmptyExpenses filter={filter} onAdd={addExpense} />
      ) : (
        expenses.map((expense) => (
          <ExpenseRow
            key={expense.id}
            expense={expense}
            data={data}
            scope={scope}
            factor={factor}
            onEdit={onEdit}
          />
        ))
      )}
    </FinanceCard>
  );
}
