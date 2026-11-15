import type { ActivityResponse } from '@bank-piggy/contracts';
import type { View } from '../plan/views.ts';
import type { Period } from '../plan/format.ts';

export type ReportMode = 'planned' | 'actual';
export type ActualView = 'flow' | 'expenses' | 'goals';

/** Limits recorded reports to views with an explicit monthly actual projection.
 * @param view Current workspace view.
 * @returns Whether the view supports the Planned and Actual choice.
 */
export function supportsActuals(view: View): view is ActualView {
  return view === 'flow' || view === 'expenses' || view === 'goals';
}

/** Keeps actuals monthly without discarding the user's retained forecast period.
 * @param view Current workspace view.
 * @param mode Chosen basis for supported reports.
 * @param plannedPeriod Valid saved forecast period.
 * @returns Calendar month for actuals, otherwise the selected forecast period.
 */
export function reportingPeriod(view: View, mode: ReportMode, plannedPeriod: Period): Period {
  return supportsActuals(view) && mode === 'actual' ? 'monthly' : plannedPeriod;
}

/** Rejects stale month, scope, plan, and mutation snapshots before displaying comparisons.
 * @param data Last accepted recorded activity.
 * @param target Current visible selection and saved plan.
 * @param target.month Selected calendar month.
 * @param target.scope Current household or personal scope.
 * @param target.revision Current saved plan revision.
 * @param stale Whether an accepted mutation still needs a reconciliation read.
 * @returns A coherent monthly snapshot, or null while replacement data is unavailable.
 */
export function currentReport(
  data: ActivityResponse | null,
  target: {
    readonly month: string;
    readonly scope: string;
    readonly revision: number;
  },
  stale: boolean,
): ActivityResponse | null {
  return !stale &&
    data?.month === target.month &&
    data.scope === target.scope &&
    data.planRevision === target.revision
    ? data
    : null;
}
