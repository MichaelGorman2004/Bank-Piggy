import type { Household } from '@bank-piggy/contracts';
import type { WorkspaceProps } from './Workspace';
import type { useWorkspace } from './useWorkspace';
import { useSharing } from './features/sharing/useSharing';

/**
 * Offers a partner link only after an explicit successful Add person save.
 * @param props Current household and authorized financial save action.
 * @param ui Workspace draft-dialog state.
 * @returns Independent invitation state and a financial save action.
 */
export function useWorkspaceSharing(props: WorkspaceProps, ui: ReturnType<typeof useWorkspace>) {
  const sharing = useSharing(props.session);
  /**
   * Saves finances first and keeps an invitation failure separate from that accepted save.
   * @param household Validated financial draft.
   * @returns Whether the financial plan was accepted.
   */
  async function save(household: Household): Promise<boolean> {
    const addedPartner =
      props.data.household.members.length === 1 && household.members.length === 2;
    const accepted = await props.onSave(household);
    if (accepted && addedPartner) {
      ui.setActive(null);
      await sharing.show(true);
    }
    return accepted;
  }
  return { ...sharing, save };
}
