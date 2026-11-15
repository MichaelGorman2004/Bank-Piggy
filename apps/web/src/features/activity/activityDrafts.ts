interface RevisionDraft {
  readonly expectedRevision: number;
  readonly edit: number;
}

/**
 * Advances an accepted draft revision without clearing field edits made during the save.
 * @template T Feature-owned draft with its original revision and local edit sequence.
 * @param current Current draft after the awaited save.
 * @param submitted Exact draft sent to the server.
 * @param revision Revision returned by that accepted save.
 * @returns No draft when unchanged; otherwise the newer edits with safe revision ownership.
 */
export function acceptActivityDraft<T extends RevisionDraft>(
  current: T | null,
  submitted: T,
  revision: number,
): T | null {
  if (!current || current.edit === submitted.edit) return null;
  if (current.expectedRevision !== submitted.expectedRevision) return current;
  return { ...current, expectedRevision: revision };
}
