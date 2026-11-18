import type { ActivityResponse, LinkTokenRequest } from '@bank-piggy/contracts';
import type { ActivityDashboardProps, ActivitySection } from './activityTypes';
import type { activityCommands } from './activityCommands';
import { ActivityTotals } from './ActivitySummary';
import { ActivitySourcePanel } from './ActivitySource';
import { PaycheckSuggestionPanel } from './PaycheckSuggestions';
import styles from './Activity.module.css';
import formStyles from '../plan/Editors.module.css';

interface BodyProps {
  readonly data: ActivityResponse;
  readonly dashboard: ActivityDashboardProps;
  readonly section: ActivitySection;
  readonly commands: ReturnType<typeof activityCommands>;
  readonly blocked: boolean;
  readonly totalsStale: boolean;
  readonly onAuthorize: (input: LinkTokenRequest) => void;
}

/**
 * Keeps all editable sections mounted while native section buttons change visibility.
 * @param props Monthly data, retained forms, and explicit mutation actions.
 * @returns The selected body with drafts preserved in hidden sections.
 */
export function ActivityBody(props: BodyProps) {
  return (
    <>
      <div hidden={props.section !== 'summary'}>
        <SummarySection {...props} />
      </div>
      <div hidden={props.section !== 'sources'}>
        <SourcesSection {...props} />
      </div>
    </>
  );
}

/**
 * Shows monthly actual totals and reviewable bank-pay suggestions.
 * @param props Current monthly facts and member names.
 * @returns Summary and explicit paycheck review panels.
 */
function SummarySection(props: BodyProps) {
  return (
    <>
      <ActivityTotals data={props.data} unavailable={props.totalsStale} />
      {props.totalsStale && (
        <p className={styles.notice}>
          Retry activity to show current totals and pay suggestions. Your draft choices are kept.
        </p>
      )}
      <section className={styles.section}>
        <h3>Pay suggestions</h3>
        <p className={styles.note}>Review pay estimates before changing your saved pay plan.</p>
        {props.data.suggestions.length ? (
          props.data.suggestions.map((suggestion) => (
            <PaycheckSuggestionPanel
              key={suggestion.id}
              suggestion={suggestion}
              members={props.dashboard.household.members}
              busy={props.blocked || props.totalsStale}
              onApply={props.commands.apply}
            />
          ))
        ) : (
          <p className={styles.note}>
            No pay suggestions are available from the recorded deposits.
          </p>
        )}
      </section>
    </>
  );
}

/**
 * Keeps consent and account-inclusion drafts separate from linked balance mappings.
 * @param props Current sources and explicit source-settings actions.
 * @returns Source forms and the Accounts entry point for an empty connection list.
 */
function SourcesSection(props: BodyProps) {
  return (
    <>
      <p className={styles.notice}>
        Choose history access and the accounts that count in totals. Shared accounts use equal
        personal shares. These settings do not change balances, budgets, or goal progress.
      </p>
      {props.data.sources.map((source) => (
        <ActivitySourcePanel
          key={source.connectionId}
          source={source}
          members={props.dashboard.household.members}
          busy={props.blocked}
          onSave={props.commands.source}
          onAuthorize={props.onAuthorize}
        />
      ))}
      {!props.data.sources.length && (
        <div className={styles.empty}>
          <p>Link an institution in Accounts, then return here to review history access.</p>
          <button className={formStyles.secondary} onClick={props.dashboard.onAccounts}>
            Open Accounts
          </button>
        </div>
      )}
    </>
  );
}
