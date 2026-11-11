import { useId, useState } from 'react';
import { findProvider, getProvider, PROVIDERS } from './catalog';
import { ProviderIcon } from './ProviderIcon';
import type { ProviderFieldProps } from './types';
import styles from './Providers.module.css';

/**
 * Searches the local service catalog while preserving a partially typed query.
 * @param props Saved provider, change handler, and optional disabled state.
 * @returns A labeled provider input with native suggestions.
 */
export function ProviderField(props: ProviderFieldProps) {
  const listId = useId();
  const [query, setQuery] = useState(() => getProvider(props.value)?.name ?? props.value ?? '');

  /**
   * Keeps free text local and saves a provider only when a catalog name matches.
   * @param input The native field whose current text needs validation.
   */
  function updateQuery(input: HTMLInputElement) {
    const value = input.value;
    const provider = findProvider(value);
    setQuery(value);
    input.setCustomValidity(
      value.trim() && !provider
        ? 'Choose a listed service or clear this field. Use the expense name for other providers.'
        : '',
    );
    props.onChange(provider?.id ?? '');
  }

  return (
    <label className={styles.field}>
      Service provider (optional)
      <span className={styles.inputRow}>
        <span className={styles.preview}>
          <ProviderIcon providerId={props.value} />
        </span>
        <input
          value={query}
          list={listId}
          autoComplete="off"
          disabled={props.disabled}
          placeholder="Search services"
          onChange={(event) => {
            updateQuery(event.currentTarget);
          }}
        />
      </span>
      <datalist id={listId}>
        {PROVIDERS.map((provider) => (
          <option key={provider.id} value={provider.name} />
        ))}
      </datalist>
    </label>
  );
}
