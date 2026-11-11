import { CarFront, ReceiptText, Repeat2 } from 'lucide-react';
import { getProvider } from './catalog';
import type { ProviderIconProps } from './types';
import styles from './Providers.module.css';

/**
 * Shows a local brand image or a category icon alongside the payment name.
 * @param props Provider identity, payment kind, and optional pixel size.
 * @returns A decorative icon whose adjacent text supplies its accessible name.
 */
export function ProviderIcon(props: ProviderIconProps) {
  const provider = getProvider(props.providerId);
  const size = props.size ?? 20;
  if (provider) {
    return (
      <img
        className={styles.logo}
        src={`/assets/providers/${provider.id}.svg`}
        width={size}
        height={size}
        alt=""
        aria-hidden="true"
      />
    );
  }
  if (props.kind === 'loan') return <CarFront size={size} aria-hidden="true" />;
  if (props.kind === 'subscription') return <Repeat2 size={size} aria-hidden="true" />;
  return <ReceiptText size={size} aria-hidden="true" />;
}
