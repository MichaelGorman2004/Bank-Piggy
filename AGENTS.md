# Bank Piggy engineering rules

Apply the global engineering rules in `~/.codex/AGENTS.md` and the agreed product
scope in `docs/product-brief.md` before implementation.

- Keep the Go backend in `apps/api`, the TypeScript frontend in `apps/web`, and
  the canonical JSON contract in `packages/contracts/openapi.json`.
- Generate API types from that contract. Keep validation, calculations,
  persistence, orchestration, and presentation with their owning features.
- Use integer cents for money. Reconcile split rounding so personal shares equal
  the household amount. Count shared expenses once in household totals.
- Keep authored functions within 50 nonblank, noncomment lines and authored
  source files within 300 physical lines. Keep cyclomatic complexity at 10 or
  below. Split by responsibility and avoid pass-through wrappers.
- Use canonical types, named constants, and feature-owned pure helpers. Add
  concise documentation to public functions and nontrivial contracts.
- Define colors in the frontend token file. Use accessible native controls,
  visible focus, responsive layouts, and readable chart labels.
- Preserve edits during requests. Handle failures, stale revisions, and cleanup
  without discarding newer data or the original error.
- Keep financial data and credentials out of logs. Check sessions and household
  membership on the server. Demo access must require an explicit demo mode and
  a loopback listener. Private mode must fail closed when auth is unconfigured.
- Use root npm scripts for checks. Run contract and provider generation checks,
  formatting, lint, type checks, unused checks, Node tests, Go race tests, and
  both builds. Use real PostgreSQL for persistence checks.
- Review the final diff and affected consumers. Verify populated, empty, error,
  mobile, and keyboard flows in a browser before handoff.
- Preserve unrelated changes. Leave work uncommitted unless Michael requests a
  commit. Report sample data, provider simulations, and unverified live paths.
