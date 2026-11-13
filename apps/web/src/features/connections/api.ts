import type {
  AccountMappingRequest,
  ConnectionList,
  ExchangeRequest,
  LinkTokenRequest,
  LinkTokenResponse,
} from '@bank-piggy/contracts';
import { contractValidator, request } from '../../api';

const isList = contractValidator<ConnectionList>('ConnectionList');
const isLinkToken = contractValidator<LinkTokenResponse>('LinkTokenResponse');
const PATH = '/api/connections';

/**
 * Reads server-owned account snapshots and reconciled balance totals.
 * @returns Current authorized household connections.
 */
export function loadConnections(): Promise<ConnectionList> {
  return request(PATH, isList);
}

/**
 * Creates one session-bound Link attempt after an explicit connect or reconnect action.
 * @param input Product kind or server-owned connection identifier.
 * @param csrfToken Current session mutation token.
 * @returns Ephemeral Link credentials held only in memory.
 */
export function createLink(input: LinkTokenRequest, csrfToken: string): Promise<LinkTokenResponse> {
  return request(`${PATH}/link-token`, isLinkToken, mutation('POST', csrfToken, input));
}

/**
 * Recovers the original attempt after an institution OAuth redirect.
 * @returns The pending Link credentials bound to this browser session.
 */
export function pendingLink(): Promise<LinkTokenResponse> {
  return request(`${PATH}/link-token/pending`, isLinkToken);
}

/**
 * Finishes Link through the server-owned attempt without accepting browser authorization IDs.
 * @param input Attempt identifier and optional temporary public token.
 * @param csrfToken Current session mutation token.
 * @returns Updated connections after server-side exchange or reconnect.
 */
export function exchangeLink(input: ExchangeRequest, csrfToken: string): Promise<ConnectionList> {
  return request(`${PATH}/exchange`, isList, mutation('POST', csrfToken, input));
}

/**
 * Requests a bounded provider refresh without claiming the snapshot is current.
 * @param id Server-owned connection identifier.
 * @param csrfToken Current session mutation token.
 * @returns Current connections and durable refresh state.
 */
export function refreshConnection(id: string, csrfToken: string): Promise<ConnectionList> {
  return request(`${PATH}/${encodeURIComponent(id)}/refresh`, isList, mutation('POST', csrfToken));
}

/**
 * Removes a connection only after the provider has accepted revocation.
 * @param id Server-owned connection identifier.
 * @param csrfToken Current session mutation token.
 * @returns Updated connections after the required provider operation succeeds.
 */
export function disconnectConnection(id: string, csrfToken: string): Promise<ConnectionList> {
  return request(`${PATH}/${encodeURIComponent(id)}`, isList, mutation('DELETE', csrfToken));
}

/**
 * Saves explicit account selection and reconciliation against its saved revision.
 * @param ids Server-owned connection and account identifiers.
 * @param ids.connection Server-owned connection identifier.
 * @param ids.account Server-owned account identifier.
 * @param input Draft mapping and original mapping revision.
 * @param csrfToken Current session mutation token.
 * @returns Authoritative mappings and balance totals.
 */
export function saveMapping(
  ids: { readonly connection: string; readonly account: string },
  input: AccountMappingRequest,
  csrfToken: string,
): Promise<ConnectionList> {
  const path = `${PATH}/${encodeURIComponent(ids.connection)}/accounts/${encodeURIComponent(ids.account)}/mapping`;
  return request(path, isList, mutation('PUT', csrfToken, input));
}

/**
 * Keeps connection writes on the existing same-origin JSON and CSRF boundary.
 * @param method Browser mutation method.
 * @param csrfToken Current session mutation token.
 * @param body Canonical request value, or an empty object.
 * @returns Request initialization for a same-origin mutation.
 */
function mutation(method: string, csrfToken: string, body: unknown = {}): RequestInit {
  return {
    method,
    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
    body: JSON.stringify(body),
  };
}
