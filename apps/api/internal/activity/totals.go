package activity

import (
	"sort"
	"strings"

	"bankpiggy/api/internal/budget"
)

func totalRecords(result *budget.ActivityResponse, plan budget.Household) error {
	categories := map[string]int64{}
	contributions := map[string]budget.ActivityContribution{}
	expenseCategories := map[string]string{}
	for _, expense := range plan.Expenses {
		expenseCategories[expense.ID] = expense.Category
	}
	for _, record := range result.Records {
		if !countedRecord(record) {
			continue
		}
		if err := totalRecord(result, record, categories, contributions, expenseCategories); err != nil {
			return err
		}
	}
	for name, amount := range categories {
		result.Categories = append(result.Categories, budget.ActivityCategoryTotal{Name: name, AmountCents: amount})
	}
	for _, row := range contributions {
		result.Contributions = append(result.Contributions, row)
	}
	sort.Slice(result.Categories, func(i, j int) bool { return result.Categories[i].Name < result.Categories[j].Name })
	sort.Slice(result.Contributions, func(i, j int) bool { return result.Contributions[i].AccountID < result.Contributions[j].AccountID })
	var err error
	result.Totals.NetInvestmentContributionsCents, err = addCents(result.Totals.InvestmentDepositsCents, -result.Totals.InvestmentWithdrawalsCents)
	return err
}

func totalRecord(result *budget.ActivityResponse, record budget.ActivityRecord, categories map[string]int64, contributions map[string]budget.ActivityContribution, expenseCategories map[string]string) error {
	cents := *record.ScopeAmountCents
	if err := addRecordTotals(&result.Totals, record, cents); err != nil {
		return err
	}
	if record.Kind == "bank" && record.Category == "spending" {
		if err := addCategory(categories, record, cents, expenseCategories); err != nil {
			return err
		}
	}
	return addContribution(contributions, record, absolute(cents))
}

func countedRecord(record budget.ActivityRecord) bool {
	return record.ScopeAmountCents != nil && record.Category != "review" && record.Category != "excluded" && !record.Pending && !record.Cancelled
}

func addCategory(categories map[string]int64, record budget.ActivityRecord, cents int64, expenseCategories map[string]string) error {
	name := categoryName(record.PrimaryCategory)
	if record.ExpenseID != nil {
		if category, exists := expenseCategories[*record.ExpenseID]; exists {
			name = category
		}
	}
	value, err := addCents(categories[name], cents)
	if err == nil {
		categories[name] = value
	}
	return err
}

func addRecordTotals(totals *budget.ActivityTotals, record budget.ActivityRecord, cents int64) error {
	var target *int64
	if record.Kind == "bank" {
		switch record.Category {
		case "paycheck", "income":
			target, cents = &totals.IncomeCents, -cents
		case "spending":
			target = &totals.SpendingCents
		}
	} else {
		switch record.Category {
		case "contribution":
			target, cents = &totals.InvestmentDepositsCents, absolute(cents)
		case "withdrawal":
			target, cents = &totals.InvestmentWithdrawalsCents, absolute(cents)
		case "investment_income":
			target, cents = &totals.InvestmentIncomeCents, -cents
		case "spending":
			target = &totals.InvestmentFeesCents
		}
	}
	if target == nil {
		return nil
	}
	value, err := addCents(*target, cents)
	if err == nil {
		*target = value
	}
	return err
}

func addContribution(rows map[string]budget.ActivityContribution, record budget.ActivityRecord, cents int64) error {
	if record.Kind != "investment" || (record.Category != "contribution" && record.Category != "withdrawal") {
		return nil
	}
	row := rows[record.AccountID]
	row.AccountID, row.AccountName, row.Ownership = record.AccountID, record.AccountName, record.Ownership
	var err error
	if record.Category == "contribution" {
		row.DepositsCents, err = addCents(row.DepositsCents, cents)
	} else {
		row.WithdrawalsCents, err = addCents(row.WithdrawalsCents, cents)
	}
	if err != nil {
		return err
	}
	row.NetContributionsCents, err = addCents(row.DepositsCents, -row.WithdrawalsCents)
	rows[record.AccountID] = row
	return err
}

func addCents(first, second int64) (int64, error) {
	if second > maxSafeCents || second < -maxSafeCents || (second > 0 && first > maxSafeCents-second) || (second < 0 && first < -maxSafeCents-second) {
		return 0, ErrInvalid
	}
	return first + second, nil
}

func categoryName(primary string) string {
	if primary == "" {
		return "Reviewed spending"
	}
	words := strings.Fields(strings.ToLower(strings.ReplaceAll(primary, "_", " ")))
	if len(words) > 0 {
		words[0] = strings.ToUpper(words[0][:1]) + words[0][1:]
	}
	return strings.Join(words, " ")
}
