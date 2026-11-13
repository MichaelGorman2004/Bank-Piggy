package banking

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	plaid "github.com/plaid/plaid-go/v50/legacy"
)

func TestDepositoryReadSuppliesBoundedCachedBalanceFreshness(t *testing.T) {
	requested := make(chan time.Time, 1)
	provider := sdkSimulation(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var request plaid.AccountsBalanceGetRequest
		if r.URL.Path != "/accounts/balance/get" || json.NewDecoder(r.Body).Decode(&request) != nil {
			t.Error("unexpected Balance request")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if request.Options == nil || request.Options.MinLastUpdatedDatetime == nil {
			writeProviderFixture(t, w, http.StatusBadRequest, `{"error_type":"INVALID_REQUEST","error_code":"INVALID_FIELD","error_message":"options.min_last_updated_datetime is required","display_message":null}`)
			return
		}
		requested <- *request.Options.MinLastUpdatedDatetime
		writeProviderFixture(t, w, http.StatusOK, `{"accounts":[{"account_id":"simulated-account","name":"Simulated","type":"depository","balances":{"current":0.29,"available":null,"iso_currency_code":"USD"}}],"item":{},"request_id":"simulated"}`)
	})
	before := time.Now().UTC().Add(-24 * time.Hour).Truncate(time.Second)
	snapshot, err := provider.Accounts(t.Context(), "simulated-token", "depository")
	after := time.Now().UTC().Add(-24 * time.Hour).Truncate(time.Second)
	if err != nil || len(snapshot.Accounts) != 1 || snapshot.Accounts[0].Account.CurrentBalanceCents == nil || *snapshot.Accounts[0].Account.CurrentBalanceCents != 29 {
		t.Fatal("required freshness option did not permit the complete exact snapshot", err)
	}
	assertCachedFreshnessBound(t, <-requested, before, after)
}

func assertCachedFreshnessBound(t *testing.T, oldest, before, after time.Time) {
	t.Helper()
	if oldest.Before(before) || oldest.After(after) || oldest.Nanosecond() != 0 || oldest.Location() != time.UTC {
		t.Fatal("cached balance bound did not preserve the independent 24-hour freshness limit")
	}
}

func TestUnavailableCachedFreshnessDoesNotFallBackToUnboundedRead(t *testing.T) {
	calls := 0
	provider := sdkSimulation(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/accounts/balance/get" {
			t.Error("cached freshness failure requested another endpoint")
		}
		w.Header().Set("Content-Type", "application/json")
		writeProviderFixture(t, w, http.StatusBadRequest, `{"error_type":"INVALID_RESULT","error_code":"LAST_UPDATED_DATETIME_OUT_OF_RANGE","error_message":"Simulated stale balance","display_message":null}`)
	})
	_, err := provider.Accounts(t.Context(), "simulated-token", "depository")
	var safe *ProviderError
	if !errors.As(err, &safe) || safe.Code != "provider_unavailable" || calls != 1 {
		t.Fatal("stale provider data was accepted or retried without the freshness bound", err)
	}
}

func TestInvestmentReadKeepsCachedAccountEndpoint(t *testing.T) {
	provider := sdkSimulation(t, func(w http.ResponseWriter, r *http.Request) {
		var request map[string]json.RawMessage
		if r.URL.Path != "/accounts/get" || json.NewDecoder(r.Body).Decode(&request) != nil || len(request["options"]) != 0 {
			t.Error("investment read changed its cached endpoint or options")
		}
		w.Header().Set("Content-Type", "application/json")
		writeProviderFixture(t, w, http.StatusOK, `{"accounts":[],"item":{},"request_id":"simulated"}`)
	})
	if _, err := provider.Accounts(t.Context(), "simulated-token", "investment"); err != nil {
		t.Fatal("cached investment request failed", err)
	}
}

func writeProviderFixture(t *testing.T, w http.ResponseWriter, status int, body string) {
	t.Helper()
	w.WriteHeader(status)
	if _, err := w.Write([]byte(body)); err != nil {
		t.Error(err)
	}
}
