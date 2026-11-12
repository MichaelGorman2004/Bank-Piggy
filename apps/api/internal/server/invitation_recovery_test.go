package server

import (
	"testing"
)

const rejectPartnerSessionSQL = `
 CREATE FUNCTION reject_partner_session() RETURNS trigger LANGUAGE plpgsql AS $$
 BEGIN
 IF NEW.member_id='member-2' THEN RAISE EXCEPTION 'simulated session failure'; END IF;
 RETURN NEW;
 END
 $$;
 CREATE TRIGGER reject_partner_session BEFORE INSERT ON sessions
 FOR EACH ROW EXECUTE FUNCTION reject_partner_session();
`

func TestSuccessfulClaimCanRecoverAfterSessionStorageFails(t *testing.T) {
	client := newPrivateSoloClient(t)
	client.app.config.Invites = []string{"one@example.com"}
	configureSecondGoogleIdentity(t, client.app)
	token := createTestInvitation(t, client)
	if _, err := client.app.store.Pool.Exec(t.Context(), rejectPartnerSessionSQL); err != nil {
		t.Fatal(err)
	}
	requireStatus(t, invitationCallback(t, client, "session-failure", tokenHash(token)), 500)
	identity, err := client.app.store.Identity(t.Context(), "private-household", "two@example.com", "second-subject")
	if err != nil || !identity.InvitationAccepted {
		t.Fatalf("session failure lost durable membership: %+v %v", identity, err)
	}
	if plan := readPlan(t, client); len(plan.Members) != 2 || plan.Revision != 2 {
		t.Fatal("session failure lost the accepted profile")
	}
	if _, err := client.app.store.Pool.Exec(t.Context(), "DROP TRIGGER reject_partner_session ON sessions; DROP FUNCTION reject_partner_session()"); err != nil {
		t.Fatal(err)
	}
	recovered := secondGoogleCallback(t, client, "recover-session")
	requireStatus(t, recovered, 302)
	requireStatus(t, useCallbackSession(t, client, recovered).request("GET", "/api/household", nil, "", ""), 200)
}
