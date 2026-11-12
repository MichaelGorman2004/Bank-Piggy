package server

import (
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"bankpiggy/api/internal/budget"
)

const todoTestPath = "/api/todos?date=2026-11-11"

func newTodoTestClient(t *testing.T) *testClient {
	t.Helper()
	client := newSoloTestClient(t)
	h := readPlan(t, client)
	h.Members[0].PaySchedule = &budget.PaySchedule{AnchorDate: "2026-11-01", SemiMonthlyDays: []budget.DayOfMonth{}}
	h.Goals = []budget.Goal{{ID: "goal", Name: "Savings", OwnerID: "member-1", MonthlyAmountCents: 10000}}
	requireStatus(t, client.request("PUT", "/api/household", planJSON(t, h), testOrigin, client.csrf), 200)
	return client
}

func todoResponse(t *testing.T, response *httptest.ResponseRecorder) budget.TodoList {
	t.Helper()
	requireStatus(t, response, 200)
	var list budget.TodoList
	if err := json.Unmarshal(response.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	return list
}

func todoBody(t *testing.T, input budget.TodoUpdate) []byte {
	t.Helper()
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestTodoCheckboxSharedAcrossSavedMembersWithoutChangingMoney(t *testing.T) {
	client := newTodoTestClient(t)
	h := readPlan(t, client)
	h.Members = append(h.Members, budget.Blank("One", "Two").Members[1])
	requireStatus(t, client.request("PUT", "/api/household", planJSON(t, h), testOrigin, client.csrf), 200)
	before := readPlan(t, client)
	list := todoResponse(t, client.request("GET", todoTestPath, nil, "", ""))
	if len(list.Tasks) != 1 || list.Tasks[0].Completed {
		t.Fatal("initial todo list is not a fresh current-period task")
	}
	second := &testClient{app: client.app}
	setTestMemberSession(t, second, "member-2")
	input := budget.TodoUpdate{Date: list.Date, PlanRevision: list.PlanRevision, Completed: true}
	updated := todoResponse(t, second.request("PUT", "/api/todos/"+list.Tasks[0].ID, todoBody(t, input), testOrigin, second.csrf))
	if !updated.Tasks[0].Completed || !todoResponse(t, client.request("GET", todoTestPath, nil, "", "")).Tasks[0].Completed {
		t.Fatal("household members did not share the checkbox state")
	}
	if !reflect.DeepEqual(readPlan(t, client), before) {
		t.Fatal("checkbox changed money, goal progress, or plan revision")
	}
}

func TestTodoRoutesRejectUnauthenticatedAndCrossOriginChanges(t *testing.T) {
	client := newTodoTestClient(t)
	list := todoResponse(t, client.request("GET", todoTestPath, nil, "", ""))
	input := budget.TodoUpdate{Date: list.Date, PlanRevision: list.PlanRevision, Completed: true}
	path := "/api/todos/" + list.Tasks[0].ID
	anonymous := &testClient{app: client.app}
	requireStatus(t, anonymous.request("GET", todoTestPath, nil, "", ""), 401)
	requireStatus(t, anonymous.request("PUT", path, todoBody(t, input), testOrigin, ""), 401)
	requireStatus(t, client.request("PUT", path, todoBody(t, input), testOrigin, ""), 403)
	requireStatus(t, client.request("PUT", path, todoBody(t, input), testOrigin+"/", client.csrf), 403)
	requireStatus(t, client.request("PUT", path, todoBody(t, input), "https://attacker.example", client.csrf), 403)
}

func TestTodoDatesAndStrictUpdateBodiesRejectInvalidInput(t *testing.T) {
	client := newTodoTestClient(t)
	for _, date := range []string{"", "2026-02-30", "2026-1-01", "1999-12-31", "2101-01-01"} {
		requireStatus(t, client.request("GET", "/api/todos?date="+date, nil, "", ""), 400)
	}
	for _, body := range []string{"", "null", `{}`, `{"date":"2026-11-11","planRevision":2,"completed":true}`, `{"date":"2026-11-11","planRevision":2,"expectedCompleted":false}`, `{"date":"2026-11-11","planRevision":2,"expectedCompleted":false,"completed":true,"memberId":"member-2"}`, strings.Repeat(" ", smallRequestLimit+1)} {
		requireStatus(t, client.request("PUT", "/api/todos/member-1:goal:goal:2026-11-01", []byte(body), testOrigin, client.csrf), 400)
	}
	input := budget.TodoUpdate{Date: "2026-11-11", PlanRevision: 0, Completed: true}
	requireStatus(t, client.request("PUT", "/api/todos/member-1:goal:goal:2026-11-01", todoBody(t, input), testOrigin, client.csrf), 400)
	input.PlanRevision = 2
	for _, id := range []string{strings.Repeat("a", maxTodoIDLength+1), "invalid%20task"} {
		requireStatus(t, client.request("PUT", "/api/todos/"+id, todoBody(t, input), testOrigin, client.csrf), 400)
	}
}

func TestTodoUpdateRejectsChangedPlanAndDeletedSource(t *testing.T) {
	client := newTodoTestClient(t)
	list := todoResponse(t, client.request("GET", todoTestPath, nil, "", ""))
	input := budget.TodoUpdate{Date: list.Date, PlanRevision: list.PlanRevision, Completed: true}
	h := readPlan(t, client)
	h.Goals = []budget.Goal{}
	requireStatus(t, client.request("PUT", "/api/household", planJSON(t, h), testOrigin, client.csrf), 200)
	path := "/api/todos/" + list.Tasks[0].ID
	requireStatus(t, client.request("PUT", path, todoBody(t, input), testOrigin, client.csrf), 409)
	input.PlanRevision++
	requireStatus(t, client.request("PUT", path, todoBody(t, input), testOrigin, client.csrf), 404)
	empty := todoResponse(t, client.request("GET", todoTestPath, nil, "", ""))
	if empty.Tasks == nil || len(empty.Tasks) != 0 {
		t.Fatal("deleted source remained in current tasks")
	}
}
