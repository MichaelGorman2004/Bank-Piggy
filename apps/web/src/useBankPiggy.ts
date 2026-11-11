import { logout } from './api';
import { useLoadedState } from './features/session/use-loaded-state';
import { usePersistence } from './features/session/use-persistence';
import { errorMessage } from './features/session/errors';

/**
 * Owns the current session and persisted plan, including safe logout and reload.
 * @returns The session, plan, request feedback, and authorized user actions.
 */
export function useBankPiggy() {
  const state = useLoadedState();
  const persistence = usePersistence(state.loaded, state.setLoaded);

  /** Revokes the server session before removing the visible household data. */
  async function signout() {
    if (!state.loaded?.session.csrfToken) return;
    try {
      await logout(state.loaded.session.csrfToken);
      state.setLoaded(null);
      await state.reload();
    } catch (failure) {
      console.error('Sign out failed', failure);
      state.setError(errorMessage(failure));
    }
  }

  /** Clears a revision conflict only after the latest plan loads successfully. */
  async function reload() {
    if (await state.reload()) {
      persistence.clearConflict();
      persistence.clearSaveError();
    }
  }

  return { ...state, ...persistence, reload, signout };
}
