import { useState } from 'react';
import type { ConnectedAccount } from '@bank-piggy/contracts';
import { mappingDraft, settleMappingDraft } from './mappingDraft';
import { saveMapping } from './api';
import { useConnectionData } from './ConnectionsContext';

export interface AccountMappingOptions {
  readonly account: ConnectedAccount;
  readonly connectionId: string;
  readonly csrfToken: string;
}

/**
 * Owns an explicit account draft and retains newer local edits after an accepted save.
 * @param options Current snapshot, connection identity, and session mutation token.
 * @returns Mapping draft, validation feedback, and save and reset actions.
 */
export function useAccountMapping(options: AccountMappingOptions) {
  const state = useConnectionData();
  const [draft, setDraft] = useState(() => mappingDraft(options.account));
  const [error, setError] = useState<string | null>(null);
  /** Saves the submitted mapping without discarding edits made during the request. */
  async function save() {
    setError(null);
    if (draft.enabled && draft.classification === 'unclassified') {
      setError('Choose the account type before including its balance.');
      return;
    }
    const submitted = draft;
    const next = await state.run(
      () =>
        saveMapping(
          { connection: options.connectionId, account: options.account.id },
          submitted,
          options.csrfToken,
        ),
      'Account settings saved.',
    );
    const accepted = next?.connections
      .find((connection) => connection.id === options.connectionId)
      ?.accounts.find((account) => account.id === options.account.id);
    if (accepted) setDraft((current) => settleMappingDraft(current, submitted, accepted));
  }
  return {
    draft,
    setDraft,
    error,
    save,
    busy: state.busy,
    changed: draft.revision !== options.account.mappingRevision,
    reset: () => {
      setDraft(mappingDraft(options.account));
      setError(null);
    },
  };
}
