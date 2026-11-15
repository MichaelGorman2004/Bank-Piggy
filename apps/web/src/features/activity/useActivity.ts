import { useCallback, useEffect, useRef, useState } from 'react';
import type { ActivityResponse, ActivitySource } from '@bank-piggy/contracts';
import { startAccountStatusUpdates } from '../connections/useAccountStatusUpdates';
import { loadActivity } from './api';
import {
  acceptActivitySource,
  mutateActivity,
  readActivity,
  type ActivityOwner,
  type ActivityQuery,
} from './activityRequests';

/**
 * Owns monthly activity reads and serialized writes independently of editable plan data.
 * @param query Selected month and household or personal scope.
 * @param revision Saved plan revision used by paycheck suggestions.
 * @param connectionVersion Saved Link completions that require updated activity status.
 * @returns Recorded activity, request feedback, and explicit read and mutation actions.
 */
export function useActivity(query: ActivityQuery, revision: number, connectionVersion: string) {
  const { data, loading, busy, error, feedback, totalsStale, setters } = useActivityState();
  const owner = useRef<ActivityOwner>(createActivityOwner(query, setters));
  const reload = useCallback(() => readActivity(owner.current), [owner]);
  useEffect(() => {
    const active = owner.current;
    active.mounted = true;
    return () => {
      active.mounted = false;
      active.generation += 1;
    };
  }, [owner]);
  useEffect(() => {
    owner.current.query = { month: query.month, scope: query.scope };
    owner.current.generation += 1;
    owner.current.setFeedback(null);
    const stop = startAccountStatusUpdates(async () => {
      await reload();
      return null;
    });
    return stop;
  }, [owner, query.month, query.scope, revision, connectionVersion, reload]);
  const current = data?.month === query.month && data.scope === query.scope ? data : null;
  return {
    data: current,
    snapshot: data,
    loading,
    busy,
    error,
    feedback,
    totalsStale,
    reload,
    acceptSource: (source: ActivitySource) => {
      acceptActivitySource(owner.current, source);
    },
    run: <T>(operation: () => Promise<T>, message: string) =>
      mutateActivity(owner.current, operation, message),
  };
}

/** Allocates one mutable request owner apart from React display state.
 * @returns Activity display state and its React setters.
 */
function useActivityState() {
  const [data, setData] = useState<ActivityResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [feedback, setFeedback] = useState<string | null>(null);
  const [totalsStale, setTotalsStale] = useState(false);
  return {
    data,
    loading,
    busy,
    error,
    feedback,
    totalsStale,
    setters: { setData, setLoading, setBusy, setError, setFeedback, setTotalsStale },
  };
}

/** Allocates mutable request ownership apart from React display state.
 * @param query Initial monthly selection.
 * @param setters Visible activity state setters.
 * @returns One activity request owner for a ref.
 */
function createActivityOwner(
  query: ActivityQuery,
  setters: ReturnType<typeof useActivityState>['setters'],
): ActivityOwner {
  return {
    mounted: true,
    generation: 0,
    locked: false,
    query,
    data: null,
    ...setters,
    load: loadActivity,
  };
}
