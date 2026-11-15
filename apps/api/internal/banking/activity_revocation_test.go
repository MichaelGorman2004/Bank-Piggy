package banking

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestAccountRevocationPurgesHistoryAndInvalidatesSurvivingPair(t *testing.T) {
	s, _, identity, connection := activityTestService(t)
	enableActivityForTest(t, s, identity, connection)
	execSQL(t, s, `INSERT INTO bank_accounts(id,household_id,connection_id,provider_id,snapshot) SELECT 'surviving-account',household_id,connection_id,'surviving-provider-account',snapshot FROM bank_accounts WHERE connection_id=$1 LIMIT 1`, connection.ID)
	out := simulatedActivity("revoked", time.Now().Format(time.DateOnly), 100)
	in := simulatedActivity("surviving", time.Now().Format(time.DateOnly), -100)
	in.ProviderAccountID = "surviving-provider-account"
	saveActivityBatchForTest(t, s, connection.ID, []ProviderActivityRecord{out, in}, nil)
	pairActivityForTest(t, s, "revoked", "surviving")
	execSQL(t, s, `INSERT INTO bank_activity_cancellations(connection_id,account_id,provider_id) VALUES($1,$2,'old-provider-id')`, connection.ID, connection.Accounts[0].ID)
	applyActivityWebhookForTest(t, s, `{"environment":"sandbox","item_id":"item-simulated","webhook_type":"ITEM","webhook_code":"USER_ACCOUNT_REVOKED","account_id":"provider-account-1"}`)
	assertActivityCheckpoint(t, s, connection.ID, "saved", 1)
	remaining := activityRecordStateForTest(t, s, "surviving")
	if remaining.peer != "" || remaining.review != "" {
		t.Fatal("account purge left a stale transfer review")
	}
	assertActivityTableCount(t, s, "bank_activity_cancellations", 0)
}

func TestRevocationAndDisconnectRejectLateActivityAndPurgeData(t *testing.T) {
	for _, mode := range []string{"revocation", "disconnect"} {
		t.Run(mode, func(t *testing.T) {
			s, p, identity, connection := activityTestService(t)
			enableActivityForTest(t, s, identity, connection)
			saveActivityBatchForTest(t, s, connection.ID, []ProviderActivityRecord{simulatedActivity("saved", time.Now().Format(time.DateOnly), 100)}, nil)
			execSQL(t, s, `UPDATE bank_activity_jobs SET next_sync_at=now() WHERE connection_id=$1`, connection.ID)
			started, release := make(chan struct{}), make(chan struct{})
			p.transactions = func(context.Context, string, string) (TransactionPage, error) {
				close(started)
				<-release
				return TransactionPage{Records: []ProviderActivityRecord{simulatedActivity("late", time.Now().Format(time.DateOnly), 29)}, NextCursor: "late", HistoryComplete: true}, nil
			}
			finished := make(chan error, 1)
			go func() { finished <- s.processActivity(t.Context(), connection.ID) }()
			<-started
			if mode == "revocation" {
				applyActivityWebhookForTest(t, s, `{"environment":"sandbox","item_id":"item-simulated","webhook_type":"ITEM","webhook_code":"USER_PERMISSION_REVOKED"}`)
			} else if err := s.Disconnect(t.Context(), identity, connection.ID); err != nil {
				t.Fatal(err)
			}
			close(release)
			if err := <-finished; !errors.Is(err, ErrConflict) {
				t.Fatal("late activity restored removed data", err)
			}
			assertActivityTableCount(t, s, "bank_activity_records", 0)
			assertActivityTableCount(t, s, "bank_activity_cancellations", 0)
			assertActivityRevokedJob(t, s, mode, connection.ID)
		})
	}
}

func assertActivityRevokedJob(t *testing.T, s *Service, mode, id string) {
	t.Helper()
	if mode == "disconnect" {
		assertActivityTableCount(t, s, "bank_activity_jobs", 0)
		return
	}
	var authorized bool
	if err := s.db.Pool.QueryRow(t.Context(), `SELECT authorized_at IS NOT NULL FROM bank_activity_jobs WHERE connection_id=$1`, id).Scan(&authorized); err != nil || authorized {
		t.Fatal("revocation retained history consent", err)
	}
}

func applyActivityWebhookForTest(t *testing.T, s *Service, body string) {
	t.Helper()
	signer, _ := webhookSigner(t)
	data := []byte(body)
	if err := s.ApplyWebhook(t.Context(), signedWebhook(t, signer, data, time.Now()), data); err != nil {
		t.Fatal(err)
	}
}

func assertActivityTableCount(t *testing.T, s *Service, table string, want int) {
	t.Helper()
	var count int
	if err := s.db.Pool.QueryRow(t.Context(), "SELECT count(*) FROM "+table).Scan(&count); err != nil || count != want {
		t.Fatal("activity purge failed", table, count, err)
	}
}
