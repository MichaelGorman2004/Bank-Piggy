package budget

import (
	"fmt"
	"reflect"
	"slices"
	"testing"
)

func dueExpense(id string, amount int64, day DayOfMonth) Expense {
	return Expense{ID: id, Name: id, Category: "Bills", Kind: "bill", OwnerID: "member-1", MonthlyAmountCents: amount, Split: Split{Method: "equal"}, DueDay: &day}
}

func TestChecklistIncludesOnlyActiveManualBillsWithKnownDueDays(t *testing.T) {
	plan := todoPlan("monthly", "2026-01-01")
	plan.Goals = []Goal{}
	plan.Expenses = []Expense{dueExpense("bill", 100, 5), dueExpense("subscription", 100, 10), dueExpense("manual-sub", 100, 11), dueExpense("auto-bill", 100, 12), dueExpense("loan", 100, 15), dueExpense("legacy", 100, 20), dueExpense("ended", 100, 21), dueExpense("future", 100, 22)}
	plan.Expenses[1].Kind = "subscription"
	plan.Expenses[2].Kind, plan.Expenses[2].PaymentMode = "subscription", "manual"
	plan.Expenses[3].PaymentMode = "automatic"
	plan.Expenses[4].Kind = "loan"
	plan.Expenses[5].Kind = ""
	plan.Expenses[6].EndMonth = "2026-10"
	plan.Expenses[7].StartMonth = "2026-12"
	if err := Validate(plan); err != nil {
		t.Fatal(err)
	}
	result := Todos(plan, testDate(t, "2026-11-11"))
	want := []string{"bill", "manual-sub", "loan", "legacy"}
	got := []string{}
	for _, task := range result.Tasks {
		got = append(got, task.SourceID)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("incorrect manual payment tasks: %+v", result.Tasks)
	}
}

func TestChecklistReportsActiveManualBillsWithMissingDueDays(t *testing.T) {
	plan := todoPlan("monthly", "2026-01-01")
	plan.Goals = []Goal{}
	plan.Expenses = []Expense{dueExpense("z-missing", 100, 5), dueExpense("a-missing", 0, 10), dueExpense("auto", 100, 15), dueExpense("ended", 100, 20), dueExpense("future", 100, 20)}
	for index := range plan.Expenses {
		plan.Expenses[index].DueDay = nil
	}
	plan.Expenses[2].Kind = "subscription"
	plan.Expenses[3].EndMonth = "2026-10"
	plan.Expenses[4].StartMonth = "2026-12"
	result := Todos(plan, testDate(t, "2026-11-11"))
	if len(result.Tasks) != 0 || !reflect.DeepEqual(result.UnscheduledExpenseIds, []string{"a-missing", "z-missing"}) {
		t.Fatalf("unknown due dates were scheduled or omitted from the warning: %+v", result)
	}
}

func TestBillOnNextPaydayBelongsOnlyToTheNextPeriod(t *testing.T) {
	plan := todoPlan("biweekly", "2026-11-06")
	plan.Goals = []Goal{}
	plan.Expenses = []Expense{dueExpense("bill", 101, 20)}
	if tasks := Todos(plan, testDate(t, "2026-11-19")).Tasks; len(tasks) != 0 {
		t.Fatal("bill on next payday was assigned to the previous period", tasks)
	}
	result := Todos(plan, testDate(t, "2026-11-20"))
	if len(result.Tasks) != 1 || result.Tasks[0].DueDate != "2026-11-20" || result.Tasks[0].AmountCents != 101 {
		t.Fatalf("bill was not assigned to its exact pay-period boundary: %+v", result.Tasks)
	}
}

func TestMonthlyCycleKeepsBothClampedDueMonthsAndStableBillIdentity(t *testing.T) {
	plan := todoPlan("monthly", "2026-01-31")
	plan.Goals = []Goal{}
	plan.Expenses = []Expense{dueExpense("bill", 101, 30)}
	result := Todos(plan, testDate(t, "2026-03-11"))
	if len(result.Tasks) != 2 || result.Tasks[0].DueDate != "2026-02-28" || result.Tasks[1].DueDate != "2026-03-30" {
		t.Fatalf("long cycle did not retain each due month: %+v", result.Tasks)
	}
	if result.Tasks[0].ID != "member-1:expense:bill:2026-02-28" || result.Tasks[0].ID == result.Tasks[1].ID {
		t.Fatal("bill identities are not stable and distinct by due date", result.Tasks)
	}
	plan.Members[0].PayFrequency = "weekly"
	plan.Members[0].PaySchedule.AnchorDate = "2026-02-26"
	moved := Todos(plan, testDate(t, "2026-02-28"))
	if len(moved.Tasks) != 1 || moved.Tasks[0].ID != result.Tasks[0].ID {
		t.Fatal("a schedule change changed the same bill identity", moved.Tasks)
	}
}

func sharedTodoPlan() Household {
	plan := todoPlan("monthly", "2026-01-01")
	second := Blank("First", "Second").Members[1]
	second.PaySchedule = &PaySchedule{AnchorDate: "2026-01-01", SemiMonthlyDays: []DayOfMonth{}}
	plan.Members = append(plan.Members, second)
	plan.Goals = []Goal{}
	plan.Expenses = []Expense{dueExpense("shared", 101, 15)}
	plan.Expenses[0].OwnerID = "shared"
	return plan
}

func TestSharedBillWithRecordedPayerCreatesOneFullPaymentTask(t *testing.T) {
	plan := sharedTodoPlan()
	plan.Expenses[0].PaidByMemberID = "member-2"
	result := Todos(plan, testDate(t, "2026-11-11"))
	if len(result.Tasks) != 1 || result.Tasks[0].MemberID != "member-2" || result.Tasks[0].AmountCents != 101 || result.Tasks[0].Name != "shared" {
		t.Fatalf("recorded payer created duplicate or partial bills: %+v", result.Tasks)
	}
	summary := Response(plan, testMonth(t, "2026-11")).Summary
	if summary.Household.MonthlyExpensesCents != 101 || summary.ExpenseShares[0].FirstMemberCents != 51 || summary.ExpenseShares[0].SecondMemberCents != 50 {
		t.Fatal("payment responsibility changed the agreed expense split", summary)
	}
}

func TestSharedBillWithoutPayerCreatesOnlyEachMembersAllocatedShare(t *testing.T) {
	for _, test := range []struct {
		method       string
		basis, first int64
	}{{"equal", 0, 51}, {"salary", 0, 67}, {"custom", 2500, 25}, {"custom", 10000, 101}} {
		plan := sharedTodoPlan()
		plan.Members[0].AnnualBaseSalaryCents, plan.Members[1].AnnualBaseSalaryCents = 200, 100
		plan.Expenses[0].Split = Split{Method: test.method, CustomFirstShareBps: test.basis}
		var total int64
		for _, task := range Todos(plan, testDate(t, "2026-11-11")).Tasks {
			if task.Name != "shared (your share)" || (task.MemberID == "member-1" && task.AmountCents != test.first) {
				t.Fatalf("shared bill did not identify the allocated share: %+v", task)
			}
			total += task.AmountCents
		}
		if total != 101 {
			t.Fatalf("%s reminders lost or duplicated bill cents: %d", test.method, total)
		}
	}
}

func TestPersonalBillPayerDoesNotChangeBudgetOwner(t *testing.T) {
	plan := sharedTodoPlan()
	plan.Expenses[0].OwnerID, plan.Expenses[0].PaidByMemberID = "member-1", "member-2"
	result := Todos(plan, testDate(t, "2026-11-11"))
	if len(result.Tasks) != 1 || result.Tasks[0].MemberID != "member-2" || result.Tasks[0].AmountCents != 101 {
		t.Fatal("personal bill was not assigned to its recorded payer", result.Tasks)
	}
	summary := Response(plan, testMonth(t, "2026-11")).Summary
	if summary.Members["member-1"].MonthlyExpensesCents != 101 || summary.Members["member-2"].MonthlyExpensesCents != 0 {
		t.Fatal("personal bill payer changed expense ownership", summary)
	}
}

func TestChecklistKeepsMaximumPlanAndDeterministicTaskOrder(t *testing.T) {
	plan := sharedTodoPlan()
	for index := range plan.Members {
		plan.Members[index].PaySchedule.AnchorDate = "2026-01-31"
	}
	plan.Expenses = []Expense{}
	for index := range MaxExpenses {
		expense := dueExpense(fmt.Sprintf("bill-%03d", index), 100, 30)
		expense.OwnerID = "shared"
		plan.Expenses = append(plan.Expenses, expense)
	}
	for index := range MaxGoals {
		plan.Goals = append(plan.Goals, Goal{ID: fmt.Sprintf("goal-%03d", index), Name: "Save", OwnerID: "shared", MonthlyAmountCents: 100})
	}
	if err := Validate(plan); err != nil {
		t.Fatal(err)
	}
	result := Todos(plan, testDate(t, "2026-03-11"))
	if len(result.Tasks) != 460 {
		t.Fatalf("maximum checklist lost tasks: %d", len(result.Tasks))
	}
	keys, seen := []string{}, map[string]bool{}
	for _, task := range result.Tasks {
		keys = append(keys, string(task.DueDate)+":"+task.MemberID+":"+task.SourceID+":"+task.Kind)
		if seen[task.ID] {
			t.Fatal("checklist contains duplicate task identities")
		}
		seen[task.ID] = true
	}
	if !slices.IsSorted(keys) {
		t.Fatal("checklist was not sorted by due date, member, and source")
	}
}

func TestManualPaymentValidationRejectsInvalidDueDaysAndModes(t *testing.T) {
	for _, test := range []struct {
		day  DayOfMonth
		mode string
	}{{0, "manual"}, {-1, "manual"}, {32, "manual"}, {15, "other"}} {
		plan := todoPlan("monthly", "2026-01-01")
		plan.Expenses = []Expense{dueExpense("bill", 101, test.day)}
		plan.Expenses[0].PaymentMode = test.mode
		if Validate(plan) == nil {
			t.Fatalf("invalid payment metadata accepted: %+v", test)
		}
	}
}
