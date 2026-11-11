# Bank Piggy API

A shared monthly finance plan for a two-person household. This layer contains the Go API, PostgreSQL persistence, generated contracts, and quality checks. The browser dashboard is added by the next review branch.

## Run locally

Use Node.js 24+, Go 1.26+, Python 3, PostgreSQL 17, and golangci-lint 2.15.0. The quality workflow uses Go 1.27.4.

```sh
npm ci
cp .env.example .env
npm run hooks:install
docker compose up -d --wait db
npm run dev
```

The API listens at `http://127.0.0.1:8080`. Demo mode is explicit and loopback-only. Private mode fails closed until Google sign-in and invitations are configured. The demo uses fictional reference data.

## API and data rules

- `POST /api/demo/session` opens the local demo session. Send `Origin: http://127.0.0.1:8080`.
- `GET /api/session` reads cookie-backed identity. `GET /api/household?month=YYYY-MM` reads an authorized plan and monthly/yearly projections.
- `PUT /api/household?month=YYYY-MM` saves a full validated plan with its current revision and the session's `X-CSRF-Token`. A stale revision returns `409`.
- `POST /api/logout` revokes the session. Private Google OpenID Connect routes are `/auth/google` and `/auth/google/callback`.
- `GET /healthz` checks PostgreSQL connectivity.

Money uses integer cents. Shared costs count once; personal shares reconcile exactly. Scheduled payments include their final month. Member account snapshots identify traditional/Roth 401(k), traditional/Roth IRA, brokerage, high-yield savings, savings, or other investments. Savings and investment totals are derived from those accounts. The atomic migration retains legacy totals under generic account types and advances saved revisions.

Monthly goal contributions remain separate from manual saved/target progress. Ordinary savings, trips, and purchases have exclusive target metadata. Trip and purchase suggestions use inclusive months and ceiling-to-cent rounding. Changing progress does not change account balances or chosen monthly contributions.

## Quality checks

```sh
TEST_DATABASE_URL=postgresql://bankpiggy:bankpiggy@127.0.0.1:5432/bankpiggy_test?sslmode=disable npm run check
npm run audit
```

The gate checks generated contracts, formatting, authored source limits, strict JavaScript/Go lint, contract types, unused dependencies, Go race tests, and the API build. The installed pre-commit hook and GitHub workflow run this gate. Integration tests require real PostgreSQL. Authentication tests use a signed local identity provider; live Google and bank connections remain unverified.

Functions stay within 50 nonblank, noncomment lines, authored source files within 300 physical lines, and cyclomatic complexity within 10. See [engineering rules](AGENTS.md) and [product scope](docs/product-brief.md).

## Repository

`apps/api` owns HTTP, authorization, financial calculations, and persistence. `packages/contracts/openapi.json` owns the canonical contract. Run `npm run contracts:generate` after a contract edit.

[MIT license](LICENSE).
