# Private Railway deployment

Run one Bank Piggy application service and one PostgreSQL service in the same Railway project and environment. The Go process serves the built React application and API on one HTTPS origin. Only PostgreSQL needs persistent storage.

## Build and runtime

The root [Dockerfile](../../Dockerfile) uses official Node.js 24.22.2 and Go 1.27.4 build images, pinned to image-index digests verified from Docker Hub on 2026-11-11. The builder checks generated contracts and provider assets, then runs the root build script. Build tools and npm packages stay in the builder. The final image contains a static Go binary, browser files, and trusted TLS certificates for Google HTTPS requests. It runs as user and group `65532`.

The image starts in private mode on `0.0.0.0:8080`. The built browser files are at `/app/web`. No `.env` file is loaded. Missing private configuration stops startup before the listener opens. Demo mode still requires a loopback listener and origin.

[`.dockerignore`](../../.dockerignore) allows only build inputs. It excludes Git data, local dependencies, build output, environment files, credentials, logs, and documentation screenshots. Supply credentials as runtime Railway variables, never as Docker build arguments.

Railway's current documentation says new services cannot use `railway.toml` or `railway.json`. Existing files stop being read on 2026-12-01. This setup uses Dockerfile detection and the dashboard settings below. [Railway Config as Code](https://docs.railway.com/config-as-code) describes this limit. A later infrastructure import can use Railway's supported [Infrastructure as Code](https://docs.railway.com/infrastructure-as-code) workflow.

## Create the services

1. Use PostgreSQL 18 in Railway and CI. The Railway service uses `ghcr.io/railwayapp-templates/postgres-ssl:18`, private DNS `postgres.railway.internal`, and the attached `postgres-volume`. Public database access is disabled. Keep the initialized volume on PostgreSQL 18 and its network private. See [Railway PostgreSQL](https://docs.railway.com/databases/postgresql).
2. Connect the application service to `MichaelGorman2004/Bank-Piggy` on the `main` branch and enable **Wait for CI**. Production uses this GitHub connection for automatic deployment. Keep the source root at the repository root. Railway must build the root `Dockerfile`. Leave custom build, start, and pre-deploy commands empty; the image supplies its entrypoint and the app performs its startup migration.
3. Set one application replica. Add an HTTPS Railway domain with target port `8080`. Use that exact origin for `APP_BASE_URL` and Google sign-in.
4. Configure a Google OAuth client of type **Web application**. Add `https://YOUR-DOMAIN/auth/google/callback` as an authorized redirect URI. The server requests only OpenID Connect, email, and profile scopes. Google permits these basic identity scopes without requiring the account to be a Testing test user. Bank Piggy's server-side invitations and saved membership enforce access; Google's test-user list is not the security boundary. Add the invited accounts as test users for setup clarity. See [Google's audience rules](https://support.google.com/cloud/answer/15549945) and [Google OpenID Connect](https://developers.google.com/identity/openid-connect/openid-connect).
5. Set the application variables below. In the database reference, replace `Postgres` with the exact PostgreSQL service name. Store the Google secret and database connection as private service variables.

| Variable                              | Value                                           |
| ------------------------------------- | ----------------------------------------------- |
| `APP_MODE`                            | `private`                                       |
| `PORT`                                | `8080`                                          |
| `LISTEN_ADDR`                         | `0.0.0.0:8080`                                  |
| `WEB_DIST_DIR`                        | `/app/web`                                      |
| `APP_BASE_URL`                        | `https://YOUR-DOMAIN`                           |
| `DATABASE_URL`                        | `${{Postgres.DATABASE_URL}}`                    |
| `GOOGLE_CLIENT_ID`                    | Google web client ID                            |
| `GOOGLE_CLIENT_SECRET`                | Google web client secret                        |
| `HOUSEHOLD_INVITES`                   | One or two distinct invited emails, comma apart |
| `RAILWAY_DEPLOYMENT_DRAINING_SECONDS` | `15`                                            |

Set `PORT` explicitly in Railway. Railway uses it for routing and health checks; the Go app reads `LISTEN_ADDR`. Keep both port values at `8080`. The app's default loopback listener remains safe for local use outside the container. [Railway health checks](https://docs.railway.com/deployments/healthchecks)

Set the application service's **Healthcheck Path** to `/healthz` and **Healthcheck Timeout** to `60` seconds. Use an **On Failure** restart policy with a maximum of `3` retries. Keep application sleep disabled for this private service. [Railway restart policies](https://docs.railway.com/deployments/restart-policy)

Startup shares a 20-second deadline across PostgreSQL migrations, Google discovery, and household initialization. A longer platform health-check window does not extend this app deadline. PostgreSQL must be ready before the app starts. A failed startup can restart under the policy above.

The Go process handles `SIGTERM` and gives requests up to 10 seconds to finish. The 15-second Railway drain setting gives the process time to complete that shutdown. See [Railway variable reference](https://docs.railway.com/variables/reference).

## Automatic deployment

Production is connected to `MichaelGorman2004/Bank-Piggy` on `main`, with **Wait for CI** enabled. The existing Railway GitHub App has access to this repository. The application builds from the repository root and root Dockerfile. The source connection preserves the runtime variables, domain, and private database connection.

The quality workflow runs on every push, including pushes to `main`. With **Wait for CI** enabled, Railway waits for GitHub Actions before it deploys a new commit. Verify the first automatic deployment against its exact commit and served asset hashes. A local commit does not trigger deployment until it is pushed to GitHub. See [Railway GitHub automatic deployments](https://docs.railway.com/deployments/github-autodeploys).

## Verify a private release

Run the full quality gate against a separate, disposable PostgreSQL 18 database before release. CI uses `postgres:18-alpine` to match the Railway major version. The earlier local full quality gate and application-container smoke check used PostgreSQL 14.15. Record separate Go race and integration test results from a fresh PostgreSQL 18 database.

```sh
TEST_DATABASE_URL='postgresql://bankpiggy:bankpiggy@127.0.0.1:5432/bankpiggy_test?sslmode=disable' npm run check
npm run audit
docker build --platform linux/amd64 --tag bank-piggy:release .
```

After deployment, record the Railway deployment ID and source commit. Confirm the following results before calling the release usable:

- `GET /healthz` returns `200`. This endpoint checks PostgreSQL. Railway uses it during deployment; it does not continuously monitor the running service.
- The HTTPS root page and its JavaScript, CSS, image, and provider assets load from the same domain. The page offers Google sign-in.
- An unauthenticated `GET /api/household` returns `401`. The demo-session endpoint is unavailable in private mode.
- Each bootstrap Google account with a saved profile can sign in. A valid household link can add and authorize the second person. An account without either authorization is rejected. Session cookies have `Secure` and `HttpOnly`.
- A new private household opens with one person and an empty plan. A manual edit survives reload and a service restart. Add a second profile, then verify its sign-in, both member views, and a conflicting edit in two windows. Preserve existing two-person plans.
- Check mobile, keyboard, empty, populated, and error flows on the deployed origin. Keep personal amounts, email addresses, session cookies, and secrets out of screenshots and logs.

Local tests with a signed provider simulate Google. A successful image build does not prove live sign-in, deployed persistence, or backups. Plaid balances have separate [release evidence](plaid-verification.md). New Activity results belong in [Activity verification](activity-verification.md); Sandbox evidence does not establish real-bank or history coverage.

Pending operational followups: configure external request limits for public routes, with specific limits for sign-in and callback requests. Add continuous uptime checks and alerts for `/healthz`, plus database capacity alerts. Railway's deployment health check alone does not provide these controls.

## Release credentials

Use credentials limited to the intended project for repeatable deployments. Keep credentials outside source files, release archives, screenshots, and logs. A general Railway CLI login can request workspace administration and persistent access beyond one deployment.

After a one-time release, revoke the temporary CLI authorization in **Account → Apps → Railway CLI → Revoke Access**. Railway states that this invalidates the app's tokens. Then run `railway logout` to clear the stored local login. Verify that the authorized-app entry is absent and an authenticated CLI command fails. Logging out alone clears local credentials; it does not prove that the server authorization was revoked. See [authorized apps](https://docs.railway.com/integrations/oauth/authorized-apps), [CLI logout](https://docs.railway.com/cli/logout), and [CLI authentication](https://docs.railway.com/cli#authentication).

Review the workspace audit log for unexpected billing, membership, SSH-key, workspace, and unrelated project changes. Record the period reviewed and its limits. Revoking the deployment CLI must leave Google sign-in, runtime variables, the database, and application services intact.

## Data safety and recovery

The Railway Hobby project's Backups page requires Pro to create native backups, set schedules, or enable point-in-time recovery. No paid plan change was made. On 2026-11-12, a manual PostgreSQL backup was created through the private console and downloaded with Michael's approval. Restore into isolated PostgreSQL 18 and the actual application migration preserved all pre-existing records. See [backup evidence](plaid-verification.md#production-release-preparation). Scheduled backups and off-device recovery remain unconfigured. Keep the encryption key ring separately from database backups, and keep fresh recovery copies after importing real bank data. PostgreSQL volume storage alone is not a backup. [Railway backups](https://docs.railway.com/volumes/backups)

The app applies schema and account upgrades in one PostgreSQL transaction at startup. It does not replace an existing plan. Back up the database before a release that changes stored data. Do not run destructive migration or reset commands on the private household.

For an application failure, select the last known working Railway deployment and redeploy it with the same private variables and database. Check database compatibility first; rolling back an image does not reverse a database upgrade. Repeat the health, sign-in, and saved-plan checks. Restore data only from a verified backup through a separate recovery plan.

When the domain changes, update both `APP_BASE_URL` and Google's redirect URI before sign-in testing. Configure one or two invitations and keep their order stable: the first maps to `member-1` and the second to `member-2`. A second bootstrap account requires its saved financial profile. An accepted household link can append that profile and does not require an environment change. The setup button offers a link after saving. An invitation binds to its first verified Google subject; changing invitation text does not reassign an existing identity.

For an invitation release, verify link creation, copy, replacement, revoke, accepted Google sign-in, and a later normal sign-in on the deployed origin. Confirm that an old link cannot block normal sign-in. For To-do, verify a saved pay schedule, manual and automatic bills, checkbox reload, source-change invalidation, and a stale write. Use test-only records or approved user inputs; do not log invitation secrets. Local signed-provider checks do not establish these live results.

## Plaid configuration

Plaid is optional and disabled by default. Keep every variable below on the Go service. Do not use Vite variables or build arguments. The application uses fixed Sandbox or Production SDK endpoints. No arbitrary provider URL is configurable.

| Variable                             | Purpose                                                                                                                                                                                                  |
| ------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `PLAID_ENABLED`                      | Set exactly `true` to enable provider calls. Omit to disable provider calls while keeping the manual planner available.                                                                                  |
| `PLAID_ENV`                          | `sandbox` for development. `production` requires the extra approval and budget settings.                                                                                                                 |
| `PLAID_CLIENT_ID`                    | Plaid team client ID.                                                                                                                                                                                    |
| `PLAID_SECRET`                       | Secret for the selected environment.                                                                                                                                                                     |
| `PLAID_ACTIVE_KEY_ID`                | ID of the current encryption key.                                                                                                                                                                        |
| `PLAID_TOKEN_KEYS`                   | Comma-separated `id:base64` key entries. Every decoded key must contain 32 random bytes. Keep old IDs during rotation.                                                                                   |
| `PLAID_PRODUCTION_APPROVED`          | Set exactly `true` only after checking the team's actual eligible Trial plan and remaining capacity.                                                                                                     |
| `PLAID_LIFETIME_ITEM_LIMIT`          | Approved remaining local production reservation budget. The application accepts 1 through 10; this is a code cap, not proof of the team's actual allowance. Do not refund deleted or uncertain attempts. |
| `PLAID_BANK_ACTIVITY_APPROVED`       | Defaults to `false`. Set exactly `true` only after checking Transactions access, consent, and costs for the actual plan.                                                                                 |
| `PLAID_INVESTMENT_ACTIVITY_APPROVED` | Defaults to `false`. Set exactly `true` only after checking separate Investments Transactions access and subscription terms.                                                                             |

Generate encryption keys with an established cryptographic generator. Store them as private runtime secrets, separately from database backups. Do not copy secret output into tickets, screenshots, command logs, or this guide. See [key rotation and recovery](plaid.md#encryption-and-operations).

The server derives the allowlisted OAuth redirect as `APP_BASE_URL/plaid-return` and the webhook as `APP_BASE_URL/api/plaid/webhook`. Configure the exact redirect in the Plaid Dashboard. Production requires the existing HTTPS origin. A local Sandbox loopback app can test application behavior, but Plaid cannot deliver webhooks to an unreachable loopback address. Use an approved HTTPS development origin for a real webhook delivery check. There is no signature bypass for development.

Before any live Link flow, inspect the actual Plaid Dashboard plan, created-Item count, and product eligibility. Coordinate with other apps on the same team. Configure only the remaining approved allowance. Local counters cannot discover external team usage. Do not upgrade the plan, accept a paid agreement, or create repeated live Items to test. Reconnect existing Items through update mode.

Activity approvals do not enable any source by themselves. Each existing connection also needs saved opt-in by its linking member. Bank history needs a session-bound activity-purpose update Link with `additional_consented_products=[transactions]`. Investment history uses existing Investments consent, but the first `/investments/transactions/get` can add the separate Transactions subscription. Show that disclosure before saving opt-in. Turning history off stops new reads; it does not end provider billing. Review actual Trial or paid terms before enabling either gate. [Plaid product initialization](https://plaid.com/docs/link/initializing-products/) and [billing](https://plaid.com/docs/account/billing/)

The single Go process runs separate balance and activity jobs. PostgreSQL owns their independent due times, leases, generations, and failures. Ordinary history and balance attempts reserve at least 30 minutes before another read, including failures and crash recovery. First import and consent repair can run immediately. Provider background updates can arrive later. Opening Activity, Sources, or Accounts reads saved data; it does not force an institution refresh. Keep the current one-replica deployment until overlap and operational checks justify a change. Shutdown cancels worker work before the database pool closes.

The additive activity migration extends Link attempts with a purpose and settings revision. It adds `bank_activity_jobs`, `bank_activity_records`, and `bank_activity_cancellations` without enabling existing sources or replacing manual plans. Back up these tables with the household, connection, account, and review data in one database snapshot. Keep the encryption key ring separately. Verify restoration into isolated PostgreSQL 18 before release. Raw activity retention is 730 days; cancellation identifiers survive removed history windows until account or Item purge. A Railway volume is not a backup. This change does not configure scheduled backups or prove a new restore result.

For rollback, retain the added tables and keys. A previous image can leave activity jobs idle while the provider subscription remains active. Close the activity approval gates and preserve the saved state; do not drop tables or keys as a rollback step. Confirm that pending Item removals still have a compatible worker. Disabling Plaid stops calls; it does not revoke consent or cancel subscriptions. Complete approved disconnects with a compatible worker or arrange Dashboard removal before retiring the feature. Item removal also ends its balance connection. Do not delete the only revocation credential before removal succeeds.

### Provider release checks

1. Pass the root gate with a disposable PostgreSQL 18 database, dependency audits, and the production Docker build. Record provider simulations separately.
2. Verify the exact deployed commit, `/healthz`, private access, database migration, and served asset hashes.
3. With the member's consent and verified remaining Trial allowance, complete Link on the deployed origin for each exact institution and account type under test.
4. Confirm ownership, classification, mapping, current and missing balances, currency, scheduled updates, freshness labels, OAuth resume, reconnect, and revocation failure/retry.
5. Deliver a real signed Sandbox webhook before relying on live delivery. Reject a tampered notification. Do not record token-bearing headers or bodies in evidence.
6. Verify restart recovery, no late synchronization after disconnect, and a restored database with the matching key ring.
7. Keep both Activity approval flags false for the initial deployment. Confirm that opening Activity/Sources and ordinary balance linking make no history reads and add no history consent.
8. With separate explicit approval, test bank update-mode history consent and existing-investment opt-in. Check the actual Dashboard product and billing state. Verify that OAuth recovery keeps the exact attempt purpose and never creates a replacement Item.
9. Check per-source ownership and inclusion, missing history, initializing/delayed/reconnect states, corrections, removals, investment cancellation, ambiguous transfer review, and cross-month paired revisions. Missing data must not appear as confirmed zero.
10. Confirm that disabling invalidates a held read, revocation/disconnect purges history, and 730-day cleanup removes raw data while keeping cancellation and surviving review state safe. Verify that balance failures do not stop eligible activity jobs.
11. Review a 180-day net-deposit pay suggestion and explicitly select a member. Confirm that only net pay, frequency, and calendar change. Reject stale evidence or plan revisions. Do not infer gross pay, taxes, 401(k) splits, matching, expense budgets, or goal progress.

Local tests do not establish these deployed or live results. No live Item or paid usage is authorized by a successful local gate.
