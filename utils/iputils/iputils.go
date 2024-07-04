package iputils

// Copyright (C) 2024 Percona LLC
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

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
)

func IsIPV6Only() bool {
	ipv6only := os.Getenv("PMM_IPV6ONLY")
	return ipv6only == "true" || ipv6only == "1"
}

func GetLoopbackAddress() string {
	if IsIPV6Only() {
		return "::1"
	}
	return "127.0.0.1"
}

func GetAllInterfacesAddress() string {
	if IsIPV6Only() {
		return "::" // Listen on all IPv6 interfaces
	}
	return "0.0.0.0" // Listen on all IPv4 interfaces
}

// ConvertLocalhostIPv4ToIPv6URL converts "127.0.0.1" to "[::1]" in a given string,
// which can be a plain IP address, IP:port, or a full URL.
func ConvertLocalhostIPv4ToIPv6URL(input string) string {
	// Attempt to parse the input as a URL.
	parsedURL, err := url.Parse(input)
	if err == nil && parsedURL.Host != "" {
		// If parsing is successful and there's a host, modify the host part.
		host, port, err := net.SplitHostPort(parsedURL.Host)
		if err != nil {
			// If there's an error, it might be just a hostname without a port.
			host = parsedURL.Host
		}
		if host == "127.0.0.1" {
			// Replace the host with IPv6 equivalent.
			newHost := "[::1]"
			if port != "" {
				newHost = fmt.Sprintf("[::1]:%s", port)
			}
			parsedURL.Host = newHost
			return parsedURL.String()
		}
	}

	// For non-URL inputs or if the URL doesn't need modification.
	return strings.Replace(input, "127.0.0.1", "[::1]", 1)
}
