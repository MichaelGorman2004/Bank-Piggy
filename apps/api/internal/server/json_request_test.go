package server

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"bankpiggy/api/internal/budget"
)

func TestSmallJSONRequestsRequireContentTypeAndAnObject(t *testing.T) {
	tests := []struct{ contentType, body string }{
		{"", `{}`},
		{"text/plain", `{}`},
		{"application/json", ""},
		{"application/json", "null"},
		{"application/json", " \n null \t"},
		{"application/json", `[]`},
		{"application/json", `{} {}`},
	}
	for _, test := range tests {
		request := httptest.NewRequest("POST", "/", bytes.NewBufferString(test.body))
		request.Header.Set("Content-Type", test.contentType)
		if err := decodeJSON(httptest.NewRecorder(), request, &struct{}{}); err == nil {
			t.Fatalf("accepted invalid JSON request %q %q", test.contentType, test.body)
		}
	}
	request := httptest.NewRequest("POST", "/", bytes.NewBufferString(`{}`))
	request.Header.Set("Content-Type", "application/json; charset=utf-8")
	if err := decodeJSON(httptest.NewRecorder(), request, &struct{}{}); err != nil {
		t.Fatal("rejected a strict empty JSON object", err)
	}
}

func TestSmallJSONRequestsRequireFalseBooleansAndRejectUnknownFields(t *testing.T) {
	for _, body := range []string{
		`{"date":"2026-11-11","planRevision":1,"completed":false}`,
		`{"date":"2026-11-11","planRevision":1,"expectedCompleted":false}`,
		`{"date":"2026-11-11","planRevision":1,"expectedCompleted":null,"completed":false}`,
		`{"date":"2026-11-11","planRevision":1,"expectedCompleted":false,"completed":false,"householdId":"other"}`,
	} {
		request := httptest.NewRequest("PUT", "/", bytes.NewBufferString(body))
		request.Header.Set("Content-Type", "application/json")
		if err := decodeJSON(httptest.NewRecorder(), request, &budget.TodoUpdate{}); err == nil {
			t.Fatal("accepted a missing, null, or foreign field")
		}
	}
}
