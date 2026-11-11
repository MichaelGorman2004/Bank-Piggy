/**
 * Returns a safe description for request feedback.
 * @param error A caught request failure.
 * @returns A message the user can act on.
 */
export function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : 'Something went wrong. Try again.';
}
