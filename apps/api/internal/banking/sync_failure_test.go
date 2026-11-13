package banking

import (
	"reflect"
	"testing"

	"bankpiggy/api/internal/budget"
)

func TestSyncDoesNotReplaceConcurrentPlanEdit(t *testing.T) {
	s, p, identity, connection := connectedService(t)
	p.readStarted, p.readRelease = make(chan struct{}), make(chan struct{})
	t.Cleanup(func() {
		select {
		case <-p.readRelease:
		default:
			close(p.readRelease)
		}
	})
	execSQL(t, s, `UPDATE bank_connections SET next_sync_at=now()`)
	finished := make(chan error, 1)
	go func() { finished <- s.processOne(t.Context(), connection.ID) }()
	<-p.readStarted
	plan, err := s.db.Read(t.Context(), identity.HouseholdID)
	if err != nil {
		t.Fatal(err)
	}
	plan.Members[0].MonthlyTakeHomeCents++
	plan.Members[0].Accounts[0].BalanceCents++
	plan.Expenses[0].MonthlyAmountCents++
	accepted, err := s.db.Update(t.Context(), identity.HouseholdID, plan)
	if err != nil {
		t.Fatal(err)
	}
	close(p.readRelease)
	if err := <-finished; err != nil {
		t.Fatal(err)
	}
	actual, err := s.db.Read(t.Context(), identity.HouseholdID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(actual, accepted) {
		t.Fatal("late provider result changed the accepted concurrent plan")
	}
}

func TestPartialSnapshotFailureRollsBackAndRecovers(t *testing.T) {
	s, p, identity, connection := connectedService(t)
	before, err := s.List(t.Context(), identity)
	if err != nil {
		t.Fatal(err)
	}
	p.snapshot.Accounts[0].Account.CurrentBalanceCents = ptr(int64(4242))
	second := p.snapshot.Accounts[0]
	second.ProviderID = "provider-second-fails"
	p.snapshot.Accounts = append(p.snapshot.Accounts, second)
	execSQL(t, s, `CREATE FUNCTION reject_second() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.provider_id='provider-second-fails' THEN RAISE EXCEPTION 'simulated partial snapshot failure'; END IF; RETURN NEW; END $$`)
	execSQL(t, s, `CREATE TRIGGER reject_second BEFORE INSERT OR UPDATE ON bank_accounts FOR EACH ROW EXECUTE FUNCTION reject_second()`)
	execSQL(t, s, `UPDATE bank_connections SET state='delayed',next_sync_at=now()`)
	if err := s.processOne(t.Context(), connection.ID); err == nil {
		t.Fatal("partial snapshot reported success")
	}
	after, err := s.List(t.Context(), identity)
	if err != nil {
		t.Fatal(err)
	}
	assertSnapshotRollback(t, before.Connections[0], after.Connections[0])
	execSQL(t, s, `DROP TRIGGER reject_second ON bank_accounts`)
	execSQL(t, s, `UPDATE bank_connections SET lease_until=now()-interval '1 second',next_sync_at=now()`)
	restarted, err := New(testConfig(), s.db, p)
	if err != nil {
		t.Fatal(err)
	}
	if err := restarted.processOne(t.Context(), connection.ID); err != nil {
		t.Fatal("restart did not recover failed snapshot", err)
	}
	recovered, err := restarted.List(t.Context(), identity)
	if err != nil {
		t.Fatal(err)
	}
	if len(recovered.Connections[0].Accounts) != 2 {
		t.Fatal("retry lost or duplicated accounts")
	}
	var reads int
	p.mu.Lock()
	reads = p.reads
	p.mu.Unlock()
	if reads != 3 {
		t.Fatal("snapshot work repeated unexpectedly")
	}
}

func TestFreshLinkRetiresOldIssuedTokenWithoutReusingMode(t *testing.T) {
	s, _, identity := setupService(t)
	first := createTestLink(t, s, identity)
	second := createTestLink(t, s, identity)
	if first.AttemptID == second.AttemptID || first.ConnectionID == second.ConnectionID {
		t.Fatal("fresh POST reused a one-use Link token")
	}
	var code *string
	if err := s.db.Pool.QueryRow(t.Context(), `SELECT error_code FROM bank_connections WHERE id=$1`, first.ConnectionID).Scan(&code); err != nil || code == nil || *code != "removal_unverified" {
		t.Fatal("old issued consent was falsely revoked")
	}
	execSQL(t, s, `UPDATE bank_link_attempts SET token_cipher=NULL WHERE id=$1`, second.AttemptID)
	if _, err := s.CreateLink(t.Context(), identity, budget.LinkTokenRequest{Kind: ptr(budget.ConnectionKind("depository"))}); err != ErrConflict {
		t.Fatal("in-flight token creation did not serialize", err)
	}
}

func assertSnapshotRollback(t *testing.T, before, after budget.Connection) {
	t.Helper()
	if !reflect.DeepEqual(before.Accounts, after.Accounts) || *before.LastSyncedAt != *after.LastSyncedAt {
		t.Fatal("partial snapshot or timestamp committed")
	}
	if after.State != "delayed" {
		t.Fatal("failed snapshot changed connection state")
	}
}
