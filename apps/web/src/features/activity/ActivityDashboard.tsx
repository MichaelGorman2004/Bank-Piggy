import { FinanceCard } from '../../components/FinanceCard';
import { ActivityHeader } from './ActivityHeader';
import { ActivityLink } from './ActivityLink';
import { ActivityBody } from './ActivityBody';
import { ActivityControls, ActivityFeedback, ActivityFooter } from './ActivityChrome';
import type { ActivityDashboardProps } from './activityTypes';
import { useActivityDashboard } from './useActivityDashboard';
import styles from './Activity.module.css';
import { monthLabel } from '../plan/month';

/**
 * Keeps monthly totals, source choices, and one permission flow in a bounded card.
 * @param props Saved plan identity and the shared scope and month controls.
 * @returns Monthly actual activity with body-owned scrolling.
 */
export function ActivityDashboard(props: ActivityDashboardProps) {
  const { state, link, commands, blocked, authorize } = useActivityDashboard(props);
  const { section, onSection } = props;
  return (
    <div className={styles.page}>
      <ActivityHeader {...props} members={props.household.members} />
      <FinanceCard
        title="Monthly actuals"
        description={`${monthLabel(props.month)} · Recorded activity stays separate from your plan`}
        actions={null}
        controls={
          <ActivityControls
            section={section}
            onSection={onSection}
            state={state}
            blocked={blocked}
          />
        }
        footer={<ActivityFooter />}
      >
        <ActivityFeedback state={state} />
        {state.snapshot && (
          <div hidden={!state.data}>
            <ActivityBody
              data={state.snapshot}
              dashboard={props}
              section={section}
              commands={commands}
              blocked={blocked}
              totalsStale={state.totalsStale}
              onAuthorize={authorize}
            />
          </div>
        )}
      </FinanceCard>
      <ActivityLink link={link} />
    </div>
  );
}
