package budget

// Sample returns fictional data used only by explicit local demo mode.
func Sample() Household {
	expenses := append(sampleHousingExpenses(), sampleOtherExpenses()...)
	expenses = append(expenses, sampleSubscriptions()...)
	return Household{
		Revision:           1,
		DefaultSplitMethod: "salary",
		Members:            sampleMembers(),
		Expenses:           expenses,
		Goals: []Goal{
			{ID: "emergency", Name: "Emergency fund", MonthlyAmountCents: 100000, OwnerID: "shared", SavingsTarget: &SavingsTarget{TargetAmountCents: 4000000, SavedAmountCents: 2000000}},
			{ID: "retirement", Name: "Extra retirement", MonthlyAmountCents: 80000, OwnerID: "shared"},
			{ID: "holiday", Name: "Summer getaway", MonthlyAmountCents: 40000, OwnerID: "shared", Trip: sampleTripBudget()},
		},
	}
}

func sampleMembers() []Member {
	return []Member{
		{
			ID: "member-1", Name: "Alex", PayFrequency: "biweekly",
			AnnualBaseSalaryCents: 10000000,
			MonthlyTakeHomeCents:  620000,
			DebtCents:             640000,
			Accounts: []AccountBalance{
				{ID: "alex-hysa", Name: "Emergency savings", Type: "high_yield_savings", BalanceCents: 2000000},
				{ID: "alex-savings", Name: "Savings", Type: "savings", BalanceCents: 450000},
				{ID: "alex-401k", Name: "Workplace retirement", Type: "traditional_401k", BalanceCents: 5000000},
				{ID: "alex-roth-ira", Name: "Roth IRA", Type: "roth_ira", BalanceCents: 1000000},
				{ID: "alex-brokerage", Name: "Brokerage", Type: "brokerage", BalanceCents: 800000},
			},
		},
		{
			ID: "member-2", Name: "Sam", PayFrequency: "semimonthly",
			AnnualBaseSalaryCents: 5000000,
			MonthlyTakeHomeCents:  340000,
			DebtCents:             320000,
			Accounts: []AccountBalance{
				{ID: "sam-hysa", Name: "High-yield savings", Type: "high_yield_savings", BalanceCents: 1380000},
				{ID: "sam-roth-401k", Name: "Workplace Roth", Type: "roth_401k", BalanceCents: 2200000},
				{ID: "sam-ira", Name: "Traditional IRA", Type: "traditional_ira", BalanceCents: 500000},
				{ID: "sam-investments", Name: "Other investments", Type: "other_investment", BalanceCents: 250000},
			},
		},
	}
}

func sampleHousingExpenses() []Expense {
	return []Expense{
		{
			ID: "rent", Name: "Rent", Category: "Housing",
			MonthlyAmountCents: 300000,
			OwnerID:            "shared", PaidByMemberID: "member-1",
			Split: Split{Method: "salary"},
		},
		{
			ID: "groceries", Name: "Groceries", Category: "Food",
			MonthlyAmountCents: 65000,
			OwnerID:            "shared", PaidByMemberID: "member-2",
			Split: Split{Method: "equal"},
		},
		{
			ID: "utilities", Name: "Utilities & internet", Category: "Bills",
			MonthlyAmountCents: 29000,
			OwnerID:            "shared", PaidByMemberID: "member-1",
			Split: Split{Method: "equal"},
		},
	}
}

func sampleOtherExpenses() []Expense {
	return []Expense{
		{
			ID: "transport", Name: "Car payment", Category: "Transport",
			MonthlyAmountCents: 45000, OwnerID: "shared",
			Split: Split{Method: "equal"}, Kind: "loan",
			StartMonth: "2026-11", EndMonth: "2029-10",
		},
		{
			ID: "dining", Name: "Dining & fun", Category: "Lifestyle",
			MonthlyAmountCents: 50000, OwnerID: "shared",
			Split: Split{Method: "custom", CustomFirstShareBps: 6000},
		},
		{
			ID: "debt-alex", Name: "Minimum debt payment", Category: "Debt",
			MonthlyAmountCents: 25000,
			OwnerID:            "member-1", PaidByMemberID: "member-1",
			Split: Split{Method: "equal"},
		},
		{
			ID: "debt-sam", Name: "Minimum debt payment", Category: "Debt",
			MonthlyAmountCents: 15000,
			OwnerID:            "member-2", PaidByMemberID: "member-2",
			Split: Split{Method: "equal"},
		},
	}
}

func sampleSubscriptions() []Expense {
	return []Expense{
		{ID: "netflix", Name: "Netflix", Category: "Bills", MonthlyAmountCents: 1600,
			OwnerID: "shared", Split: Split{Method: "equal"}, Kind: "subscription", ProviderID: "netflix"},
		{ID: "amazon-prime", Name: "Amazon Prime", Category: "Bills", MonthlyAmountCents: 1400,
			OwnerID: "shared", Split: Split{Method: "equal"}, Kind: "subscription", ProviderID: "amazon-prime"},
	}
}

func sampleTripBudget() *TripBudget {
	return &TripBudget{
		SavedAmountCents: 120000, TargetMonth: "2027-06",
		Costs: []TripCost{
			{Category: "flights", AmountCents: 140000},
			{Category: "lodging", AmountCents: 180000},
			{Category: "food", AmountCents: 60000},
			{Category: "transport", AmountCents: 20000},
			{Category: "activities", AmountCents: 60000},
			{Category: "other", AmountCents: 20000},
		},
	}
}

// Blank returns an empty plan with the same fixed two-member identities.
func Blank(first, second string) Household {
	return Household{
		Revision:           1,
		DefaultSplitMethod: "equal",
		Members: []Member{
			{ID: "member-1", Name: first, PayFrequency: "monthly", Accounts: []AccountBalance{}},
			{ID: "member-2", Name: second, PayFrequency: "monthly", Accounts: []AccountBalance{}},
		},
		Expenses: []Expense{},
		Goals:    []Goal{},
	}
}
