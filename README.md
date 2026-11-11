# Bank Piggy

**A shared monthly money plan for a two-person household.** Bank Piggy shows where take-home pay goes, how shared costs divide, and what remains for savings, investments, and extra debt payments.

![Bank Piggy default Overview with fictional household data](docs/system-design/images/overview.png)

The first version is a working Go and TypeScript application with PostgreSQL persistence. It combines a warm, responsive dashboard with server-owned financial calculations, invite-only Google sign-in, and explicit protection against stale saves.

## What works today

- Build a monthly plan with income, regular expenses, goals, and manually entered balances.
- Track named savings, retirement, and investment accounts by type. Derive savings and investment totals from those snapshots.
- Switch between the household view and the signed-in member's allocated share.
- Divide shared expenses evenly, by custom percentage, or by annual base salary. Record the payer separately.
- Add, edit, and remove expenses and goals. Keep several goals active in the same budget.
- Schedule subscriptions, bills, and loan payments with a start month and an inclusive final month or finite term.
- Choose from 100 service names and local logos, or keep a custom expense name.
- Build trip budgets from six cost categories or enter a purchase target. Record the amount already saved and calculate a monthly saving suggestion.
- Optionally record saved progress toward an ordinary goal. Goals without a target show their planned contribution.
- Read a Sankey money flow, summary totals, goal allocations, and balance cards from the same server calculation.
- Change the planning month to update the forecast automatically. View its next 12 months with payment schedules or an average paycheck estimate for a member with regular pay frequency.
- Save plans in PostgreSQL. Detect concurrent edits instead of overwriting another window's changes.
- Run an explicit loopback-only demo, or configure private access for exactly two invited Google accounts.

Each navigation view has one task. The household scope, period, and planning month stay consistent across them.

| View               | Content                                                            |
| ------------------ | ------------------------------------------------------------------ |
| Overview (default) | Plan totals, recorded inputs, and balance snapshots.               |
| Money flow         | Sankey of income and planned allocations.                          |
| Expenses           | Bills, subscriptions, regular costs, and loans with type filters.  |
| Goals              | Ordinary savings, investment, and debt contributions.              |
| Trips & purchases  | Manual targets, saved progress, deadlines, and saving suggestions. |

The application frame fits the visible viewport. Only card bodies scroll, using thin taupe scrollbars with transparent tracks. Card headers, actions, filters, and footers stay visible. Overview has Plan totals, At a glance, and Balances sections. At a glance compares both members in compact income and balance tables. Narrow or short layouts show one section at a time. The Money flow chart groups smaller categories when needed and preserves their full amount.

This version uses manual plans and fictional demo data. Target progress comes from the saved amount the user enters. Goal allocations and period totals are forecasts. Both invited members can read and edit the complete household plan. Bank connections, transaction history, service cancellation, loan repayment, and investment advice are outside this version.

[Mobile Overview](docs/system-design/images/mobile.png) · [Money flow](docs/system-design/images/money-flow.png) · [Mobile Money flow](docs/system-design/images/mobile-money-flow.png) · [Expenses](docs/system-design/images/expenses.png) · [Mobile Expenses](docs/system-design/images/mobile-expenses.png) · [Goals](docs/system-design/images/goals.png) · [Mobile Goals](docs/system-design/images/mobile-goals.png)

## Run locally

Use Node.js 24 or later, Go 1.26 or later, Python 3, and PostgreSQL 17. The quality workflow uses Go 1.27.4. Docker with Compose can provide PostgreSQL. Install `golangci-lint` 2.15.0 on your `PATH` to run the complete checks.

```sh
npm ci
cp .env.example .env
npm run hooks:install
docker compose up -d --wait db
npm run dev
```

Open **http://127.0.0.1:5173** and select **Open household plan**. Use `127.0.0.1` consistently: the configured origin must match the browser origin.

The development script reads `.env`, starts the Go API on port 8080, and starts Vite on port 5173. Vite proxies API and sign-in requests to Go. Closing the development process stops both servers. The Compose database listens only on loopback.

Demo sessions use the same persisted sample household. Changes survive sign-out and server restarts. Demo mode has a separate household from private mode. It does not isolate data by visitor or reset the sample on each visit.

The interface uses household names and labels in both modes. Local reference data remains fictional, and the server still requires explicit demo mode and loopback access.

### Private sign-in

Set the following values in `.env` for development, or supply them as environment variables to the built server:

| Variable                                   | Purpose                                                                                     |
| ------------------------------------------ | ------------------------------------------------------------------------------------------- |
| `APP_MODE=private`                         | Enables Google sign-in and disables the demo-session endpoint. This is the backend default. |
| `DATABASE_URL`                             | PostgreSQL connection string.                                                               |
| `APP_BASE_URL`                             | Exact browser origin. HTTPS is required outside loopback.                                   |
| `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET` | Credentials for the configured Google application.                                          |
| `HOUSEHOLD_INVITES`                        | Exactly two different email addresses, separated by a comma.                                |
| `LISTEN_ADDR`                              | Go listener, such as `127.0.0.1:8080` behind a local HTTPS proxy.                           |
| `WEB_DIST_DIR`                             | Built frontend path, relative to the process working directory.                             |

The callback path is `/auth/google/callback` on `APP_BASE_URL`. The first successful sign-in binds each invitation to a verified Google subject. A new private household starts with an empty plan. The frontend opens the plan setup when income is zero and no expenses or goals exist.

To serve the frontend and API from the built Go server, run the following from the repository root after exporting the private environment variables above:

```sh
npm run build
WEB_DIST_DIR=apps/web/dist ./dist/bank-piggy
```

The binary does not read `.env` itself. `npm run dev` does. The default frontend path, `../web/dist`, fits the development API working directory; the root-level binary command needs the override shown above. Hosting, HTTPS termination, backups, and a live Google login have not been verified in a deployed environment.

Startup upgrades older saved balance totals to generic Savings and Other investments accounts in one database transaction. Existing typed accounts keep their values and order. The upgrade advances affected plan revisions once, so an editor opened before the upgrade must reload. See [data upgrades](docs/system-design/data-model.md#account-upgrade).

## Validate changes

Create a disposable test database once:

```sh
docker compose exec -T db createdb -U bankpiggy bankpiggy_test
```

Run the complete quality gate with real PostgreSQL integration tests:

```sh
TEST_DATABASE_URL='postgresql://bankpiggy:bankpiggy@127.0.0.1:5432/bankpiggy_test?sslmode=disable' npm run check
```

Each integration test creates and removes its own schema. The test database user must have permission to create schemas. Without `TEST_DATABASE_URL`, database-backed tests skip; a passing command alone then does not prove persistence behavior.

`npm run check` checks generated contracts and provider assets, Prettier and Go formatting, source size, ESLint, Go lint, TypeScript types, unused code, Node regression tests, Go tests with the race detector, and both builds. The [pre-commit hook](.githooks/pre-commit) runs this gate after `npm run hooks:install`. Fresh clones need that installation step. The [quality workflow](.github/workflows/quality.yml) configures PostgreSQL and also runs `npm run audit`.

The audit command checks npm advisories and runs the pinned official Go vulnerability checker. Weekly dependency-update configuration covers npm, Go modules, and GitHub Actions. The workflow and update configuration become active after the repository changes are pushed.

| Evidence                       | Boundary                                                                                                                                      |
| ------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------- |
| Go calculation and HTTP tests  | Split conservation, payment dates, 12-month forecasts, target projections, validation, sessions, and API behavior.                            |
| Node regression tests          | Draft conversion, inclusive payment terms, exact cents, month links, and local provider assets.                                               |
| PostgreSQL integration tests   | Real persistence, identity binding, expiry, and competing revisions.                                                                          |
| Local signed identity provider | Signature and claim verification, nonce, PKCE, and callback replay checks. This simulates Google.                                             |
| Browser review                 | Local sample data and manual plans. See the current verification note in [system design](docs/system-design/README.md#verification-boundary). |
| Live services                  | No deployment, live Google sign-in, or connected bank account is claimed.                                                                     |

## Read the design

- [Architecture and request flow](docs/system-design/README.md)
- [Data model and financial rules](docs/system-design/data-model.md)
- [Security boundaries and defenses](docs/system-design/security.md)
- [Design decisions and tradeoffs](docs/system-design/decisions.md)
- [User stories and acceptance criteria](docs/user-stories.md)
- [Product brief and longer-term scope](docs/product-brief.md)
- [Generated visual assets](docs/assets.md)

## Repository map

```text
apps/api/             Go HTTP server, calculations, authentication, PostgreSQL
apps/web/             React, TypeScript, Vite, and ECharts dashboard
packages/contracts/   Canonical OpenAPI contract and generated client schemas
scripts/              Development runner and source-size check
docs/                 Product scope, design, and user stories
```

The canonical API contract is [openapi.json](packages/contracts/openapi.json). Run `npm run contracts:generate` after changing it. Generated Go types, TypeScript types, and the browser's runtime schema must stay in sync.

## License

[MIT](LICENSE).
