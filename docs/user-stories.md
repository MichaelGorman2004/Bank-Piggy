# User stories

These stories describe the implemented first version. Acceptance criteria state observable behavior. They do not claim a live deployment or a completed provider integration. The [product brief](product-brief.md) remains the source for longer-term scope.

## 1. Open the local household plan

**As a local evaluator, I want to open an empty household plan so I can try the application with my own inputs.**

- Explicit demo mode starts only with a loopback listener and browser origin.
- The entry action is **Open household plan** and creates a session for the first local member.
- The interface shows **Our household**, the saved member names, **Household member**, and **Household plan**.
- Edits persist in PostgreSQL and remain after reload or sign-out.
- A new local household starts with one person and no financial records. All demo sessions use that same saved local household. Private mode uses a different household and rejects demo-session creation.

**Evidence boundary:** local mode does not establish Google identity or bank access. Automated tests use isolated populated fixtures. Startup preserves existing plans in both modes.

## 2. Sign in as an invited household member

**As an invited member, I want my own Google sign-in so the server can identify me and enforce household access.**

- Private mode requires one or two distinct bootstrap emails and complete Google configuration. A partner can also accept a household link.
- The callback requires a valid one-use sign-in attempt and a verified Google identity.
- A verified invited email binds to a stable Google subject. A different subject cannot claim that invitation later.
- An account without a bootstrap invitation or accepted link cannot obtain a household session.
- Expired sessions and bootstrap sessions inconsistent with their configured invitation cannot read or edit the plan. Accepted-link identities use their persisted membership.
- Invited members with a saved profile can read and edit the complete household plan.
- A second bootstrap account receives a clear rejection until its profile is saved. This rejection does not bind an identity or create a session. A valid link can append a missing second profile during acceptance.
- Adding a financial profile offers a link after saving. Sessions must still match a current saved member and a pinned, authorized identity.

**Evidence boundary:** signed local identity-provider tests and real PostgreSQL tests cover this flow. One invited Google account has passed live sign-in, sign-out, and repeat sign-in. The second invited login and live rejection of an uninvited account remain unverified.

## 3. Create the first useful monthly plan

**As a member, I want a short setup flow so I can see what our income can support.**

- A household with zero income and no expenses or goals opens plan setup.
- Three steps collect income, expenses, and goals with balances.
- New households show one profile. **Add person** below it adds the optional second profile and moves keyboard focus to its name field.
- Adding a person preserves all entered profile, account, expense, and goal fields. The button disappears at two people.
- Saving and reopening a one-person plan preserves one person. Existing two-person plans retain both profiles, even when the second person has no income.
- New expenses and goals in a one-person plan belong to that person. Adding a partner does not change their ownership.
- Split controls appear only when there are two people. A one-person plan allocates the full amount to its only member.
- Each member can enter monthly take-home pay, pay frequency, annual base salary, typed account balances, and debt.
- Salary is used for salary-based splits; no tax estimate is deducted from entered take-home pay.
- The saved result shows take-home pay minus living expenses, then subtracts planned goal contributions.
- Missing income or an empty expense list shows a visible completeness notice.
- Invalid values return to the relevant step, show an error, and preserve the entered draft.

## 4. Read the household money flow

**As a member, I want one view of the household plan so I can see where our combined income goes.**

- The default scope is the combined household.
- Money flow contains the Sankey. The default Overview shows Plan totals, At a glance, and Balances. At a glance shows the saved members in compact income and balance tables.
- Expenses are grouped by category in the flow. Goals and money left to plan are separate destinations.
- The chart measures available height and keeps the largest categories visible. Smaller categories combine into `Other costs (N)` without losing cents. Expenses retains all detailed records.
- A shared expense appears once in the household total.
- A plan that exceeds income shows the exact funding gap as a labeled shortfall.
- An empty plan shows a useful prompt. Accessible text communicates the chart totals outside the visual chart.

## 5. Understand my allocated share

**As a signed-in member, I want to switch to my share so I can understand my part of the shared plan.**

- The scope selector offers the household and the signed-in member's view.
- Cards, expense rows, goals, balances, and money flow follow the selected scope.
- Personal expenses belong entirely to their owner.
- Shared expenses use the server's reconciled shares. Shared goals divide evenly.
- The saved members' allocated totals add to the household totals. A one-person plan assigns the full total to that person.
- Changing the visible scope does not change permissions or save a new plan.

## 6. Agree on a shared expense

**As a member, I want a clear split rule and a separate payer so our plan reflects our agreement.**

- A shared expense supports equal, salary-based, and custom percentage splits.
- Custom percentages include 0% and 100% and support two decimal places.
- A $1.01 equal split produces $0.51 and $0.50. The shares always equal the full expense.
- Salaries of $100,000 and $50,000 split a $3,000 bill into $2,000 and $1,000.
- Both salaries zero prevent a shared salary split. A zero salary for one member is valid when the combined salary is positive.
- Recording another member as payer does not change the agreed allocation.
- A household default initializes new expenses. Existing expenses retain their saved method and custom percentage.
- Editing salary recalculates salary-based shares for the current plan. Historical split snapshots are outside this version.

## 7. Adjust expenses without losing work

**As a member, I want to add, edit, or remove an expense so the plan can change as our estimates improve.**

- The editor collects name, category, monthly amount, owner, split rule, and optional payer.
- Money input accepts nonnegative decimal text with up to two decimal places; an expense must be greater than zero in the editor.
- Invalid input stays visible with a useful error.
- A save disables editing and closing controls until the request finishes.
- A successful save updates the dashboard. A rejected or disconnected save keeps the draft open.
- Deleting an expense saves a new complete household revision and updates its derived totals.

## 8. Keep several goals in one budget

**As a member, I want to plan several contributions so savings, investments, and extra debt payments can share the same available money.**

- Goals can be added, edited, removed, and assigned to a member or the household.
- Monthly contributions subtract from the amount available after living expenses.
- The interface shows whether all goal contributions fit within income.
- Goals contains ordinary contributions, with optional manually recorded target progress. Trips & purchases contains dated target goals.
- An untargeted row shows its planned contribution. A targeted row shows saved amount of target, a bounded progress bar, and the contribution separately.
- Entered savings, investment, and debt balances remain separate from contributions.
- A planned contribution does not claim a completed transfer or progress against a historical target.

## 9. Compare plan periods

**As a member, I want a selected planning month and period estimates so I can see when scheduled costs affect our plan.**

- The planning month defaults to the current UTC month and accepts `0001-01` through `9998-12`.
- A valid change applies automatically. The accepted plan stays visible and the month field stays editable while the read is pending.
- A failed latest read restores the accepted month without overwriting a newer selection.
- Later month selections win over earlier responses, including a return to the currently accepted month.
- The themed month picker supports year entry, month buttons, and keyboard navigation. Out-of-range months are disabled. Required fields have no clear action; optional fields offer Clear month.
- The accepted month stays in the URL and is retained across saves and reloads.
- Changing the month reads a forecast without changing the plan revision.
- Yearly estimates add the selected month and next 11 months, including each payment's start and final month.
- Yearly member shares add the repeated rounded monthly shares. Ended payments stop contributing, and balances stay unchanged.
- A member with regular pay frequency can view an average paycheck estimate.
- Weekly, biweekly, semimonthly, and monthly schedules use 52, 26, 24, and 12 paychecks per year.
- Household and irregular-pay paycheck options are disabled.
- Returning to household scope resets an active paycheck estimate to monthly.
- Balances remain snapshots. The views do not invent pay dates or actual annual activity.

## 10. Resolve a competing save safely

**As either member, I want stale saves to be rejected so I cannot silently erase a newer household plan.**

- Every write includes the revision from which the user started editing.
- Two requests for one revision produce one accepted save and one conflict.
- The accepted save advances the revision once.
- A conflict keeps the draft and explains that the latest plan must be reloaded.
- Further writes remain blocked until an explicit reload succeeds.
- A failed reload keeps the conflict visible. An older response cannot replace newer loaded state or another session's state.

## 11. Use the app on a small screen or with a keyboard

**As a member, I want readable controls and predictable focus so I can manage the plan on my device.**

- The application frame fits the visible viewport and provides mobile navigation.
- Long lists scroll inside their card bodies. The document, main content area, and active-view container stay still.
- Card headers, actions, filters, and footers remain visible while the body scrolls. Each body has a name and can receive keyboard focus.
- Short layouts use compact controls and retain a scrollable card body.
- Native select controls expose scope and period with accessible names.
- Dialogs keep keyboard focus inside the editor and restore focus when closed.
- Dialog headings and close actions stay visible while their inset form bodies scroll. Field focus rings remain visible inside that area.
- The mobile drawer contains Tab navigation, supports Escape, and restores its trigger.
- Interactive controls show keyboard focus. Chart totals are available as text.

## 12. End my session

**As a member, I want sign-out to revoke access so this browser cannot continue using the current session.**

- Sign-out uses the authenticated mutation checks.
- The server deletes the current session before clearing the cookie.
- A failed revocation shows an error and does not claim success.
- After successful sign-out, the interface returns to the unauthenticated state.

## 13. Schedule recurring payments

**As a member, I want to record when subscriptions, bills, and loan payments start and end so future plan totals include the right costs.**

- An expense can be a subscription, bill, or loan payment, with an optional provider.
- The provider field searches 100 local catalog entries and displays local logos. A custom payment can use its own expense name and a generic icon.
- An unmatched provider query must be corrected or cleared before submission. Other providers can be named in the expense name.
- A payment can continue indefinitely, end after a selected month, or run for 1–1,200 monthly payments.
- A finite term requires a first month. A 36-payment term starting November 2026 stores October 2029 as the inclusive final month.
- Invalid months, reversed dates, and invalid terms show an error and preserve the draft.
- An undated existing expense remains active.
- Upcoming and ended payments stay editable in the management view and have zero shares outside their active months.
- Expenses combines all payment types. All types, Subscriptions, Bills & costs, and Loans filters keep ended, upcoming, and zero-share records reachable.
- Rows sort by the selected scope and period's allocated cents, largest first. Ties preserve saved order, and filtering or sorting does not mutate the plan.
- The card and empty-state add actions use the selected type; All types defaults to a bill. The header's Add expense always starts as a bill.
- Schedule edits save through the same revision and session checks as other plan edits.
- Scheduling a final payment changes forecast totals. External service cancellation and actual loan payments remain the user's responsibility.

## 14. Build a trip budget

**As a member, I want an itemized trip target and a saving suggestion so I can choose a monthly contribution that fits our plan.**

- Trip goals appear in Trips & purchases and are excluded from ordinary Goals.
- A trip goal records flights, lodging, food, local transport, activities, and other costs.
- The target is their sum. The user enters the amount already saved and target month.
- The remaining amount is clamped at zero. The suggestion divides it across months including the planning month and target month, rounding up to a whole cent.
- A $100 remaining target from November through January suggests $33.34 per month across three periods.
- An unfunded past deadline shows overdue with no future monthly suggestion. A funded target remains funded after the deadline.
- The chosen contribution can be zero. It remains under the user's control and is the only trip amount deducted from monthly cash flow.
- Shared trip contributions split evenly. The whole trip target and saved amount stay visible as shared context in the personal view.
- Saved trip progress does not create income or change account balances.

## 15. Save for a purchase

**As a member, I want a purchase target and deadline so I can plan a contribution without changing my recorded account balances.**

- Trips & purchases can create a purchase with a target amount, amount already saved, and target month.
- A goal can have trip or purchase details. A request containing both is rejected.
- Purchase targets use the same inclusive periods, ceiling-to-cent suggestion, and funded or overdue states as trips.
- A $2,000.01 target with $500 saved and a February 2027 deadline has four periods from November 2026. Its $1,500.01 remainder suggests $375.01 per month.
- The user chooses the actual monthly contribution, including zero. A $125.50 allocation adds $125.50 to the budget and leaves savings balances unchanged.
- A failed save retains the draft. Save, reload, and reopen preserve the target fields.

## 16. Keep each view focused

**As a member, I want one clear task per view so I can find the right records and use the available screen space.**

- Overview is the default and shows Plan totals, At a glance, and Balances. At a glance fits the saved members' core facts in two compact tables. Narrow or short layouts use three section buttons to show one section at a time.
- Money flow shows only the Sankey below the shared controls and notices.
- Expenses shows bills, subscriptions, regular costs, and loans in one filtered list.
- Goals shows ordinary goals. Trips & purchases shows trips and purchases, including funded, overdue, and zero-contribution targets.
- Scope, period, and planning month stay consistent when switching views.
- Switching views filters presentation without deleting, rewriting, or changing the ownership of saved records.

## 17. Track typed account balances

**As a member, I want named account balances by type so I can understand our savings and investments without changing our monthly plan.**

- Plan setup can add, edit, and remove up to 20 accounts per member. Each account has a stable ID, name, type, and whole-cent balance.
- Types include Traditional and Roth 401(k), Traditional and Roth IRA, Brokerage, High-yield savings, Savings, and Other investment.
- Savings and high-yield savings contribute to savings; the other types contribute to investments. Personal and household totals count each record once and remain unchanged across period estimates.
- IDs must be unique within the member. Invalid accounts and excessive account or group balances reject the save and retain the draft.
- Account balances stay separate from income, debt, contributions, and saved-goal progress.
- Upgrading legacy totals preserves their amounts in generic Savings and Other investments accounts. It keeps already typed accounts and order. A repeated startup makes no additional change.
- An affected plan's revision advances once. A stale editor must reload; a failed migration leaves saved plans unchanged.

## 18. Record ordinary goal progress

**As a member, I want to record an amount saved toward a goal so I can see progress independently of its planned monthly contribution.**

- Ordinary goals offer Monthly contribution only or Saved amount and target.
- A target must be positive. The saved amount can be zero or exceed the target.
- A tracked goal shows saved amount of target and a native progress bar capped at the target. Overfunded text retains the actual saved amount.
- An untracked goal has no progress bar. Turning tracking off saves only the chosen contribution and other ordinary goal fields.
- Ordinary progress has no deadline or monthly suggestion and cannot coexist with trip or purchase details.
- A tracked goal can have a zero contribution. Editing saved progress leaves account balances, income, and monthly allocations unchanged.
- Shared progress describes the whole goal, while the personal view shows the member's contribution separately.

## 19. Share my household with a partner

**As the first member, I want to copy a private link so my partner can join without an email integration.**

- Overview offers household sharing. Saving a new second profile offers the link automatically.
- Creating a link returns its secret once. The server stores only a hash. A new link replaces the previous pending link.
- The interface explains the shared access and seven-day expiry. Copy failure leaves a manual copy option.
- The first member can revoke a pending link. The second member cannot replace the owner's access or invite a third person.
- The recipient's URL fragment is removed before application resources load. A limited preview contains names and expiry, with no finances.
- A verified Google identity accepts the link once. Acceptance preserves an existing second profile or appends an empty one under the household lock.
- Revoked, expired, replaced, and used links fail. Concurrent claims have one winner. Identity binding failure rolls back the claim.
- Later sign-ins return to the same household without adding the recipient's email to deployment variables.
- If link creation fails after setup saves, the plan stays saved and invitation creation can be retried separately.

**Evidence boundary:** Google tests use a signed local provider. These criteria do not establish live Google acceptance or a deployed release.

## 20. Track payroll contributions without counting them twice

**As a member, I want to record my retirement deductions and employer match so I can see what is saved before take-home pay arrives.**

- Monthly take-home is labeled after taxes and automatic payroll investments.
- Traditional and Roth 401(k) percentages are separate. Their combined rate cannot exceed 100%.
- Employer matching supports ordered tiers or an entered monthly amount. Invalid or mixed rules fail without discarding the draft.
- Payroll estimates appear separately in At a glance. They do not change spendable totals or account balances.
- A match of 100% through 3% and 50% through 5% produces a 4% salary match at a 5% employee contribution.
- Existing plans without payroll details remain valid and unchanged.

## 21. Plan an annual Roth IRA contribution

**As a member, I want an annual IRA goal so I can compare entered contributions with my allowed target.**

- A Roth IRA preset uses a personal owner, contribution year, annual target, and contributions so far.
- The user enters the allowed target. The app does not infer eligibility from salary or hardcode an annual legal maximum.
- The chosen monthly contribution is the only cash allocation. The target and prior contributions do not become new income or account balances.
- An IRA goal cannot also contain trip or purchase details. It can have a zero monthly allocation.
- To-do reminders apply only in the contribution year, stop at a funded entered target, and do not exceed the entered remaining amount.

## 22. Work through a pay-period checklist

**As a member, I want dated checkboxes so I can record the contributions and manual bills I need to handle before the next payday.**

- An actual payday anchors a regular cadence. Monthly pay supports an explicit last-day option. Semimonthly pay requires two distinct valid days.
- Expected dates handle leap years and short months. They do not infer holiday adjustments. Irregular pay has no expected schedule.
- To-do has an As-of date, household/person scope, period dates, task amounts, and due dates. It does not multiply the list by the financial dashboard period.
- Goal tasks divide a month's contribution across its actual paydays and preserve every cent.
- Automatic charges create no tasks. A manual bill needs a due day. Missing days produce a visible setup notice.
- A recorded payer gets the full bill reminder. Otherwise, members get their allocated shares. A bill appears once per due month.
- Native checkboxes cross out completed tasks. Completion survives reload and is shared by household members.
- A changed amount or due date makes the task incomplete. An unrelated edit or name change preserves completion.
- A stale plan revision or unavailable source rejects a checkbox write. A failed request does not discard a newer choice.
- Checking a task does not execute a payment or update goal savings. Earlier periods remain reachable by date; unfinished earlier tasks do not automatically carry forward.

## 23. Connect accounts without changing my plan

**As a member, I want automatic balance snapshots so I can reduce manual balance entry.**

- The Overview balance area opens compact connected-account management in the existing product theme.
- Manual planning remains available when Plaid is disabled or unavailable. Production starts empty.
- Official Link handles institution consent. The application never asks for bank passwords or verification codes.
- New linking and reconnect attempts bind to the current session, member, and household. Expired or repeated exchange attempts fail.
- The operator must verify actual Trial eligibility and remaining team capacity before enabling production. A new Link reserves local lifetime capacity. Removing an Item never restores it.
- Bank and investment connections use separate minimal product flows. The application does not fetch transaction history or execute financial actions.
- An imported account starts excluded. A member explicitly chooses ownership, classification, and inclusion. Traditional, Roth, and high-yield status are not guessed from names.
- Mapping a selected eligible account to a manual snapshot replaces that snapshot in effective totals only. Conflicting mappings fail. The saved plan and its revision stay unchanged.
- A shared account counts once in the household and divides exact cents between personal views. Duplicate copies through another login must remain excluded.
- Missing, unsupported, inactive, or non-USD balances stay explicit. Missing values do not become zero. Current balance determines totals; available balance is separate.
- The view shows connection state, successful synchronization time, and provider freshness when supplied. Investment reads are cached. Scheduled updates are not described as real time.
- Both members can read household balances and change account mappings. Only the connecting member can reconnect or disconnect the connection.
- A stale mapping cannot erase a newer mapping. Failed requests and background reads preserve current editor drafts.
- Reconnect uses the existing Item. Disconnect must revoke provider access before deleting the credential. A failed revocation remains visible and retryable.
- A late synchronization cannot restore a disconnected connection or replace newer data. Expired worker leases recover after restart.
- Provider errors and invalid webhooks reveal no credentials or raw provider payloads.
- Desktop, narrow, short-screen, and keyboard flows keep controls, focus, and dialog actions reachable.

**Evidence boundary:** isolated provider simulations and PostgreSQL checks prove local application behavior. Official Sandbox, real institution consent, exact account coverage, Trial capacity, and deployed release verification require separate evidence. See [Plaid design](system-design/plaid.md).

## 24. See all linked accounts together

**As a member, I want one Accounts tab with the latest balances returned by my linked institutions.**

- Accounts appears in desktop and mobile navigation. It shows all household accounts without planning-month, period, or personal-share controls.
- Every returned account remains visible, including excluded, unclassified, inactive, non-USD, credit, and loan accounts. Viewing an account does not silently select it for planning totals.
- Account names, plain account types, and balances are easy to scan. Native account details expose owner confirmation, provider type, inclusion, manual replacement, and provider data time.
- Current and available balances remain separate. Credit and loan current balances are amounts owed; credit availability is labeled available credit. The view does not add debts into assets or infer a complete net worth.
- The direct Connect accounts action opens the bank/investment choice. Continue to Plaid starts the deliberate consent flow. Existing connection management remains available.
- Plaid is authoritative for displayed linked balances. A valid explicitly mapped snapshot replaces its manual copy once in existing totals. Unmatched manual records, income assumptions, budgets, contribution targets, and saved goal progress remain separate.
- The view shows successful synchronization and provider freshness separately. Missing amounts stay unavailable, including values that cannot be represented in whole cents. It does not promise instant institution updates.
- While visible, the view reads stored server snapshots on entry, on foreground changes, and at a bounded interval. Reads share request ownership, preserve revision and mutation reconciliation, and stop scheduling work when the view closes.
- Reload failures retain earlier rows with clear feedback. Mapping drafts and open account details survive status updates.
- Empty, populated, error, narrow, short-screen, and keyboard states keep account values, update information, and connection actions reachable.

Monthly spending and net investment contributions require separate activity history. A balance increase is not proof of either spending or contributions. The agreed future contribution metric is cash added minus withdrawals, with trades and market gains separate.

## Future stories

These are product directions, not implemented or verified features:

| Actor and desired action                                                | Intended value                       | Work still required                                                          |
| ----------------------------------------------------------------------- | ------------------------------------ | ---------------------------------------------------------------------------- |
| A member reads actual spending and debt activity.                       | Compare the plan with what happened. | Transaction model, transfers, deduplication, corrections, sync, and history. |
| A member reads monthly net investment contributions.                    | Measure new money invested.          | Investment activity, withdrawals, transfer matching, history, and coverage.  |
| A member automatically tracks contributions toward a target.            | Measure real progress over time.     | Verified contribution records and deliberate goal reconciliation.            |
| A member confirms pay dates from a provider.                            | Reconcile expected and received pay. | Payroll or transaction identification and immutable deposit history.         |
| A member asks an optional finance assistant about authorized summaries. | Explain their own figures.           | Consent, read-only tools, data boundaries, and grounded answers.             |

See [system design](system-design/README.md#verification-boundary) for evidence limits and [design decisions](system-design/decisions.md) for the current tradeoffs.
