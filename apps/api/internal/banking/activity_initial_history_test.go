package banking

import (
	"encoding/json"
	"net/http"
	"testing"

	plaid "github.com/plaid/plaid-go/v50/legacy"
)

func TestInitialTransactionSyncRequestsHistoryWithoutUpdateModeProductParameters(t *testing.T) {
	provider := sdkSimulation(t, func(w http.ResponseWriter, r *http.Request) {
		var request plaid.TransactionsSyncRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.GetCursor() == "" {
			if request.Options == nil || request.Options.GetDaysRequested() != activityHistoryDays {
				t.Error("first sync did not request bounded history")
			}
		} else if request.Options != nil {
			t.Error("incremental sync changed product initialization options")
		}
		w.Header().Set("Content-Type", "application/json")
		writeProviderFixture(t, w, http.StatusOK, transactionProviderFixture())
	})
	for _, cursor := range []string{"", "saved-cursor"} {
		if _, err := provider.TransactionsSync(t.Context(), "simulated-token", cursor); err != nil {
			t.Fatal(err)
		}
	}
}
