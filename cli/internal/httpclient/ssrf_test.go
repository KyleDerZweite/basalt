// SPDX-License-Identifier: AGPL-3.0-or-later

package httpclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"
)

func TestPrivateNetworkBlockingRejectsLoopback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("request reached private server")
	}))
	defer server.Close()

	_, err := New(WithPrivateNetworkBlocking()).Do(context.Background(), server.URL, nil)
	if err == nil || !strings.Contains(err.Error(), "non-public") {
		t.Fatalf("expected non-public address error, got %v", err)
	}
}

func TestPublicURLValidationUsesDNSCache(t *testing.T) {
	cache := NewDNSCache(time.Minute)
	cache.entries["cached.invalid"] = &dnsEntry{
		addrs:    []string{"8.8.8.8"},
		resolved: time.Now(),
		ttl:      time.Minute,
	}
	target, err := neturl("https://cached.invalid/profile")
	if err != nil {
		t.Fatal(err)
	}
	if err := validatePublicURL(context.Background(), target, cache); err != nil {
		t.Fatalf("expected cached public address to validate: %v", err)
	}
}

func TestPublicAddressClassification(t *testing.T) {
	tests := []struct {
		address string
		public  bool
	}{
		{"8.8.8.8", true},
		{"2606:4700:4700::1111", true},
		{"127.0.0.1", false},
		{"10.0.0.1", false},
		{"169.254.169.254", false},
		{"100.64.0.1", false},
		{"192.0.2.1", false},
		{"198.18.0.1", false},
		{"198.51.100.1", false},
		{"203.0.113.1", false},
		{"2001:db8::1", false},
		{"::ffff:127.0.0.1", false},
	}
	for _, test := range tests {
		t.Run(test.address, func(t *testing.T) {
			address := netip.MustParseAddr(test.address)
			if got := isPublicAddress(address); got != test.public {
				t.Fatalf("isPublicAddress(%s) = %v, want %v", address, got, test.public)
			}
		})
	}
}
