// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package validators

import (
	"errors"
	"net"
	"net/url"
	"strings"
)

// reservedCIDRs lists private, loopback, link-local and otherwise reserved IP ranges.
// Kept in sync with checks/funcs.go privateAddressBlocks (full list:
// https://en.wikipedia.org/wiki/Reserved_IP_addresses).
var reservedCIDRs = []string{
	"0.0.0.0/8", // "this host" / current network; 0.0.0.0 routes to loopback on many systems
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"100.64.0.0/10",
	"192.0.0.0/24",
	"198.18.0.0/15",
	"169.254.0.0/16",
	"224.0.0.0/24",
	"127.0.0.0/8",
	"255.255.255.255/32", // limited broadcast

	"::/128", // IPv6 unspecified; routes to loopback
	"fc00::/7",
	"fe80::/10",
	"::1/128",
}

var reservedNetworks []*net.IPNet

//nolint:gochecknoinits
func init() {
	for _, b := range reservedCIDRs {
		_, network, err := net.ParseCIDR(b)
		if err != nil {
			panic(err)
		}
		reservedNetworks = append(reservedNetworks, network)
	}
}

// Errors returned by the URL validators. Callers wrap these with context (e.g. the field/env name).
var (
	// ErrURLEmpty is returned when the URL is empty after trimming.
	ErrURLEmpty = errors.New("URL must not be empty")
	// ErrURLScheme is returned when the URL scheme is not http or https.
	ErrURLScheme = errors.New("URL must use http or https scheme")
	// ErrURLNoHost is returned when the URL has no host.
	ErrURLNoHost = errors.New("URL must have a host")
	// ErrURLNotHTTPS is returned when https is required but the URL uses http.
	ErrURLNotHTTPS = errors.New("URL must use https")
	// ErrURLPlaintext is returned when plaintext http is used for a non-local destination.
	ErrURLPlaintext = errors.New("URL must use https; plaintext http is only allowed for localhost or in-cluster addresses")
	// ErrURLPrivateHost is returned when an external URL points to a non-public destination.
	ErrURLPrivateHost = errors.New("URL must point to a public host (private, reserved, loopback or in-cluster addresses are not allowed)")
	// ErrURLInvalid is returned when the URL cannot be parsed. It is intentionally static (it never
	// echoes the raw input) so embedded basic-auth credentials cannot leak into error responses or logs.
	ErrURLInvalid = errors.New("URL is not valid")
)

// parseHTTPURL trims, parses and requires an http/https scheme and a non-empty host.
// Embedded credentials (user:pass@) are tolerated for clients that use Basic auth.
func parseHTTPURL(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, ErrURLEmpty
	}
	u, err := url.Parse(raw)
	if err != nil {
		// Static error: url.Parse embeds the raw input in its message, which may contain credentials.
		return nil, ErrURLInvalid
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, ErrURLScheme
	}
	if u.Hostname() == "" {
		return nil, ErrURLNoHost
	}
	return u, nil
}

// isReservedIP reports whether ip falls in a private, loopback, link-local or reserved range.
func isReservedIP(ip net.IP) bool {
	for _, network := range reservedNetworks {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// inClusterSuffixes are IANA special-use / non-public DNS suffixes — none can ever be publicly
// delegated — so a host ending in one is treated as in-cluster: RFC 6762 .local (which also covers
// Kubernetes *.svc.cluster.local), RFC 6761 .localhost, and ICANN private-use .internal. Non-reserved
// suffixes (e.g. the bare k8s ".svc" short form) are deliberately excluded so they can't become a
// plaintext-http bypass if ever delegated; use the full FQDN or the dev opt-in for those.
var inClusterSuffixes = []string{".local", ".internal", ".localhost"}

// isLocalOrInClusterHost reports whether host is safe to reach over plaintext http: localhost, a
// loopback/private/reserved IP literal, a single-label hostname (e.g. "holmesgpt"), or a dotted name
// in a non-public suffix (e.g. "holmes.monitoring.svc.cluster.local"). A dotted public FQDN or a
// public IP literal returns false. This is a no-DNS heuristic: it never resolves names.
func isLocalOrInClusterHost(host string) bool {
	host = strings.TrimSpace(host)
	if host == "" {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return isReservedIP(ip)
	}
	// Single-label names (no dot) are in-cluster service names.
	if !strings.Contains(host, ".") {
		return true
	}
	// Dotted names in a non-public suffix (e.g. Kubernetes *.svc.cluster.local) are in-cluster.
	lower := strings.ToLower(strings.TrimSuffix(host, "."))
	for _, s := range inClusterSuffixes {
		if strings.HasSuffix(lower, s) {
			return true
		}
	}
	return false
}

// RequireSecureServiceURL validates a URL for an internal/operator-run service (HolmesGPT, the PMM
// callback URL). https is required for public destinations, but plaintext http is permitted to
// local/in-cluster addresses (loopback, private/reserved IP literals and single-label hostnames such
// as "holmesgpt") so the documented http://holmesgpt:8080 deployment keeps working. allowInsecure (an
// explicit dev/lab opt-in) permits http to any host.
func RequireSecureServiceURL(raw string, allowInsecure bool) (*url.URL, error) {
	u, err := parseHTTPURL(raw)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "https" || allowInsecure {
		return u, nil
	}
	if isLocalOrInClusterHost(u.Hostname()) {
		return u, nil
	}
	return nil, ErrURLPlaintext
}

// RequireSecureExternalURL validates a URL for an external SaaS (ServiceNow): https is required (no
// http) and the host must be public — loopback/private/reserved IP literals, localhost, single-label
// and in-cluster names are rejected as an SSRF guard. A hostname that *resolves* to a private address
// (DNS rebinding) needs a resolve-then-check dialer and is handled separately (F8).
func RequireSecureExternalURL(raw string) (*url.URL, error) {
	u, err := parseHTTPURL(raw)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "https" {
		return nil, ErrURLNotHTTPS
	}
	if isLocalOrInClusterHost(u.Hostname()) {
		return nil, ErrURLPrivateHost
	}
	return u, nil
}
