# User stories

These stories describe the implemented first version. Acceptance criteria state observable behavior. They do not claim a live deployment or a completed provider integration. The [product brief](product-brief.md) remains the source for longer-term scope.

## 1. Open the local household plan

**As a local evaluator, I want to open an empty household plan so I can try the application with my own inputs.**

- Explicit demo mode starts only with a loopback listener and browser origin.
- The entry action is **Open household plan** and creates a session for the first local member.
- The interface shows **Our household**, the saved member names, **Household member**, and **Household plan**.
- Edits persist in PostgreSQL and remain after reload or sign-out.
- A new local household has no financial records. All demo sessions use that same saved local household. Private mode uses a different household and rejects demo-session creation.

**Evidence boundary:** local mode does not establish Google identity or bank access. Automated tests use isolated populated fixtures. Startup preserves existing plans in both modes.

## 2. Sign in as an invited household member

**As an invited member, I want my own Google sign-in so the server can identify me and enforce household access.**

- Private mode requires two distinct invited emails and complete Google configuration.
- The callback requires a valid one-use sign-in attempt and a verified Google identity.
- A verified invited email binds to a stable Google subject. A different subject cannot claim that invitation later.
- An uninvited account cannot obtain a household session.
- Expired sessions and sessions inconsistent with the configured invitations cannot read or edit the plan.
- Both invited members can read and edit the complete household plan.

**Evidence boundary:** signed local identity-provider tests and real PostgreSQL tests cover this flow. One invited Google account has passed live sign-in, sign-out, and repeat sign-in. The second invited login and live rejection of an uninvited account remain unverified.

## 3. Create the first useful monthly plan

**As a member, I want a short setup flow so I can see what our income can support.**

- A household with zero income and no expenses or goals opens plan setup.
- Three steps collect income, expenses, and goals with balances.
- Each member can enter monthly take-home pay, pay frequency, annual base salary, typed account balances, and debt.
- Salary is used for salary-based splits; no tax estimate is deducted from entered take-home pay.
- The saved result shows take-home pay minus living expenses, then subtracts planned goal contributions.
- Missing income or an empty expense list shows a visible completeness notice.
- Invalid values return to the relevant step, show an error, and preserve the entered draft.

## 4. Read the household money flow

**As a member, I want one view of the household plan so I can see where our combined income goes.**

- The default scope is the combined household.
- Money flow contains the Sankey. The default Overview shows Plan totals, At a glance, and Balances. At a glance compares both members in compact income and balance tables.
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
- The two members' allocated totals add to the household totals.
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

- Overview is the default and shows Plan totals, At a glance, and Balances. At a glance fits both members' core facts in two compact tables. Narrow or short layouts use three section buttons to show one section at a time.
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

## Future stories

These are product directions, not implemented or verified features:

| Actor and desired action                                                | Intended value                       | Work still required                                                                             |
| ----------------------------------------------------------------------- | ------------------------------------ | ----------------------------------------------------------------------------------------------- |
| A member connects a bank or brokerage with consent.                     | Reduce manual entry.                 | Provider selection, secure token storage, live account support, reconnect and revocation flows. |
| A member reads actual spending and debt activity.                       | Compare the plan with what happened. | Transaction model, transfers, deduplication, corrections, sync, and history.                    |
| A member automatically tracks contributions toward a target.            | Measure real progress over time.     | Actual contribution records and historical balances.                                            |
| A member uses actual pay dates.                                         | Plan cash between real deposits.     | Payroll or transaction identification and calendar periods.                                     |
| A member asks an optional finance assistant about authorized summaries. | Explain their own figures.           | Consent, read-only tools, data boundaries, and grounded answers.                                |

See [system design](system-design/README.md#verification-boundary) for evidence limits and [design decisions](system-design/decisions.md) for the current tradeoffs.
