package activity

import (
	"testing"

	"bankpiggy/api/internal/budget"
)

func TestBankFlowBalancesRefundsReversalsAndSpendingGap(t *testing.T) {
	flow, err := bankFlow(-200, []budget.ActivityCategoryTotal{{Name: "Food", AmountCents: -300}, {Name: "Rent", AmountCents: 500}})
	if err != nil {
		t.Fatal(err)
	}
	if flow.TotalCents != 700 || len(flow.Inflows) != 2 || len(flow.Outflows) != 2 {
		t.Fatalf("unexpected signed bank flow: %+v", flow)
	}
	if flow.Inflows[0].ID != "refund-Food" || flow.Inflows[1].ID != "inflow-gap" || flow.Inflows[1].AmountCents != 400 || flow.Outflows[0].ID != "income-reversals" {
		t.Fatal("refunds, reversals, or gap were misclassified")
	}
	requireBalancedFlow(t, flow)
}

func TestBankFlowRetainsSurplusAndRejectsUnsafeGrossFlow(t *testing.T) {
	flow, err := bankFlow(1000, []budget.ActivityCategoryTotal{{Name: "Food", AmountCents: 200}})
	if err != nil || len(flow.Outflows) != 2 || flow.Outflows[1].ID != "surplus" || flow.Outflows[1].AmountCents != 800 {
		t.Fatal("recorded surplus was lost", err)
	}
	requireBalancedFlow(t, flow)
	if _, err := bankFlow(maxSafeCents, []budget.ActivityCategoryTotal{{Name: "Refund", AmountCents: -1}}); err != ErrInvalid {
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
