package banking

import (
	"testing"

	"bankpiggy/api/internal/budget"
)

func TestLegacySavedTransferDoesNotCreateNewFunding(t *testing.T) {
	s, identity, connection := serviceActivityFixture(t, true)
	second := addServiceInvestmentAccount(t, s, identity, connection)
	saveServiceActivity(t, s, identity, connection, "outgoing", connection.Accounts[0].ID, "2026-10-31", 10001, func(r *budget.ActivityRecord) { r.ActivitySubtype = "withdrawal" })
	saveServiceActivity(t, s, identity, connection, "incoming", second, "2026-11-01", -10001, func(r *budget.ActivityRecord) { r.ActivitySubtype = "contribution" })
	execSQL(t, s, `UPDATE bank_activity_records SET review_category='transfer',transfer_peer_id=CASE id WHEN 'outgoing' THEN 'incoming' ELSE 'outgoing' END,revision=2`)
	projection := serviceActivityMonth(t, s, identity, "household")
	if projection.Totals.NetInvestmentContributionsCents != 0 {
		t.Fatal("saved internal transfer changed household funding")
	}
	var pairs int
	if err := s.db.Pool.QueryRow(t.Context(), `SELECT count(*) FROM bank_activity_records WHERE transfer_peer_id IS NOT NULL AND review_category='transfer'`).Scan(&pairs); err != nil || pairs != 2 {
		t.Fatal("aggregate read changed saved transfer pairs", err)
	}
}
