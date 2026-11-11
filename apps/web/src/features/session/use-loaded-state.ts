import {
  useEffect,
  useRef,
  useState,
  type Dispatch,
  type RefObject,
  type SetStateAction,
} from 'react';
import { loadHousehold, loadSession, startDemo } from '../../api';
import { errorMessage } from './errors';
import type { LoadedState, LoadedStateSetter } from './types';
import { locationMonth, rememberMonth } from './planning-month';

/**
 * Reads the plan only after the server confirms the current identity.
 * @param month Optional selected projection month.
 * @returns A session and its authorized household, when signed in.
 */
async function readState(month = locationMonth()): Promise<LoadedState> {
  const session = await loadSession();
  return { session, data: session.authenticated ? await loadHousehold(month) : null };
}

/**
 * Loads the initial session and ignores results after unmount or a newer request.
 * @param setLoaded Saved session state setter.
 * @param setLoading Request progress setter.
 * @param setError Request feedback setter.
 * @param sequenceRef Generation shared with later reload requests.
 */
function useInitialState(
  setLoaded: LoadedStateSetter,
  setLoading: Dispatch<SetStateAction<boolean>>,
  setError: Dispatch<SetStateAction<string | null>>,
  sequenceRef: RefObject<number>,
) {
  useEffect(() => {
    const version = ++sequenceRef.current;
    void readState()
      .then((result) => {
        if (version === sequenceRef.current) setLoaded(result);
      })
      .catch((failure: unknown) => {
        console.error('Session load failed', failure);
        if (version === sequenceRef.current) setError(errorMessage(failure));
      })
      .finally(() => {
        if (version === sequenceRef.current) setLoading(false);
      });
    return () => {
      sequenceRef.current += 1;
    };
  }, [setLoaded, setLoading, setError, sequenceRef]);
}

/**
 * Loads server state with a retry path and protection against stale responses.
 * @returns The current data, feedback, and loading actions.
 */
export function useLoadedState() {
  const [loaded, setLoaded] = useState<LoadedState | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const sequenceRef = useRef(0);
  const monthRef = useRef<string | undefined>(undefined);
  useInitialState(setLoaded, setLoading, setError, sequenceRef);
  useEffect(() => {
    monthRef.current = loaded?.data?.planningMonth;
    rememberMonth(monthRef.current);
  }, [loaded]);

  /**
   * Replaces state only if this is still the newest read.
   * @param enterDemo Whether to open the local session first, default false.
   * @param month Calendar month to preview; retains the current selection when absent.
   * @returns Whether the newest state loaded successfully.
   */
  async function reload(enterDemo = false, month = monthRef.current): Promise<boolean> {
    const version = ++sequenceRef.current;
    setLoading(true);
    setError(null);
    try {
      if (enterDemo) await startDemo();
      const next = await readState(month);
      if (version !== sequenceRef.current) return false;
      monthRef.current = next.data?.planningMonth;
      setLoaded(next);
      return true;
    } catch (failure) {
      console.error('Household load failed', failure);
      if (version === sequenceRef.current) setError(errorMessage(failure));
      return false;
    } finally {
      if (version === sequenceRef.current) setLoading(false);
    }
  }

  return {
    loaded,
    setLoaded,
    loading,
    error,
    setError,
    reload,
    demo: () => reload(true),
    changeMonth: (month: string) => reload(false, month),
  };
}
