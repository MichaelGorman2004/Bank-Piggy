package server

import (
	"sync"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
)

func TestPublicAuthLimitResetsAtWindowBoundaryWithoutSleep(t *testing.T) {
	var limit publicAuthLimiter
	now := time.Date(2026, 11, 11, 0, 0, 0, 0, time.UTC)
	for index := 0; index < publicAuthRequestLimit; index++ {
		if allowed, _ := limit.allow(now); !allowed {
			t.Fatal("allowed request was rejected")
		}
	}
	if allowed, retry := limit.allow(now.Add(time.Second)); allowed || retry != 59 {
		t.Fatalf("limit did not reject with exact retry time: %v %d", allowed, retry)
	}
	if allowed, _ := limit.allow(now.Add(publicAuthWindow)); !allowed {
		t.Fatal("limit did not reset at the next window")
	}
}

func TestPublicAuthLimitHasOneSharedConcurrentBudget(t *testing.T) {
	var limit publicAuthLimiter
	now := time.Now()
	results := make(chan bool, 60)
	var ready sync.WaitGroup
	for index := 0; index < 60; index++ {
		ready.Go(func() {
			allowed, _ := limit.allow(now)
			results <- allowed
		})
	}
	ready.Wait()
	close(results)
	accepted := 0
	for allowed := range results {
		if allowed {
			accepted++
		}
	}
	if accepted != publicAuthRequestLimit {
		t.Fatalf("accepted %d requests, want %d", accepted, publicAuthRequestLimit)
	}
}

func TestInvitationPrepareAndGoogleStartShareRateLimit(t *testing.T) {
	client := newPrivateSoloClient(t)
	client.app.verifier = &oidc.IDTokenVerifier{}
	for index := 0; index < publicAuthRequestLimit; index++ {
		requireStatus(t, client.request("POST", "/api/household/invitation/prepare", []byte(`{}`), testOrigin, ""), 400)
	}
	for _, target := range []string{"/auth/google", "/api/household/invitation/prepare"} {
		method := "GET"
		if target != "/auth/google" {
			method = "POST"
		}
		response := client.request(method, target, []byte(`{}`), testOrigin, "")
		requireStatus(t, response, 429)
		if response.Header().Get("Retry-After") == "" {
			t.Fatal("rate limit omitted Retry-After")
		}
	}
}
