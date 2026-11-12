import { useEffect, useRef, useState } from 'react';
import type { InvitationCreated, InvitationStatus, Session } from '@bank-piggy/contracts';
import { errorMessage } from '../session/errors';
import { createInvitation, loadInvitation, revokeInvitation } from './api';

type SharingAction = 'read' | 'create' | 'revoke';

interface SharingState {
  readonly open: boolean;
  readonly busy: boolean;
  readonly status: InvitationStatus | null;
  readonly created: InvitationCreated | null;
  readonly error: string | null;
  readonly feedback: string | null;
  readonly planSaved: boolean;
  readonly failedAction: SharingAction | null;
}
interface ShareOptions {
  readonly session: Session;
  readonly available: boolean;
  readonly status: InvitationStatus | null;
  readonly generation: { current: number };
  readonly lock: { current: boolean };
  readonly update: (change: Partial<SharingState>) => void;
}

/**
 * Keeps invitation URLs in memory and separates invitation failures from plan saves.
 * @param session Current cookie-backed identity and private-mode availability.
 * @returns Dialog state and explicit invitation, revocation, and copy actions.
 */
export function useSharing(session: Session) {
  const [state, setState] = useState<SharingState>({
    open: false,
    busy: false,
    status: null,
    created: null,
    error: null,
    feedback: null,
    planSaved: false,
    failedAction: null,
  });
  const generation = useRef(0);
  const lock = useRef(false);
  const available = session.mode === 'private' && session.googleConfigured;
  useEffect(
    () => () => {
      generation.current += 1;
    },
    [session.csrfToken],
  );
  /**
   * Merges feedback and results without overwriting newer dialog state.
   * @param change Fields owned by the completed operation.
   */
  function update(change: Partial<SharingState>) {
    setState((current) => ({ ...current, ...change }));
  }
  const options = { session, available, status: state.status, generation, lock, update };
  return {
    ...state,
    available,
    show: (afterSave = false) => showSharing(options, afterSave),
    create: () => runShare(options, 'create', (version) => issueInvitation(options, version)),
    revoke: () => revokeSharing(options),
    retry: () => retrySharing(options, state.failedAction, state.planSaved),
    copy: () => copySharing(options, state.created),
    close: () => {
      if (lock.current) return;
      generation.current += 1;
      update({ open: false, created: null, feedback: null, error: null });
    },
  };
}

/**
 * Serializes invitation operations and ignores responses from older dialogs or sessions.
 * @param options Current identity, request owner, and dialog updater.
 * @param action User action whose failure must keep the same retry intent.
 * @param operation Explicit read or write that updates the current dialog.
 */
async function runShare(
  options: ShareOptions,
  action: SharingAction,
  operation: (version: number) => Promise<void>,
) {
  if (options.lock.current) return;
  options.lock.current = true;
  const version = ++options.generation.current;
  options.update({ busy: true, error: null, feedback: null, failedAction: action });
  try {
    await operation(version);
  } catch (failure) {
    if (version === options.generation.current) options.update({ error: errorMessage(failure) });
  } finally {
    options.lock.current = false;
    if (version === options.generation.current) options.update({ busy: false });
  }
}

/**
 * Creates a link only after an explicit invitation or successful Add person action.
 * @param options Current identity and dialog updater.
 * @param version Request generation that owns the result.
 */
async function issueInvitation(options: ShareOptions, version: number) {
  if (!options.session.csrfToken || !options.available) return;
  options.update({ created: null });
  const created = await createInvitation(options.session.csrfToken);
  if (version !== options.generation.current) return;
  options.update({
    created,
    status: options.status
      ? { ...options.status, pending: true, expiresAt: created.expiresAt }
      : null,
  });
}

/**
 * Opens sharing and reads status before optionally issuing a partner link.
 * @param options Current identity and dialog updater.
 * @param afterSave Whether the user just saved a newly added financial profile.
 */
async function showSharing(options: ShareOptions, afterSave: boolean) {
  if (options.lock.current) return;
  options.update({
    open: true,
    planSaved: afterSave,
    created: null,
    status: null,
    error: null,
    feedback: null,
  });
  if (!options.available) return;
  await runShare(options, 'read', async (version) => {
    const status = await loadInvitation();
    if (version !== options.generation.current) return;
    options.update({ status });
    if (afterSave && status.canInvite) {
      options.update({ failedAction: 'create' });
      await issueInvitation({ ...options, status }, version);
    }
  });
}

/**
 * Revokes a pending link before removing its visible URL.
 * @param options Current identity and dialog updater.
 */
async function revokeSharing(options: ShareOptions) {
  await runShare(options, 'revoke', async (version) => {
    if (!options.session.csrfToken) return;
    await revokeInvitation(options.session.csrfToken);
    if (version !== options.generation.current) return;
    options.update({
      created: null,
      status: options.status ? { ...options.status, pending: false, expiresAt: undefined } : null,
      feedback: 'Link revoked.',
    });
  });
}

/**
 * Copies the current link or gives the user a manual-copy action without logging it.
 * @param options Current request owner and dialog updater.
 * @param created Current raw invitation URL, held only in memory.
 */
async function copySharing(options: ShareOptions, created: InvitationCreated | null) {
  if (!created) return;
  const version = options.generation.current;
  try {
    await navigator.clipboard.writeText(created.url);
    if (version === options.generation.current) options.update({ feedback: 'Link copied.' });
  } catch {
    if (version === options.generation.current)
      options.update({
        feedback: 'Copy did not work. Select the link below and copy it manually.',
      });
  }
}

/**
 * Repeats the failed sharing operation without replacing a revoke request with link creation.
 * @param options Current identity and dialog updater.
 * @param action Operation recorded before the failed request.
 * @param afterSave Whether an accepted financial save originally opened sharing.
 * @returns Completion of the same explicit operation.
 */
function retrySharing(
  options: ShareOptions,
  action: SharingAction | null,
  afterSave: boolean,
): Promise<void> {
  if (action === 'revoke') return revokeSharing(options);
  if (action === 'create')
    return runShare(options, 'create', (version) => issueInvitation(options, version));
  return showSharing(options, afterSave);
}
