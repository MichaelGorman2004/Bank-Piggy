import type { Household } from '@bank-piggy/contracts';
import type { ActivityHeaderProps } from './ActivityHeader';
import type { useActivity } from './useActivity';

export interface ActivityDashboardProps extends Omit<ActivityHeaderProps, 'members'> {
  readonly household: Household;
  readonly onReload: () => void;
  readonly onAccounts: () => void;
  readonly section: ActivitySection;
  readonly onSection: (section: ActivitySection) => void;
}
export type ActivitySection = 'summary' | 'sources';
export type ActivityState = ReturnType<typeof useActivity>;
