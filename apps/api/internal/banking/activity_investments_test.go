package banking

import (
	"context"
	"testing"
	"time"

	"bankpiggy/api/internal/budget"
)

func investmentActivityTestService(t *testing.T) (*Service, *activitySimulation, Identity, budget.Connection) {
	t.Helper()
	s, p, identity, connection := activityTestService(t)
	execSQL(t, s, `UPDATE bank_connections SET kind='investment' WHERE id=$1`, connection.ID)
	execSQL(t, s, `UPDATE bank_accounts SET snapshot=jsonb_set(snapshot,'{providerType}','"investment"') WHERE connection_id=$1`, connection.ID)
	connection.Kind = "investment"
	return s, p, identity, connection
}

func investmentActivity(id, date string, amount int64) ProviderActivityRecord {
	value := simulatedActivity(id, date, amount)
	value.Record.Kind = "investment"
	value.Record.ActivityType = "cash"
	value.Record.ActivitySubtype = "contribution"
	return value
}

func TestInvestmentOptInUsesExistingConsentAndStagesCompleteWindow(t *testing.T) {
	s, p, identity, connection := investmentActivityTestService(t)
	beforeLinks := p.links
	source := enableActivityForTest(t, s, identity, connection)
	if source.NeedsConsent || p.links != beforeLinks {
		t.Fatal("investment source added another Link product")
	}
	date := time.Now().Format(time.DateOnly)
	requested := []int{}
	p.investments = func(_ context.Context, _, _, _ string, offset int) (InvestmentPage, error) {
		requested = append(requested, offset)
		id := "first"
		if offset == 1 {
			id = "second"
		}
		return InvestmentPage{Total: 2, Records: []ProviderActivityRecord{investmentActivity(id, date, 29)}}, nil
	}
	if err := s.processActivity(t.Context(), connection.ID); err != nil {
		t.Fatal(err)
	}
	if len(requested) != 2 || requested[0] != 0 || requested[1] != 1 {
		t.Fatal("investment paging skipped or repeated an offset")
	}
	assertActivityCheckpoint(t, s, connection.ID, "", 2)
	if activitySourcesForTest(t, s, identity)[0].HistoryStart == nil {
		t.Fatal("complete investment window did not establish coverage")
	}
}

func TestInconsistentInvestmentPagesKeepPreviousWindow(t *testing.T) {
	for _, mode := range []string{"total", "duplicate", "empty"} {
		t.Run(mode, func(t *testing.T) {
			s, p, identity, connection := investmentActivityTestService(t)
			enableActivityForTest(t, s, identity, connection)
			date := time.Now().Format(time.DateOnly)
			saveActivityBatchForTest(t, s, connection.ID, []ProviderActivityRecord{investmentActivity("existing", date, 100)}, nil)
			execSQL(t, s, `UPDATE bank_activity_jobs SET next_sync_at=now() WHERE connection_id=$1`, connection.ID)
			p.investments = func(_ context.Context, _, _, _ string, offset int) (InvestmentPage, error) {
				return inconsistentInvestmentPage(mode, offset, date), nil
			}
			if err := s.processActivity(t.Context(), connection.ID); err == nil {
				t.Fatal("inconsistent window reported success")
			}
			assertActivityCheckpoint(t, s, connection.ID, "saved", 1)
			if activityRecordStateForTest(t, s, "existing").cancelled {
				t.Fatal("failed window removed previously saved data")
			}
		})
	}
}

func inconsistentInvestmentPage(mode string, offset int, date string) InvestmentPage {
	if offset == 0 {
		return InvestmentPage{Total: 2, Records: []ProviderActivityRecord{investmentActivity("staged", date, 29)}}
	}
	switch mode {
	case "total":
		return InvestmentPage{Total: 3, Records: []ProviderActivityRecord{investmentActivity("second", date, 29)}}
	case "duplicate":
		return InvestmentPage{Total: 2, Records: []ProviderActivityRecord{investmentActivity("staged", date, 29)}}
	default:
		return InvestmentPage{Total: 2}
	}
}

func TestInvestmentCancellationSurvivesWindowAndPayloadRetention(t *testing.T) {
	s, _, identity, connection := investmentActivityTestService(t)
	enableActivityForTest(t, s, identity, connection)
	old := investmentActivity("older-target", time.Now().AddDate(0, 0, -200).Format(time.DateOnly), 100)
	saveActivityBatchForTest(t, s, connection.ID, []ProviderActivityRecord{old}, nil)
	cancellation := investmentActivity("cancellation", time.Now().Format(time.DateOnly), -100)
	cancellation.CancelledProviderID = "older-target"
	cancellation.Record.ActivityType, cancellation.Record.Cancelled = "cancel", true
	saveActivityBatchForTest(t, s, connection.ID, []ProviderActivityRecord{cancellation}, nil)
	if !activityRecordStateForTest(t, s, "older-target").cancelled {
		t.Fatal("cancellation lost target outside replacement window")
	}
	execSQL(t, s, `UPDATE bank_activity_records SET posted_on=CURRENT_DATE-731 WHERE provider_id='cancellation'`)
	saveActivityBatchForTest(t, s, connection.ID, nil, nil)
	before := activityRecordStateForTest(t, s, "older-target")
	saveActivityBatchForTest(t, s, connection.ID, []ProviderActivityRecord{old}, nil)
	if saved := activityRecordStateForTest(t, s, "older-target"); saved != before || !saved.cancelled {
		t.Fatal("retained cancellation identifier did not preserve exact state")
	}
}

func TestInvestmentRawCancellationInvalidatesBothTransferReviews(t *testing.T) {
	s, _, identity, connection := investmentActivityTestService(t)
	enableActivityForTest(t, s, identity, connection)
	date := time.Now().Format(time.DateOnly)
	out, in := investmentActivity("out", date, 100), investmentActivity("in", date, -100)
	saveActivityBatchForTest(t, s, connection.ID, []ProviderActivityRecord{out, in}, nil)
	pairActivityForTest(t, s, "out", "in")
	out.Record.Cancelled, out.Record.ActivityType = true, "cancel"
	saveActivityBatchForTest(t, s, connection.ID, []ProviderActivityRecord{out, in}, nil)
	if saved := activityRecordStateForTest(t, s, "out"); !saved.cancelled || saved.peer != "" || saved.review != "" {
		t.Fatal("raw cancellation left the original review")
	}
	if saved := activityRecordStateForTest(t, s, "in"); saved.cancelled || saved.peer != "" || saved.review != "" {
		t.Fatal("raw cancellation left the paired review")
	}
}
