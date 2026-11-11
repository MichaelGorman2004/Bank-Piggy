import Ajv from 'ajv';
import type { Household, HouseholdResponse, Session } from '@bank-piggy/contracts';
import schema from '@bank-piggy/contracts/schema.json';

const validator = new Ajv({ allErrors: false });
validator.addSchema(schema, 'bank-piggy');
const isSession = validator.compile<Session>({ $ref: 'bank-piggy#/$defs/Session' });
const isHouseholdResponse = validator.compile<HouseholdResponse>({
  $ref: 'bank-piggy#/$defs/HouseholdResponse',
});

/** Keeps HTTP status available for expired sessions and revision conflicts. */
export class ApiError extends Error {
  /**
   * Creates a safe request error that keeps the HTTP result available to callers.
   * @param message User-facing failure description.
   * @param status HTTP status, or zero when the connection failed.
   */
  constructor(
    message: string,
    readonly status: number,
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

/**
 * Reads an error body without trusting the response payload's shape.
 * @param body Unvalidated JSON from the server.
 * @returns A safe error description.
 */
function responseError(body: unknown): string {
  if (
    typeof body !== 'object' ||
    body === null ||
    !('error' in body) ||
    typeof body.error !== 'string'
  )
    return 'The request could not be completed. Try again.';
  return body.error;
}

/**
 * Validates successful responses against the canonical generated API schema.
 * @template T Successful response type owned by the API contract.
 * @param path Same-origin API path.
 * @param validate Canonical runtime schema validator.
 * @param init Optional request method, body, and headers.
 * @returns The validated response.
 * @throws {ApiError} When the connection, status, or response schema is invalid.
 */
async function request<T>(
  path: string,
  validate: (value: unknown) => value is T,
  init?: RequestInit,
): Promise<T> {
  let response: Response;
  try {
    response = await fetch(path, { credentials: 'same-origin', ...init });
  } catch {
    throw new ApiError('Bank Piggy could not connect. Check your connection and try again.', 0);
  }
  const body: unknown = await response.json().catch(() => null);
  if (!response.ok) throw new ApiError(responseError(body), response.status);
  if (!validate(body))
    throw new ApiError(
      'Bank Piggy received an unexpected response. Reload the page and try again.',
      502,
    );
  return body;
}

/**
 * Reads the cookie-backed session.
 * @returns The current identity and available sign-in methods.
 */
export function loadSession(): Promise<Session> {
  return request('/api/session', isSession);
}
/**
 * Opens the loopback-only sample household session.
 * @returns The new sample session.
 */
export function startDemo(): Promise<Session> {
  return request('/api/demo/session', isSession, { method: 'POST' });
}
/**
 * Reads the server-calculated household plan and member allocations.
 * @param month Optional calendar month for both monthly and yearly projections.
 * @returns The saved plan and derived totals.
 */
export function loadHousehold(month?: string): Promise<HouseholdResponse> {
  return request(householdPath(month), isHouseholdResponse);
}
/**
 * Saves a complete plan using its revision and the session's CSRF token.
 * @param household Plan with the revision that the user started editing.
 * @param csrfToken Server-issued token for this session's writes.
 * @param month Planning month retained after the save.
 * @returns The updated plan and calculated totals.
 */
export function saveHousehold(
  household: Household,
  csrfToken: string,
  month?: string,
): Promise<HouseholdResponse> {
  return request(householdPath(month), isHouseholdResponse, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
    body: JSON.stringify(household),
  });
}
/**
 * Ends the current cookie-backed session.
 * @param csrfToken Server-issued token for this session's writes.
 * @returns Completion after the server revokes the session.
 * @throws {ApiError} When revocation fails.
 */
export async function logout(csrfToken: string): Promise<void> {
  const response = await fetch('/api/logout', {
    method: 'POST',
    credentials: 'same-origin',
    headers: { 'X-CSRF-Token': csrfToken },
  });
  if (!response.ok) throw new ApiError('Sign out failed. Try again.', response.status);
}

/**
 * Keeps reads and writes on the same explicit planning month.
 * @param month Optional calendar month in YYYY-MM format.
 * @returns Same-origin household endpoint with the selected projection.
 */
function householdPath(month?: string): string {
  return month ? `/api/household?month=${encodeURIComponent(month)}` : '/api/household';
}
