ALTER TABLE bank_link_attempts ADD COLUMN IF NOT EXISTS purpose text NOT NULL DEFAULT 'balances'
 CHECK (purpose IN ('balances','activity'));
ALTER TABLE bank_link_attempts ADD COLUMN IF NOT EXISTS activity_revision bigint;

CREATE TABLE IF NOT EXISTS bank_activity_jobs (
 connection_id text PRIMARY KEY REFERENCES bank_connections(id) ON DELETE CASCADE,
 household_id text NOT NULL REFERENCES households(id),
 revision bigint NOT NULL DEFAULT 1,
 enabled boolean NOT NULL DEFAULT false,
 authorized_at timestamptz,
 accounts jsonb NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(accounts)='array'),
 cursor text NOT NULL DEFAULT '',
 history_start date,
 history_end date,
 history_complete boolean NOT NULL DEFAULT false,
 generation bigint NOT NULL DEFAULT 1,
 next_sync_at timestamptz NOT NULL DEFAULT now(),
 last_synced_at timestamptz,
 lease_owner text,
 lease_until timestamptz,
 failures integer NOT NULL DEFAULT 0,
 error_code text
);
CREATE INDEX IF NOT EXISTS bank_activity_due ON bank_activity_jobs(next_sync_at)
 WHERE enabled AND authorized_at IS NOT NULL;

CREATE TABLE IF NOT EXISTS bank_activity_records (
 id text PRIMARY KEY,
 household_id text NOT NULL REFERENCES households(id),
 connection_id text NOT NULL REFERENCES bank_connections(id) ON DELETE CASCADE,
 account_id text NOT NULL REFERENCES bank_accounts(id) ON DELETE CASCADE,
 provider_id text NOT NULL,
 cancelled_provider_id text,
 kind text NOT NULL CHECK (kind IN ('bank','investment')),
 posted_on date NOT NULL,
 data jsonb NOT NULL,
 cancelled boolean NOT NULL DEFAULT false,
 review_category text,
 transfer_peer_id text REFERENCES bank_activity_records(id) ON DELETE SET NULL,
 revision bigint NOT NULL DEFAULT 1,
 UNIQUE(connection_id,provider_id)
);
CREATE INDEX IF NOT EXISTS bank_activity_month ON bank_activity_records(household_id,posted_on,id);
CREATE INDEX IF NOT EXISTS bank_activity_cancel_references ON bank_activity_records(connection_id,cancelled_provider_id)
 WHERE cancelled_provider_id IS NOT NULL;

ALTER TABLE bank_activity_records ADD COLUMN IF NOT EXISTS expense_id text;
ALTER TABLE bank_activity_records ADD COLUMN IF NOT EXISTS goal_id text;
ALTER TABLE bank_activity_records ADD COLUMN IF NOT EXISTS allocation jsonb
 CHECK (allocation IS NULL OR jsonb_typeof(allocation)='object');

-- Keep cancellation identifiers after a bounded raw-history window is removed.
CREATE TABLE IF NOT EXISTS bank_activity_cancellations (
 connection_id text NOT NULL REFERENCES bank_connections(id) ON DELETE CASCADE,
 account_id text NOT NULL REFERENCES bank_accounts(id) ON DELETE CASCADE,
 provider_id text NOT NULL,
 PRIMARY KEY(connection_id,provider_id)
);
