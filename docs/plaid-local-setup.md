# Try Plaid locally

This guide covers local Sandbox testing and optional local production setup. The deployed production service is configured separately in Railway. No Railway deployment is required for local testing.

The local Plaid app uses [http://localhost:5188](http://localhost:5188). Its private, ignored `.env` already contains Sandbox credentials and an encryption key. Michael completed Sandbox consent, and 14 simulated accounts were imported. These are not real bank balances. The separate local database is `bankpiggy_local` in the PostgreSQL 18 container. If Docker has stopped, start Docker Desktop and run `docker start bank-piggy-plaid-pg18` before starting the app.

## 1. Create your Plaid account

1. Open [Plaid signup](https://dashboard.plaid.com/signup).
2. Create the account and verify your email. You complete any terms, identity checks, and security attestations yourself.
3. For real banks, open the [free Trial application](https://dashboard.plaid.com/trial-plan). Select the free Trial path if available. Do not upgrade to a paid plan for this setup.
4. Wait for the Dashboard to confirm production Trial access. Record the created-Item count shown there before any real connection.

Sandbox uses simulated accounts and is free. It does not connect your real bank. The eligible Trial plan permits 10 created production Items across the team. An Item is one institution login and can include several accounts. Removing it does not restore capacity. Your actual approval and remaining capacity must be checked. See [Plaid billing and Trial rules](https://plaid.com/docs/account/billing/).

## 2. Add two Sandbox values

In [Plaid Dashboard](https://dashboard.plaid.com/), open the API Keys section. Copy your Client ID and **Sandbox** secret into the repository-root `.env` file:

```dotenv
PLAID_CLIENT_ID=your-client-id
PLAID_SECRET=your-sandbox-secret
```

The rest of the initial setup is already present:

```dotenv
APP_MODE=demo
LISTEN_ADDR=127.0.0.1:5188
APP_BASE_URL=http://localhost:5188
PLAID_ENABLED=true
PLAID_ENV=sandbox
```

Keep the generated `PLAID_TOKEN_KEYS` value. It encrypts stored provider credentials. Enter API secrets only in this private file. The frontend has no secret fields.

Open [Plaid API settings](https://dashboard.plaid.com/team/api). Add this exact allowed redirect URI:

```text
http://localhost:5188/plaid-return
```

Use `localhost` consistently for this setup. Plaid permits HTTP localhost redirects in Sandbox. [Official OAuth guide](https://plaid.com/docs/link/oauth/)

Save the redirect before selecting **Connect accounts**. Plaid rejects Link creation when the redirect is not configured. The app keeps credentials and raw provider errors out of its messages.

## 3. Start and try Sandbox

From the repository root:

```sh
npm run start:local
```

The compiled application and API share one origin. This command loads `.env`. It refuses to start enabled Plaid without the two credentials; it does not pretend that linking is enabled.

1. Open [http://localhost:5188](http://localhost:5188).
2. Select **Open household plan**. You can close the empty plan setup.
3. Select **Accounts** in the navigation. It shows all balances returned by linked institutions, including accounts excluded from planning totals.
4. Select **Connect accounts**. Choose bank accounts or investments, then select **Continue to Plaid**. Add each institution separately. After saving one connection, use **Connect accounts** again to add another institution. The existing manager is also available from **Overview → Balances → Connected accounts**.
5. Use Plaid's Sandbox test login. Standard test credentials are `user_good` and `pass_good`; the standard test verification code is `1234`. Some test institutions use their own prompts. [Plaid Quickstart](https://plaid.com/docs/quickstart/)
6. After linking, use **Reload status** when the first background check finishes. Accounts start excluded. Choose their owner and type before including them.
7. If an imported account represents an existing manual account, select that manual balance as its replacement. Include a joint account once; exclude duplicate copies from another login.

The shared workspace checks saved provider values every 30 minutes while visible and on foreground return. Accounts and Overview share the same balance projection. Accounts shows one current amount per account and labels credit balances as amounts owed. Supported untouched USD accounts enter totals automatically under the member who linked them. Review ownership and match any duplicate manual account in Connected accounts; a saved exclusion remains excluded. Opening the view or selecting **Reload status** reads saved snapshots; it does not force an immediate bank update. Scheduled bank reads also run every 30 minutes. **Request update** cannot bypass that read or retry deadline. First import and consent repair can read immediately. Monthly spending and net investment contributions require transaction history and are not calculated from these snapshots.

Press Control-C in the terminal to stop this app. Run the same command after editing `.env`. The database is separate from automated test schemas and does not reset when the app restarts.

Plaid cannot deliver webhooks to your Mac's localhost address. Scheduled reads can still exercise balance updates. Real webhook delivery requires a reachable HTTPS endpoint and remains a separate check.

## 4. Connect a real bank locally

Sandbox approval and a successful test login do not grant real-bank access. Complete the free Trial approval first. Real-bank mode also requires private Google sign-in and HTTPS. The app deliberately rejects production Plaid in anonymous demo mode.

A local-only HTTPS address can use [Caddy's local HTTPS support](https://caddyserver.com/docs/automatic-https#local-https). The included [local Caddy configuration](examples/Caddyfile.local) binds only to loopback. It does not publish the app on the internet. Caddy has not been installed or started by this setup, and local certificate trust has not been changed.

1. Install Caddy from its official distribution or Homebrew: `brew install caddy`.
2. Run `caddy run --config docs/examples/Caddyfile.local --adapter caddyfile` in a separate terminal. Trust its local certificate authority through the normal macOS prompt. If Safari or Chrome shows a certificate warning, finish certificate setup before continuing.
3. Open [Google Cloud credentials](https://console.cloud.google.com/apis/credentials). Use the existing Bank Piggy Web application client, or create a separate local Web application client. Add this authorized redirect URI while retaining the existing deployed callback:

   ```text
   https://localhost:5443/auth/google/callback
   ```

4. Put that client's ID and secret in the matching `.env` fields. Set `HOUSEHOLD_INVITES` to your Google email address. The app's existing member and session checks remain active.
5. In Plaid API settings, add:

   ```text
   https://localhost:5443/plaid-return
   ```

6. After verifying approved Trial access and the remaining team allowance, change these `.env` fields:

   ```dotenv
   APP_MODE=private
   APP_BASE_URL=https://localhost:5443
   PLAID_ENV=production
   PLAID_SECRET=your-production-secret
   GOOGLE_CLIENT_ID=your-google-web-client-id
   GOOGLE_CLIENT_SECRET=your-google-web-client-secret
   HOUSEHOLD_INVITES=your-google-email
   PLAID_PRODUCTION_APPROVED=true
   PLAID_LIFETIME_ITEM_LIMIT=1
   ```

   Start with one approved new Item. A canceled or uncertain Link attempt can consume the application's conservative reservation. Do not keep creating attempts to test. Reconnect existing Items through update mode. An operator must reconcile an uncertain reservation with the actual Dashboard count before any budget change.

   Check Plaid Dashboard Account Verification settings as well. Dashboard settings can override per-Link Auth options. Keep manual verification and microdeposit flows disabled for this read-only integration.

7. Back up the database and keep the encryption key separately. Verify a restore before importing real financial data. The local database and key exist; an off-device backup and restore have not been configured by this setup.
8. Restart `npm run start:local`. Open [https://localhost:5443](https://localhost:5443), sign in with the invited Google account, then open **Connected accounts** and complete your own institution consent.

Use the Mac's browser for these localhost links. A phone's localhost points to the phone. Physical-phone access needs a separately configured secure origin.

The account names and balances must be checked against the institution. Capital One, Robinhood, Principal, Huntington, BMO US, and each requested account type remain coverage targets until tested. Local production-mode consent does not prove external webhook delivery, a deployed release, or continuous operation while the Mac sleeps.
