package budget

import "fmt"

// Blank returns an empty plan in fixed member order; supply one or two names.
func Blank(names ...string) Household {
	members := make([]Member, len(names))
	for index, name := range names {
		members[index] = Member{ID: fmt.Sprintf("member-%d", index+1), Name: name, PayFrequency: "monthly", Accounts: []AccountBalance{}}
	}
	return Household{
		Revision:           1,
		DefaultSplitMethod: "equal",
		Members:            members,
		Expenses:           []Expense{},
		Goals:              []Goal{},
	}
}

// HasMember reports whether a saved member ID belongs to the plan.
func HasMember(h Household, id string) bool {
	for _, member := range h.Members {
		if member.ID == id {
			return true
		}
	}
	return false
}
