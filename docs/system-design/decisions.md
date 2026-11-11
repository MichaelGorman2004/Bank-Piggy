# Design decisions

The first version optimizes for a correct, reviewable plan for two people. These decisions describe the code that exists, the benefit it provides, and the cost it accepts.

## 1. One Go application and a React frontend

**Decision.** Keep HTTP routing, Google sign-in, financial rules, and PostgreSQL access in one Go application. Build the React and TypeScript frontend with Vite. Let Go serve the built frontend from the same origin.

**Reason.** The main operation is a short request that reads or replaces one household plan. A single backend keeps permission and transaction boundaries clear. Same-origin requests fit cookie sessions and exact-origin mutation checks.

**Tradeoff.** Backend responsibilities still need feature boundaries. Go packages separate budget rules, persistence, and server orchestration. Development uses two processes and a Vite proxy. Runtime deployment does not require two services.

**Revisit when.** A measured workload needs separate execution, such as long-running provider synchronization. That workload does not exist yet.

## 2. A canonical contract with generated types and runtime validation

**Decision.** Generate Go types, TypeScript types, and the browser JSON Schema from [one OpenAPI document](../../packages/contracts/openapi.json). Reject generated drift in the quality gate.

**Reason.** A household plan crosses language and network boundaries. A shared field name or type should not require separate manual edits in each language. Ajv validates the actual JSON response rather than relying on a TypeScript assertion.

**Tradeoff.** The local generator supports only the schema features the application currently uses. Domain rules remain in Go because a shape alone cannot prove owner membership, duplicate IDs, salary availability, or revision safety. Generator changes require review when the schema gains new features.

## 3. Integer cents with one reconciled allocation

**Decision.** Store money as integer cents. Calculate the first proportional share once, then assign the remainder to the second member. Use arbitrary-precision integers for the intermediate ratio multiplication.

**Reason.** Every personal view must reconcile with the household view. Independently rounded shares can create or lose cents. Binary floating-point money can make equality unreliable. The maximum accepted inputs can also overflow an `int64` multiplication even though the final share is safe.

**Tradeoff.** The deterministic tie rule assigns the extra cent to member 1 when a split lands exactly on a half cent. The rule is simple, documented, and tested. Currency is US dollars; multiple currencies and exchange rates are outside this contract.

## 4. One JSONB plan with optimistic revision checks

**Decision.** Store the current plan as one JSONB document. Protect whole-plan replacement with a row lock and an expected revision. Keep identities, sessions, and sign-in attempts in separate relational tables.

**Reason.** The UI edits a small aggregate and needs all of its totals to agree. One commit saves income, expenses, goals, and revision together. A two-person household does not require independent records or event streams for each draft field.

**Tradeoff.** Different edits to the same revision conflict even if they touch different expenses. There is no automatic merge, per-field history, or audit log. Queries across historical transactions would need a different model because this plan is not a ledger.

**Revisit when.** Actual activity, imports, or history become implemented requirements. Add those records deliberately instead of treating a revised budget as a past transaction.

## 5. Server-owned identity and invite-only access

**Decision.** Use the established Go OpenID Connect library to verify Google identities. Bind two configured invitations to stable Google subjects. Issue opaque, server-stored sessions and authorize every household request.

**Reason.** Google proves identity; Bank Piggy controls household access and session revocation. The browser cannot grant itself a household by changing a request field. Server-side sessions make logout and invitation changes enforceable on the next request.

**Tradeoff.** Startup in private mode depends on Google provider discovery. Each protected request depends on PostgreSQL. The current model serves one fixed household, and both members may edit all plan fields. A personal view provides allocated figures, not private storage from the other household member.

## 6. Explicit local demo and honest estimates

**Decision.** Require an explicit demo mode bound to loopback. Keep its local household separate from the private household. Start both modes with empty plans and keep populated fixtures in test code. Label planned contributions, scheduled forecasts, and paycheck estimates. Show target progress from the user's manual saved amount.

**Reason.** The local application can exercise finance and persistence flows with user-entered data. Automated tests use populated fixtures without inserting example finances into the running application. Clear labels explain where the figures come from. A labeled Sankey shortfall makes a funding gap visible.

**Tradeoff.** Demo users share one saved plan in the local database. Existing records remain intact during startup; the app cannot safely infer whether an older local record is still an example or has been edited. The demo is not a public hosted sandbox. Salary edits recalculate the current plan, and year/paycheck estimates contain no historical activity or real pay dates.

## 7. Strict checks with a real persistence boundary

**Decision.** Use one root quality command, a local pre-commit hook, and a checked-in CI workflow. Include Go and frontend formatting, contract and provider-asset drift, source size, strict lint, types, unused code, Node regressions, Go race tests, and builds. Run persistence tests against PostgreSQL.

**Reason.** Mock SQL cannot prove locking, uniqueness, expiry queries, or transaction behavior. The source-size and function-complexity checks make responsibility growth visible. Generated-file checks keep consumers aligned with the contract.

**TypeScript 7.** Use its native compiler for builds and type checks, plus Oxlint and `tsgolint` for strict typed lint. Keep Babel-backed ESLint for the existing syntax, documentation, React Hooks, and source-structure rules. Knip v6 uses OXC for unused-code analysis. These tools do not need a TypeScript 6 compiler. Pin `eslint-plugin-jsdoc` to `64.2.1`; later releases introduce the older TypeScript API dependency. Explicit unused-variable options retain checks for underscore parameters and runtime values used only by types. Negative lint-policy tests verify that the migration retains the required rejections. See [native typed lint](https://oxc.rs/docs/guide/usage/linter/type-aware.html), [Knip v6](https://knip.dev/blog/knip-v6), and the [JSDoc package manifest](https://github.com/gajus/eslint-plugin-jsdoc/blob/v64.2.1/package.json).

**Tradeoff.** The full gate needs Node, Go, Python, `golangci-lint`, and a test database. Database tests explicitly skip without `TEST_DATABASE_URL`; reviewers must distinguish that run from real integration evidence. A hook can be bypassed, so its presence does not prove that a hosted workflow passed.

## 8. Calculate schedules from one current plan

**Decision.** Add optional dates and payment metadata to expenses. Store a finite term as its inclusive final month. Calculate monthly and 12-month forecasts from the selected planning month. Let a goal optionally record ordinary saved progress or one dated trip/purchase target while retaining its chosen contribution.

**Reason.** Subscriptions, bills, and loan payments already belong in living expenses. A dated expense can stop contributing without losing its editable record. A rolling forecast preserves repeated monthly cent allocations. Shared target math handles trip and purchase deadlines. Ordinary progress needs no deadline projection. A recorded saved amount supports a real progress bar; a monthly contribution alone does not establish progress.

**Tradeoff.** This is a forecast from the current plan. It has no actual payment log, automatic cancellation, interest schedule, or historical salary snapshots. Each month uses the current income and chosen goal contributions. Target suggestions use equal monthly saving periods and a manual saved amount. Separate projection arrays preserve the existing trip response and identify purchases explicitly.

## 9. Serve provider logos as local assets

**Decision.** Keep 100 providers in a frontend catalog. Generate their SVG files from Simple Icons and checked-in archival assets. Offer a native searchable input and resolve IDs through the catalog before building an image path.

**Reason.** A familiar logo helps identify a recurring service. Local images keep payment names out of third-party image requests and make the display independent of a remote logo service.

**Tradeoff.** The catalog needs deliberate updates and attribution. A selected logo represents a manual label. It grants no provider access and does not verify or cancel a subscription. Unknown providers can use the expense name and a generic payment icon.

## 10. Keep one task in the active viewport

**Decision.** Open Overview by default and keep Money flow, Expenses, Goals, and Trips & purchases focused on their tasks. Combine payment types in one filtered list. Keep scope, period, and planning month shared. Fit the frame to `100dvh` and scroll bounded card bodies. At a glance uses compact income and balance tables; narrow or short Overview layouts select one section through native buttons.

**Reason.** Overview gives immediate access to the plan totals and inputs. The separate Money flow chart can use the available space. One expense list keeps subscriptions and other payments in the same context, while sorting surfaces the largest current allocation. Fixed card controls remain reachable during scrolling. Grouping small chart categories preserves all cents while keeping labels readable; detail remains in Expenses.

**Tradeoff.** A narrow chart can combine several categories into one labeled total. Users open the relevant payment view for detail. Short screens need compact controls and a usable body height. Browser review must check body scrolling and stationary outer regions in isolated emulated profiles and in the actual handoff window.

## 11. Store typed snapshots with a safe legacy upgrade

**Decision.** Make named, member-owned account records the canonical balance input. Derive savings and investment totals. Upgrade old aggregate totals inside the same startup transaction as schema creation.

**Reason.** One account list supports separate retirement and savings types without duplicated totals drifting apart. Legacy amounts lack account-type evidence, so migration uses generic Savings and Other investments. Users supply more specific types when known.

**Tradeoff.** Balances remain manual snapshots with no provider reconciliation or joint-account model. Startup can change an affected plan's revision. Row locks, matching JSON/row revisions, idempotent predicates, and transaction rollback preserve coherent saved state. Group bounds limit totals as well as individual accounts.

See [user stories](../user-stories.md) for observable behavior, [data rules](data-model.md) for numerical examples, and [security](security.md) for trust boundaries and rejection paths.
