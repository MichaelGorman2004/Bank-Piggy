import { MoreHorizontal } from 'lucide-react';
import type { Goal, GoalProjection } from '@bank-piggy/contracts';
import { money } from '../features/plan/format';
import { goalProgress } from '../features/plan/goalProgress';
import { GoalTargetProgress } from './GoalTargetProgress';
import styles from './Dashboard.module.css';
import ownStyles from './GoalPanel.module.css';

interface GoalRowProps {
  readonly goal: Goal;
  readonly memberCount: number;
  readonly amount: number;
  readonly factor: number;
  readonly projection?: GoalProjection;
  readonly onEdit: (goal: Goal) => void;
}

/**
 * Shows one chosen contribution with a direct edit action.
 * @param props Goal allocation and editing action.
 * @returns Goal name, contribution, and accessible edit button.
 */
function GoalRowHeading(props: GoalRowProps) {
  const progress = goalProgress(props.goal, props.projection);
  return (
    <div className={ownStyles.goalRowTop}>
      <span className={ownStyles.goalName}>
        <i className={ownStyles.dot} />
        {props.goal.name}
      </span>
      <span className={ownStyles.goalValue}>
        {progress
          ? `${money(progress.savedAmountCents)} of ${money(progress.targetAmountCents)}`
          : money(props.amount, props.factor)}
        <button
          className={styles.iconButton}
          aria-label={`Edit ${props.goal.name}`}
          onClick={() => {
            props.onEdit(props.goal);
          }}
        >
          <MoreHorizontal size={14} />
        </button>
      </span>
    </div>
  );
}

/**
 * Shows manual saved-to-target progress separately from the chosen contribution.
 * @param props Scoped contribution, editing action, and optional target projection.
 * @returns Goal row with manual target planning when configured.
 */
export function GoalRow(props: GoalRowProps) {
  const progress = goalProgress(props.goal, props.projection);
  const progressLabel = props.goal.contributionYear
    ? `contributed for ${String(props.goal.contributionYear)}`
    : 'saved';
  return (
    <div className={ownStyles.goalRow}>
      <GoalRowHeading {...props} />
      <p className={ownStyles.contribution}>
        {progress
          ? `${money(props.amount, props.factor)} planned contribution`
          : 'Planned contribution'}
        {props.goal.contributionYear && ` · ${String(props.goal.contributionYear)} Roth IRA`}
        {progress && props.memberCount > 1 && props.goal.ownerId === 'shared' && ' · Shared target'}
      </p>
      {progress && (
        <progress
          className={ownStyles.goalTrack}
          aria-label={`${props.goal.name} ${progressLabel} toward target`}
          aria-valuetext={`${money(progress.savedAmountCents)} of ${money(progress.targetAmountCents)} ${progressLabel}`}
          max={progress.max}
          value={progress.value}
        />
      )}
      <GoalTargetProgress
        goal={props.goal}
        memberCount={props.memberCount}
        projection={props.projection}
      />
    </div>
  );
}
