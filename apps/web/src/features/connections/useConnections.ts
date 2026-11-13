import { useCallback, useEffect, useRef, useState } from 'react';
import type { ConnectionList } from '@bank-piggy/contracts';
import { loadConnections } from './api';
import {
  connectionMutation,
  connectionRead,
  type ConnectionFeedback,
  type ConnectionReadState,
} from './connectionRequests';

/**
 * Owns authoritative balance snapshots independently of editable household drafts.
 * Reads and mutations cannot replace newer results or clear a failed account draft.
 * @param revision Saved household revision used to refresh manual reconciliation.
 * @returns Connection data, safe feedback, and serialized request operations.
 */
export function useConnections(revision: number) {
  const snapshot = useConnectionRead(revision);
  const [busy, setBusy] = useState(false);
  const [feedback, setFeedback] = useState<string | null>(null);
  const options = { ...snapshot, setBusy, setFeedback };
  const { read } = snapshot;
  const reload = useCallback(() => read(), [read]);
  return {
    data: snapshot.data,
    balances: snapshot.data?.planRevision === revision ? snapshot.data.effectiveBalances : null,
    loading: snapshot.loading,
    error: snapshot.error,
    busy,
    feedback,
    setError: snapshot.setError,
    setFeedback,
    reload,
    run: (operation: () => Promise<ConnectionList>, message: ConnectionFeedback) =>
      connectionMutation(options, operation, message),
  };
}

/**
 * Reads snapshots after plan changes and defers reads while a mutation owns the request state.
 * @param revision Saved household revision.
 * @returns Current projection and request ownership shared with mutation orchestration.
 */
function useConnectionRead(revision: number) {
  const [data, setData] = useState<ConnectionList | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const generation = useRef(0);
  const lock = useRef(false);
  const pendingRead = useRef(false);
  const mounted = useRef(true);
  const reads = useRef<ConnectionReadState>({ active: null, queued: null });
  const read = useCallback(
    (preservedError?: string | null, fresh = false) =>
      connectionRead(
        { generation, mounted, lock, pendingRead, reads, setData, setError, setLoading },
        loadConnections,
        preservedError,
        fresh,
      ),
    [],
  );
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
      generation.current += 1;
    };
  }, []);
  useEffect(() => {
    void read(undefined, true);
  }, [revision, read]);
  const ownership = { generation, mounted, lock, pendingRead, reads };
  const setters = { setLoading, setError, setData };
  return {
    data,
    loading,
    error,
    read,
    ...ownership,
    ...setters,
  };
}
