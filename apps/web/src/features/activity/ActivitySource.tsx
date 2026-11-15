import { useState } from 'react';
import type {
  ActivitySettingsRequest,
  ActivitySource,
  LinkTokenRequest,
  Member,
} from '@bank-piggy/contracts';
import {
  activityAccountListChanged,
  canSaveActivitySource,
  reconcileActivityAccounts,
} from './activitySourceDraft';
import { acceptActivityDraft } from './activityDrafts';
import { SourceAccounts } from './SourceAccounts';
import {
  InvestmentHistoryCost,
  SourceAuthorize,
  SourcePermissionNotes,
} from './ActivitySourcePermission';
import { activityDate, sourceState } from './presentation';
import styles from './Activity.module.css';
import formStyles from '../plan/Editors.module.css';

interface SourceProps {
  readonly source: ActivitySource;
  readonly members: ReadonlyArray<Member>;
  readonly busy: boolean;
  readonly onSave: (
    source: ActivitySource,
    input: ActivitySettingsRequest,
  ) => Promise<ActivitySource | null>;
  readonly onAuthorize: (input: LinkTokenRequest) => void;
}
interface SourceDraft extends ActivitySettingsRequest {
  readonly edit: number;
}

/**
 * Reviews explicit history opt-in separately from balance account mappings.
 * @param props Saved source, member names, and authorized write actions.
 * @returns Source status and a retained consent and account-selection form.
 */
export function ActivitySourcePanel(props: SourceProps) {
  const { source } = props;
  return (
    <section className={styles.source}>
      <div className={styles.rowHeading}>
        <h3>{source.institutionName || 'Linked institution'}</h3>
        <span className={styles.state}>{sourceState(source.state)}</span>
      </div>
      <p className={styles.note}>
        {source.kind === 'depository' ? 'Bank and card activity' : 'Investment activity'}
        {source.lastSyncedAt
          ? ` · Last saved ${activityDate(source.lastSyncedAt)}`
          : ' · No history saved yet'}
      </p>
      {source.historyStart && source.historyEnd && (
        <p className={styles.note}>
          Saved history: {activityDate(source.historyStart)} through{' '}
          {activityDate(source.historyEnd)}.
        </p>
      )}
      {!source.allowed && (
        <p className={styles.notice}>
          Activity history is not enabled for this connection environment.
        </p>
      )}
      {!source.canManage && (
        <p className={styles.notice}>
          The member who linked this institution can change its history settings.
        </p>
      )}
      <SourceForm {...props} />
    </section>
  );
}

/**
 * Keeps original source revisions and newer account edits through asynchronous saves.
 * @param props Saved source and explicit settings and permission actions.
 * @returns Read-only history consent and selected account ownership.
 */
function SourceForm(props: SourceProps) {
  const { draft, value, disabled, change, save, reset } = useSourceDraft(props);
  const { source } = props;
  return (
    <form
      className={styles.form}
      onSubmit={(event) => {
        event.preventDefault();
        void save();
      }}
    >
      {draft && activityAccountListChanged(draft.accounts, source.accounts) && (
        <p className={styles.notice}>
          The account list changed. Your existing choices are kept. New accounts start excluded.
          Removed or unavailable accounts no longer count.
        </p>
      )}
      <SourceConsent source={source} value={value} disabled={disabled} onChange={change} />
      <SourceAccounts
        source={source}
        members={props.members}
        accounts={value.accounts}
        disabled={disabled || !source.allowed}
        onChange={(accounts) => {
          change({ accounts });
        }}
      />
      <SourceActions
        source={source}
        draft={draft}
        busy={props.busy}
        disabled={disabled}
        reset={reset}
        onAuthorize={props.onAuthorize}
      />
    </form>
  );
}

/** Retains original revisions and newer source edits across a save.
 * @param props Current source and explicit settings save.
 * @returns Source draft and field, save, and reset actions.
 */
function useSourceDraft(props: SourceProps) {
  const [draft, setDraft] = useState<SourceDraft | null>(null);
  const { source } = props;
  const initial = {
    expectedRevision: source.revision,
    enabled: source.enabled,
    accounts: reconcileActivityAccounts(source.accounts, source.accounts),
    edit: 0,
  };
  const value = draft
    ? { ...draft, accounts: reconcileActivityAccounts(draft.accounts, source.accounts) }
    : initial;
  const disabled = !source.canManage;
  /** Retains the original revision when a source draft changes.
   * @param patch Changed source fields.
   */
  function change(patch: Partial<SourceDraft>) {
    setDraft((previous) => ({
      ...(previous ?? initial),
      ...patch,
      edit: (previous?.edit ?? 0) + 1,
    }));
  }
  /** Saves this draft without discarding newer account choices. */
  async function save() {
    if (!draft || !canSaveActivitySource(source, value) || props.busy) return;
    const input = {
      expectedRevision: value.expectedRevision,
      enabled: value.enabled,
      accounts: value.accounts,
    };
    const saved = await props.onSave(source, input);
    if (saved) setDraft((current) => acceptActivityDraft(current, value, saved.revision));
  }
  return {
    draft,
    value,
    disabled,
    change,
    save,
    reset: () => {
      setDraft(null);
    },
  };
}

/** Explains read-only consent and the scope of account inclusion.
 * @param props Current source draft and field action.
 * @param props.source Saved history source.
 * @param props.value Current settings draft.
 * @param props.disabled Whether settings can be edited.
 * @param props.onChange Explicit source field action.
 * @returns History opt-in and plain consent scope.
 */
function SourceConsent(props: {
  readonly source: ActivitySource;
  readonly value: SourceDraft;
  readonly disabled: boolean;
  readonly onChange: (patch: Partial<SourceDraft>) => void;
}) {
  const { value, disabled } = props;
  return (
    <>
      <p className={styles.note}>
        Enable read-only history. Account choices control activity totals. Plaid can return history
        for other consented accounts. Review ownership below. This does not change your budget,
        balances, or goal progress.
      </p>
      {props.source.kind === 'investment' && !props.source.enabled && <InvestmentHistoryCost />}
      <label className={styles.checkbox}>
        <input
          type="checkbox"
          checked={value.enabled}
          disabled={disabled || (!props.source.allowed && !value.enabled)}
          onChange={(event) => {
            props.onChange({ enabled: event.target.checked });
          }}
        />
        Use read-only activity history
      </label>
    </>
  );
}

/** Keeps save, reset, and official permission actions explicit.
 * @param props Source revision, current draft, and permission actions.
 * @returns Deliberate source actions and stale-revision guidance.
 */
function SourceActions(
  props: Pick<SourceProps, 'source' | 'busy' | 'onAuthorize'> & {
    readonly draft: SourceDraft | null;
    readonly disabled: boolean;
    readonly reset: () => void;
  },
) {
  const { source, draft, busy, disabled, reset, onAuthorize } = props;
  return (
    <>
      {draft && draft.expectedRevision !== source.revision && (
        <p className={styles.notice}>
          These settings changed. Your draft is kept. Reset the draft to review the latest settings.
        </p>
      )}
      <div className={styles.actions}>
        <button
          className={formStyles.button}
          disabled={busy || !canSaveActivitySource(source, draft)}
        >
          Save history settings
        </button>
        <button
          type="button"
          className={formStyles.secondary}
          disabled={!draft || busy}
          onClick={() => {
            reset();
          }}
        >
          Reset draft
        </button>
        <SourceAuthorize
          source={source}
          draft={draft}
          busy={busy}
          disabled={disabled}
          onAuthorize={onAuthorize}
        />
      </div>
      <SourcePermissionNotes source={source} />
    </>
  );
}
