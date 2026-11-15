import type { ActivityResponse, ActivitySource } from '@bank-piggy/contracts';
import { errorMessage } from '../session/errors.ts';

export interface ActivityQuery {
  readonly month: string;
  readonly scope: string;
}
export interface ActivityOwner {
  mounted: boolean;
  generation: number;
  locked: boolean;
  query: ActivityQuery;
  data: ActivityResponse | null;
  setData: (value: ActivityResponse) => void;
  setLoading: (value: boolean) => void;
  setBusy: (value: boolean) => void;
  setError: (value: string | null) => void;
  setFeedback: (value: string | null) => void;
  setTotalsStale: (value: boolean) => void;
  load: (query: ActivityQuery) => Promise<ActivityResponse>;
}

/**
 * Reads monthly aggregates and rejects stale responses after a selection change or mutation.
 * @param owner Feature request ownership and visible state setters.
 * @param preservedError Mutation error that a reconciliation read must retain.
 * @returns Accepted response, or null after a stale result or failure.
 */
export async function readActivity(
  owner: ActivityOwner,
  preservedError: string | null = null,
): Promise<ActivityResponse | null> {
  if (!owner.mounted || owner.locked) return null;
  const generation = ++owner.generation;
  const query = { ...owner.query };
  owner.setLoading(true);
  try {
    const next = await owner.load(query);
    if (!ownsActivity(owner, generation)) return null;
    owner.data = next;
    owner.setData(next);
    owner.setTotalsStale(false);
    owner.setError(preservedError);
    return next;
  } catch (failure) {
    if (ownsActivity(owner, generation)) owner.setError(preservedError ?? errorMessage(failure));
    return null;
  } finally {
    if (ownsActivity(owner, generation)) owner.setLoading(false);
  }
}

/**
 * Serializes mutations and reconciles the current selection without clearing form drafts.
 * @template T Authoritative mutation result.
 * @param owner Current feature request owner.
 * @param operation Explicit server mutation.
 * @param feedback Plain success message.
 * @returns Saved result, or null after rejection, failure, or unmount.
 */
export async function mutateActivity<T>(
  owner: ActivityOwner,
  operation: () => Promise<T>,
  feedback: string,
): Promise<T | null> {
  if (!owner.mounted || owner.locked) return null;
  owner.locked = true;
  owner.generation += 1;
  owner.setBusy(true);
  owner.setTotalsStale(true);
  owner.setError(null);
  owner.setFeedback(null);
  let failureMessage: string | null = null;
  let saved: T | null = null;
  try {
    saved = await operation();
    if (isMounted(owner)) owner.setFeedback(feedback);
  } catch (failure) {
    failureMessage = errorMessage(failure);
    if (isMounted(owner)) owner.setError(failureMessage);
  } finally {
    owner.locked = false;
    if (isMounted(owner)) {
      await readActivity(owner, failureMessage);
      if (isMounted(owner)) owner.setBusy(false);
    }
  }
  return isMounted(owner) ? saved : null;
}

/**
 * Checks lifetime and generation after awaited activity work.
 * @param owner Current request owner.
 * @param generation Generation captured before the request.
 * @returns Whether this response still owns visible state.
 */
function ownsActivity(owner: ActivityOwner, generation: number): boolean {
  return owner.mounted && owner.generation === generation;
}

/** Checks mutable lifetime again after an awaited operation.
 * @param owner Current component owner.
 * @returns Whether this owner is still mounted.
 */
function isMounted(owner: ActivityOwner): boolean {
  return owner.mounted;
}

/** Publishes an authoritative source save before the optional reconciliation read.
 * @param owner Current activity request owner.
 * @param source Source returned by the accepted settings mutation.
 */
export function acceptActivitySource(owner: ActivityOwner, source: ActivitySource): void {
  if (!owner.mounted || !owner.data) return;
  const previous = owner.data.sources.find((item) => item.connectionId === source.connectionId);
  if (previous && previous.revision > source.revision) return;
  const sources = previous
    ? owner.data.sources.map((item) => (item.connectionId === source.connectionId ? source : item))
    : [...owner.data.sources, source];
  const next = { ...owner.data, sources };
  owner.data = next;
  owner.setData(next);
  owner.setTotalsStale(true);
}
