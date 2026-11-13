import { useEffect, useRef, useState } from 'react';
import type { ExchangeRequest, LinkTokenRequest, LinkTokenResponse } from '@bank-piggy/contracts';
import { createLink, exchangeLink, pendingLink } from './api';
import { clearPlaidReturn, plaidReturn } from './oauthReturn';
import { errorMessage } from '../session/errors';
import type { useConnections } from './useConnections';
import { hasCompletedAttempt } from './connectionRequests';
import { linkFailureMessage } from './presentation';

type ConnectionData = ReturnType<typeof useConnections>;
export interface LinkSession {
  readonly attempt: LinkTokenResponse;
  readonly redirectUri?: string;
  readonly reconnect: boolean;
}

/**
 * Holds ephemeral Link credentials in memory and recovers server-bound OAuth attempts.
 * @param csrfToken Current session mutation token.
 * @param data Shared connection request owner.
 * @returns Link lifecycle and account-dialog visibility.
 */
export function useConnectionLink(csrfToken: string, data: ConnectionData) {
  const [open, setOpen] = useState(Boolean(plaidReturn()));
  const [link, setLink] = useState<LinkSession | null>(null);
  const [preparing, setPreparing] = useState(false);
  const [external, setExternal] = useState(false);
  const { lock, generation } = useLinkResume(setLink, data.setError);
  const completion = useLinkCompletion(csrfToken, data);
  return {
    open,
    link,
    preparing,
    external,
    pendingSave: completion.pending,
    retry: completion.retry,
    reload: completion.reload,
    start: (input: LinkTokenRequest) =>
      link || completion.pending
        ? Promise.resolve()
        : startLink({ input, csrfToken, data, lock, generation, setPreparing, setLink }),
    finish: async (publicToken: string | null) => {
      if (!link) return;
      setExternal(false);
      setLink(null);
      await completion.save(link, publicToken);
    },
    show: () => {
      setOpen(true);
      void completion.reload();
    },
    close: () => {
      if (!data.busy && !preparing) {
        setOpen(false);
        setLink(null);
      }
    },
    activate: () => {
      setExternal(true);
    },
    exit: (code: string | null) => {
      setExternal(false);
      setLink(null);
      clearPlaidReturn();
      const message = linkFailureMessage(code);
      if (message) data.setError(message);
    },
  };
}

/**
 * Recovers the original OAuth Link token with bounded component ownership.
 * @param setLink Receives the current session-bound Link attempt.
 * @param setError Receives safe recovery failures.
 * @returns Request generation invalidated when the account manager unmounts.
 */
function useLinkResume(
  setLink: (value: LinkSession | null) => void,
  setError: (value: string | null) => void,
) {
  const generation = useRef(0);
  const lock = useRef(false);
  useEffect(() => {
    const version = ++generation.current;
    const redirectUri = plaidReturn();
    if (redirectUri)
      void pendingLink()
        .then((attempt) => {
          if (version === generation.current) setLink({ attempt, redirectUri, reconnect: false });
        })
        .catch(() => {
          if (version === generation.current)
            setError(
              'The bank connection could not resume. Reload the connection list before starting again.',
            );
        });
    return () => {
      generation.current += 1;
    };
  }, [setLink, setError]);
  return { generation, lock };
}

interface StartOptions {
  readonly input: LinkTokenRequest;
  readonly csrfToken: string;
  readonly data: ConnectionData;
  readonly lock: { current: boolean };
  readonly generation: { current: number };
  readonly setPreparing: (value: boolean) => void;
  readonly setLink: (value: LinkSession | null) => void;
}

/**
 * Prepares official Link only after a deliberate connect or reconnect action.
 * @param options Product choice, request ownership, and current session.
 */
async function startLink(options: StartOptions) {
  if (options.lock.current || options.data.busy) return;
  options.lock.current = true;
  const version = ++options.generation.current;
  options.setPreparing(true);
  options.data.setError(null);
  try {
    const attempt = await createLink(options.input, options.csrfToken);
    if (version === options.generation.current)
      options.setLink({ attempt, reconnect: Boolean(options.input.connectionId) });
  } catch (failure) {
    if (version === options.generation.current) options.data.setError(errorMessage(failure));
  } finally {
    options.lock.current = false;
    if (version === options.generation.current) options.setPreparing(false);
  }
}

/**
 * Retains a temporary exchange request after failure for an explicit save retry.
 * @param csrfToken Current session mutation token.
 * @param data Shared request owner.
 * @returns An awaited completion action and retry state kept only in memory.
 */
function useLinkCompletion(csrfToken: string, data: ConnectionData) {
  const [pending, setPending] = useState<{
    readonly request: ExchangeRequest;
    readonly connectionId: string;
  } | null>(null);
  /**
   * Completes the server-owned attempt and retains a failed exchange only in memory.
   * @param input Transient request and its server-owned connection identity.
   */
  async function send(input: NonNullable<typeof pending>) {
    setPending(input);
    const result = await data.run(
      () => exchangeLink(input.request, csrfToken),
      'Connection saved. Review and select the accounts to include.',
    );
    if (result) setPending(null);
    clearPlaidReturn();
  }
  return {
    pending,
    save: (link: LinkSession, token: string | null) =>
      send({
        connectionId: link.attempt.connectionId,
        request: {
          attemptId: link.attempt.attemptId,
          ...(!link.reconnect && token ? { publicToken: token } : {}),
        },
      }),
    retry: async () => {
      if (pending) await send(pending);
    },
    reload: async () => {
      const result = await data.reload();
      if (pending && hasCompletedAttempt(result, pending.connectionId, pending.request.attemptId)) {
        setPending(null);
        data.setFeedback('The connection is saved. Review and select the accounts to include.');
      }
    },
  };
}
