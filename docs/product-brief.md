# Bank Piggy Product Brief

Bank Piggy is a private finance tracker for Michael and his girlfriend. Each person signs in, answers a short set of questions, and gets a monthly view of where the household's money goes. Each member can switch from the combined household view to their own personal view. The dashboard puts spending, budgeting, savings, investments, and debt in one place. The longer-term product connects to banks and brokerages for automatic updates.

## Dashboard and money flow

The signed-in app has five focused views behind a left navigation bar. Overview opens by default with summary cards, recorded plan inputs, and balance snapshots. Money flow contains the Sankey. Expenses combines bills, subscriptions, regular costs, and loans. Goals contains ordinary savings, investment, and debt contributions. Trips & purchases contains manual target plans.

Each Sankey stream's width shows its amount. The default scope combines the household. The person selector switches to the signed-in member's allocated share. Keep scope, period, and planning month consistent across views. Month is the default. The yearly projection covers the selected month and following eleven months, including scheduled payment changes. The paycheck view shows an average for regular pay schedules. Actual spending history and real pay dates belong to a later phase.

Keep the app frame within the visible viewport. Scroll only card bodies; keep their headers, actions, filters, and footers fixed. The document and outer workspace stay still, including on short screens. Overview has Plan totals, At a glance, and Balances sections. At a glance uses two compact income and balance tables for both members instead of a long member list. On narrow or short screens, three native section buttons show one section at a time. Use thin taupe scrollbars with transparent tracks.

Measure the Money flow chart's available height and let its body scroll. Projection assumptions are available through the month control's native disclosure. Keep the largest expense categories visible and combine smaller categories into an explicit `Other costs (N)` row when needed. Preserve every cent. Detailed records remain in Expenses.

Use one themed month picker for planning, payment dates, and target deadlines. Its popover supports year entry, month buttons, keyboard navigation, and touch. Keep it inside the visible viewport. Required fields cannot clear their value; optional fields offer Clear month. Apply a planning-month selection automatically, keep it editable while a forecast loads, and restore a failed latest choice without replacing a newer selection.

The current money flow starts from entered take-home pay. A future gross-pay view could show $100,000 splitting into $15,000 of withholding and $85,000 of take-home pay, with $20,000 then going to rent. Add that view only when the user or a payroll source supplies the deductions.

Keep the budget and future actual activity distinct. When transaction views are added, count card purchases once and classify payments for those purchases as transfers. Show card payments and changes in debt in the future debt activity view. Keep savings and investment contributions visible as allocations. Money moved between tracked accounts does not create new income or spending.

## Visual direction

Use brown and cream surfaces, orange accents, and translucent glass panels over a background image. The supplied Glassy Dashboard V7 + CRM and V24 recordings show warm interior photography behind rounded panels, a left sidebar, compact summary cards, and charts. Keep a visible gap between the sidebar and dashboard so the image shows through. Put the pink cartoon banker pig in a suit and tie near the top-left corner.

Prioritize iPhone use. Keep controls readable at narrow widths, use comfortable touch targets and native form text sizes, preserve keyboard focus, and account for safe areas. Chart labels must stay within the diagram and remain readable when a stream is too thin to contain its label.

Payment icon colors follow the payment type and stay consistent after filtering or sorting.

Use household labels consistently: Our household, the current member names, Household member, and Household plan. The local entry action is Open household plan. These labels do not change runtime access rules or turn seeded reference data into live financial data.

## Onboarding and the first monthly plan

- Ask each member for take-home pay and pay frequency, plus housing costs, groceries, other regular expenses, and minimum debt payments. Collect annual base salary when members choose salary-based expense splits or want to show gross salary in the money flow. Convert recurring amounts to a monthly estimate. Let the user revise estimates as they learn more. Ask for actual paycheck deductions when the user wants to expand the money flow to show them.
- Let each member enter named, typed account balances and outstanding debt when they want to track balances. Keep saved-goal progress as a separate manual input.
- Ask for state when a feature uses it. Take-home pay already accounts for payroll withholding, so the first cash-flow calculation can work without state. If Bank Piggy later estimates taxes from gross pay, define that as a separate feature with its own inputs and assumptions.

The first useful result is estimated monthly take-home pay minus living expenses and minimum debt payments. Show the amount available for goals or the size of the shortfall. If the user skips a material cost, show that the estimate is incomplete. Keep planned amounts, actual spending, and account balances distinct. A bank feed can inform the estimate, but the user still decides what to budget for groceries next month.

## Shared expenses

Support three split methods for shared expenses.

- **Even split.** Each of the two members pays half.
- **Custom split.** The members choose a percentage for the first member and the other member receives the remainder. Fixed-dollar custom entry remains a future option.
- **Base salary split.** Divide each member's annual base salary by the combined annual base salary. With salaries of $100,000 and $50,000, Michael's share is two-thirds and his girlfriend's share is one-third. A $3,000 shared bill becomes $2,000 and $1,000 in their personal views. The household view shows the $3,000 bill once.

Propose a household default with an override for each shared expense. Record who paid separately from each person's share. A payment by one member does not change the agreed split. Transfers between members to settle a shared bill keep the original expense total intact.

Calculate shares from the full salary ratio and round the resulting money amounts to cents. Assign any remaining cent consistently so the personal shares always add up to the full expense. If the combined base salary is zero or a salary is missing, ask the members to choose an even or custom split. Salary-based splitting is available once both nonnegative base salaries are supplied and their total is positive.

Propose keeping each recorded expense's split when a salary or household default changes. Apply new rules to future expenses and let members explicitly recalculate earlier expenses when needed. Joint account ownership is a separate decision from expense splitting.

## Several goals can share one budget

Let users keep savings, extra debt payments, and retirement contributions active at the same time. Make goal details optional during onboarding. For each active goal, let the user set a monthly amount and add a target balance or date when it helps.

Use the monthly amount left over to show how the goals fit together. For example, with $600 available, a user could assign $250 to savings, $200 to extra debt payments, and $150 to retirement. If the total exceeds $600, show the gap so the user can adjust the amounts. Count minimum debt payments in regular expenses and extra payments in goals. If a retirement contribution already comes out of a paycheck, avoid deducting it again from take-home pay.

An ordinary goal can optionally track a positive target amount and money already saved. Show saved amount of target with a bounded progress bar. Without a target, show only the planned contribution. Saved progress does not change account balances, income, or the chosen contribution. Ordinary progress has no deadline or saving-rate suggestion and cannot coexist with trip or purchase details on one goal.

## Recurring payments

Manage subscriptions, bills, and loan payments together in Expenses. Offer All types, Subscriptions, Bills & costs, and Loans filters. Show the largest allocated amount first for the selected scope and period. Retain upcoming, ended, and zero-share records; keep saved order when amounts tie. The card's add action starts with the selected payment type, while the header's Add expense starts as a bill.

Support an optional first month, a final month, or a fixed number of monthly payments. An $800 payment for 36 months contributes to the first through the thirty-sixth month. The following month excludes it from projected costs. Retain the record for review and editing.

Show recognizable service logos where available and category icons for other payments, including a car icon for a loan. The first catalog has 100 service options with local logo assets. Custom payments remain supported. Schedules affect the plan. Users manage the actual service or loan with its provider.

## Trip budgets and purchase targets

Let a trip goal hold expected flights, lodging, food, local transport, activities, and other costs. The sum becomes the savings target. Record money already saved and a target month. Calculate the amount remaining and a suggested monthly contribution using the months from the selected planning month through the target month, inclusive.

Let a purchase goal hold one target amount, money already saved, and a target month. A goal can be ordinary, a trip, or a purchase. Keep trip and purchase details mutually exclusive.

Use the same saving-period rules for trips and purchases. Clamp the remaining amount at zero and round a monthly suggestion up to a whole cent. A past, unfunded target shows overdue with no future monthly suggestion. A funded target remains funded after its deadline.

Keep the user's chosen monthly contribution editable. That contribution is the amount allocated in the household budget. Manual target progress does not create income or change account balances. Shared contributions divide evenly, while the full shared target remains visible in a personal view.

## Manual account balances

Each member can keep up to 20 accounts with a stable ID, name, type, and integer-cent balance. Support Traditional 401(k), Roth 401(k), Traditional IRA, Roth IRA, Brokerage, High-yield savings, Savings, and Other investment. Savings and high-yield savings form the savings total; the other types form investments. Keep debt as a separate entered balance.

Derive totals from accounts rather than storing duplicate savings and investment totals on each member. Preserve older aggregate balances as generic Savings and Other investments during upgrade. Existing data does not identify a Roth, retirement, or brokerage account. Users choose those details when they edit the accounts.

## Future connected accounts and automatic updates

Track checking, savings, credit cards, brokerage and retirement accounts, and health savings accounts (HSAs). For cards, track spending, balances, and credit usage when the credit limit is available. Let people enter balances and contributions manually for accounts that do not connect. Specific bank and HSA providers will be selected after the first implementation.

Use read-only Plaid connections as the first integration to evaluate. Transactions can supply bank and card activity. Investments can supply holdings and investment activity. Plaid lists Robinhood among the institutions that provide tax lot details for holdings. Confirm the exact banks, Robinhood account types, and available data in a live test. Plaid refreshes data at intervals that vary by institution. Show when each connection last updated and whether it needs to reconnect. [Plaid Transactions API](https://plaid.com/docs/api/products/transactions/) · [Plaid transaction refresh timing](https://support.plaid.com/hc/en-us/articles/15373384920599-Do-the-transactions-get-and-transactions-sync-endpoints-provide-real-time-data) · [Plaid Investments](https://plaid.com/docs/investments/) · [Plaid changelog](https://plaid.com/docs/changelog/#june-18-2026)

Plaid's free Trial plan for eligible US and Canadian developers includes Transactions and Investments. The plan allows up to 10 live institution connections, called Items. Removing an Item does not restore a slot. Start with test data in Sandbox and use the trial to verify the exact accounts when ready. [Plaid Trial plan](https://support.plaid.com/hc/en-us/articles/39994173227159-What-is-the-Plaid-Trial-plan) · [Plaid billing docs](https://plaid.com/docs/account/billing/)

Use Plaid Link for consent and keep institution credentials inside the Plaid or institution sign-in flow. Request only the data the app needs, encrypt access tokens and keep them on the server, and keep financial data out of logs and analytics. Verify incoming Plaid notifications and apply transaction additions, corrections, and removals without duplicating records on retry. Disconnecting should revoke access. Decide how long to keep imported history and let users delete it. [Plaid Quickstart](https://plaid.com/docs/quickstart/) · [Plaid transaction updates](https://plaid.com/docs/transactions/transactions-data/) · [Webhook verification](https://plaid.com/docs/api/webhooks/webhook-verification/) · [Removing an Item](https://plaid.com/docs/api/items/#itemremove)

## Status and a finance assistant

Start with statuses based on the user's own plan. For example, “You saved $400 of your $500 monthly target” or “Dining was $80 over budget this month.” Show the amount, period, and target behind each status. Use “ahead,” “on track,” or “behind” when a target date or agreed pace makes the comparison meaningful. A cash surplus can support a scenario such as “What happens if I save $100 more each month?” Investment risk depends on time horizon and risk tolerance. Ask for those inputs before considering guidance about taking more risk. [Investor.gov asset allocation guidance](https://www.investor.gov/introduction-investing/getting-started/asset-allocation)

Later, an optional assistant could answer “Where could I cut spending?” The app would give it read-only tools that return authorized summaries such as spending totals by category and month. Keep the underlying transactions, account identifiers, and provider credentials inside Bank Piggy. Explain which summary data goes to the model before a user enables it. Ground answers in the same calculations the dashboard uses.

## Stack and deployment

Keep the Go backend and TypeScript frontend in one monorepo. The first implementation uses React, Vite, ECharts, and PostgreSQL. Hosting remains a proposal.

- **Backend.** Use one Go application with PostgreSQL. Go owns sign-in, permissions, finance calculations, and synchronization. Plaid maintains an official Go client. [Plaid client libraries](https://plaid.com/docs/api/libraries/)
- **Frontend.** Use React and TypeScript with Vite, which builds the browser app. Use Apache ECharts for the Sankey, bar, and line charts. React suits the shared period filters, onboarding steps, and multiple dashboard views. Go can serve the frontend build and the application API from the same domain. [Vite React and TypeScript support](https://vite.dev/guide/) · [Apache ECharts Sankey](https://github.com/apache/echarts-doc/blob/master/en/option/series/sankey.md) · [Go embed package](https://pkg.go.dev/embed)
- **Hosting.** Start with Railway for the Go application and PostgreSQL. Cloudflare's free plan can provide DNS and HTTPS for a custom domain. The Go app still enforces sign-in and account permissions. [Railway PostgreSQL](https://docs.railway.com/databases/postgresql) · [Cloudflare DNS](https://developers.cloudflare.com/dns/faq/) · [Cloudflare free HTTPS certificates](https://developers.cloudflare.com/ssl/edge-certificates/universal-ssl/)
- **Rust.** Keep the first release in Go and TypeScript. If learning Rust is a goal, try it on an offline transaction import tool with sample data once the import rules are clear. Consider a separate service when there is a concrete workload that benefits from one.

## Private sign-in and data ownership

Use Google sign-in through OpenID Connect, a standard way to verify identity. Keep users, household membership, and sessions in PostgreSQL. Start with invite-only sign-up and two separate logins. Use an established Go library to validate the sign-in and link the verified Google identity to an invited Bank Piggy user. Bank Piggy issues its own session. [Google OpenID Connect](https://developers.google.com/identity/openid-connect/openid-connect) · [Go OpenID Connect library](https://pkg.go.dev/github.com/coreos/go-oidc/v3/oidc)

Both members can access the combined household view. Attribute personal records to their owner and count joint accounts once in combined totals. Personal views show each person's allocated share of shared expenses. Check household membership and the permitted view on the server for each data request. Use secure session cookies, protect requests that change data, and keep the database private. Define joint account ownership before dividing those balances between personal views.

## First build and private release

1. The manual first version includes invite-only sign-in code, onboarding, focused navigation, shared-expense splits, scheduled payments, and trip and purchase targets. Use editable inputs or sample data to prove that the Sankey and plan totals agree. Complete live Google sign-in and hosting verification before a private hosted release.
2. Select providers and test Plaid against the intended bank and Robinhood accounts in a later phase. Before relying on connected data, verify that balances and activity refresh automatically, household permissions hold, and reconnecting or repeating a sync preserves correct totals.
3. Add actual spending history, connected balances, real pay dates, and automatic target progress as those data become available. Retain the existing manual snapshots and period forecasts. Use the same authorized totals for the assistant if that feature moves forward.

## Open decisions

1. How should ownership of joint account balances appear in the personal views? Expense splits are defined separately.
2. Choose the hosting environment and configure live Google sign-in before a private hosted release. Railway remains the suggested target.
