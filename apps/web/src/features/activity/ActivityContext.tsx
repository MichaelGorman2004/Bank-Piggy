import { createContext, useCallback, useContext, type ReactNode } from 'react';
import { useConnectionData } from '../connections/ConnectionsContext';
import { useActivity } from './useActivity';

type ActivityData = ReturnType<typeof useActivity>;
const Context = createContext<ActivityData | null>(null);

/** Shares one monthly activity reader and mutation owner across reports and source forms.
 * @param props Saved month, scope, plan revision, and the financial workspace.
 * @param props.month Selected calendar month.
 * @param props.scope Household or personal share.
 * @param props.revision Current saved plan revision.
 * @param props.onReloadPlan Reads the current saved plan alongside an explicit activity reload.
 * @param props.children Views that consume recorded activity.
 * @returns Workspace views with one activity refresh lifecycle.
 */
export function ActivityProvider(props: {
  readonly month: string;
  readonly scope: string;
  readonly revision: number;
  readonly onReloadPlan: () => void;
  readonly children: ReactNode;
}) {
  const connections = useConnectionData();
  const connectionVersion =
    connections.data?.connections
      .map((item) => `${item.id}:${item.lastCompletedAttemptId ?? ''}`)
      .join('|') ?? '';
  const state = useActivity(
    { month: props.month, scope: props.scope },
    props.revision,
    connectionVersion,
  );
  const { onReloadPlan } = props;
  const readActivity = state.reload;
  const reload = useCallback(() => {
    onReloadPlan();
    return readActivity();
  }, [onReloadPlan, readActivity]);
  const totalsStale =
    state.totalsStale || Boolean(state.data && state.data.planRevision !== props.revision);
  return <Context value={{ ...state, reload, totalsStale }}>{props.children}</Context>;
}

/** Reads the shared monthly snapshot without starting a second refresh lifecycle.
 * @returns Current activity, mutation serialization, and safe request feedback.
 */
export function useActivityData(): ActivityData {
  const state = useContext(Context);
  if (!state) throw new Error('Activity views require their household provider.');
  return state;
}
