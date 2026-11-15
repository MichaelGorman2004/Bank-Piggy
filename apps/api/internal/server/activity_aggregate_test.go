package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"bankpiggy/api/internal/banking"
	"bankpiggy/api/internal/budget"
)

func TestActivityHTTPSerializesOnlyAggregateFinancialFacts(t *testing.T) {
	client, provider := newBankHTTPClient(t)
	client.signIn(t)
	config := banking.Config{Enabled: true, Environment: "sandbox", BankActivityApproved: true, ClientID: "simulation", Secret: "simulation", ActiveKeyID: "simulation", Keys: map[string][]byte{"simulation": make([]byte, 32)}}
	service, err := banking.New(config, client.app.store, provider)
	if err != nil {
		t.Fatal(err)
	}
	client.app.banking = service
	seedHTTPAggregateActivity(t, client)
	response := client.request("GET", "/api/activity?month="+time.Now().UTC().Format("2006-01"), nil, "", "")
	requireStatus(t, response, 200)
	var result budget.ActivityResponse
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Totals.SpendingCents != 1250 || len(result.Suggestions) != 1 {
		t.Fatal("populated history did not produce aggregate spending and a pay estimate")
	}
	requireAggregateActivityBody(t, response.Body.Bytes())
	body := []byte(`{"expectedRevision":1,"category":"income"}`)
	requireStatus(t, client.request("PUT", "/api/activity/records/private-purchase-record", body, testOrigin, client.csrf), 404)
	if provider.links.Load() != 0 || provider.exchanges.Load() != 0 {
		t.Fatal("aggregate read or removed review route reached the provider")
	}
}

func seedHTTPAggregateActivity(t *testing.T, client *testClient) {
	t.Helper()
	statements := []string{
		`INSERT INTO bank_connections(id,household_id,member_id,kind,state) VALUES('aggregate-connection','demo-household','member-1','depository','connected')`,
		`INSERT INTO bank_accounts(id,household_id,connection_id,provider_id,snapshot) VALUES('aggregate-account','demo-household','aggregate-connection','aggregate-provider-account','{"name":"Test account","providerType":"depository","currency":"USD","active":true}')`,
		`INSERT INTO bank_activity_jobs(connection_id,household_id,enabled,authorized_at,accounts) VALUES('aggregate-connection','demo-household',true,now(),'[{"accountId":"aggregate-account","enabled":true,"ownership":"member-1"}]')`,
	}
	for _, statement := range statements {
		if _, err := client.app.store.Pool.Exec(t.Context(), statement); err != nil {
			t.Fatal(err)
		}
	}
	today := time.Now().UTC()
	currency, purchaseCents, payCents := "USD", int64(1250), int64(-120000)
	purchase := budget.ActivityRecord{Name: "Private purchase name", Counterparty: "Private merchant", AmountCents: &purchaseCents, Currency: &currency, PrimaryCategory: "FOOD_AND_DRINK", CategoryConfidence: "VERY_HIGH"}
	insertHTTPAggregateRecord(t, client, "private-purchase-record", today, purchase)
	for index := range 4 {
		pay := budget.ActivityRecord{Name: "Test pay", Counterparty: "Test employer", AmountCents: &payCents, Currency: &currency, PrimaryCategory: "INCOME", DetailedCategory: "INCOME_WAGES", CategoryConfidence: "VERY_HIGH"}
		insertHTTPAggregateRecord(t, client, fmt.Sprintf("pay-%d", index), today.AddDate(0, 0, -7*index), pay)
	}
}

func insertHTTPAggregateRecord(t *testing.T, client *testClient, id string, date time.Time, record budget.ActivityRecord) {
	t.Helper()
	record.Kind, record.Date = "bank", budget.CalendarDate(date.Format(time.DateOnly))
	_, err := client.app.store.Pool.Exec(t.Context(), `INSERT INTO bank_activity_records(id,household_id,connection_id,account_id,provider_id,kind,posted_on,data) VALUES($1,'demo-household','aggregate-connection','aggregate-account',$1,'bank',$2,$3)`, id, record.Date, bankJSON(t, record))
	if err != nil {
		t.Fatal(err)
	}
}

func requireAggregateActivityBody(t *testing.T, body []byte) {
	t.Helper()
	for _, field := range []string{"records", "categories", "contributions", "reviewCount", "totalRecordCount", "nextCursor", "expenses", "investmentGoals", "unmatchedSpendingCents", "unmatchedSpendingCount", "unmatchedInvestmentNetCents", "deposits"} {
		if bytes.Contains(body, []byte(`"`+field+`":`)) {
			t.Fatalf("aggregate response exposed detail field %s", field)
		}
	}
	for _, detail := range []string{"private-purchase-record", "Private purchase name", "Private merchant", "FOOD_AND_DRINK"} {
		if bytes.Contains(body, []byte(detail)) {
			t.Fatal("aggregate response exposed an individual purchase")
		}
	}
}
