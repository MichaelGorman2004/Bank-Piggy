import { useEffect, useRef, useState } from 'react';
import type { Household, HouseholdResponse } from '@bank-piggy/contracts';
import { ApiError, saveHousehold } from '../../api';
import { errorMessage } from './errors';
import type { LoadedState, LoadedStateSetter } from './types';

/**
 * Applies an accepted revision only to the session that submitted it.
 * @param current Latest client state when React processes the response.
 * @param data Accepted plan and its updated totals.
 * @param csrfToken Request token captured when the save started.
 * @returns The newest state with a current viewer name.
 */
function applySavedPlan(
  current: LoadedState | null,
  data: HouseholdResponse,
  csrfToken: string,
): LoadedState | null {
  if (current?.session.csrfToken !== csrfToken) return current;
  if (current.data && current.data.household.revision > data.household.revision) return current;
  const viewer = current.session.viewer;
  const member = data.household.members.find((item) => item.id === viewer?.id);
  const session =
    viewer && member
      ? { ...current.session, viewer: { ...viewer, name: member.name } }
      : current.session;
  return { ...current, data, session };
}

/**
 * Retains save failures and revision conflicts until the user resolves them.
 * @returns Save feedback and explicit reset actions.
 */
function useSaveFeedback() {
  const [saveError, setSaveError] = useState<string | null>(null);
  const [conflict, setConflict] = useState(false);

  /**
   * Shows a failed write without replacing the user's draft.
   * @param failure Caught API or connection error.
   */
  function reportFailure(failure: unknown) {
    console.error('Plan save failed', failure);
    const isConflict = failure instanceof ApiError && failure.status === 409;
    setConflict(isConflict);
    setSaveError(
      isConflict
        ? 'This plan changed in another window. Close this editor and reload the latest plan before editing again.'
        : errorMessage(failure),
    );
  }

  return {
    saveError,
    conflict,
    reportFailure,
    clearSaveError: () => {
      setSaveError(null);
    },
    clearConflict: () => {
      setConflict(false);
    },
  };
}

/**
 * Serializes writes and ignores results belonging to an earlier session or revision.
 * @param loaded The current authenticated household state.
 * @param setLoaded Functional state setter shared with session loading.
 * @returns Saving feedback and the plan write action.
 */
export function usePersistence(loaded: LoadedState | null, setLoaded: LoadedStateSetter) {
  const [saving, setSaving] = useState(false);
  const inFlight = useRef(false);
  const sessionRef = useRef(loaded?.session.csrfToken);
  const feedback = useSaveFeedback();
  useEffect(() => {
    sessionRef.current = loaded?.session.csrfToken;
  }, [loaded?.session.csrfToken]);

  /**
   * Saves one revision while preserving any newer state already loaded.
   * @param household User-edited plan with its original revision.
   * @returns Whether the server accepted the write.
   */
  async function save(household: Household): Promise<boolean> {
    if (!loaded?.session.csrfToken || inFlight.current || feedback.conflict) return false;
    const csrfToken = loaded.session.csrfToken;
    inFlight.current = true;
    setSaving(true);
    feedback.clearSaveError();
    try {
      const data = await saveHousehold(household, csrfToken, loaded.data?.planningMonth);
      setLoaded((current) => applySavedPlan(current, data, csrfToken));
      return true;
    } catch (failure) {
      if (sessionRef.current === csrfToken) feedback.reportFailure(failure);
      return false;
    } finally {
      inFlight.current = false;
      setSaving(false);
    }
  }

  return { ...feedback, saving, save };
}
