import { useEffect, useRef } from 'react';
import { usePlaidLink } from 'react-plaid-link';
import type { LinkSession } from './useConnectionLink';

interface LaunchProps {
  readonly session: LinkSession;
  readonly active: boolean;
  readonly onActivate: () => void;
  readonly onSuccess: (token: string | null) => Promise<void>;
  readonly onExit: (code: string | null) => void;
}

/**
 * Uses official Plaid Link and releases its iframe on unmount.
 * The native account dialog suspends first so the provider iframe can receive focus.
 * @param props Ephemeral Link session and lifecycle callbacks.
 * @returns No app-owned bank credential form.
 */
export function PlaidLaunch(props: LaunchProps) {
  const { active, onActivate, onExit } = props;
  const opened = useRef(false);
  const { ready, open, error } = usePlaidLink({
    token: props.session.attempt.linkToken,
    receivedRedirectUri: props.session.redirectUri,
    onSuccess: (token) => {
      void props.onSuccess(token);
    },
    onExit: (failure) => {
      props.onExit(failure?.error_code ?? null);
    },
  });
  useEffect(() => {
    if (error) onExit('LINK_LOAD_FAILED');
  }, [error, onExit]);
  useEffect(() => {
    if (ready && !active) onActivate();
    if (ready && active && !opened.current) {
      opened.current = true;
      open();
    }
  }, [ready, active, onActivate, open]);
  return null;
}
