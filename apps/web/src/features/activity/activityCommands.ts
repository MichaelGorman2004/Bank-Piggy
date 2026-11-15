import type {
  ActivityRecord,
  ActivityReviewRequest,
  ActivitySource,
  ActivitySettingsRequest,
  PaycheckSuggestion,
} from '@bank-piggy/contracts';
import type { ActivityDashboardProps, ActivityState } from './activityTypes';
import { applyPaycheck, reviewActivity, saveActivitySource } from './api';

/**
 * Binds explicit activity edits to the session and the saved plan revision.
 * @param props Current saved plan and planning month.
 * @param state Serialized activity request owner.
 * @param csrfToken Current session mutation token.
 * @returns Explicit review, settings, and confirmed pay-plan mutations.
 */
export function activityCommands(
  props: ActivityDashboardProps,
  state: ActivityState,
  csrfToken: string,
) {
  return {
    review: async (record: ActivityRecord, input: ActivityReviewRequest) =>
      state.run(async () => {
        const saved = await reviewActivity(record.id, input, csrfToken);
        state.acceptRecord(saved);
        return saved;
      }, 'Activity review saved.'),
    source: async (source: ActivitySource, input: ActivitySettingsRequest) =>
      state.run(async () => {
        const saved = await saveActivitySource(source.connectionId, input, csrfToken);
        state.acceptSource(saved);
        return saved;
      }, 'History settings saved. Review any required institution permission.'),
    apply: async (suggestion: PaycheckSuggestion, memberId: 'member-1' | 'member-2') => {
      const saved = await state.run(
        () =>
          applyPaycheck(
            { id: suggestion.id, month: props.month },
            { expectedPlanRevision: props.household.revision, memberId },
            csrfToken,
          ),
        'Pay suggestion applied to the saved plan.',
      );
      if (saved) props.onReload();
      return Boolean(saved);
    },
  };
}
