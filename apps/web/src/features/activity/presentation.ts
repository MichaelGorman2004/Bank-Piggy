import type { ActivityResponse, ConnectionKind } from '@bank-piggy/contracts';
import type { ActivityQuery } from './activityRequests';

/**
 * Uses the server's view-specific availability after source permission and account checks.
 * @param data Authorized monthly response.
 * @param kind Source kind required by this total.
 * @returns Whether this source kind has available recorded amounts for the selected view.
 */
export function hasActivityHistory(data: ActivityResponse, kind: ConnectionKind): boolean {
  return kind === 'depository'
    ? data.comparison.bankAvailable
    : data.comparison.investmentAvailable;
}

/**
 * Requires current monthly data before an explicit pay-plan confirmation can run.
 * @param data Last accepted monthly aggregates, or no current selection.
 * @param query Selected calendar month and authorized scope.
 * @param revision Current saved plan revision.
 * @param id Suggestion identity selected for confirmation.
 * @returns Whether this suggestion belongs to the selected month, scope, and saved plan.
 */
export function isCurrentPaycheck(
  data: ActivityResponse | null,
  query: ActivityQuery,
  revision: number,
  id: string,
): boolean {
  return (
    data !== null &&
    data.month === query.month &&
    data.scope === query.scope &&
    data.planRevision === revision &&
    data.suggestions.some((suggestion) => suggestion.id === id)
  );
}

/**
 * Names the saved history state without exposing provider error codes.
 * @param state Source history state.
 * @returns Short user-facing status.
 */
export function sourceState(state: ActivityResponse['sources'][number]['state']): string {
  const labels = {
    disabled: 'History off',
    consent_required: 'Permission needed',
    initializing: 'History loading',
    ready: 'History available',
    delayed: 'Update delayed',
    reconnect: 'Reconnect needed',
    unsupported: 'History unavailable',
  };
  return labels[state];
}

/**
 * Formats a calendar date without a timezone shift.
 * @param date Canonical calendar date or timestamp.
 * @returns Readable date using local calendar fields.
 */
export function activityDate(date: string): string {
  return new Intl.DateTimeFormat('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  }).format(new Date(`${date.slice(0, 10)}T12:00:00`));
}
