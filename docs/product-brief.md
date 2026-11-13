# Bank Piggy Product Brief

Bank Piggy is a private finance tracker for Michael and his girlfriend. A new household starts with one person and can add a second profile during setup or later. Each invited person with a saved profile signs in and gets a monthly view of where the household's money goes. Each member can switch from the household view to their own personal view. The dashboard puts spending, budgeting, savings, investments, and debt in one place. The longer-term product connects to banks and brokerages for automatic updates.

## Dashboard and money flow

The signed-in app has seven focused views behind a left navigation bar. Overview opens by default with summary cards, recorded plan inputs, and balance snapshots. Accounts lists every account returned by linked institutions and its latest recorded balance. Money flow contains the Sankey. Expenses combines bills, subscriptions, regular costs, and loans. Goals contains ordinary savings, investment, debt, and annual Roth IRA contributions. Trips & purchases contains manual target plans. To-do contains pay-period contribution and manual payment reminders.

Each Sankey stream's width shows its amount. The default scope combines the household. The person selector switches to the signed-in member's allocated share. Keep scope, period, and planning month consistent across financial views. Month is the default. The yearly projection covers the selected month and following eleven months, including scheduled payment changes. The paycheck view shows an average for regular pay schedules. To-do uses an explicit As-of date and an entered payday anchor. Actual spending history and provider-confirmed payroll activity belong to a later phase.

Keep the app frame within the visible viewport. Scroll only card bodies; keep their headers, actions, filters, and footers fixed. The document and outer workspace stay still, including on short screens. Overview has Plan totals, At a glance, and Balances sections. At a glance uses two compact income and balance tables for the saved members instead of a long member list. On narrow or short screens, three native section buttons show one section at a time. Use thin taupe scrollbars with transparent tracks.

Measure the Money flow chart's available height and let its body scroll. Projection assumptions are available through the month control's native disclosure. Keep the largest expense categories visible and combine smaller categories into an explicit `Other costs (N)` row when needed. Preserve every cent. Detailed records remain in Expenses.

Use one themed month picker for planning, payment dates, and target deadlines. Its popover supports year entry, month buttons, keyboard navigation, and touch. Keep it inside the visible viewport. Required fields cannot clear their value; optional fields offer Clear month. Apply a planning-month selection automatically, keep it editable while a forecast loads, and restore a failed latest choice without replacing a newer selection.

The current money flow starts from entered take-home pay. A future gross-pay view could show $100,000 splitting into $15,000 of withholding and $85,000 of take-home pay, with $20,000 then going to rent. Add that view only when the user or a payroll source supplies the deductions.

Keep the budget and future actual activity distinct. When transaction views are added, count card purchases once and classify payments for those purchases as transfers. Show card payments and changes in debt in the future debt activity view. Keep savings and investment contributions visible as allocations. Money moved between tracked accounts does not create new income or spending.

## Visual direction

Use brown and cream surfaces, orange accents, and translucent glass panels over a background image. The supplied Glassy Dashboard V7 + CRM and V24 recordings show warm interior photography behind rounded panels, a left sidebar, compact summary cards, and charts. Keep a visible gap between the sidebar and dashboard so the image shows through. Put the pink cartoon banker pig in a suit and tie near the top-left corner.

Prioritize iPhone use. Keep controls readable at narrow widths, use comfortable touch targets and native form text sizes, preserve keyboard focus, and account for safe areas. Chart labels must stay within the diagram and remain readable when a stream is too thin to contain its label.

Payment icon colors follow the payment type and stay consistent after filtering or sorting.

Use household labels consistently: Our household, the current member names, Household member, and Household plan. The local entry action is Open household plan. New households start empty. Populated examples belong in tests and documentation, outside the running application. These labels do not change runtime access rules.

## Onboarding and the first monthly plan

- Start with one person. Place a plus-icon **Add person** button below the profile. It adds the optional second profile, focuses its name field, and preserves the current draft. Existing saved profiles remain intact, including a second person with no income. Profile removal is outside this version. A successful save after adding a person offers an invitation link. Only verified acceptance grants Google sign-in access.
- Ask each member for take-home pay and pay frequency, plus housing costs, groceries, other regular expenses, and minimum debt payments. Collect annual base salary when members choose salary-based expense splits or want to show gross salary in the money flow. Convert recurring amounts to a monthly estimate. Let the user revise estimates as they learn more. Ask for actual paycheck deductions when the user wants to expand the money flow to show them.
- Let each member enter named, typed account balances and outstanding debt when they want to track balances. Keep saved-goal progress as a separate manual input.
- Ask for state when a feature uses it. Take-home pay already accounts for payroll withholding, so the first cash-flow calculation can work without state. If Bank Piggy later estimates taxes from gross pay, define that as a separate feature with its own inputs and assumptions.

The first useful result is estimated monthly take-home pay minus living expenses and minimum debt payments. Show the amount available for goals or the size of the shortfall. If the user skips a material cost, show that the estimate is incomplete. Keep planned amounts, actual spending, and account balances distinct. A bank feed can inform the estimate, but the user still decides what to budget for groceries next month.

## Shared expenses

For a one-person household, assign the full amount to that person and hide split controls. New expenses and goals belong to that person by default. Adding a second profile keeps their ownership, so existing personal costs do not become shared. Existing shared records remain shared.

For a two-person household, support three split methods for shared expenses.

- **Even split.** Each of the two members pays half.
- **Custom split.** The members choose a percentage for the first member and the other member receives the remainder. Fixed-dollar custom entry remains a future option.
- **Base salary split.** Divide each member's annual base salary by the combined annual base salary. With salaries of $100,000 and $50,000, Michael's share is two-thirds and his girlfriend's share is one-third. A $3,000 shared bill becomes $2,000 and $1,000 in their personal views. The household view shows the $3,000 bill once.

Propose a household default with an override for each shared expense. Record who paid separately from each person's share. A payment by one member does not change the agreed split. Transfers between members to settle a shared bill keep the original expense total intact.

Calculate shares from the full salary ratio and round the resulting money amounts to cents. Assign any remaining cent consistently so the personal shares always add up to the full expense. If the combined base salary is zero or a salary is missing, ask the members to choose an even or custom split. Salary-based splitting is available once both nonnegative base salaries are supplied and their total is positive.

Propose keeping each recorded expense's split when a salary or household default changes. Apply new rules to future expenses and let members explicitly recalculate earlier expenses when needed. Joint account ownership is a separate decision from expense splitting.

## Sharing a household

The first member can copy a one-use link for the second person. No email delivery is required. The recipient sees a limited preview, continues with Google, and joins the same saved household. Future sign-ins use that pinned membership. If no second financial profile exists, acceptance appends an empty one without replacing the first person's plan.

Links expire after seven days. Creating a new link replaces the previous pending link. The first member can revoke it. Store only a hash of the secret, remove the URL fragment on entry, and bind the invitation to the one-use Google sign-in attempt. Consume it with identity binding under the household row lock. Existing two-person identities cannot be replaced by a link. Explain that anyone with the link can accept it and access the household plan.

## Payroll and retirement contributions

Label monthly take-home as cash after taxes and automatic payroll investments. Record separate Traditional and Roth 401(k) percentages. Record employer matching as ordered contribution tiers or an entered monthly amount for an unusual rule. Show estimated employee and employer contributions separately from spendable income. Never subtract them from entered take-home again or add them to an account balance automatically.

An annual Roth IRA goal has a personal owner, contribution year, entered allowed target, and entered contributions so far. The user chooses its monthly cash allocation. It uses ordinary goal progress and cannot also be a trip or purchase. Do not infer legal eligibility or a maximum from salary alone.

## Pay-period To-do

Record one actual payday and the existing pay frequency. Weekly and biweekly advance by seven or fourteen calendar days. Monthly pay keeps the day number, with an explicit last-day-of-month option. Semimonthly pay has two explicit days that stay distinct in short months. Expected dates do not adjust for holidays or employer changes. Irregular pay has no inferred dates.

To-do has its own As-of date and keeps the household/person scope. Generate the current pay period for each configured member. Divide monthly goal allocations across that month's actual paydays with exact cent reconciliation. Show the contribution amount and a due date before the next payday. Annual IRA reminders apply only in their contribution year and do not exceed the entered remaining target.

Manual bills need an explicit due day. Subscriptions default to automatic payment and create no tasks unless changed to manual. A recorded payer receives the full payment reminder; otherwise, shared bills use each person's share. Show missing due days as setup work instead of assuming a due date.

Use native checkboxes and cross out completed tasks. Persist completions by household, member, source, and period. A completion acknowledges the task's amount and dates; changes to those fields make it incomplete again. Checkbox writes check the current plan revision and task state. They do not move money, update goal progress, or alter plan revisions. Prior periods can be selected by date; this is not an immutable payment ledger or an automatic carryover list.

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

## Connected accounts and automatic balance updates

The first Plaid release connects bank and investment accounts for the existing private household. The app stays free to users. It has no checkout, billing, or paywall. Plaid access has its own plan limits.

Use official Plaid Link for consent. Keep access tokens encrypted on the server. New bank connections initialize Auth and use scheduled Balance reads. Investment connections initialize Investments and read cached account balances. The app does not retrieve transaction history, holdings, or activity. It cannot transfer money or place trades. See the [implemented Plaid design](system-design/plaid.md) for product choices, consent scope, failure handling, and costs.

Untouched supported USD accounts enter totals automatically under the person who linked the institution. The server uses the provider type and subtype for checking, savings, HSA, and brokerage defaults. Other investment accounts use a neutral Other investment type; unknown bank types stay excluded. Saving account settings replaces these defaults, including a saved exclusion. Members can correct ownership, select a specific retirement type, or exclude duplicate copies. A member can map an imported account to an existing manual account. The effective balance view then counts the connected snapshot once and leaves the editable plan intact. Shared ownership counts one account in household totals and divides its cents between personal views. Exclude duplicate copies linked through another login. Matching names or last four digits do not prove identity.

Accounts shows the full provider balances even before an account is classified or included in planning totals. It includes returned credit and loan accounts, with debt labeled as an amount owed. Show one current balance per account, with debt labeled Amount owed. Keep any available balance in the provider snapshot without showing a second amount in the account list. Do not sum card debt into assets or claim a complete net worth from the current asset-only projection. Keep institution and account identity, account type, source, and connection state visible. Omit the account-details disclosure, secondary available balance, account count, and per-institution update timestamps. This view covers all household accounts and has no planning month, period multiplier, or personal-share selector.

For a matched manual account, a valid supported Plaid current balance is authoritative. It replaces that manual snapshot once in displayed totals. A missing or unrepresentable balance remains unavailable and cannot silently replace a known manual value with zero. Keep the original manual record for fallback. A refresh updates balance facts, not income assumptions, expense budgets, contribution targets, or goal progress. Manual debt is still an aggregate without individual account identity; it cannot be reconciled automatically from one returned credit-card balance.

The shared financial workspace reloads saved server snapshots every 30 minutes while visible, on entry, and when the user returns to it. Accounts, Overview, and matched manual-account editor rows use this same projection. These status checks do not force a new institution read. Scheduled provider reads also use a 30-minute cadence; ordinary refresh requests and failed reads cannot bypass that deadline. First import and consent repair can read immediately. Institution availability determines data freshness. State the approximate 30-minute check cadence and keep delayed or reconnect states visible; these values are latest snapshots rather than instant prices. Card-only institution linking requires a separate product choice; showing a card returned by an existing bank connection does not establish standalone card coverage.

Do not infer Traditional, Roth, or high-yield status from an account name. Keep missing balances distinct from zero. Show only current balances in the account list. Only supported USD current balances enter USD totals. Checking and HSA balances remain distinct from savings and investments. Refreshes cannot change income, budgets, payroll assumptions, goal progress, or To-do completion.

Show institution, account name, account type, and connection state. Keep owner and explicit manual matching in account settings. A matched manual account editor shows the authoritative Plaid balance read-only and retains its original value as a labeled manual fallback. Provide update-mode reconnect, delayed-update feedback, explicit mapping, and disconnect with provider revocation. Scheduled updates are not real-time data. Manual accounts remain available when Plaid is disabled, unavailable, or unsupported.

Capital One, Robinhood, Principal retirement accounts, Huntington, and BMO US are coverage targets. Checking, savings, brokerage, Traditional and Roth IRAs and 401(k)s, and HSAs require exact institution and account checks. They are not a verified compatibility list.

Use Sandbox for development. The eligible Trial plan permits 10 created production Items across the Plaid team. An Item is an institution login and can contain several accounts. Removal does not restore the lifetime allowance. Before live consent, verify the actual team's eligibility and remaining capacity. Production linking requires explicit operator configuration and a conservative local reservation budget. The application must not upgrade a paid plan or exceed the approved allowance. [Trial rules](https://plaid.com/docs/account/billing/#trial-plans)

Transaction history, actual-spending reconciliation, automatic goal progress, credit-card activity, and provider-confirmed payroll belong to later phases. They need deliberate data models and lifecycle rules.

For future monthly activity, Michael defines invested money as **net contributions**: cash added to investment accounts minus withdrawals. Trades, dividends, and market gains remain separate. A transfer from checking into an investment account is a contribution; a transfer between two investment accounts must not become new household investment funding. Monthly spending needs purchase and refund activity, with transfers and credit-card payments reconciled so a purchase counts once. Balances alone cannot establish either metric. Transaction products, account coverage, consent, update latency, retention, and any production costs must be checked before this additional history is enabled.

## Status and a finance assistant

Start with statuses based on the user's own plan. For example, “You saved $400 of your $500 monthly target” or “Dining was $80 over budget this month.” Show the amount, period, and target behind each status. Use “ahead,” “on track,” or “behind” when a target date or agreed pace makes the comparison meaningful. A cash surplus can support a scenario such as “What happens if I save $100 more each month?” Investment risk depends on time horizon and risk tolerance. Ask for those inputs before considering guidance about taking more risk. [Investor.gov asset allocation guidance](https://www.investor.gov/introduction-investing/getting-started/asset-allocation)

Later, an optional assistant could answer “Where could I cut spending?” The app would give it read-only tools that return authorized summaries such as spending totals by category and month. Keep the underlying transactions, account identifiers, and provider credentials inside Bank Piggy. Explain which summary data goes to the model before a user enables it. Ground answers in the same calculations the dashboard uses.

## Stack and deployment

Keep the Go backend and TypeScript frontend in one monorepo. The first implementation uses React, Vite, ECharts, and PostgreSQL. The private application and PostgreSQL are hosted on Railway.

- **Backend.** Use one Go application with PostgreSQL. Go owns sign-in, permissions, finance calculations, and synchronization. Plaid maintains an official Go client. [Plaid client libraries](https://plaid.com/docs/api/libraries/)
- **Frontend.** Use React and TypeScript with Vite, which builds the browser app. Use Apache ECharts for the Sankey, bar, and line charts. React suits the shared period filters, onboarding steps, and multiple dashboard views. Go can serve the frontend build and the application API from the same domain. [Vite React and TypeScript support](https://vite.dev/guide/) · [Apache ECharts Sankey](https://github.com/apache/echarts-doc/blob/master/en/option/series/sankey.md) · [Go embed package](https://pkg.go.dev/embed)
- **Hosting.** Start with Railway for the Go application and PostgreSQL. Cloudflare's free plan can provide DNS and HTTPS for a custom domain. The Go app still enforces sign-in and account permissions. [Railway PostgreSQL](https://docs.railway.com/databases/postgresql) · [Cloudflare DNS](https://developers.cloudflare.com/dns/faq/) · [Cloudflare free HTTPS certificates](https://developers.cloudflare.com/ssl/edge-certificates/universal-ssl/)
- **Rust.** Keep the first release in Go and TypeScript. If learning Rust is a goal, try it on an offline transaction import tool with sample data once the import rules are clear. Consider a separate service when there is a concrete workload that benefits from one.

## Private sign-in and data ownership

Use Google sign-in through OpenID Connect, a standard way to verify identity. Keep users, household membership, and sessions in PostgreSQL. Start with invite-only sign-up and two separate logins. Use an established Go library to validate the sign-in and link the verified Google identity to an invited Bank Piggy user. Bank Piggy issues its own session. [Google OpenID Connect](https://developers.google.com/identity/openid-connect/openid-connect) · [Go OpenID Connect library](https://pkg.go.dev/github.com/coreos/go-oidc/v3/oidc)

Both members can access the combined household view. Attribute personal records to their owner and count joint accounts once in combined totals. Personal views show each person's allocated share of shared expenses. Check household membership and the permitted view on the server for each data request. Use secure session cookies, protect requests that change data, and keep the database private. Define joint account ownership before dividing those balances between personal views.

## First build and private release

1. The manual first version includes invite-only sign-in code, onboarding, focused navigation, shared-expense splits, scheduled payments, and trip and purchase targets. Use editable inputs or sample data to prove that the Sankey and plan totals agree. Complete live Google sign-in and hosting verification before a private hosted release.
2. Implement read-only Plaid balances locally. Before live use, verify the actual Trial allowance, exact institution and account coverage, deployed OAuth consent, automatic balance updates, household permissions, reconnect, and revocation.
3. Add actual spending history, real pay dates, and automatic target progress as those data become available. Retain the existing manual snapshots and period forecasts. Use the same authorized totals for the assistant if that feature moves forward.

## Open decisions

1. Verify exact institution and account coverage and the actual Plaid team allowance before live linking. Shared connected balances use reconciled equal personal shares; expense splits remain separate.
2. Verify the second invited Google login and select a backup method with a tested restore. Private Railway hosting and the first invited login have passed.
