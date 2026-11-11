package budget_test

import (
	"testing"
	"time"
)

func testMonth(t *testing.T, value string) time.Time {
	t.Helper()
	month, err := time.Parse("2006-01", value)
	if err != nil {
		t.Fatal(err)
	}
	return month
}
