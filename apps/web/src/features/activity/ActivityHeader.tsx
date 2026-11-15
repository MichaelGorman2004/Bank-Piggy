import type { Member, Session } from '@bank-piggy/contracts';
import { PlanningMonth } from '../../components/PlanningMonth';
import { HOUSEHOLD_SCOPE } from '../plan/format';
import styles from './Activity.module.css';

export interface ActivityHeaderProps {
  readonly members: ReadonlyArray<Member>;
  readonly session: Session;
  readonly month: string;
  readonly scope: string;
  readonly monthLoading: boolean;
  readonly onMonth: (month: string) => Promise<boolean>;
  readonly onScope: (scope: string) => void;
}

/**
 * Keeps actual activity on the shared monthly selection without plan period multipliers.
 * @param props Shared month, authorized scope choices, and month read action.
 * @returns Monthly Activity heading with native scope and calendar controls.
 */
export function ActivityHeader(props: ActivityHeaderProps) {
  const viewer = props.members.find((member) => member.id === props.session.viewer?.id);
  return (
    <header className={styles.header}>
      <div>
        <h1>Activity</h1>
        <p>Recorded monthly income, spending, and investment contributions.</p>
      </div>
      <div className={styles.filters}>
        <label className={styles.scope}>
          View
          <select
            value={props.scope}
            onChange={(event) => {
              props.onScope(event.target.value);
            }}
          >
            <option value={HOUSEHOLD_SCOPE}>Our household</option>
            {viewer && <option value={viewer.id}>{viewer.name}’s view</option>}
          </select>
        </label>
        <PlanningMonth
          month={props.month}
          period="monthly"
          disabled={false}
          loading={props.monthLoading}
          onMonth={props.onMonth}
        />
      </div>
    </header>
  );
}
