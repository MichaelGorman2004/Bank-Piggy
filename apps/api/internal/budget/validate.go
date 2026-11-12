package budget

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// MaxMoneyCents bounds each entered amount to one billion dollars.
const MaxMoneyCents int64 = 100_000_000_000

// MaxExpenses bounds plan size and calculation work.
const MaxExpenses = 100

// MaxGoals bounds active monthly allocations.
const MaxGoals = 30

// MaxMembers limits a plan to the first person and one optional partner.
const MaxMembers = 2

var validProvider = regexp.MustCompile(`^[a-z0-9-]{1,64}$`)

var validID = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

func validMoney(value int64) bool { return value >= 0 && value <= MaxMoneyCents }
func validText(value string, max int) bool {
	return utf8.ValidString(value) && strings.TrimSpace(value) != "" && utf8.RuneCountInString(value) <= max
}
func validMethod(value string) bool {
	return value == "equal" || value == "salary" || value == "custom"
}
func validOwner(value string, h Household) bool {
	return value == "shared" || HasMember(h, value)
}

func validateMember(member Member, index int) error {
	if member.ID != fmt.Sprintf("member-%d", index+1) || !validText(member.Name, 80) {
		return errors.New("member identity or name is invalid")
	}
	amounts := []int64{member.AnnualBaseSalaryCents, member.MonthlyTakeHomeCents, member.DebtCents}
	for _, amount := range amounts {
		if !validMoney(amount) {
			return errors.New("money must be whole cents between zero and 100000000000")
		}
	}
	switch member.PayFrequency {
	case "weekly", "biweekly", "semimonthly", "monthly", "irregular":
		return validateMemberDetails(member)
	default:
		return errors.New("invalid pay frequency")
	}
}

func validateMemberDetails(member Member) error {
	if err := validateAccounts(member.Accounts); err != nil {
		return err
	}
	if err := validatePayroll(member.Payroll); err != nil {
		return err
	}
	return validatePaySchedule(member)
}

func validateSplit(h Household, expense Expense) error {
	if !validMethod(expense.Split.Method) || expense.Split.CustomFirstShareBps < 0 || expense.Split.CustomFirstShareBps > 10000 {
		return errors.New("invalid expense split")
	}
	if len(h.Members) == MaxMembers && expense.OwnerID == "shared" && expense.Split.Method == "salary" && h.Members[0].AnnualBaseSalaryCents+h.Members[1].AnnualBaseSalaryCents == 0 {
		return errors.New("salary split needs a positive combined base salary")
	}
	payer := expense.PaidByMemberID
	if payer != "" && !HasMember(h, payer) {
		return errors.New("invalid expense payer")
	}
	return nil
}

func validateExpenses(h Household) error {
	seen := map[string]bool{}
	for _, expense := range h.Expenses {
		if !validID.MatchString(expense.ID) || seen[expense.ID] {
			return errors.New("invalid or duplicate expense ID")
		}
		if !validText(expense.Name, 100) || !validText(expense.Category, 60) {
			return errors.New("invalid expense name or category")
		}
		if !validMoney(expense.MonthlyAmountCents) || !validOwner(expense.OwnerID, h) {
			return errors.New("invalid expense amount or owner")
		}
		if err := validateSchedule(expense); err != nil {
			return err
		}
		if err := validateSplit(h, expense); err != nil {
			return err
		}
		seen[expense.ID] = true
	}
	return nil
}

func validateGoals(h Household) error {
	seen := map[string]bool{}
	for _, goal := range h.Goals {
		if !validID.MatchString(goal.ID) || seen[goal.ID] {
			return errors.New("invalid or duplicate goal ID")
		}
		if !validText(goal.Name, 100) || !validMoney(goal.MonthlyAmountCents) || !validOwner(goal.OwnerID, h) {
			return errors.New("invalid goal name, amount, or owner")
		}
		if err := validateGoalTarget(goal); err != nil {
			return err
		}
		seen[goal.ID] = true
	}
	return nil
}

func validateShape(h Household) error {
	if h.Revision < 1 || len(h.Members) < 1 || len(h.Members) > MaxMembers {
		return errors.New("invalid revision or member count")
	}
	if len(h.Expenses) > MaxExpenses || len(h.Goals) > MaxGoals {
		return errors.New("too many expenses or goals")
	}
	if h.Expenses == nil || h.Goals == nil {
		return errors.New("expenses and goals must be arrays")
	}
	return nil
}

// Validate rejects incomplete records and values that cannot be allocated safely.
func Validate(h Household) error {
	if err := validateShape(h); err != nil {
		return err
	}
	if !validMethod(h.DefaultSplitMethod) {
		return errors.New("invalid default split method")
	}
	for index, member := range h.Members {
		if err := validateMember(member, index); err != nil {
			return err
		}
	}
	if err := validateExpenses(h); err != nil {
		return err
	}
	return validateGoals(h)
}
