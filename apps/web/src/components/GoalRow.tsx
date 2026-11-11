import { MoreHorizontal } from 'lucide-react';
import type { Goal, GoalProjection } from '@bank-piggy/contracts';
import type { CSSProperties } from 'react';
import { money } from '../features/plan/format';
import { GoalTargetProgress } from './GoalTargetProgress';
import styles from './Dashboard.module.css';
import ownStyles from './GoalPanel.module.css';

interface GoalRowProps {
  readonly goal: Goal;
  readonly amount: number;
  readonly total: number;
  readonly factor: number;
  readonly projection?: GoalProjection;
  readonly onEdit: (goal: Goal) => void;
}
interface ProgressStyle extends CSSProperties {
  readonly '--goal-width': string;
}

/**
 * Shows one chosen contribution with a direct edit action.
 * @param props Goal allocation and editing action.
 * @returns Goal name, contribution, and accessible edit button.
 */
function GoalRowHeading(props: GoalRowProps) {
  return (
    <div className={ownStyles.goalRowTop}>
      <span className={ownStyles.goalName}>
        <i className={ownStyles.dot} />
        {props.goal.name}
      </span>
      <span className={ownStyles.goalValue}>
        {money(props.amount, props.factor)}
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
 * Keeps the monthly allocation bar separate from trip or purchase target progress.
 * @param props Goal contribution, budget total, and optional target projection.
 * @returns Goal row with manual target planning when configured.
 */
export function GoalRow(props: GoalRowProps) {
  const style: ProgressStyle = {
    '--goal-width': `${String(props.total > 0 ? (props.amount / props.total) * 100 : 0)}%`,
  };
  return (
    <div className={ownStyles.goalRow}>
      <GoalRowHeading {...props} />
      <div className={ownStyles.goalTrack}>
        <div style={style} className={ownStyles.goalFill} />
      </div>
      <GoalTargetProgress goal={props.goal} projection={props.projection} />
    </div>
  );
}
