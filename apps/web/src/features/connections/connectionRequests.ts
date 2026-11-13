import type { ConnectionList } from '@bank-piggy/contracts';
import { errorMessage } from '../session/errors.ts';

type ReadResult = ConnectionList | null;
interface ActiveRead {
  readonly version: number;
  readonly promise: Promise<ReadResult>;
}
interface QueuedRead {
  preservedError: string | null;
  readonly promise: Promise<ReadResult>;
}

/** One shared GET and at most one fresh follow-up belong to a connection provider. */
export interface ConnectionReadState {
  active: ActiveRead | null;
  queued: QueuedRead | null;
}

export interface MutationOptions {
  readonly generation: { current: number };
  readonly mounted: { current: boolean };
  readonly lock: { current: boolean };
  readonly pendingRead: { current: boolean };
  readonly reads: { current: ConnectionReadState };
  readonly read: (preservedError?: string | null, fresh?: boolean) => Promise<unknown>;
  readonly setBusy: (value: boolean) => void;
  readonly setLoading: (value: boolean) => void;
  readonly setError: (value: string | null) => void;
  readonly setFeedback: (value: string | null) => void;
  readonly setData: (value: ConnectionList) => void;
}
export type ConnectionFeedback = string | ((response: ConnectionList) => string);
type ReadOptions = Pick<
  MutationOptions,
  | 'generation'
  | 'mounted'
  | 'lock'
  | 'pendingRead'
  | 'reads'
  | 'setData'
  | 'setError'
  | 'setLoading'
>;

/**
 * Shares current GETs and queues one fresh read after older work or a mutation.
 * @param options Request ownership and loaded-state setters.
 * @param load Authorized server snapshot read.
 * @param preservedError Explicit mutation outcome; omitted callers retain a queued failure.
 * @param fresh Whether this intent must read after an already active request.
 * @returns Accepted snapshot, or null when deferred, stale, or unsuccessful.
 */
export function connectionRead(
  options: ReadOptions,
  load: () => Promise<ConnectionList>,
  preservedError?: string | null,
  fresh = false,
): Promise<ConnectionList | null> {
  if (!options.mounted.current) return Promise.resolve(null);
  if (options.lock.current) {
    options.pendingRead.current = true;
    return Promise.resolve(null);
  }
  const queued = options.reads.current.queued;
  if (queued) return updateQueuedRead(options, queued, preservedError, fresh);
  const active = options.reads.current.active;
  if (active) {
    if (!fresh && active.version === options.generation.current) return active.promise;
    options.generation.current += 1;
    return queueConnectionRead(options, load, active, preservedError ?? null);
  }
  return startConnectionRead(options, load, preservedError ?? null);
}

/**
 * Coalesces new read intents without losing the latest explicit mutation outcome.
 * @param options Shared request owner.
 * @param queued Fresh GET waiting for the active GET to settle.
 * @param preservedError Explicit mutation outcome, or no change when omitted.
 * @param fresh Whether this intent invalidates the active result.
 * @returns The shared fresh GET result.
 */
function updateQueuedRead(
  options: ReadOptions,
  queued: QueuedRead,
  preservedError: string | null | undefined,
  fresh: boolean,
): Promise<ReadResult> {
  if (fresh) options.generation.current += 1;
  if (preservedError !== undefined) queued.preservedError = preservedError;
  return queued.promise;
}

/**
 * Queues one GET whose callers receive the accepted fresh result and mutation outcome.
 * @param options Shared request owner.
 * @param load Authorized snapshot read.
 * @param active Older GET that must settle before the queued GET starts.
 * @param preservedError Mutation outcome retained by the fresh GET.
 * @returns The accepted fresh result after older work settles.
 */
function queueConnectionRead(
  options: ReadOptions,
  load: () => Promise<ConnectionList>,
  active: ActiveRead,
  preservedError: string | null,
): Promise<ReadResult> {
  const queued: QueuedRead = {
    preservedError,
    promise: active.promise.then(() => {
      const next = options.reads.current.queued;
      options.reads.current.queued = null;
      return connectionRead(options, load, next?.preservedError);
    }),
  };
  options.reads.current.queued = queued;
  return queued.promise;
}

/**
 * Dispatches the sole active GET and releases its ownership on every result.
 * @param options Shared request owner and visible state setters.
 * @param load Authorized snapshot read.
 * @param preservedError Mutation failure retained after successful reconciliation.
 * @returns The accepted snapshot, or null after a failure or stale result.
 */
function startConnectionRead(
  options: ReadOptions,
  load: () => Promise<ConnectionList>,
  preservedError: string | null,
): Promise<ReadResult> {
  const version = ++options.generation.current;
  options.setLoading(true);
  let request: Promise<ConnectionList>;
  try {
    request = load();
  } catch (failure) {
    const error =
      failure instanceof Error ? failure : new Error(errorMessage(failure), { cause: failure });
    request = Promise.reject(error);
  }
  const promise = request
    .then((next) => {
      if (!ownsRequest(options, version)) return null;
      options.setData(next);
      options.setError(preservedError);
      return next;
    })
    .catch((failure: unknown) => {
      if (ownsRequest(options, version)) options.setError(preservedError ?? errorMessage(failure));
      return null;
    })
    .finally(() => {
      if (options.reads.current.active?.version === version) options.reads.current.active = null;
      if (ownsRequest(options, version)) options.setLoading(false);
    });
  options.reads.current.active = { version, promise };
  return promise;
}

/**
 * Serializes provider mutations and preserves their failure across a deferred plan read.
 * @param options Request ownership and state setters.
 * @param operation Authorized provider mutation.
 * @param feedback Safe success feedback.
 * @returns Accepted response, or null after failure or loss of request ownership.
 */
export async function connectionMutation(
  options: MutationOptions,
  operation: () => Promise<ConnectionList>,
  feedback: ConnectionFeedback,
): Promise<ConnectionList | null> {
  if (options.lock.current || !ownsRequest(options, options.generation.current)) return null;
  options.lock.current = true;
  const version = ++options.generation.current;
  options.setBusy(true);
  options.setError(null);
  options.setFeedback(null);
  let failureMessage: string | null = null;
  try {
    const next = await operation();
    if (!ownsRequest(options, version)) return null;
    options.setData(next);
    options.setFeedback(typeof feedback === 'string' ? feedback : feedback(next));
    return next;
  } catch (failure) {
    failureMessage = errorMessage(failure);
    if (ownsRequest(options, version)) options.setError(failureMessage);
    return null;
  } finally {
    options.lock.current = false;
    if (options.mounted.current) {
      options.setBusy(false);
      options.setLoading(false);
      await reconcileConnectionRead(options, failureMessage);
    }
  }
}

/**
 * Preserves the latest mutation outcome when older reads already have a follow-up queued.
 * @param options Shared read and mutation ownership.
 * @param failure Latest mutation failure, or null after an accepted mutation.
 */
async function reconcileConnectionRead(options: MutationOptions, failure: string | null) {
  if (!options.pendingRead.current && !options.reads.current.queued) return;
  options.pendingRead.current = false;
  await options.read(failure, true);
}

/**
 * Checks component lifetime and request order again after awaited provider work.
 * @param options Current request owner.
 * @param version Generation captured by the request.
 * @returns Whether this request can still update the current session.
 */
function ownsRequest(
  options: Pick<MutationOptions, 'mounted' | 'generation'>,
  version: number,
): boolean {
  return options.mounted.current && options.generation.current === version;
}

/**
 * Resolves a lost response only from server proof that the exact Link attempt completed.
 * @param result Latest authorized connection list.
 * @param id Connection identity from the session-bound Link attempt.
 * @param attemptId Exact attempt whose successful completion must be proved.
 * @returns Whether the server confirmed this exact new or update attempt.
 */
export function hasCompletedAttempt(
  result: ConnectionList | null,
  id: string,
  attemptId: string,
): boolean {
  const connection = result?.connections.find((item) => item.id === id);
  if (connection?.lastCompletedAttemptId !== attemptId) return false;
  return (
    connection.state === 'connected' ||
    connection.state === 'delayed' ||
    connection.state === 'reconnect'
  );
}
