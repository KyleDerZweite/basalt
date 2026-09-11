// SPDX-License-Identifier: AGPL-3.0-or-later

package httpclient

import (
	"context"
	"testing"
)

func TestDomainRateLimiterCreatesIndependentBuckets(t *testing.T) {
	limiter := NewDomainRateLimiter(1000, 1)
	limiter.jitter = 0

	if err := limiter.Wait(context.Background(), "https://one.example/profile"); err != nil {
		t.Fatal(err)
	}
	if err := limiter.Wait(context.Background(), "https://two.example/profile"); err != nil {
		t.Fatal(err)
	}

	if len(limiter.limiters) != 2 {
		t.Fatalf("expected two domain buckets, got %d", len(limiter.limiters))
	}
	if limiter.limiters["one.example"] == limiter.limiters["two.example"] {
		t.Fatal("expected independent limiters for unrelated domains")
	}
}
