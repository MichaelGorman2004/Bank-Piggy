package store

const (
	queryLockPlan             = `SELECT plan FROM households WHERE id=$1 FOR UPDATE`
	queryPruneExpiredSessions = `
 DELETE FROM sessions WHERE token_hash IN (
  SELECT token_hash FROM sessions WHERE expires_at <= now()
  ORDER BY expires_at LIMIT $1
 )
 `
	queryPruneExpiredOIDCAttempts = `
 DELETE FROM oidc_attempts WHERE token_hash IN (
  SELECT token_hash FROM oidc_attempts WHERE expires_at <= now()
  ORDER BY expires_at LIMIT $1
 )
 `
	querySeedHousehold = `
 INSERT INTO households (id,revision,plan)
 VALUES ($1,$2,$3)
 ON CONFLICT DO NOTHING
 `
	queryReadHousehold = `
 SELECT plan
 FROM households
 WHERE id=$1
 `
	queryUpdateHousehold = `
 UPDATE households
 SET revision=$2,plan=$3,updated_at=now()
 WHERE id=$1
 `
	queryCreateSession = `
 INSERT INTO sessions (token_hash,household_id,member_id,csrf_token,expires_at)
 VALUES ($1,$2,$3,$4,$5)
 `
	queryDeleteSession = `
 DELETE
 FROM sessions
 WHERE token_hash=$1
 `
	queryBindIdentity = `
 INSERT INTO household_users (household_id,member_id,email,google_subject)
 VALUES ($1,$2,$3,$4)
 ON CONFLICT (household_id,member_id)
 DO UPDATE
 SET email=EXCLUDED.email
 WHERE household_users.google_subject=EXCLUDED.google_subject AND household_users.email=EXCLUDED.email
 `
	queryCreateOIDCAttempt = `
 INSERT INTO oidc_attempts (token_hash,state,nonce,verifier,expires_at,invitation_hash)
 VALUES ($1,$2,$3,$4,$5,$6)
 `
	queryConsumeOIDCAttempt = `
 DELETE
 FROM oidc_attempts
 WHERE token_hash=$1 AND expires_at>now()
 RETURNING state,nonce,verifier,expires_at,invitation_hash
 `
	queryLockHousehold = `
 SELECT revision,jsonb_array_length(plan->'members')
 FROM households
 WHERE id=$1 FOR UPDATE
 `
	queryReadSession = `
 SELECT s.household_id,s.member_id,s.csrf_token,s.expires_at,COALESCE(u.email,''),COALESCE(u.invitation_accepted,false)
 FROM sessions s
 JOIN households h ON h.id=s.household_id
 LEFT JOIN household_users u ON u.household_id=s.household_id AND u.member_id=s.member_id
 WHERE s.token_hash=$1 AND s.expires_at>now()
 AND EXISTS (
  SELECT 1 FROM jsonb_array_elements(h.plan->'members') AS member
  WHERE member->>'id'=s.member_id
 )
 `
)
