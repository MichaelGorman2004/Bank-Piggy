import type { InvitationCreated, InvitationPreview, InvitationStatus } from '@bank-piggy/contracts';
import { contractValidator, request } from '../../api';

const isStatus = contractValidator<InvitationStatus>('InvitationStatus');
const isCreated = contractValidator<InvitationCreated>('InvitationCreated');
const isPreview = contractValidator<InvitationPreview>('InvitationPreview');
const INVITATION_PATH = '/api/household/invitation';

/**
 * Reads linked-account and pending-invitation status without issuing a new link.
 * @returns Authorized household invitation status.
 */
export function loadInvitation(): Promise<InvitationStatus> {
  return request(INVITATION_PATH, isStatus);
}

/**
 * Creates a short-lived partner link after an explicit invitation action.
 * @param csrfToken Token owned by this authenticated session.
 * @returns One invitation URL whose raw token stays in transient memory.
 */
export function createInvitation(csrfToken: string): Promise<InvitationCreated> {
  return request(INVITATION_PATH, isCreated, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
    body: '{}',
  });
}

/**
 * Revokes a pending link without changing financial profiles.
 * @param csrfToken Token owned by this authenticated session.
 * @returns Completion when the server has revoked the link.
 */
export async function revokeInvitation(csrfToken: string): Promise<void> {
  await request(INVITATION_PATH, (value): value is null => value === null, {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
    body: '{}',
  });
}

/**
 * Exchanges the scrubbed fragment token for a secure pending-invitation cookie.
 * @param token Raw invitation token held only in transient memory.
 * @returns Nonfinancial preview for the Google join confirmation.
 */
export function prepareInvitation(token: string): Promise<InvitationPreview> {
  return request(`${INVITATION_PATH}/prepare`, isPreview, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ token }),
  });
}
