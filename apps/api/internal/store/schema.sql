CREATE TABLE IF NOT EXISTS households (
 id text PRIMARY KEY,
 revision bigint NOT NULL CHECK (revision > 0),
 plan jsonb NOT NULL,
 updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS household_users (
 household_id text NOT NULL REFERENCES households(id),
 member_id text NOT NULL,
 email text NOT NULL,
 google_subject text NOT NULL,
 PRIMARY KEY (household_id, member_id),
 UNIQUE (household_id, email),
 UNIQUE (household_id, google_subject)
);
CREATE TABLE IF NOT EXISTS sessions (
 token_hash text PRIMARY KEY,
 household_id text NOT NULL REFERENCES households(id),
 member_id text NOT NULL,
 csrf_token text NOT NULL,
 expires_at timestamptz NOT NULL
);
CREATE INDEX IF NOT EXISTS sessions_expiry_idx ON sessions(expires_at);
CREATE TABLE IF NOT EXISTS oidc_attempts (
 token_hash text PRIMARY KEY,
 state text NOT NULL,
 nonce text NOT NULL,
 verifier text NOT NULL,
 expires_at timestamptz NOT NULL
);
CREATE INDEX IF NOT EXISTS oidc_attempts_expiry_idx ON oidc_attempts(expires_at);
ALTER TABLE household_users ADD COLUMN IF NOT EXISTS invitation_accepted boolean NOT NULL DEFAULT false;
ALTER TABLE oidc_attempts ADD COLUMN IF NOT EXISTS invitation_hash text NOT NULL DEFAULT '';
CREATE TABLE IF NOT EXISTS household_invitations (
 household_id text PRIMARY KEY REFERENCES households(id),
 token_hash text NOT NULL UNIQUE,
 expires_at timestamptz NOT NULL
);
CREATE TABLE IF NOT EXISTS todo_completions (
 household_id text NOT NULL REFERENCES households(id),
 member_id text NOT NULL,
 task_id text NOT NULL,
 completed boolean NOT NULL,
 PRIMARY KEY (household_id,member_id,task_id)
);
ALTER TABLE todo_completions ADD COLUMN IF NOT EXISTS task_signature text NOT NULL DEFAULT '';
