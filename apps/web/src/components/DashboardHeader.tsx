import { ChevronDown, Info, Plus, SlidersHorizontal } from 'lucide-react';
import type { Member, Session } from '@bank-piggy/contracts';
import { HOUSEHOLD_SCOPE, type Period } from '../features/plan/format';
import { VIEW_DETAILS, type View } from '../features/plan/views';
import styles from './Dashboard.module.css';
import ownStyles from './DashboardHeader.module.css';

interface DashboardHeaderProps {
  readonly session: Session;
  readonly members: ReadonlyArray<Member>;
  readonly scope: string;
  readonly period: Period;
  readonly view: View;
  readonly onScope: (scope: string) => void;
  readonly onPeriod: (period: Period) => void;
  readonly onAdd: () => void;
}
const PERIOD_LABELS: Record<Period, string> = {
  monthly: 'Month',
  yearly: 'Year',
  paycheck: 'Paycheck',
};

/**
 * Sizes the native period control to its selected label and keeps its arrow inset.
 * @param props Current period, scope, and period action.
 * @returns Native period selector.
 */
function PeriodFilter(props: DashboardHeaderProps) {
  const selected = props.members.find((item) => item.id === props.scope);
  return (
    <span className={`${ownStyles.selectField} ${ownStyles.periodField}`}>
      <span className={ownStyles.selectSizer} aria-hidden="true">
        {PERIOD_LABELS[props.period]}
      </span>
      <select
        aria-label="Plan period"
        className={ownStyles.select}
        value={props.period}
        onChange={(event) => {
          const value = event.target.value;
          if (value === 'monthly' || value === 'yearly' || value === 'paycheck')
            props.onPeriod(value);
        }}
      >
        <option value="monthly">{PERIOD_LABELS.monthly}</option>
        <option value="yearly">{PERIOD_LABELS.yearly}</option>
        <option value="paycheck" disabled={!selected || selected.payFrequency === 'irregular'}>
          {PERIOD_LABELS.paycheck}
        </option>
      </select>
      <ChevronDown className={ownStyles.selectChevron} size={14} aria-hidden="true" />
    </span>
  );
}

/**
 * Provides one consistent household/member scope and estimated plan period.
 * @param props Component values and event handlers.
 * @returns Rendered interface.
 */
function HeaderFilters(props: DashboardHeaderProps) {
  const member = props.members.find((item) => item.id === props.session.viewer?.id);
  return (
    <div className={ownStyles.headerControls}>
      <span className={`${ownStyles.selectField} ${ownStyles.scopeField}`}>
        <select
          aria-label="View household or my share"
          className={ownStyles.select}
          value={props.scope}
          onChange={(event) => {
            props.onScope(event.target.value);
          }}
        >
          <option value={HOUSEHOLD_SCOPE}>Our household</option>
          {member && <option value={member.id}>{member.name}’s view</option>}
        </select>
        <ChevronDown className={ownStyles.selectChevron} size={14} aria-hidden="true" />
      </span>
      {props.view !== 'todos' && <PeriodFilter {...props} />}
    </div>
  );
}

/**
 * Opens the editor associated with the focused workspace view.
 * @param props Current view and editor action.
 * @returns A named primary action with the matching edit or add icon.
 */
function HeaderAction(props: DashboardHeaderProps) {
  const ActionIcon = ['overview', 'flow', 'todos'].includes(props.view) ? SlidersHorizontal : Plus;
  const actionLabel = VIEW_DETAILS[props.view].actionLabel;
  return (
    <button
      className={`${styles.button} ${ownStyles.headerAction}`}
      onClick={props.onAdd}
      aria-label={actionLabel}
    >
      <ActionIcon size={15} />
      <span>{actionLabel}</span>
    </button>
  );
}

/**
 * Names the selected plan and explains every nonmonthly estimate.
 * @param props Component values and event handlers.
 * @returns Rendered interface.
 */
export function DashboardHeader(props: DashboardHeaderProps) {
  const selected = props.members.find((item) => item.id === props.scope);
  return (
    <>
      <header className={ownStyles.header}>
        <div className={ownStyles.heading}>
          <h1>{VIEW_DETAILS[props.view].title}</h1>
          <p>
            {props.view === 'todos'
              ? 'Manual payments and goal contributions for each pay period.'
              : props.view === 'flow'
                ? 'Income and planned allocations.'
                : props.scope === HOUSEHOLD_SCOPE
                  ? `${props.members.map((item) => item.name).join(' & ')}’s household plan`
                  : `${selected?.name ?? 'Your'}’s share of the plan`}
          </p>
        </div>
        <HeaderAction {...props} />
        <HeaderFilters {...props} />
      </header>
      {props.view !== 'todos' && props.period !== 'monthly' && (
        <p className={ownStyles.periodNote}>
          <Info size={12} />
          {props.period === 'yearly'
            ? 'Next 12 months with scheduled payment dates. This is an estimate, not actual activity.'
            : 'Average plan per paycheck, based on your pay frequency. Actual pay dates can vary.'}
        </p>
      )}
    </>
  );
}
