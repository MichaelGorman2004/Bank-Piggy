package store_test

import (
	"testing"

	"bankpiggy/api/internal/testbudget"
	"bankpiggy/api/internal/testdb"
)

func TestActivityMigrationLeavesExistingItemsUnconsentedAndIsRepeatable(t *testing.T) {
	database := testdb.Open(t)
	if err := database.Seed(t.Context(), "household", testbudget.Sample()); err != nil {
		t.Fatal(err)
	}
	_, err := database.Pool.Exec(t.Context(), `DROP TABLE bank_activity_cancellations,bank_activity_records,bank_activity_jobs;
 ALTER TABLE bank_link_attempts DROP COLUMN purpose,DROP COLUMN activity_revision;
 INSERT INTO bank_connections(id,household_id,member_id,kind,state,token_cipher)
 VALUES('existing','household','member-1','depository','connected','simulated-encrypted-token');
 INSERT INTO bank_link_attempts(id,household_id,member_id,session_hash,connection_id,is_update,expires_at,state)
 VALUES('old-attempt','household','member-1','old-session','existing',false,now(),'completed')`)
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		migrated := reopenStore(t, database)
		var purpose string
		var jobs, records, cancellations int
		err := migrated.Pool.QueryRow(t.Context(), `SELECT purpose,
 (SELECT count(*) FROM bank_activity_jobs),(SELECT count(*) FROM bank_activity_records),
 (SELECT count(*) FROM bank_activity_cancellations) FROM bank_link_attempts WHERE id='old-attempt'`).Scan(&purpose, &jobs, &records, &cancellations)
		if err != nil || purpose != "balances" || jobs != 0 || records != 0 || cancellations != 0 {
			t.Fatal("migration activated history or changed a previous attempt", err)
		}
	}
}

func TestActivityHistoryAndCancellationIdentifiersFollowAccountDeletion(t *testing.T) {
	database := testdb.Open(t)
	if err := database.Seed(t.Context(), "household", testbudget.Sample()); err != nil {
		t.Fatal(err)
	}
	_, err := database.Pool.Exec(t.Context(), `INSERT INTO bank_connections(id,household_id,member_id,kind) VALUES('item','household','member-1','investment');
 INSERT INTO bank_accounts(id,household_id,connection_id,provider_id,snapshot) VALUES('account','household','item','provider-account','{}');
 INSERT INTO bank_activity_jobs(connection_id,household_id) VALUES('item','household');
 INSERT INTO bank_activity_records(id,household_id,connection_id,account_id,provider_id,kind,posted_on,data)
 VALUES('record','household','item','account','provider-record','investment',CURRENT_DATE,'{}');
 INSERT INTO bank_activity_cancellations(connection_id,account_id,provider_id) VALUES('item','account','cancelled-provider-record');
 DELETE FROM bank_accounts WHERE id='account'`)
	if err != nil {
		t.Fatal(err)
	}
	var records, cancellations int
	if err := database.Pool.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM bank_activity_records),(SELECT count(*) FROM bank_activity_cancellations)`).Scan(&records, &cancellations); err != nil || records != 0 || cancellations != 0 {
		t.Fatal("account revocation retained activity payload or identifiers", err)
	}
}

func TestActivityMatchMigrationPreservesExistingReviewsWithoutInventedMatches(t *testing.T) {
	database := testdb.Open(t)
	if err := database.Seed(t.Context(), "household", testbudget.Sample()); err != nil {
		t.Fatal(err)
	}
	_, err := database.Pool.Exec(t.Context(), `ALTER TABLE bank_activity_records DROP COLUMN expense_id,DROP COLUMN goal_id,DROP COLUMN allocation;
 INSERT INTO bank_connections(id,household_id,member_id,kind) VALUES('item','household','member-1','depository');
 INSERT INTO bank_accounts(id,household_id,connection_id,provider_id,snapshot) VALUES('account','household','item','provider-account','{}');
 INSERT INTO bank_activity_records(id,household_id,connection_id,account_id,provider_id,kind,posted_on,data,review_category,revision)
 VALUES('record','household','item','account','provider-record','bank',CURRENT_DATE,'{}','spending',7)`)
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		migrated := reopenStore(t, database)
		var category string
		var revision int64
		var unmatched bool
		err := migrated.Pool.QueryRow(t.Context(), `SELECT review_category,revision,expense_id IS NULL AND goal_id IS NULL AND allocation IS NULL FROM bank_activity_records WHERE id='record'`).Scan(&category, &revision, &unmatched)
		if err != nil || category != "spending" || revision != 7 || !unmatched {
			t.Fatal("match migration changed an existing review or invented a match", err)
		}
	}
}
