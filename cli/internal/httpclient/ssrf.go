// SPDX-License-Identifier: AGPL-3.0-or-later

package httpclient

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

func validatePublicURL(ctx context.Context, target *url.URL, cache *DNSCache) error {
	if target.Scheme != "http" && target.Scheme != "https" {
		return fmt.Errorf("blocked URL scheme %q", target.Scheme)
	}
	host := strings.TrimSuffix(target.Hostname(), ".")
	if host == "" || strings.EqualFold(host, "localhost") {
		return fmt.Errorf("blocked non-public host %q", host)
	}
	if address, err := netip.ParseAddr(host); err == nil {
		if !isPublicAddress(address) {
			return fmt.Errorf("blocked non-public address %s", address)
		}
		return nil
	}
	addresses, err := lookupAddresses(ctx, host, cache)
	if err != nil {
		return fmt.Errorf("resolving %s: %w", host, err)
	}
	if len(addresses) == 0 {
		return fmt.Errorf("resolving %s: no addresses", host)
	}
	for _, rawIP := range addresses {
		address, err := netip.ParseAddr(rawIP)
		if err != nil || !isPublicAddress(address) {
			return fmt.Errorf("blocked non-public address %s for %s", rawIP, host)
		}
	}
	return nil
}

var reservedNetworks = []netip.Prefix{
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("2001:10::/28"),
	netip.MustParsePrefix("2001:20::/28"),
	netip.MustParsePrefix("2001:db8::/32"),
}

func isPublicAddress(address netip.Addr) bool {
	address = address.Unmap()
	if !address.IsValid() || !address.IsGlobalUnicast() || address.IsPrivate() || address.IsLoopback() || address.IsLinkLocalUnicast() {
		return false
	}
	for _, network := range reservedNetworks {
		if network.Contains(address) {
			return false
		}
	}
	return true
}

func lookupAddresses(ctx context.Context, host string, cache *DNSCache) ([]string, error) {
	if cache != nil {
		return cache.LookupAll(ctx, host)
	}
	addresses, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(addresses))
	for _, address := range addresses {
		out = append(out, address.String())
	}
	return out, nil
}

func guardedDialContext(timeout time.Duration, cache *DNSCache) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		rawAddresses, err := lookupAddresses(ctx, host, cache)
		if err != nil {
			return nil, err
		}
		for _, rawIP := range rawAddresses {
			address, err := netip.ParseAddr(rawIP)
			if err != nil || !isPublicAddress(address) {
				return nil, fmt.Errorf("blocked non-public address %s for %s", rawIP, host)
			}
		}
		if len(rawAddresses) == 0 {
			return nil, fmt.Errorf("resolving %s: no addresses", host)
		}
		dialer := &net.Dialer{Timeout: timeout, KeepAlive: 30 * time.Second}
		return dialer.DialContext(ctx, network, net.JoinHostPort(rawAddresses[0], port))
	}
}
