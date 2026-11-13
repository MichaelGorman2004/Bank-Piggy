# Plaid local verification

Verification dates: 2026-11-11 and 2026-11-12. The development sections record the original local verification boundary. Production release evidence is recorded at the end of this document. No live bank consent or paid-plan change was performed by the agent.

## Automated checks

The root `npm run check` passed with `TEST_DATABASE_URL` pointing to a separate PostgreSQL 18.6 container. Each persistence test created and removed its own schema. Existing local services and the unrelated `output/` folder were preserved.

The gate included contract generation, all 100 provider assets, formatting, source limits, frontend and Go lint, types, unused code, Node tests, Go race tests, and both builds. All 89 Node tests passed; none were skipped. Go tests passed in banking, budget, server, and store. The banking package contains 34 tests. Source checks accepted 287 authored files with the existing 300-line limit. No gate, assertion, hook, CI setting, or lint policy was weakened.

`docker build --platform linux/amd64 --tag bank-piggy:plaid-review .` passed. The local image was built from this working tree. It was not published or deployed.

`npm run audit` passed. npm reported zero vulnerabilities. The pinned Go vulnerability checker reported no vulnerabilities.

The frontend build retains a nonfatal chunk-size warning: the main browser bundle is about 519 kB before gzip and 154 kB after gzip. The Plaid Link adapter loads separately. The warning threshold was not changed.

## Persistence and provider boundary

The tests use real PostgreSQL and labeled provider simulations. SDK transport tests exercise the official Plaid client against local HTTP responses. They prove decimal parsing and request/error handling, not provider availability.

Covered cases include:

- Missing sessions, foreign household/member/connection/account/manual identifiers, connecting-member management, exact origin, and CSRF rejection.
- Fresh Link retries, same-token OAuth continuation, expired and repeated exchange, session binding, and production reservations before Link.
- Incomplete encryption configuration, authenticated ciphertext, record-context binding, old-key recovery, and token confidentiality.
- Exact decimal cents, missing and negative balances, unsupported types and currencies, explicit mappings, shared cent reconciliation, and duplicate manual replacement rejection.
- Stable account IDs and classifications, current plan revision in the projection, preserved plan and task data, and exact attempt completion markers.
- Signed webhook validation, stale/future/tampered inputs, expired keys, rotated-key bounds, alternate ECDSA representations, duplicate delivery, account revocation, and consent renewal.
- Provider reconnect and removal failures, durable cleanup after failed token attachment, and retained revocation credentials.
- Controlled Link creation, token exchange, and synchronization races with disconnect. A late result cannot restore removed data.
- A held provider read while a newer manual plan edit commits. The complete accepted plan remains unchanged.
- A PostgreSQL trigger failure on the second imported account. The complete snapshot and success timestamp roll back. Restart/retry applies the complete snapshot once.
- Persistent snapshot-write failures stop provider reads after the durable five-failure budget. Stale workers do not penalize newer work.
- Expired leases, stopped automatic retries, restart recovery, additive migration preservation, Sandbox/Production separation, and disabled-provider manual fallback.

## Independent reviews

An independent security/correctness reviewer and an independent UI reviewer reviewed the changes. Findings were fixed and reviewed again. Neither reviewer authored implementation changes.

Security fixes included provider error extraction, environment isolation, one-use production Link handling, fresh retry versus OAuth continuation, consistent lifecycle lock order, late-exchange compensation, retained revocation credentials, webhook key expiry and replay handling, persistent consent notices, and bounded retries after snapshot-write failures.

UI fixes included a revision guard for balance projections, visible account management after an OAuth return on short screens, exact completion proof for reconnect recovery, and blocked overlapping Link actions while recovery remains unresolved. Mapping forms remain mounted while newer balance projections load.

The final review reported no open security, correctness, or UI findings within the checked local scope.

## Compiled browser review

The review used the built frontend served by the real Go application on an isolated loopback origin. No Vite hot reload was active. The disabled/empty baseline used the real API and PostgreSQL. Populated bank rows and provider interactions were browser-only simulations. No fictional bank records were inserted into the running private product.

Checks passed at 320×568, 390×844, 844×390, and 1366×768. The document stayed within the viewport. Long lists scrolled inside the dialog. Its header and close control stayed visible. Native controls, visible focus, Tab containment, Escape, and focus restoration passed.

The reviewer checked empty, loading, error/retry, populated, long-name and 40-account views; ownership and manual mapping; current versus available values; negative and missing values; unsupported currency; reconnect; failed removal and retry; stale mapping and read responses; and edits made while saving. Newer drafts remained intact.

The official Link adapter was tested with a simulated provider iframe. Native dialogs suspended before the iframe opened. Simulated OAuth return, exit, and success passed on narrow and short viewports without a hidden modal or automatic setup collision. Lost completion responses resolved only when the exact completed attempt was proved. An unchanged reconnect-required row did not count as success.

The final reviewed frontend bundle was `index-C2wpBpNM.js`. Review scripts, screenshots, and command logs are local artifacts under `/tmp/bank-piggy-plaid-review`; they are not a permanent release record.

## Separate evidence boundaries

| Evidence                                             | Result                                                                                                                             |
| ---------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| Local automated checks                               | Passed with real PostgreSQL 18.6, labeled provider simulations, and SDK transport tests.                                           |
| Production container build                           | Passed locally for `linux/amd64`; image `bank-piggy:plaid-review`.                                                                 |
| Compiled browser verification                        | Passed for the states and viewports above. Provider consent was simulated.                                                         |
| Official Plaid Sandbox API                           | Michael completed Sandbox consent. The worker saved 14 accounts, and account mapping plus current-balance totals passed in Safari. |
| Live institutions and exact account types            | Not run. No bank consent or live Item was created.                                                                                 |
| Actual team Trial eligibility and remaining capacity | Not inspected. Required before any production linking.                                                                             |
| Deployed release and production OAuth/webhooks       | Not run. This change was not deployed.                                                                                             |
| Backup restore and key recovery in hosting           | Not verified. Deployment and recovery checks remain required.                                                                      |

Cross-login joint-account identity still needs explicit user reconciliation. Include one copy and exclude duplicates. External consent and token-exchange crash windows remain documented in [Plaid design](plaid.md). Local tests do not establish security compliance or live institution coverage.

## Local setup and mobile follow-up

A later local follow-up added `npm run start:local`, a private ignored `.env` with a generated encryption key, a separate `bankpiggy_local` database, and the [local setup guide](../plaid-local-setup.md). No Plaid account, API credential, Trial application, or bank connection was created. The launcher rejects missing provider credentials with the exact field names. A disabled-provider launch and graceful shutdown were checked against PostgreSQL 18.

The To-do date input now fits its reserved label width. Today keeps its own width and a 44-pixel minimum height. Chrome checks at 320×568, 390×844, and 844×390 found no overlap or viewport overflow. Native keyboard date entry and Today activation passed. Safari responsive mode also showed separate controls at 320×568 and 390×844. These are desktop browser checks, not a physical iPhone test.

The root quality gate passed again after these changes: 89 Node tests, Go race tests with real PostgreSQL 18, strict checks, and both builds. It accepted 288 authored source files. The new frontend bundle was `index-B3NUC3Sl.js`. The earlier Docker image and independent Plaid review refer to the preceding feature state; no deployment was performed for this follow-up.

## First configured Sandbox request

Michael entered his Client ID and Sandbox secret directly into the private `.env` file. The agent checked only whether the fields were populated. Credential values were not displayed.

The first actual Link request exposed a constructor defect that simulated SDK servers had hidden: the official SDK expects full environment URLs, while application configuration uses `sandbox` and `production`. The constructor now selects the official `plaid.Sandbox` and `plaid.Production` constants. New tests exercise `NewProvider` itself and inspect actual SDK request destinations and credential headers. A separate test proves a redirect cannot forward credentials to another host. These are transport simulations without live credentials. The banking package now contains 36 tests.

An independent reviewer confirmed the mapping defect and fix, inspected the SDK configuration, and ran the Go race suite against PostgreSQL 18. The root quality gate then passed with all 89 Node tests, all 36 banking tests, Go race tests with real PostgreSQL, strict checks, and both builds.

The configured Sandbox API initially returned `INVALID_FIELD` because the OAuth redirect was missing from Dashboard. Michael saved `http://localhost:5188/plaid-return`. After the rebuilt app restarted, Link-token creation succeeded and the official Plaid iframe opened in Chrome. Its screen explicitly showed Sandbox mode and the Bank Piggy consent introduction. The page was left open for Michael to continue. A screenshot is at `/tmp/bank-piggy-plaid-review/official-sandbox-ready.png`.

At this stage, diagnostics did not display API credentials, access tokens, raw API responses, or financial data. Consent acceptance, test-account import, and an actual balance fetch were still unverified. No Sandbox Item or live Item was created by the agent. The app used `http://localhost:5188` in loopback-only demo mode with Sandbox enabled. Production linking remained disabled.

## Single entry and completed Sandbox import

Michael completed consent in Safari. The application saved the connection, but its first update failed. A safe existing-Item diagnostic identified two independent causes:

- Plaid returned `INVALID_FIELD` because this Item required `options.min_last_updated_datetime`. Balance requests now send UTC now minus the 24-hour synchronization interval. Investments retain the cached account endpoint. A stale-cache rejection does not trigger an unbounded read or a fallback request.
- One supported USD investment account contained a fractional-cent current value. Exact decimal conversion still rejects fractional cents. The snapshot now retains that account and marks only the unrepresentable field unavailable. It never rounds or replaces it with zero. Independent current and available values survive. Malformed and out-of-range values still reject the complete snapshot, including fractional values beyond the range limit.

The running worker then imported all 14 Sandbox accounts. Thirteen current balances were representable in cents; one remained unavailable. Safari showed **Connected**, the institution name, a successful synchronization time, and a check scheduled for the following day. Unsupported credit and loan accounts remain outside the supported balance mapping scope.

A temporary inclusion test selected the Sandbox checking account. Both Overview balance views showed its $110 current balance rather than its $100 available balance. The test then restored the original excluded, unclassified mapping. A protected baseline comparison confirmed that the saved household plan and plan revision were unchanged. The final database count was 14 imported accounts, one unavailable current balance, and zero selected accounts. Mapping revisions advanced through the reversible save test. No new provider Item was created by the agent, and the user's connection was not disconnected.

The account manager now has one **Connect accounts** entry. It opens a native **Bank accounts** or **Investments** choice, then **Continue to Plaid**. Each institution uses its own consent and allowance reservation. Native Multi-Item Link remains disabled because its shared product filter would narrow bank/brokerage coverage and its documented API supplies no per-session Item cap. See the [design rationale](plaid.md#product-choice-and-cost-boundary).

Missing selected current balances now produce a review notice above both summary balance views. The notice follows personal and joint ownership and is hidden while the browser is using manual fallback. It does not recalculate totals. Incomplete local attempts are labeled **Connection incomplete**. **Request update** is shown only in states the server can refresh.

An independent review found no unresolved issues. The root quality gate passed with 92 Node tests, 42 banking tests, Go race tests with PostgreSQL 18, contract and provider generation checks, formatting, source limits, lint, types, unused checks, and both builds. It accepted 292 authored source files. A final label shortening was followed by successful formatting, web lint, type checks, and both builds. The final browser bundle is `index-DdpPvZQz.js`. The existing nonfatal bundle-size warning remains; its threshold was not changed.

Browser checks used the compiled app and real API. Safari verified the existing Sandbox import, account exclusion, classification, saving, current versus available totals, and restoration of the original mapping. A separate PostgreSQL review schema with clearly labeled simulated accounts and placeholder credentials exercised missing-balance warnings and provider startup failure without contacting Plaid. The chosen account type survived that failure. Cancel restored focus, Enter opened the chooser, and Escape restored the account-manager trigger. Chrome checks at 320×568 and 390×844 found no page or dialog overflow; the launch and cancel controls were at least 44 pixels high. Safari responsive views at those sizes verified the summary-warning layout visually. These were desktop browser checks, not a physical iPhone test.

The isolated review app, its temporary browser tabs, its schema, and the diagnostic executable and baseline were removed after verification. Safari was returned to normal browsing with the user's real local Sandbox account manager open at `http://localhost:5188`. Screenshots and check logs remain under `/tmp/bank-piggy-plaid-review`, including `single-connect-safari-ready.png`, `sandbox-current-balance-verified.png`, and the labeled simulation/mobile views. Changes remain uncommitted and local. Real-bank coverage, production Trial capacity, deployed OAuth/webhooks, and backup restore remain unverified.

## Accounts dashboard follow-up

The new Accounts navigation view lists every returned provider account, including excluded, inactive, credit, loan, and non-USD accounts. It separates current and available values. Credit and loan current balances are labeled **Amount owed**. Missing values remain **Unavailable**. The view does not sum unlike currencies, infer retirement tax treatment from account names, or change saved classifications. Supported balances replace explicitly matched manual snapshots once; planning amounts remain separate.

Accounts reloads saved status on entry, foreground return, and every 30 seconds while visible. These GET requests do not request new provider data. Provider checks retain the existing 24-hour schedule. An independent review identified overlapping-read and stale-result risks introduced by polling. Shared read ownership now coalesces requests, queues a fresh read after older work, and preserves mutation failures during reconciliation. Controlled-interleaving tests cover these cases, plan revisions, hidden pages, and cleanup. A second independent audit and the root source review found no unresolved issues.

The full root quality gate passed: 112 Node tests, all 42 banking tests, all Go race packages with real PostgreSQL 18, contract and provider generation checks, formatting, source limits, lint, types, unused checks, and both builds. It accepted 304 authored source files. The final frontend bundle is `index-BPoxlMqi.js`; the existing nonfatal bundle-size warning remains. The log is `/tmp/bank-piggy-plaid-review/accounts-check.log`.

Native Safari verified all 14 accounts from Michael's existing official Sandbox connection, credit and loan labels, missing balances, details, and a successful status reload that retained expanded details. The Accounts **Connect accounts** action opens directly at the Bank accounts / Investments choice. Opening and closing that choice left the existing Link-attempt count unchanged at three. No new Link request, Item, provider product, or account mapping was created during this follow-up.

A disposable review schema and loopback server used placeholder credentials, blocked provider traffic, and four clearly labeled simulated accounts. Browser checks covered the empty view, a long account name, credit balances, EUR values, unavailable investment balances, and expanded details retained during reload. A stopped-server check kept all four saved rows and displayed a readable error; restarting the isolated server and reloading cleared that error. Enter opened the account choice, Escape restored the Connect accounts trigger, and mobile navigation worked by keyboard. Chrome checks at 320×568, 390×844, and 844×390 found no horizontal page or account-body overflow. Action controls were at least 44 pixels high, and short screens retained a scrolling account body. These were desktop browser checks, not physical-phone checks.

The isolated server, review schema, simulated records, and test browser tab were removed. The temporary viewport override was reset. Safari remains open on Accounts at `http://localhost:5188`. Screenshots include `accounts-safari-ready.png`, `accounts-simulated-320.jpg`, `accounts-simulated-390.jpg`, `accounts-simulated-landscape.jpg`, and `accounts-simulated-error-390.jpg` under `/tmp/bank-piggy-plaid-review`.

Michael confirmed that future monthly investing means net contributions: deposits minus withdrawals, separate from trades and market returns. Balances alone do not establish monthly spending or contributions. Transaction-history products and metrics were not enabled in this change. Changes remain uncommitted and local; live institution coverage and production readiness remain unverified.

## Staged Railway configuration

Michael reported production access and requested browser-only credential setup with placeholders. Safari opened the existing Bank Piggy production application service in Railway. Eight new variables were staged: placeholder `PLAID_CLIENT_ID` and `PLAID_SECRET`, `PLAID_ENV=production`, `PLAID_ENABLED=false`, `PLAID_PRODUCTION_APPROVED=false`, `PLAID_LIFETIME_ITEM_LIMIT=1`, `PLAID_ACTIVE_KEY_ID=production-v1`, and a newly generated 32-byte encryption key in `PLAID_TOKEN_KEYS`. The encryption value was submitted directly through the variable form without displaying it in tool output, screenshots, or this document. Existing variables were preserved.

Railway showed 18 service variables and **Apply 8 changes**. Every value was masked in the final screenshot, `/tmp/bank-piggy-plaid-review/railway-plaid-placeholders.png`. No Deploy action was taken. The running release remains unchanged. Michael will replace the two credential placeholders himself; the production secret has not been inspected. Actual plan terms and remaining team allowance, backup restoration, the code release, and deployed provider checks remain pending before enabling production Plaid.

## Production release preparation

Michael saved the production credentials in Railway and authorized enabling and deploying the integration. The actual Plaid Dashboard confirmed **Free trial**, **0/20 connections used**, and automatic bank access. This team-specific allowance differs from the earlier published ten-Item description. Bank Piggy retains its conservative initial reservation limit of one; this observation does not authorize a paid upgrade or repeated test Items.

The final Linux AMD64 production container built successfully with the same `index-BPoxlMqi.js` asset. Its image manifest-list digest is `sha256:42de6b7ebf0f86a5cabef6a918f8344e4fab533c78ca7ae3856520afa51fcedb`. Both dependency audits passed with no reported vulnerabilities. A fresh independent read-only security review found no unresolved code release blocker. Its attempted focused test run lacked Go in that agent's PATH; the earlier full root gate and subsequent release CI are the executable test evidence.

A custom PostgreSQL backup was created through the existing Railway private database console. After Michael explicitly approved the payload and local destination, the backup was downloaded to this Mac and protected with owner-only permissions. Its SHA-256 is `19dd5514a64cce889aa0c324d89b30d0efed40a350cca95329ae44668fa6048b`. A restore into a new, isolated local PostgreSQL 18 database succeeded. Running the application's actual store migration added all five banking tables. Hashes of every row in the six pre-existing application tables matched before and after migration. The temporary migration-check executable source was removed. No financial values, session credentials, or backup contents were logged or added to Git.

This is a manual release backup and verified restore. It does not configure scheduled backups or off-device recovery. Railway still requires Pro for new native backups; no plan change was made. Runtime enabling, the deployed code, and live provider acceptance remain separate steps.

## Production activation

PR #16 was merged as `991e611918f8b71942af3e3c80d0dae91a2f5429`. Both pull-request checks and the merged main-branch quality workflow passed, including the production Docker build and dependency audits. Railway's configuration deployment `a124019b-3a46-4834-9a19-a569fe88bf72` became active with the expected `index-BPoxlMqi.js` asset.

The two credential placeholders were replaced by Michael. Browser checks validated only presence and format without displaying the credential values. The encryption key decoded to 32 bytes. `PLAID_ENABLED=true` and `PLAID_PRODUCTION_APPROVED=true` were verified; production mode and the initial one-connection reservation budget were retained. Existing Google and database settings were preserved. No credential, backup, or mascot output file was included in the release commit.

The exact HTTPS redirect `https://bank-piggy-production.up.railway.app/plaid-return` was saved in Plaid after the existing Google identity was verified. The localhost Sandbox redirect was retained. Plaid's Account Verification page had no Auth rule override configured. No verification methods, paid products, or new account permissions were added.

On the deployed origin, `/healthz` returned 200, unauthenticated household and connection reads returned 401, the demo-session endpoint returned 404, and a malformed webhook signature returned 401. Safari resumed the existing authorized household session and opened Accounts. The empty production view and direct bank/investment choice worked without a Sandbox notice. The agent did not select Continue to Plaid, reserve a production Link attempt, create an Item, or complete bank consent. Actual institution balances, signed webhook delivery, OAuth bank return, and reconnect/revocation still need the member's first live connection.

Michael then requested Accounts as the last navigation tab. The shared desktop/mobile order was changed to place it below Trips & purchases. A Sol 6.1 medium agent made the one-line move; the root reviewed the exact diff. The full root gate passed again. Local browser checks verified the desktop order, the 390-pixel mobile drawer, and keyboard activation without horizontal overflow. The temporary browser tab and viewport override were removed.

PR #17 merged as `d485334054d95fe1fd92a6779d4c08441c8009f6`. Both PR checks and the main-branch quality workflow passed. Railway deployment `5f7d147a-4bd8-4a23-bfc5-21999d990562` became active with `index-4Fgy-DXQ.js`. The public asset matched the locally built asset. Safari reloaded the production origin, confirmed Accounts after Trips & purchases, and opened the enabled bank/investment choice without a Sandbox notice. The dialog was closed without creating a Link attempt. The production view showed zero linked accounts and was left open in Safari. The final proof screenshot is `/tmp/bank-piggy-plaid-review/accounts-bottom-production-safari.jpg`; the local mobile screenshot is `accounts-bottom-mobile-local.jpg` in the same directory.

Post-release checks retained healthy PostgreSQL, private API access, an unavailable demo endpoint, and rejection of an invalid webhook signature. No real-bank consent was completed by the agent. The remaining bank-specific acceptance starts with the member's first institution. These final acceptance notes and guide corrections are local documentation changes after the source releases; the deployed application code is the merged PR #17 state. Unrelated mascot output remains untouched.

## First real institution and cadence follow-up

A member completed the first real depository connection. A read-only production check confirmed completed consent, a saved connection, imported accounts with present current balances, and no synchronization failure. The deployment logs showed no application crash. Institution identity, account counts, financial values, credentials, and raw provider responses are excluded from this repository evidence. This establishes the first real consent, token exchange, and balance import; it does not establish other institutions or all account types.

The initial one-institution application reservation limit was consumed. A second new institution is rejected before Plaid Link starts, with no second attempt or Item saved by Bank Piggy. Actual remaining free Trial capacity was checked before permitting the next distinct institution with a limit of two. The existing reservation is not refunded or hidden. Brokerage accounts require the Investments choice; their actual consent and import remain unverified.

Michael requested a 30-minute cadence. The former 30-second Accounts status polling is now 30 minutes. Bank reads change from daily to every 30 minutes, with the same minimum interval after failed or interrupted reads. Manual refresh and ordinary balance webhooks cannot bypass the saved deadline. Consent repair and first import remain immediate. Revocation cleanup keeps its prior faster retry behavior, and Capital One's supported cache-age bound stays 24 hours independently of polling. A second independent read-only audit found no unresolved code issue. Existing healthy production rows need a guarded one-time change from their exact old 24-hour deadline after the new worker is deployed; active leases, failure retries, and newer schedules must be preserved.

## Thirty-minute production deployment

PR #18 merged as `f95f854c8b3556272cb7b1b46a44886a8bac0f6f` after both PR checks passed. The main quality workflow also passed. The root gate covered real PostgreSQL race tests, contract/provider generation, formatting, source limits, lint, types, unused checks, Node tests, and both builds. Railway's automatic deployment `fdfee20a-8448-4fe1-9f7f-d019dec39bec` was replaced by configuration deployment `7e452bb0-a700-4626-9310-8b26acc7c78f`, which is active. The expected `index-Vf9TCvxQ.js` asset matches the local build byte for byte. The two-institution reservation limit was verified in the service variables. No new Link session or Item was created by the agent.

The deployed health endpoint returns 200. Unauthenticated household and connection reads return 401, demo-session creation returns 404, and an unsigned webhook returns 401. Provider logs confirm successful consent handoff, credential exchange, and the first balance read. They do not contain the earlier failed institution-selection event, so the cause of that failure is unconfirmed. The saved connection used the bank product flow; investment consent remains a separate unverified path.

The one-time production schedule adjustment is **pending**. Railway's private console lost its WebSocket connection; reconnecting and a separate console view failed. The database viewer also remained at its connection step. No schedule mutation was submitted. Existing exact daily deadlines therefore remain until the next successful scheduled read, after which the new worker uses 30 minutes. A later operator can advance only healthy connected production rows with saved access, no active lease, no failures, and an exact `next_sync_at=last_synced_at+interval '24 hours'` deadline. Set that deadline to `GREATEST(now(),last_synced_at+interval '30 minutes')`; do not alter retries, active leases, or newer deadlines. The app remains healthy; this console failure does not establish an application database outage.

## Simplified Accounts and shared balance projection

Michael requested one primary balance per account without the account-details disclosure, available balance, last/next update times, or linked-account count. Accounts and its manager now use the same primary balance formatter. Missing amounts say not provided; zero remains zero; credit and loan balances remain amounts owed. Connection problems and recovery actions remain visible.

Untouched supported USD accounts now receive read-time defaults under the institution's connecting member. The canonical returned mapping and effective totals resolve the same defaults. Provider type/subtype can establish checking, savings, brokerage, or HSA; other investments remain neutral Other investment. Unknown bank subtypes, unsupported currencies, credit, and loans stay out of asset totals. Saved mapping revisions, including exclusions and manual replacements, are preserved. No read writes defaults, snapshots, or plan values to the database. Manual matching stays explicit; names and masks do not establish identity.

One shared connection provider now wraps workspace views and plan dialogs. Its single 30-minute status lifecycle keeps Overview and Accounts on one projection. Overview withholds totals when the plan revision does not match, with a reload notice. A matched manual editor row displays the projected Plaid amount read-only and retains the original manual draft as fallback. Manual debt is labeled separately because an aggregate cannot identify which linked debts it contains. Budgets, income, allocations, and goal progress remain plan inputs.

The full root quality gate passed with real PostgreSQL race tests, contract/provider generation, formatting, source limits, lint, types, unused checks, Node tests, and both builds. The bundle is `index-DTfTdaQR.js`; the existing nonfatal size warning remains. An independent read-only audit found no unresolved issue.

Browser checks used an isolated PostgreSQL schema and loopback app with placeholder credentials and no provider access. Clearly labeled simulations covered known asset types, missing and zero balances, foreign currency, credit debt, a long account name, and an explicit manual match. Overview's household/member figures agreed with the server projection. Saving the household from the browser changed only its revision; every manual input stayed identical. A server failure retained saved account rows and showed a retryable error; restarting and reloading cleared the error. Empty Accounts guidance passed. A separately advanced plan revision hid stale totals and showed a reload notice. At 320 and 390 pixels, account rows had no horizontal overflow and actions remained 44 pixels high. Enter opened the account chooser; Escape closed it and restored focus. These are desktop browser checks, not a physical-phone test. The simulated fixture and browser images are private temporary review artifacts, not running production data.

The schedule transition is now an embedded startup data upgrade after the banking tables exist. It advances only future, healthy, unleased deadlines equal to the prior success time plus exactly 24 hours. Already-due work stays due; failures, leases, other states, and newer schedules stay unchanged. This resolves the previously pending console-only rollout step when the new release starts. A real PostgreSQL test checks 19 cases, preserves all other connection metadata and saved financial records, and requires exact equality after two further startup migrations. All Go race packages, API lint, API build, and source limits passed. The independent audit found no issue in the guarded statement. The tests cover lease guards and repeated execution; they do not reproduce a concurrent waiting worker. Neutral automatic retirement classifications retain readable provider IRA/401(k) labels without asserting Roth or Traditional status. The UI describes 30-minute checks, not a guarantee of new institution data.

The final full root gate passed again after the schedule upgrade and label changes. Its frontend asset is `index-D5OzwOHu.js`.

PR #19 merged as `bc97ea6cf4faeb3f277e37f1bd6b42b030751c61`. Both pull-request checks passed for source `dbe63444bb17b8123491ec44d3086c443802b056`; main workflow `43375703930` also passed, including audits and the Linux production image. Railway deployment `532ec1e6-c8bf-4fc5-803b-ca82b4cbb14a` is active, and the previous deployment is removed. The deployed `index-D5OzwOHu.js` matches the local asset byte for byte.

Signed-in Safari verified the simplified production Accounts screen, retained connection states, primary balances, and 30-minute check copy. Overview displayed connected sources without a pending projection or plan-revision warning. A separate signed-in, read-only API view confirmed that the legacy connection's last/next synchronization interval changed from 24 hours to 30 minutes and remained connected without a provider-unavailable error. This completes the earlier pending schedule transition. No live values, account identities, or institution names from this acceptance are recorded here.

Live health remains 200; private household and connection reads require authentication (401), the demo endpoint remains unavailable (404), and an unsigned webhook is rejected (401). The agent created no Link session or Item and changed no production manual mapping. The isolated review server and schema were removed, the browser viewport was reset, and Safari was left on production Accounts. Its proof image is private under `/tmp/bank-piggy-plaid-review/accounts-cleanup-production-safari.png` with owner-only permissions. These acceptance notes remain local after the code release. Unrelated mascot output remains untouched.
