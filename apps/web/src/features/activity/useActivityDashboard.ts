import type { LinkTokenRequest } from '@bank-piggy/contracts';
import { useConnectionData } from '../connections/ConnectionsContext';
import { useConnectionLink } from '../connections/useConnectionLink';
import { activityCommands } from './activityCommands';
import { useActivityData } from './ActivityContext';
import type { ActivityDashboardProps } from './activityTypes';

/** Owns one activity request owner and official Link flow.
 * @param props Current saved plan and selected calendar scope.
 * @returns Data, explicit actions, and request availability.
 */
export function useActivityDashboard(props: ActivityDashboardProps) {
  const connections = useConnectionData();
  const state = useActivityData();
  const link = useConnectionLink(connections.csrfToken, connections);
  /** Opens official read-only history permission after saved source opt-in.
   * @param input Server-owned institution identity and explicit permission purpose.
   */
  function authorize(input: LinkTokenRequest) {
    link.show();
    void link.start(input);
  }
  const commands = activityCommands(props, state, connections.csrfToken);
  const blocked =
    state.busy ||
    connections.busy ||
    link.preparing ||
    Boolean(link.link) ||
    Boolean(link.pendingSave);
  return { state, link, commands, blocked, authorize };
}
