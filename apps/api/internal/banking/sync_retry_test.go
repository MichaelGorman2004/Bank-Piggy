package banking

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestSnapshotPersistenceFailuresStopProviderReadsAtRetryLimit(t *testing.T) {
	s, p, _, connection := connectedService(t)
	second := p.snapshot.Accounts[0]
	second.ProviderID = "persistently-rejected-account"
	p.snapshot.Accounts = append(p.snapshot.Accounts, second)
	execSQL(t, s, `CREATE FUNCTION reject_snapshot() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.provider_id='persistently-rejected-account' THEN RAISE EXCEPTION 'simulated snapshot failure'; END IF; RETURN NEW; END $$`)
	execSQL(t, s, `CREATE TRIGGER reject_snapshot BEFORE INSERT OR UPDATE ON bank_accounts FOR EACH ROW EXECUTE FUNCTION reject_snapshot()`)
	for attempt := 0; attempt < MaxFailures; attempt++ {
		execSQL(t, s, `UPDATE bank_connections SET next_sync_at=now() WHERE id=$1`, connection.ID)
		if err := s.processOne(t.Context(), connection.ID); err == nil {
			t.Fatal("failed snapshot persistence reported success")
		}
	}
	execSQL(t, s, `UPDATE bank_connections SET next_sync_at=now() WHERE id=$1`, connection.ID)
	if err := s.processOne(t.Context(), connection.ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("retry limit did not stop billed provider work", err)
	}
	var failures int
	if err := s.db.Pool.QueryRow(t.Context(), `SELECT failures FROM bank_connections WHERE id=$1`, connection.ID).Scan(&failures); err != nil {
		t.Fatal(err)
	}
	if failures != MaxFailures {
		t.Fatal("persistence failures were not durable")
	}
	p.mu.Lock()
	reads := p.reads
	p.mu.Unlock()
	if reads != 1+MaxFailures {
		t.Fatal("provider was read after the retry budget was exhausted")
	}
}
