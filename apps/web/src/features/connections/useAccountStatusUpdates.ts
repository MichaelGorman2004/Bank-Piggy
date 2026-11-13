import { useEffect } from 'react';
import type { ConnectionList } from '@bank-piggy/contracts';

const ACCOUNT_STATUS_POLL_INTERVAL = 30_000;
type ReloadAccounts = () => Promise<ConnectionList | null>;
type StatusPage = Pick<Document, 'visibilityState' | 'addEventListener' | 'removeEventListener'>;
type StatusBrowser = Pick<
  Window,
  'addEventListener' | 'removeEventListener' | 'setInterval' | 'clearInterval'
>;

/**
 * Keeps an open Accounts view current with stored server snapshots, without provider refreshes.
 * @param reload Stable connection read that owns its own request-error feedback.
 */
export function useAccountStatusUpdates(reload: ReloadAccounts): void {
  useEffect(() => startAccountStatusUpdates(reload), [reload]);
}

/**
 * Reads on entry and foreground changes, with bounded polling and no overlapping reads.
 * @param reload Authorized snapshot read; it owns error presentation and mutation serialization.
 * @param page Current document visibility and events.
 * @param browser Current window focus events and timers.
 * @returns Cleanup that prevents queued reads and removes all registered resources.
 */
export function startAccountStatusUpdates(
  reload: ReloadAccounts,
  page: StatusPage = document,
  browser: StatusBrowser = window,
): () => void {
  let stopped = false;
  let reading = false;
  /** Releases the read after its owner handles success or failure. */
  const finished = () => {
    reading = false;
  };
  /** Queues one foreground read without overlapping an earlier request. */
  const readLatest = () => {
    if (stopped || reading || page.visibilityState !== 'visible') return;
    reading = true;
    void Promise.resolve()
      .then(() => (stopped || page.visibilityState !== 'visible' ? null : reload()))
      .then(finished, finished);
  };
  page.addEventListener('visibilitychange', readLatest);
  browser.addEventListener('focus', readLatest);
  const timer = browser.setInterval(readLatest, ACCOUNT_STATUS_POLL_INTERVAL);
  readLatest();
  return () => {
    stopped = true;
    browser.clearInterval(timer);
    page.removeEventListener('visibilitychange', readLatest);
    browser.removeEventListener('focus', readLatest);
  };
}
