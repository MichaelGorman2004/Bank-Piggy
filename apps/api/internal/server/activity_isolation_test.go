package server

import (
	"bytes"
	"testing"

	"bankpiggy/api/internal/budget"
)

func TestActivityHTTPHidesForeignHouseholdRecordsAndSources(t *testing.T) {
	client, _ := newBankHTTPClient(t)
	client.signIn(t)
	seedForeignHTTPActivity(t, client)
	response := client.request("GET", "/api/activity?month=2026-11", nil, "", "")
	requireStatus(t, response, 200)
	for _, private := range []string{"foreign-record", "Foreign private purchase", "foreign-connection"} {
		if bytes.Contains(response.Body.Bytes(), []byte(private)) {
			t.Fatal("activity read exposed another household")
		}
	}
	review := []byte(`{"expectedRevision":1,"category":null,"transferPeerId":null,"expectedPeerRevision":null,"expectedPlanRevision":1,"expenseId":null,"goalId":null}`)
	requireStatus(t, client.request("PUT", "/api/activity/records/foreign-record", review, testOrigin, client.csrf), 404)
	settings := []byte(`{"expectedRevision":0,"enabled":false,"accounts":[]}`)
	requireStatus(t, client.request("PUT", "/api/connections/foreign-connection/activity", settings, testOrigin, client.csrf), 404)
	paycheck := []byte(`{"expectedPlanRevision":1,"memberId":"member-1"}`)
	requireStatus(t, client.request("POST", "/api/activity/paychecks/foreign-record/apply?month=2026-11", paycheck, testOrigin, client.csrf), 404)
	foreign := bankMemberClient(t, client, "foreign-household", "member-1")
	requireStatus(t, foreign.request("GET", "/api/activity", nil, "", ""), 401)
	var revision int64
	if err := client.app.store.Pool.QueryRow(t.Context(), `SELECT revision FROM bank_activity_records WHERE id='foreign-record'`).Scan(&revision); err != nil {
		t.Fatal(err)
	}
	if revision != 1 {
		t.Fatal("foreign record changed")
	}
}

func seedForeignHTTPActivity(t *testing.T, client *testClient) {
	t.Helper()
	if err := client.app.store.Seed(t.Context(), "foreign-household", budget.Blank("Foreign")); err != nil {
		t.Fatal(err)
	}
	statements := []string{
		`INSERT INTO bank_connections(id,household_id,member_id,kind,state) VALUES('foreign-connection','foreign-household','member-1','depository','connected')`,
		`INSERT INTO bank_accounts(id,household_id,connection_id,provider_id,snapshot) VALUES('foreign-account','foreign-household','foreign-connection','foreign-provider-account','{"name":"Foreign private account","providerType":"depository","currency":"USD","active":true}')`,
		`INSERT INTO bank_activity_jobs(connection_id,household_id,enabled,authorized_at,accounts) VALUES('foreign-connection','foreign-household',true,now(),'[{"accountId":"foreign-account","enabled":true,"ownership":"member-1"}]')`,
		`INSERT INTO bank_activity_records(id,household_id,connection_id,account_id,provider_id,kind,posted_on,data) VALUES('foreign-record','foreign-household','foreign-connection','foreign-account','foreign-provider-record','bank','2026-11-01','{"name":"Foreign private purchase","kind":"bank","date":"2026-11-01","amountCents":1250,"currency":"USD"}')`,
	}
	for _, statement := range statements {
		if _, err := client.app.store.Pool.Exec(t.Context(), statement); err != nil {
			t.Fatal(err)
		}
	}
}
