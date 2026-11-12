import type { InvitationPreview } from '@bank-piggy/contracts';
import { prepareInvitation } from './api';

declare global {
  interface Window {
    consumeBankPiggyInvitation?: () => string | null;
  }
}

let token = window.consumeBankPiggyInvitation?.() ?? null;
delete window.consumeBankPiggyInvitation;
let preparation: Promise<InvitationPreview> | null = null;

/**
 * Owns one preparation request across React Strict Mode effect remounts.
 * The early document script removed the fragment before any network assets loaded.
 * @returns A nonfinancial invitation preview after the pending cookie is set.
 */
export function prepareJoin(): Promise<InvitationPreview> {
  if (preparation) return preparation;
  if (!token)
    return Promise.reject(
      new Error('This invitation link is missing or invalid. Ask your partner for a new link.'),
    );
  preparation = prepareInvitation(token)
    .then((preview) => {
      token = null;
      return preview;
    })
    .catch((failure: unknown) => {
      preparation = null;
      throw failure;
    });
  return preparation;
}
