import catalog from './catalog.json';

export type Provider = (typeof catalog)[number];
export const PROVIDERS: ReadonlyArray<Provider> = catalog;

/**
 * Resolves a catalog entry before an identifier can become an image path.
 * @param id Optional saved provider identifier.
 * @returns The known provider, or undefined for custom payments.
 */
export function getProvider(id: string | undefined): Provider | undefined {
  return PROVIDERS.find((provider) => provider.id === id);
}

/**
 * Matches a service selected through the native searchable options.
 * @param name Entered provider name or identifier.
 * @returns A known entry when the input is an exact case-insensitive match.
 */
export function findProvider(name: string): Provider | undefined {
  const normalized = name.trim().toLocaleLowerCase('en-US');
  return PROVIDERS.find(
    (provider) =>
      provider.name.toLocaleLowerCase('en-US') === normalized || provider.id === normalized,
  );
}
