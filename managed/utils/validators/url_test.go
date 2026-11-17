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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequireSecureServiceURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		raw           string
		allowInsecure bool
		wantErr       error
	}{
		{"https public", "https://holmes.example.com", false, nil},
		{"https public with port", "https://holmes.example.com:8443/api", false, nil},
		{"http single-label in-cluster", "http://holmesgpt:8080", false, nil},
		{"http k8s fqdn", "http://holmes.monitoring.svc.cluster.local:8080", false, nil},
		{"http internal suffix", "http://pmm.internal", false, nil},
		{"http mdns local", "http://printer.local", false, nil},
		{"http localhost suffix", "http://app.localhost:8080", false, nil},
		{"http fqdn trailing dot", "http://holmes.svc.cluster.local.:8080", false, nil},
		{"http bare svc short form rejected", "http://holmes.monitoring.svc", false, ErrURLPlaintext},
		{"http two-label no suffix rejected", "http://holmes.monitoring", false, ErrURLPlaintext},
		{"http localhost", "http://localhost:9093", false, nil},
		{"http loopback ip", "http://127.0.0.1:8080", false, nil},
		{"http private ip", "http://10.1.2.3", false, nil},
		{"https public ip", "https://1.2.3.4", false, nil},
		{"https embedded creds", "https://user:pass@holmes/", false, nil},
		{"http embedded creds single-label", "http://user:pass@holmesgpt:8080", false, nil},
		{"http public fqdn rejected", "http://holmes.example.com", false, ErrURLPlaintext},
		{"http public ip rejected", "http://1.2.3.4", false, ErrURLPlaintext},
		{"http public fqdn with allowInsecure", "http://holmes.example.com", true, nil},
		{"bad scheme", "ftp://holmes.example.com", false, ErrURLScheme},
		{"no scheme", "holmesgpt:8080", false, ErrURLScheme},
		{"missing host", "https://", false, ErrURLNoHost},
		{"empty", "   ", false, ErrURLEmpty},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := RequireSecureServiceURL(tt.raw, tt.allowInsecure)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, got)
		})
	}
}

func TestRequireSecureExternalURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		raw     string
		wantErr error
	}{
		{"https saas", "https://acme.service-now.com/api/create", nil},
		{"https public ip", "https://1.2.3.4", nil},
		{"http rejected", "http://acme.service-now.com", ErrURLNotHTTPS},
		{"link-local rejected", "https://169.254.169.254", ErrURLPrivateHost},
		{"private ip rejected", "https://10.0.0.1", ErrURLPrivateHost},
		{"loopback rejected", "https://127.0.0.1", ErrURLPrivateHost},
		{"ula v6 rejected", "https://[fc00::1]", ErrURLPrivateHost},
		{"unspecified v4 rejected", "https://0.0.0.0", ErrURLPrivateHost},
		{"current-net v4 rejected", "https://0.0.0.1", ErrURLPrivateHost},
		{"unspecified v6 rejected", "https://[::]", ErrURLPrivateHost},
		{"broadcast rejected", "https://255.255.255.255", ErrURLPrivateHost},
		{"localhost rejected", "https://localhost", ErrURLPrivateHost},
		{"single-label rejected", "https://servicenow", ErrURLPrivateHost},
		{"internal suffix rejected", "https://sn.corp.internal", ErrURLPrivateHost},
		{"bad scheme", "ftp://acme.service-now.com", ErrURLScheme},
		{"missing host", "https://", ErrURLNoHost},
		{"empty", "", ErrURLEmpty},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := RequireSecureExternalURL(tt.raw)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErr), "got %v, want %v", err, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, got)
		})
	}
}

// TestURLValidators_doNotLeakCredentials checks that a malformed URL carrying basic-auth credentials
// is rejected with a static error that never echoes the raw input (so creds can't leak to logs/responses).
func TestURLValidators_doNotLeakCredentials(t *testing.T) {
	t.Parallel()
	const raw = "http://user:secretpass@%zz" // %zz is an invalid percent-escape → url.Parse fails

	_, errSvc := RequireSecureServiceURL(raw, false)
	require.ErrorIs(t, errSvc, ErrURLInvalid)
	assert.NotContains(t, errSvc.Error(), "secretpass")

	_, errExt := RequireSecureExternalURL(raw)
	require.ErrorIs(t, errExt, ErrURLInvalid)
	assert.NotContains(t, errExt.Error(), "secretpass")
}
