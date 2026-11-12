package server

import (
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const publicAuthRequestLimit = 30
const publicAuthWindow = time.Minute

// publicAuthLimiter bounds anonymous sign-in work without storing client addresses.
type publicAuthLimiter struct {
	mu      sync.Mutex
	resetAt time.Time
	used    int
}

func (limit *publicAuthLimiter) allow(now time.Time) (bool, int) {
	limit.mu.Lock()
	defer limit.mu.Unlock()
	if limit.resetAt.IsZero() || !now.Before(limit.resetAt) {
		limit.resetAt = now.Add(publicAuthWindow)
		limit.used = 0
	}
	if limit.used >= publicAuthRequestLimit {
		return false, int(math.Ceil(limit.resetAt.Sub(now).Seconds()))
	}
	limit.used++
	return true, 0
}

func (s *Server) allowPublicAuth(w http.ResponseWriter) bool {
	allowed, retryAfter := s.publicAuth.allow(time.Now())
	if !allowed {
		w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
		fail(w, http.StatusTooManyRequests, "sign_in_rate_limit", "Too many sign-in requests. Wait a minute and try again.")
	}
	return allowed
}
