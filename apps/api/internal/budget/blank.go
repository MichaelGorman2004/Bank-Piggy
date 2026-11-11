package budget

// Blank returns an empty plan with the fixed two-member identities.
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
