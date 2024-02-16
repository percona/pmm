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
	"os"
	"testing"
)

func TestIsIPV6Only(t *testing.T) {
	tests := []struct {
		name      string
		envValue  string
		wantValue bool
	}{
		{"IPV6Only true", "true", true},
		{"IPV6Only 1", "1", true},
		{"IPV6Only false", "false", false},
		{"IPV6Only empty", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set environment variable
			os.Setenv("PMM_IPV6ONLY", tc.envValue)
			// Test IsIPV6Only
			if got := IsIPV6Only(); got != tc.wantValue {
				t.Errorf("IsIPV6Only() = %v, want %v", got, tc.wantValue)
			}
			// Clean up environment variable
			os.Unsetenv("PMM_IPV6ONLY")
		})
	}
}

func TestGetLoopbackAddress(t *testing.T) {
	// Setting the environment variable to simulate IPv6 only
	os.Setenv("PMM_IPV6ONLY", "true")
	if got := GetLoopbackAddress(); got != "::1" {
		t.Errorf("GetLoopbackAddress() with IPV6ONLY=true = %v, want ::1", got)
	}

	// Unsetting the environment variable to test IPv4
	os.Unsetenv("PMM_IPV6ONLY")
	if got := GetLoopbackAddress(); got != "127.0.0.1" {
		t.Errorf("GetLoopbackAddress() with IPV6ONLY unset = %v, want 127.0.0.1", got)
	}
}

func TestGetAllInterfacesAddress(t *testing.T) {
	os.Setenv("PMM_IPV6ONLY", "true")
	if got := GetAllInterfacesAddress(); got != "::" {
		t.Errorf("GetAllInterfacesAddress() with IPV6ONLY=true = %v, want ::", got)
	}

	os.Unsetenv("PMM_IPV6ONLY")
	if got := GetAllInterfacesAddress(); got != "0.0.0.0" {
		t.Errorf("GetAllInterfacesAddress() with IPV6ONLY unset = %v, want 0.0.0.0", got)
	}
}

func TestConvertLocalhostIPv4ToIPv6URL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"Plain IP", "127.0.0.1", "[::1]"},
		{"IP with port", "127.0.0.1:8080", "[::1]:8080"},
		{"HTTP URL", "http://127.0.0.1", "http://[::1]"},
		{"HTTPS URL", "https://127.0.0.1:443", "https://[::1]:443"},
		{"URL with path", "http://127.0.0.1/prometheus", "http://[::1]/prometheus"},
		{"URL with port and path", "http://127.0.0.1:9090/prometheus/", "http://[::1]:9090/prometheus/"},
		{"Non-localhost IP", "http://192.168.1.1", "http://192.168.1.1"},               // Should remain unchanged
		{"IPv6 URL", "http://[::1]:9090/prometheus/", "http://[::1]:9090/prometheus/"}, // Should remain unchanged
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ConvertLocalhostIPv4ToIPv6URL(tc.input); got != tc.want {
				t.Errorf("ConvertLocalhostIPv4ToIPv6URL(%v) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}
