export interface ProviderIconProps {
  readonly providerId?: string | undefined;
  readonly kind?: string | undefined;
  readonly size?: number;
}

export interface ProviderFieldProps {
  readonly value: string | undefined;
  readonly onChange: (providerId: string) => void;
  readonly disabled?: boolean;
}
