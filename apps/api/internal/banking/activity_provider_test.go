package banking

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"testing"

	plaid "github.com/plaid/plaid-go/v50/legacy"
)

func TestActivityLinkRequestsTransactionsOnlyInExplicitUpdateMode(t *testing.T) {
	provider := sdkSimulation(t, func(w http.ResponseWriter, r *http.Request) {
		var request plaid.LinkTokenCreateRequest
		if r.URL.Path != "/link/token/create" || json.NewDecoder(r.Body).Decode(&request) != nil {
			t.Error("unexpected activity Link request")
		}
		if request.GetAccessToken() != "simulated-existing-token" || request.Update == nil || len(request.Products) != 0 {
			t.Error("activity Link created a new product Item")
		}
		if !reflect.DeepEqual(request.AdditionalConsentedProducts, []plaid.Products{plaid.PRODUCTS_TRANSACTIONS}) || request.Transactions != nil {
			t.Error("activity Link did not use consent-only update mode")
		}
		w.Header().Set("Content-Type", "application/json")
		writeProviderFixture(t, w, http.StatusOK, `{"link_token":"simulated-activity-link","expiration":"2030-01-01T00:00:00Z","request_id":"simulated"}`)
	})
	if _, _, err := provider.ActivityLink(t.Context(), "simulated-user", "simulated-existing-token"); err != nil {
		t.Fatal(err)
	}
}

func TestTransactionsProviderKeepsExactUnavailableAndRemovedFacts(t *testing.T) {
	provider := sdkSimulation(t, func(w http.ResponseWriter, r *http.Request) {
		var request plaid.TransactionsSyncRequest
		if r.URL.Path != "/transactions/sync" || json.NewDecoder(r.Body).Decode(&request) != nil || request.GetCursor() != "original" || request.GetCount() != activityPageSize {
			t.Error("transaction cursor or bounded count was lost")
		}
		w.Header().Set("Content-Type", "application/json")
		writeProviderFixture(t, w, http.StatusOK, transactionProviderFixture())
	})
	page, err := provider.TransactionsSync(t.Context(), "simulated-token", "original")
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Records) != 3 {
		t.Fatal("provider records were lost")
	}
	if !reflect.DeepEqual([]*int64{page.Records[0].Record.AmountCents, page.Records[1].Record.AmountCents, page.Records[2].Record.AmountCents}, []*int64{ptr(int64(29)), nil, nil}) {
		t.Fatal("provider decimal values were rounded or missing values became zero")
	}
	if !reflect.DeepEqual([]any{page.NextCursor, page.HasMore, page.HistoryComplete, page.Removed}, []any{"complete", false, true, []string{"removed"}}) {
		t.Fatal("cursor lifecycle fields were lost")
	}
	record := page.Records[0].Record
	if !reflect.DeepEqual([]string{record.PrimaryCategory, record.DetailedCategory, record.Counterparty, record.CategoryConfidence}, []string{"INCOME", "INCOME_WAGES", "Simulated employer", "VERY_HIGH"}) {
		t.Fatal("provider evidence was discarded")
	}
}

func transactionProviderFixture() string {
	return `{"transactions_update_status":"HISTORICAL_UPDATE_COMPLETE","accounts":[],"added":[
 {"transaction_id":"exact","account_id":"account","date":"2026-11-10","name":"PAYROLL","merchant_name":"Simulated employer","amount":0.29,"iso_currency_code":"USD","pending":false,"personal_finance_category":{"primary":"INCOME","detailed":"INCOME_WAGES","confidence_level":"VERY_HIGH"}},
 {"transaction_id":"fractional","account_id":"account","date":"2026-11-10","name":"Fractional","amount":0.001,"iso_currency_code":"USD","pending":false}],
 "modified":[{"transaction_id":"missing","account_id":"account","date":"2026-11-10","name":"Missing","amount":null,"iso_currency_code":null,"pending":true}],
 "removed":[{"transaction_id":"removed"}],"next_cursor":"complete","has_more":false,"request_id":"simulated"}`
}

func TestInvestmentProviderPreservesCancellationAndContributionEvidence(t *testing.T) {
	provider := sdkSimulation(t, func(w http.ResponseWriter, r *http.Request) {
		var request plaid.InvestmentsTransactionsGetRequest
		if json.NewDecoder(r.Body).Decode(&request) != nil {
			t.Error("invalid investment request")
		}
		if !reflect.DeepEqual([]any{r.URL.Path, request.Options.GetOffset(), request.Options.GetCount(), request.StartDate, request.EndDate}, []any{"/investments/transactions/get", int32(500), int32(activityPageSize), "2026-06-01", "2026-11-13"}) {
			t.Error("bounded investment window or offset was lost")
		}
		w.Header().Set("Content-Type", "application/json")
		writeProviderFixture(t, w, http.StatusOK, `{"accounts":[],"securities":[],"investment_transactions":[
 {"investment_transaction_id":"buy","account_id":"account","date":"2026-11-10","name":"Simulated contribution","amount":0.29,"iso_currency_code":"USD","type":"buy","subtype":"contribution","cancel_transaction_id":null},
 {"investment_transaction_id":"cancel","account_id":"account","date":"2026-11-11","name":"Simulated cancellation","amount":-0.29,"iso_currency_code":"USD","type":"cancel","subtype":"contribution","cancel_transaction_id":"older-id"}],"total_investment_transactions":502,"request_id":"simulated"}`)
	})
	page, err := provider.InvestmentTransactions(t.Context(), "simulated-token", "2026-06-01", "2026-11-13", 500)
	if err != nil || len(page.Records) != 2 || page.Total != 502 {
		t.Fatal("investment page was lost", err)
	}
	if !reflect.DeepEqual([]any{page.Records[0].Record.AmountCents, page.Records[0].Record.ActivityType, page.Records[0].Record.ActivitySubtype}, []any{ptr(int64(29)), "buy", "contribution"}) {
		t.Fatal("investment evidence or exact cents changed")
	}
	if page.Records[1].CancelledProviderID != "older-id" || !page.Records[1].Record.Cancelled {
		t.Fatal("cross-window cancellation reference was lost")
	}
}

func TestTransactionMutationProviderErrorIsSafeAndSpecific(t *testing.T) {
	provider := sdkSimulation(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		writeProviderFixture(t, w, http.StatusBadRequest, `{"error_type":"TRANSACTIONS_ERROR","error_code":"TRANSACTIONS_SYNC_MUTATION_DURING_PAGINATION","error_message":"simulated-secret","request_id":"simulated"}`)
	})
	_, err := provider.TransactionsSync(t.Context(), "simulated-token", "original")
	var safe *ProviderError
	if !errors.As(err, &safe) || safe.Code != "activity_pagination_changed" || err.Error() != "activity_pagination_changed" {
		t.Fatal("provider pagination error lost safe retry code", err)
	}
}
