import type { ActivityRecord, ActivityResponse, ActivitySource } from '@bank-piggy/contracts';
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
  load: (query: ActivityQuery & { cursor?: string }) => Promise<ActivityResponse>;
}

/**
 * Reads one month or next page and rejects stale responses after a scope change or mutation.
 * @param owner Feature request ownership and visible state setters.
 * @param cursor Optional cursor owned by the currently displayed first page.
 * @param preservedError Mutation error that a reconciliation read must retain.
 * @returns Accepted response, or null after a stale result or failure.
 */
export async function readActivity(
  owner: ActivityOwner,
  cursor?: string,
  preservedError: string | null = null,
): Promise<ActivityResponse | null> {
  if (!canRead(owner, cursor)) return null;
  const previous = owner.data;
  const generation = ++owner.generation;
  const query = { ...owner.query, ...(cursor ? { cursor } : {}) };
  owner.setLoading(true);
  try {
    const next = await loadActivityWindow(owner, query, generation);
    if (!ownsActivity(owner, generation)) return null;
    const accepted = acceptActivityPage(previous, next, cursor);
    owner.data = accepted;
    owner.setData(accepted);
    owner.setTotalsStale(false);
    owner.setError(preservedError);
    return accepted;
  } catch (failure) {
    if (ownsActivity(owner, generation)) owner.setError(preservedError ?? errorMessage(failure));
    return null;
  } finally {
    if (ownsActivity(owner, generation)) owner.setLoading(false);
  }
}

/**
 * Requires pagination to belong to the current month and its latest accepted cursor.
 * @param owner Current request owner.
 * @param cursor Optional requested page cursor.
 * @returns Whether this read can start.
 */
function canRead(owner: ActivityOwner, cursor?: string): boolean {
  if (!owner.mounted || owner.locked) return false;
  if (!cursor) return true;
  return (
    owner.data?.month === owner.query.month &&
    owner.data.scope === owner.query.scope &&
    owner.data.nextCursor === cursor
  );
}

/**
 * Reloads previously visible pages so a status update does not remove later-page drafts.
 * @param owner Current request owner and accepted records.
 * @param query Selected month, scope, and optional page cursor.
 * @param generation Request generation checked before every follow-up page.
 * @returns Fresh visible page window with full-month totals.
 */
async function loadActivityWindow(
  owner: ActivityOwner,
  query: ActivityQuery & { cursor?: string },
  generation: number,
): Promise<ActivityResponse> {
  const current = owner.data;
  const count =
    current?.month === query.month && current.scope === query.scope ? current.records.length : 0;
  let next = await owner.load(query);
  if (query.cursor) return next;
  while (next.nextCursor && next.records.length < count && ownsActivity(owner, generation)) {
    const page = await owner.load({ ...query, cursor: next.nextCursor });
    next = appendActivityPage(next, page);
  }
  return next;
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
      await readActivity(owner, undefined, failureMessage);
      if (isMounted(owner)) owner.setBusy(false);
    }
  }
  return isMounted(owner) ? saved : null;
}

/**
 * Appends unique records while retaining full-month totals from the latest response.
 * @param previous Current monthly result with loaded pages.
 * @param next Next page returned for the same month and scope.
 * @returns New response with records in server page order.
 */
function appendActivityPage(previous: ActivityResponse, next: ActivityResponse): ActivityResponse {
  const ids = new Set(previous.records.map((record) => record.id));
  const updates = new Map(next.records.map((record) => [record.id, record]));
  return {
    ...next,
    records: [
      ...previous.records.map((record) => updates.get(record.id) ?? record),
      ...next.records.filter((record) => !ids.has(record.id)),
    ],
  };
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

/** Appends only deliberate pagination reads, retaining the captured previous page window.
 * @param previous Accepted records before the read started.
 * @param next Newly returned monthly response.
 * @param cursor Optional explicit next-page cursor.
 * @returns Accepted monthly response.
 */
function acceptActivityPage(
  previous: ActivityResponse | null,
  next: ActivityResponse,
  cursor?: string,
): ActivityResponse {
  return cursor && previous ? appendActivityPage(previous, next) : next;
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

/** Publishes an accepted record review while retaining unrelated record drafts.
 * @param owner Current activity request owner.
 * @param record Record returned by the accepted review mutation.
 */
export function acceptActivityRecord(owner: ActivityOwner, record: ActivityRecord): void {
  if (!owner.mounted || !owner.data) return;
  const previous = owner.data.records.find((item) => item.id === record.id);
  if (!previous || previous.revision > record.revision) return;
  const next = {
    ...owner.data,
    records: owner.data.records.map((item) => (item.id === record.id ? record : item)),
  };
  owner.data = next;
  owner.setData(next);
  owner.setTotalsStale(true);
}
