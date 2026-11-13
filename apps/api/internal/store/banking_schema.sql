CREATE TABLE IF NOT EXISTS bank_connections (
 id text PRIMARY KEY,
 household_id text NOT NULL REFERENCES households(id),
 environment text NOT NULL DEFAULT 'sandbox' CHECK(environment IN ('sandbox','production')),
 member_id text NOT NULL,
 kind text NOT NULL CHECK (kind IN ('depository','investment')),
 item_id text,
 token_cipher text,
 institution_name text NOT NULL DEFAULT 'Connected institution',
 state text NOT NULL DEFAULT 'pending',
 generation bigint NOT NULL DEFAULT 1,
 last_synced_at timestamptz,
 next_sync_at timestamptz NOT NULL DEFAULT now(),
 lease_owner text,
 lease_until timestamptz,
 failures integer NOT NULL DEFAULT 0,
 error_code text,
 created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(environment,item_id)
);
CREATE INDEX IF NOT EXISTS bank_connections_due ON bank_connections(next_sync_at) WHERE token_cipher IS NOT NULL;
CREATE TABLE IF NOT EXISTS bank_link_attempts (
 id text PRIMARY KEY,
 household_id text NOT NULL REFERENCES households(id),
 member_id text NOT NULL,
 session_hash text NOT NULL,
 connection_id text NOT NULL REFERENCES bank_connections(id),
 is_update boolean NOT NULL,
 token_cipher text,
 public_token_cipher text,
 state text NOT NULL DEFAULT 'pending',
 expires_at timestamptz NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS bank_attempts_session ON bank_link_attempts(session_hash,created_at);
CREATE INDEX IF NOT EXISTS bank_completed_attempt ON bank_link_attempts(connection_id,created_at DESC,id DESC) WHERE state='completed';
CREATE TABLE IF NOT EXISTS bank_accounts (
 id text PRIMARY KEY,
 household_id text NOT NULL REFERENCES households(id),
 environment text NOT NULL DEFAULT 'sandbox',
 connection_id text NOT NULL REFERENCES bank_connections(id),
 provider_id text NOT NULL,
 snapshot jsonb NOT NULL,
 active boolean NOT NULL DEFAULT true,
 mapping jsonb NOT NULL DEFAULT '{"enabled":false,"ownership":"member-1","classification":"unclassified"}',
 mapping_revision bigint NOT NULL DEFAULT 1,
 manual_member_id text,
 manual_account_id text,
 UNIQUE(connection_id,provider_id)
);
CREATE UNIQUE INDEX IF NOT EXISTS bank_manual_mapping ON bank_accounts(environment,household_id,manual_member_id,manual_account_id) WHERE manual_account_id IS NOT NULL;
CREATE TABLE IF NOT EXISTS bank_item_budget (
 environment text PRIMARY KEY,
 reservations integer NOT NULL DEFAULT 0 CHECK (reservations >= 0)
);
CREATE TABLE IF NOT EXISTS bank_webhook_receipts (
 digest text PRIMARY KEY,
 received_at timestamptz NOT NULL DEFAULT now()
);
