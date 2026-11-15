package banking

import (
	"net/http"
	"strings"
	"testing"
)

func TestMalformedInvestmentResponseCannotCertifyAnEmptyWindow(t *testing.T) {
	for _, body := range []string{`{}`, `{"investment_transactions":[],"total_investment_transactions":null}`, `{"investment_transactions":null,"total_investment_transactions":0}`} {
		provider := sdkSimulation(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			writeProviderFixture(t, w, http.StatusOK, body)
		})
		if _, err := provider.InvestmentTransactions(t.Context(), "simulated-token", "2026-06-01", "2026-11-13", 0); err == nil {
			t.Fatal("missing history response fields became confirmed zero")
		}
	}
}

func TestNotReadyTransactionHistoryAllowsAnEmptyInitialCheckpoint(t *testing.T) {
	provider := sdkSimulation(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		writeProviderFixture(t, w, http.StatusOK, `{"added":[],"modified":[],"removed":[],"next_cursor":"","has_more":false,"transactions_update_status":"NOT_READY"}`)
	})
	page, err := provider.TransactionsSync(t.Context(), "simulated-token", "")
	if err != nil || page.HistoryComplete || page.NextCursor != "" {
		t.Fatal("initial not-ready history became an error or confirmed zero", err)
	}
}

func TestEmptyInitialCursorCannotCertifyHistoryOrCarryUpdates(t *testing.T) {
	for _, body := range []string{
		`{"added":[],"modified":[],"removed":[],"next_cursor":"","has_more":false,"transactions_update_status":"HISTORICAL_UPDATE_COMPLETE"}`,
		strings.ReplaceAll(transactionProviderFixture(), `"next_cursor":"complete"`, `"next_cursor":""`),
	} {
		provider := sdkSimulation(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			writeProviderFixture(t, w, http.StatusOK, body)
		})
		if _, err := provider.TransactionsSync(t.Context(), "simulated-token", ""); err == nil {
			t.Fatal("not-ready cursor certified history or carried updates")
		}
	}
}

func TestMalformedTransactionResponseCannotReplaceSavedCheckpoint(t *testing.T) {
	for _, body := range []string{`{}`, `{"added":[],"modified":[],"removed":[],"next_cursor":"","has_more":false}`, `{"added":null,"modified":[],"removed":[],"next_cursor":"next","has_more":false}`} {
		provider := sdkSimulation(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			writeProviderFixture(t, w, http.StatusOK, body)
		})
		if _, err := provider.TransactionsSync(t.Context(), "simulated-token", "saved"); err == nil {
			t.Fatal("malformed history replaced the saved checkpoint")
		}
	}
}

func TestCompleteEmptyProviderHistoryRemainsValid(t *testing.T) {
	provider := sdkSimulation(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		body := `{"investment_transactions":[],"total_investment_transactions":0}`
		if r.URL.Path == "/transactions/sync" {
			body = `{"added":[],"modified":[],"removed":[],"next_cursor":"next","has_more":false,"transactions_update_status":"HISTORICAL_UPDATE_COMPLETE"}`
		}
		writeProviderFixture(t, w, http.StatusOK, body)
	})
	if page, err := provider.TransactionsSync(t.Context(), "simulated-token", ""); err != nil || !page.HistoryComplete {
		t.Fatal("valid empty bank history was rejected", err)
	}
	if page, err := provider.InvestmentTransactions(t.Context(), "simulated-token", "2026-06-01", "2026-11-13", 0); err != nil || page.Total != 0 {
		t.Fatal("valid empty investment history was rejected", err)
	}
}
