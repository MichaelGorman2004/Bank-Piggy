import type {
  ActivitySettingsRequest,
  ActivitySource,
  LinkTokenRequest,
} from '@bank-piggy/contracts';
import { sourceAuthorization } from './sourceAuthorization';
import styles from './Activity.module.css';
import formStyles from '../plan/Editors.module.css';

/**
 * Shows separate investment-history billing before an initial source opt-in can be saved.
 * @returns Product and billing scope without guessed prices or Trial coverage promises.
 */
export function InvestmentHistoryCost() {
  return (
    <p className={styles.notice}>
      Plaid Investments transaction history is a separate product. Your Trial allowance may cover
      it. On a paid plan, enabling history can start a separate subscription. Turning history off
      here stops reads. It does not cancel provider billing.
    </p>
  );
}

/**
 * Explains reconnect order, separate bank permission, and the effect of disabling reads.
 * @param props Current saved history source.
 * @param props.source Read-only source permission state.
 * @returns Permission and disable notices.
 */
export function SourcePermissionNotes(props: { readonly source: ActivitySource }) {
  const { source } = props;
  return (
    <>
      {source.enabled && source.state === 'reconnect' && (
        <p className={styles.notice}>
          {source.kind === 'investment'
            ? 'Reconnect the institution to restore read-only investment access.'
            : 'Reconnect the institution first. After reconnect, bank history may need a separate permission.'}
        </p>
      )}
      {source.enabled &&
        source.state !== 'reconnect' &&
        source.kind === 'depository' &&
        source.needsConsent && (
          <p className={styles.notice}>
            Save your account choices, then authorize read-only history in official Plaid Link.
          </p>
        )}
      <p className={styles.note}>
        Turning history off stops new history reads. It does not remove institution access or
        confirm that a provider subscription ended.
      </p>
    </>
  );
}

/**
 * Uses ordinary reconnect before any separate bank-history consent.
 * @param props Saved source, current draft, and explicit Link action.
 * @param props.source Saved source permission state.
 * @param props.draft Current settings draft, if any.
 * @param props.busy Whether another mutation owns the feature.
 * @param props.disabled Whether the current member can manage this source.
 * @param props.onAuthorize Explicit official Link start action.
 * @returns The relevant permission action after account choices are saved.
 */
export function SourceAuthorize(props: {
  readonly source: ActivitySource;
  readonly draft: ActivitySettingsRequest | null;
  readonly busy: boolean;
  readonly disabled: boolean;
  readonly onAuthorize: (input: LinkTokenRequest) => void;
}) {
  const authorization = sourceAuthorization(props.source);
  if (!authorization) return null;
  return (
    <button
      type="button"
      className={formStyles.secondary}
      disabled={props.disabled || props.busy || Boolean(props.draft)}
      onClick={() => {
        props.onAuthorize(authorization.input);
      }}
    >
      {authorization.label}
    </button>
  );
}
