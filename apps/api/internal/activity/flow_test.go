package activity

import (
	"testing"

	"bankpiggy/api/internal/budget"
)

func TestBankFlowBalancesNetRefundsAndIncomeReversals(t *testing.T) {
	flow, err := bankFlow(-200, -300)
	if err != nil {
		t.Fatal(err)
	}
	if flow.TotalCents != 300 || len(flow.Inflows) != 1 || len(flow.Outflows) != 2 {
		t.Fatalf("unexpected signed bank flow: %+v", flow)
	}
	if flow.Inflows[0].ID != "net-refunds" || flow.Outflows[0].ID != "income-reversals" || flow.Outflows[1].ID != "surplus" || flow.Outflows[1].AmountCents != 100 {
		t.Fatal("refunds, reversals, or gap were misclassified")
	}
	requireBalancedFlow(t, flow)
}

func TestBankFlowRetainsSurplusAndRejectsUnsafeGrossFlow(t *testing.T) {
	flow, err := bankFlow(1000, 200)
	if err != nil || len(flow.Outflows) != 2 || flow.Outflows[1].ID != "surplus" || flow.Outflows[1].AmountCents != 800 {
		t.Fatal("recorded surplus was lost", err)
	}
	requireBalancedFlow(t, flow)
	if _, err := bankFlow(maxSafeCents, -1); err != ErrInvalid {
		t.Fatal("unsafe gross inflow passed because its net amount was safe")
	}
}

func requireBalancedFlow(t *testing.T, flow budget.ActivityBankFlow) {
	t.Helper()
	in, inErr := sumFlow(flow.Inflows)
	out, outErr := sumFlow(flow.Outflows)
	if inErr != nil || outErr != nil || in != flow.TotalCents || out != flow.TotalCents {
		t.Fatal("bank flow did not conserve cents")
	}
	for _, row := range append(flow.Inflows, flow.Outflows...) {
		if row.AmountCents <= 0 {
			t.Fatal("flow contains a nonpositive edge")
		}
	}
}

func TestAggregateBankFlowHasOneSpendingEdgeAndBalancesGap(t *testing.T) {
	flow, err := bankFlow(100, 300)
	if err != nil || flow.TotalCents != 300 || len(flow.Inflows) != 2 || len(flow.Outflows) != 1 {
		t.Fatal("aggregate spending gap was lost", err)
	}
	if flow.Outflows[0].ID != "spending" || flow.Outflows[0].AmountCents != 300 || flow.Inflows[1].ID != "inflow-gap" || flow.Inflows[1].AmountCents != 200 {
		t.Fatal("aggregate spending was split into details or used an invalid balancing edge")
	}
	requireBalancedFlow(t, flow)
}
