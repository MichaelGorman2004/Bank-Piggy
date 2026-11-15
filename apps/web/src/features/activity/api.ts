import type {
  ActivityResponse,
  ActivitySettingsRequest,
  ActivitySource,
  ApplyPaycheckRequest,
  HouseholdResponse,
} from '@bank-piggy/contracts';
import { contractValidator, request } from '../../api';

const isActivity = contractValidator<ActivityResponse>('ActivityResponse');
const isSource = contractValidator<ActivitySource>('ActivitySource');
const isHousehold = contractValidator<HouseholdResponse>('HouseholdResponse');

/**
 * Reads recorded monthly activity without requesting provider history.
 * @param query Selected month and scope.
 * @param query.month Selected calendar month.
 * @param query.scope Authorized household or personal scope.
 * @returns Validated monthly totals, pay estimates, and source settings.
 */
export function loadActivity(query: { month: string; scope: string }): Promise<ActivityResponse> {
  return request(`/api/activity?${new URLSearchParams(query).toString()}`, isActivity);
}

/**
 * Saves explicit history consent and account ownership against the source revision.
 * @param id Server-owned connection identity.
 * @param input Reviewed source choices.
 * @param csrfToken Current session mutation token.
 * @returns Authoritative source settings.
 */
export function saveActivitySource(
  id: string,
  input: ActivitySettingsRequest,
  csrfToken: string,
): Promise<ActivitySource> {
  return request(
    `/api/connections/${encodeURIComponent(id)}/activity`,
    isSource,
    mutation('PUT', input, csrfToken),
  );
}

/**
 * Applies a confirmed net-pay suggestion to one saved member without changing other plan fields.
 * @param target Suggestion identity and selected planning month.
 * @param target.id Server-owned suggestion identity.
 * @param target.month Selected calendar month.
 * @param input Explicit member and saved plan revision.
 * @param csrfToken Current session mutation token.
 * @returns Updated household plan for the selected month.
 */
export function applyPaycheck(
  target: { id: string; month: string },
  input: ApplyPaycheckRequest,
  csrfToken: string,
): Promise<HouseholdResponse> {
  return request(
    `/api/activity/paychecks/${encodeURIComponent(target.id)}/apply?month=${encodeURIComponent(target.month)}`,
    isHousehold,
    mutation('POST', input, csrfToken),
  );
}

/**
 * Uses the existing same-origin JSON and session mutation boundary.
 * @param method HTTP mutation method.
 * @param input Canonical request payload.
 * @param csrfToken Current session mutation token.
 * @returns JSON request options.
 */
function mutation(method: string, input: unknown, csrfToken: string): RequestInit {
  return {
    method,
    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
    body: JSON.stringify(input),
  };
}
