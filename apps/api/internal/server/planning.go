package server

import (
	"errors"
	"net/http"
	"time"

	"bankpiggy/api/internal/budget"
)

func planningMonth(r *http.Request) (time.Time, error) {
	month := r.URL.Query().Get("month")
	if month == "" {
		month = time.Now().UTC().Format("2006-01")
	}
	if !budget.ValidMonth(month) || month >= "9999-01" {
		return time.Time{}, errors.New("choose a valid planning month from 0001-01 through 9998-12")
	}
	return time.Parse("2006-01", month)
}
