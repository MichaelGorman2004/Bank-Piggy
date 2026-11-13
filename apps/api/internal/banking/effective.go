package banking

import "bankpiggy/api/internal/budget"

// EffectiveBalances overlays only explicit mappings and never mutates the saved plan.
// A shared account appears once in household totals; personal shares reconcile cents.
func EffectiveBalances(plan budget.Household, connections []budget.Connection) budget.EffectiveBalances {
	result := budget.EffectiveBalances{Members: map[string]budget.BalanceTotals{}, Accounts: []budget.EffectiveAccount{}}
	hidden := map[string]bool{}
	for _, member := range plan.Members {
		result.Members[member.ID] = budget.BalanceTotals{}
	}
	for _, connection := range connections {
		if connection.State == "disconnecting" || connection.State == "disconnected" {
			continue
		}
		for _, account := range connection.Accounts {
			appendConnected(&result, account, hidden)
		}
	}
	for _, member := range plan.Members {
		for _, account := range member.Accounts {
			included := !hidden[member.ID+"/"+account.ID]
			row := budget.EffectiveAccount{ID: member.ID + "/" + account.ID, Name: account.Name, Ownership: budget.AccountOwnership(member.ID), Classification: budget.ConnectedAccountType(account.Type), Source: "manual", BalanceCents: ptr(account.BalanceCents), Currency: ptr("USD"), Included: included}
			result.Accounts = append(result.Accounts, row)
			if included {
				addEffective(&result, row)
			}
		}
	}
	return result
}

func appendConnected(result *budget.EffectiveBalances, account budget.ConnectedAccount, hidden map[string]bool) {
	mapping := account.Mapping
	included := supportedAccount(account) && account.Active && mapping.Enabled && account.Currency != nil && *account.Currency == "USD" && account.CurrentBalanceCents != nil && mapping.Classification != "unclassified"
	row := budget.EffectiveAccount{ID: account.ID, Name: account.Name, Ownership: mapping.Ownership, Classification: mapping.Classification, Source: "plaid", BalanceCents: account.CurrentBalanceCents, Currency: account.Currency, Included: included}
	result.Accounts = append(result.Accounts, row)
	if !included {
		return
	}
	if mapping.ManualMemberID != nil && mapping.ManualAccountID != nil {
		hidden[*mapping.ManualMemberID+"/"+*mapping.ManualAccountID] = true
	}
	addEffective(result, row)
}

func addEffective(result *budget.EffectiveBalances, row budget.EffectiveAccount) {
	cents := *row.BalanceCents
	result.Household = addTotal(result.Household, row.Classification, cents)
	if row.Ownership != "shared" {
		id := string(row.Ownership)
		result.Members[id] = addTotal(result.Members[id], row.Classification, cents)
		return
	}
	first := cents/int64(len(result.Members)) + cents%int64(len(result.Members))
	result.Members["member-1"] = addTotal(result.Members["member-1"], row.Classification, first)
	if len(result.Members) > 1 {
		result.Members["member-2"] = addTotal(result.Members["member-2"], row.Classification, cents-first)
	}
}

func addTotal(total budget.BalanceTotals, kind budget.ConnectedAccountType, cents int64) budget.BalanceTotals {
	switch kind {
	case "checking":
		total.CheckingCents += cents
	case "savings", "high_yield_savings":
		total.SavingsCents += cents
	case "hsa":
		total.HsaCents += cents
	default:
		total.InvestmentsCents += cents
	}
	return total
}
