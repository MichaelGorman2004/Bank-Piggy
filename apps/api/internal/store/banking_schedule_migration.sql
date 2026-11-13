-- Upgrade only untouched successful daily deadlines. The exact old deadline
-- makes this repeatable without changing worker leases, retries, or newer jobs.
UPDATE bank_connections
SET next_sync_at = GREATEST(now(), last_synced_at + interval '30 minutes')
WHERE state = 'connected'
 AND token_cipher IS NOT NULL
 AND failures = 0
 AND last_synced_at IS NOT NULL
 AND next_sync_at = last_synced_at + interval '24 hours'
 AND next_sync_at > now()
 AND lease_owner IS NULL
 AND (lease_until IS NULL OR lease_until <= now());
