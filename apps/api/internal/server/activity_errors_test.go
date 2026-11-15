package server

import (
	"bytes"
	"fmt"
	"net/http/httptest"
	"testing"

	"bankpiggy/api/internal/banking"
	"bankpiggy/api/internal/store"
)

func TestActivityFailureUsesSafeMessages(t *testing.T) {
	cases := []struct {
		err    error
		status int
	}{
		{banking.ErrInvalid, 400},
		{banking.ErrNotFound, 404},
		{banking.ErrConflict, 409},
		{store.ErrConflict, 409},
		{banking.ErrUnavailable, 503},
		{banking.ErrLimit, 503},
	}
	for _, item := range cases {
		response := httptest.NewRecorder()
		activityFailure(response, fmt.Errorf("private financial payload: %w", item.err))
		requireStatus(t, response, item.status)
		if bytes.Contains(response.Body.Bytes(), []byte("private financial payload")) {
			t.Fatal("activity error exposed request data")
		}
	}
}
