package store

const (
	queryLinkedPartner = `SELECT EXISTS (
 SELECT 1 FROM household_users WHERE household_id=$1 AND member_id='member-2'
 )`
	queryInvitationStatus = `
 SELECT EXISTS (SELECT 1 FROM household_users WHERE household_id=$1 AND member_id='member-2'),
 (SELECT expires_at FROM household_invitations WHERE household_id=$1 AND expires_at>clock_timestamp())
 `
	queryInvitationPreview = `
 SELECT h.plan#>>'{members,0,name}',COALESCE(h.plan#>>'{members,1,name}','New person'),i.expires_at
 FROM household_invitations i JOIN households h ON h.id=i.household_id
 WHERE i.household_id=$1 AND i.token_hash=$2 AND i.expires_at>clock_timestamp()
 AND NOT EXISTS (SELECT 1 FROM household_users WHERE household_id=$1 AND member_id='member-2')
 `
	querySetInvitation = `
 INSERT INTO household_invitations (household_id,token_hash,expires_at) VALUES ($1,$2,$3)
 ON CONFLICT (household_id) DO UPDATE SET token_hash=EXCLUDED.token_hash,expires_at=EXCLUDED.expires_at
 `
	queryRevokeInvitation  = `DELETE FROM household_invitations WHERE household_id=$1`
	queryConsumeInvitation = `
 DELETE FROM household_invitations WHERE household_id=$1 AND token_hash=$2 AND expires_at>clock_timestamp()
 `
	queryAcceptInvitation = `
 UPDATE household_users SET invitation_accepted=true WHERE household_id=$1 AND member_id='member-2'
 `
)
