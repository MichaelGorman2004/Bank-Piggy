package banking

import (
	"errors"
	"testing"

	"bankpiggy/api/internal/budget"
)

func TestActivityReviewServicePairsAndUnpairsAtomically(t *testing.T) {
	s, identity, connection := serviceActivityFixture(t, true)
	second := addServiceInvestmentAccount(t, s, identity, connection)
	saveServiceActivity(t, s, identity, connection, "outgoing", connection.Accounts[0].ID, "2026-11-03", 10001, func(r *budget.ActivityRecord) { r.ActivitySubtype = "withdrawal" })
	saveServiceActivity(t, s, identity, connection, "incoming", second, "2026-11-04", -10001, func(r *budget.ActivityRecord) { r.ActivitySubtype = "contribution" })
	request := budget.ActivityReviewRequest{ExpectedPlanRevision: 1, ExpectedRevision: 1, Category: ptr(budget.ActivityCategory("transfer")), TransferPeerID: ptr("incoming"), ExpectedPeerRevision: ptr(int64(1))}
	got, err := s.ReviewActivity(t.Context(), identity, "outgoing", request)
	if err != nil {
		t.Fatal(err)
	}
	requireServiceReviewRow(t, s, "outgoing", "incoming", 2)
	if got.Category != "transfer" {
		t.Fatal("review did not pair the outgoing record")
	}
	requireServiceReviewRow(t, s, "incoming", "outgoing", 2)
	projection := serviceActivityMonth(t, s, identity, "household", "")
	if projection.Totals.NetInvestmentContributionsCents != 0 || projection.ReviewCount != 0 {
		t.Fatal("confirmed internal transfer changed household funding")
	}
	if _, err := s.ReviewActivity(t.Context(), identity, "outgoing", request); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale review: %v", err)
	}
	got, err = s.ReviewActivity(t.Context(), identity, "outgoing", budget.ActivityReviewRequest{ExpectedPlanRevision: 1, ExpectedRevision: 2})
	if err != nil {
		t.Fatal(err)
	}
	requireServiceReviewRow(t, s, "outgoing", "", 3)
	if got.IsReviewed {
		t.Fatal("clearing review did not clear the pair")
	}
	requireServiceReviewRow(t, s, "incoming", "", 3)
}

func requireServiceReviewRow(t *testing.T, s *Service, id, peer string, revision int64) {
	t.Helper()
	var gotPeer *string
	var gotRevision int64
	if err := s.db.Pool.QueryRow(t.Context(), `SELECT transfer_peer_id,revision FROM bank_activity_records WHERE id=$1`, id).Scan(&gotPeer, &gotRevision); err != nil {
		t.Fatal(err)
	}
	if gotRevision != revision || (peer == "" && gotPeer != nil) || (peer != "" && (gotPeer == nil || *gotPeer != peer)) {
		t.Fatal("partner update was not atomic")
	}
}

func TestActivityReviewServiceRejectsStalePairWithoutPartialWrite(t *testing.T) {
	s, identity, connection := serviceActivityFixture(t, true)
	second := addServiceInvestmentAccount(t, s, identity, connection)
	saveServiceActivity(t, s, identity, connection, "outgoing", connection.Accounts[0].ID, "2026-11-03", 10001, func(r *budget.ActivityRecord) { r.ActivitySubtype = "withdrawal" })
	saveServiceActivity(t, s, identity, connection, "incoming", second, "2026-11-04", -10001, func(r *budget.ActivityRecord) { r.ActivitySubtype = "contribution" })
	execSQL(t, s, `UPDATE bank_activity_records SET revision=2 WHERE id='incoming'`)
	request := budget.ActivityReviewRequest{ExpectedPlanRevision: 1, ExpectedRevision: 1, Category: ptr(budget.ActivityCategory("transfer")), TransferPeerID: ptr("incoming"), ExpectedPeerRevision: ptr(int64(1))}
	if _, err := s.ReviewActivity(t.Context(), identity, "outgoing", request); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale partner: %v", err)
	}
	requireServiceReviewRow(t, s, "outgoing", "", 1)
	requireServiceReviewRow(t, s, "incoming", "", 2)
	request.ExpectedPeerRevision = ptr(int64(2))
	request.TransferPeerID = ptr("foreign-record")
	if _, err := s.ReviewActivity(t.Context(), identity, "outgoing", request); !errors.Is(err, ErrNotFound) {
		t.Fatalf("foreign partner: %v", err)
	}
	requireServiceReviewRow(t, s, "outgoing", "", 1)
}

func TestActivityReviewServiceRejectsUnsupportedClassification(t *testing.T) {
	cases := []struct {
		name, category string
		modify         func(*budget.ActivityRecord)
	}{
		{"unknown category", "made-up", nil},
		{"bank trade", "trade", nil},
		{"pending", "spending", func(r *budget.ActivityRecord) { r.Pending = true }},
		{"non USD", "spending", func(r *budget.ActivityRecord) { r.Currency = ptr("EUR") }},
		{"cancelled", "spending", func(r *budget.ActivityRecord) { r.Cancelled = true }},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			s, identity, connection := serviceActivityFixture(t, false)
			saveServiceActivity(t, s, identity, connection, "record", connection.Accounts[0].ID, "2026-11-03", 101, item.modify)
			if item.name == "cancelled" {
				execSQL(t, s, `UPDATE bank_activity_records SET cancelled=true WHERE id='record'`)
			}
			request := budget.ActivityReviewRequest{ExpectedPlanRevision: 1, ExpectedRevision: 1, Category: ptr(budget.ActivityCategory(item.category))}
			if _, err := s.ReviewActivity(t.Context(), identity, "record", request); !errors.Is(err, ErrInvalid) {
				t.Fatalf("invalid review: %v", err)
			}
			requireServiceReviewRow(t, s, "record", "", 1)
		})
	}
}

func TestActivityReviewServiceRequiresCurrentMember(t *testing.T) {
	s, identity, connection := serviceActivityFixture(t, false)
	saveServiceActivity(t, s, identity, connection, "record", connection.Accounts[0].ID, "2026-11-03", 101, nil)
	identity.MemberID = "absent-member"
	if _, err := s.ReviewActivity(t.Context(), identity, "record", budget.ActivityReviewRequest{ExpectedPlanRevision: 1, ExpectedRevision: 1}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("absent member: %v", err)
	}
	requireServiceReviewRow(t, s, "record", "", 1)
}

func TestActivityReviewServiceConcurrentRevisionHasOneWinner(t *testing.T) {
	s, identity, connection := serviceActivityFixture(t, false)
	saveServiceActivity(t, s, identity, connection, "record", connection.Accounts[0].ID, "2026-11-03", 101, nil)
	start := make(chan struct{})
	results := make(chan error, 2)
	for _, category := range []budget.ActivityCategory{"spending", "income"} {
		go func() {
			<-start
			_, err := s.ReviewActivity(t.Context(), identity, "record", budget.ActivityReviewRequest{ExpectedPlanRevision: 1, ExpectedRevision: 1, Category: &category})
			results <- err
		}()
	}
	close(start)
	successes, conflicts := 0, 0
	for range 2 {
		switch err := <-results; {
		case err == nil:
			successes++
		case errors.Is(err, ErrConflict):
			conflicts++
		default:
			t.Fatalf("concurrent review: %v", err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatal("concurrent reviews did not preserve the expected revision")
	}
	requireServiceReviewRow(t, s, "record", "", 2)
}
