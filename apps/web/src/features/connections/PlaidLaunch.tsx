import { useEffect, useEffectEvent, useRef, useState } from 'react';
import { usePlaidLink } from 'react-plaid-link';
import type { LinkSession } from './useConnectionLink';

const LINK_LOAD_TIMEOUT_MS = 25_000;

interface LaunchProps {
  readonly session: LinkSession;
  readonly active: boolean;
  readonly onActivate: () => void;
  readonly onSuccess: (token: string | null) => Promise<void>;
  readonly onExit: (code: string | null) => void;
}

/**
 * Uses official Plaid Link and releases its iframe on unmount.
 * Keeps the native dialog visible until Plaid confirms that its iframe has loaded.
 * @param props Ephemeral Link session and lifecycle callbacks.
 * @returns No app-owned bank credential form.
 */
export function PlaidLaunch(props: LaunchProps) {
  const { active, onActivate, onExit } = props;
  const [loaded, setLoaded] = useState(false);
  const phase = useRef<'loading' | 'opened' | 'settled'>('loading');
  const fail = useEffectEvent(() => {
    if (phase.current === 'settled') return;
    phase.current = 'settled';
    onExit('LINK_LOAD_FAILED');
  });
  const { ready, open, error } = usePlaidLink({
    token: props.session.attempt.linkToken,
    receivedRedirectUri: props.session.redirectUri,
    onLoad: () => {
      setLoaded(true);
    },
    onSuccess: (token) => {
      if (phase.current === 'settled') return;
      phase.current = 'settled';
      void props.onSuccess(token);
    },
    onExit: (failure) => {
      if (phase.current === 'settled') return;
      phase.current = 'settled';
      props.onExit(failure?.error_code ?? null);
    },
  });
  // The SDK can report ready before its content has finished loading.
  const canOpen = ready && loaded;
  useEffect(() => {
    if (error) {
      fail();
      return;
    }
    if (canOpen) return;
    const timeout = window.setTimeout(() => {
      fail();
    }, LINK_LOAD_TIMEOUT_MS);
    return () => {
      window.clearTimeout(timeout);
    };
  }, [canOpen, error]);
  useEffect(() => {
    if (!canOpen || phase.current !== 'loading') return;
    if (!active) onActivate();
    else {
      phase.current = 'opened';
      open();
    }
  }, [canOpen, active, onActivate, open]);
  return null;
}
