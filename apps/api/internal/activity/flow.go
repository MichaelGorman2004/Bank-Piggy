package activity

import "bankpiggy/api/internal/budget"

func bankFlow(income, spending int64) (budget.ActivityBankFlow, error) {
	result := budget.ActivityBankFlow{Inflows: []budget.ActivityFlowAmount{}, Outflows: []budget.ActivityFlowAmount{}}
	if income > 0 {
		result.Inflows = append(result.Inflows, budget.ActivityFlowAmount{ID: "income", Name: "Recorded income", AmountCents: income})
	} else if income < 0 {
		result.Outflows = append(result.Outflows, budget.ActivityFlowAmount{ID: "income-reversals", Name: "Income reversals", AmountCents: -income})
	}
	if spending > 0 {
		result.Outflows = append(result.Outflows, budget.ActivityFlowAmount{ID: "spending", Name: "Spending", AmountCents: spending})
	} else if spending < 0 {
		result.Inflows = append(result.Inflows, budget.ActivityFlowAmount{ID: "net-refunds", Name: "Net refunds", AmountCents: -spending})
	}
	return balanceBankFlow(result)
}

func balanceBankFlow(flow budget.ActivityBankFlow) (budget.ActivityBankFlow, error) {
	in, err := sumFlow(flow.Inflows)
	if err != nil {
		return flow, err
	}
	out, err := sumFlow(flow.Outflows)
	if err != nil {
		return flow, err
	}
	if in > out {
		flow.Outflows = append(flow.Outflows, budget.ActivityFlowAmount{ID: "surplus", Name: "Recorded surplus", AmountCents: in - out})
	} else if out > in {
		flow.Inflows = append(flow.Inflows, budget.ActivityFlowAmount{ID: "inflow-gap", Name: "Spending above recorded inflow", AmountCents: out - in})
	}
	flow.TotalCents = max(in, out)
	return flow, nil
}

func sumFlow(rows []budget.ActivityFlowAmount) (int64, error) {
	var total int64
	for _, row := range rows {
		var err error
		total, err = addCents(total, row.AmountCents)
		if err != nil {
			return 0, err
		}
	}
	return total, nil
}
