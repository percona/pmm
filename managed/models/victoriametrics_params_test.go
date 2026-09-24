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

package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVictoriaMetricsParams(t *testing.T) {
	t.Run("read non exist baseConfigFile", func(t *testing.T) {
		_, err := NewVictoriaMetricsParams("nonExistConfigFile.yml", VMBaseURL)
		require.NoError(t, err)
	})
	t.Run("check params for VMAlert", func(t *testing.T) {
		vmp, err := NewVictoriaMetricsParams("../testdata/victoriametrics/prometheus.external.alerts.yml", VMBaseURL)
		require.NoError(t, err)
		require.Equal(t, []string{"--rule=/srv/external_rules/rul1.yml", "--rule=/srv/external_rules/rule2.yml", "--evaluationInterval=10s"}, vmp.VMAlertFlags)
	})
	t.Run("ParsedURL returns a copy", func(t *testing.T) {
		vmp, err := NewVictoriaMetricsParams(BasePrometheusConfigPath, "https://user:pass@vm:8428/")
		require.NoError(t, err)
		u := vmp.ParsedURL()
		u.User = nil
		assert.Equal(t, "https://vm:8428/", u.String())
		assert.Equal(t, "https://user:pass@vm:8428/", vmp.URL())
	})
	t.Run("check external VM", func(t *testing.T) {
		tests := []struct {
			url  string
			want bool
		}{
			{
				"http://127.0.0.1:9090/prometheus",
				false,
			},
			{
				"http://127.0.0.1:9090/prometheus/",
				false,
			},
			{
				"http://victoriametrics:8428/",
				true,
			},
			{
				"https://example.com:9090/",
				true,
			},
		}
		for _, tt := range tests {
			t.Run(tt.url, func(t *testing.T) {
				vmp, err := NewVictoriaMetricsParams(BasePrometheusConfigPath, tt.url)
				require.NoError(t, err)
				assert.Equalf(t, tt.want, vmp.ExternalVM(), "ExternalVM()")
			})
		}
	})
}

func TestParseVictoriaMetricsURL(t *testing.T) {
	t.Run("valid URLs get a trailing slash", func(t *testing.T) {
		for raw, want := range map[string]string{
			VMBaseURL:                           VMBaseURL,
			"http://victoriametrics:8428":       "http://victoriametrics:8428/",
			"https://user:pass@vm:8428/path":    "https://user:pass@vm:8428/path/",
			"http://pmm-ha-vmauth.pmm.svc:8427": "http://pmm-ha-vmauth.pmm.svc:8427/",
		} {
			u, err := ParseVictoriaMetricsURL(raw)
			require.NoError(t, err, raw)
			assert.Equal(t, want, u.String(), raw)
		}
	})

	t.Run("URLs without an http scheme and a host are rejected", func(t *testing.T) {
		for _, raw := range []string{"", "vm:8428", "//vm:8428/", "vm.example.com", "ftp://vm:8428", "http:///path"} {
			_, err := ParseVictoriaMetricsURL(raw)
			require.Error(t, err, raw)
			assert.Contains(t, err.Error(), "invalid VictoriaMetrics URL", raw)
		}
	})

	t.Run("errors never echo credentials", func(t *testing.T) {
		_, err := ParseVictoriaMetricsURL("http://user:secret@[::1")
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "secret")

		_, err = ParseVictoriaMetricsURL("ftp://user:secret@vm:8428")
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "secret")

		// Redacted() keeps the username, so the scheme and host check drops the whole userinfo.
		_, err = ParseVictoriaMetricsURL("ftp://vmadmin:secret@vm:8428")
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "secret")
		assert.NotContains(t, err.Error(), "vmadmin")

		// A missing scheme is the likeliest way to get this variable wrong, and it is the one
		// shape url.Parse leaves in Opaque, where there is no URL.User to clear.
		_, err = ParseVictoriaMetricsURL("vmadmin:secret@vm.example.com/prometheus")
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "secret")
		assert.NotContains(t, err.Error(), "vmadmin")
		assert.Contains(t, err.Error(), "vm.example.com")
	})
}

func TestRedactURLCredentials(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		value string
		want  string
	}{
		{
			name:  "no credentials is left alone",
			value: "https://vm.example.com/api/v1/write",
			want:  "https://vm.example.com/api/v1/write",
		},
		{
			name:  "userinfo is replaced",
			value: "https://user:secret@vm.example.com/api/v1/write",
			want:  "https://<redacted>@vm.example.com/api/v1/write",
		},
		{
			name:  "a username alone is replaced",
			value: "https://user@vm.example.com/api/v1/write",
			want:  "https://<redacted>@vm.example.com/api/v1/write",
		},
		{
			name:  "a scheme-less value is parsed as an authority",
			value: "user:secret@vm.example.com/prometheus",
			want:  "<redacted>@vm.example.com/prometheus",
		},
		{
			name:  "every element of a list is redacted",
			value: "https://u1:s1@vm1.example.com/api/v1/write,https://u2:s2@vm2.example.com/api/v1/write",
			want:  "https://<redacted>@vm1.example.com/api/v1/write,https://<redacted>@vm2.example.com/api/v1/write",
		},
		{
			name:  "a list redacts the elements that carry credentials",
			value: "https://vm1.example.com/api/v1/write,https://u2:s2@vm2.example.com/api/v1/write",
			want:  "https://vm1.example.com/api/v1/write,https://<redacted>@vm2.example.com/api/v1/write",
		},
		{
			name:  "an element that cannot be parsed is redacted whole",
			value: "https://u1:s1@vm1.example.com/write,https://u2:s2@[::1",
			want:  "https://<redacted>@vm1.example.com/write,<redacted>",
		},
		// A comma is legal inside userinfo, so a password that carries one must not be split into
		// elements: the part in front of the comma stops looking like a credential and would be
		// printed as it stands. The third case is the one that hides best, because "user:1234"
		// parses as a host and a port all by itself.
		{
			name:  "a comma in the password does not split the URL",
			value: "http://user:pa,ss@vmauth:8427/",
			want:  "http://<redacted>@vmauth:8427/",
		},
		{
			name:  "several commas in the password do not split the URL",
			value: "http://user:p,a,s,s@vmauth:8427/",
			want:  "http://<redacted>@vmauth:8427/",
		},
		{
			name:  "a password whose first segment parses as a port",
			value: "http://user:1234,5678@host:8427/",
			want:  "http://<redacted>@host:8427/",
		},
		{
			name:  "a comma in the password of a scheme-less value",
			value: "user:pa,ss@vmauth:8427",
			want:  "<redacted>@vmauth:8427",
		},
		// A comma outside the userinfo is not a reason to lose the rest of the message.
		{
			name:  "a comma in the query keeps the URL readable",
			value: "http://u:p@vm.example.com/api/v1/write?tags=a,b",
			want:  "http://<redacted>@vm.example.com/api/v1/write?tags=a,b",
		},
		{
			name:  "an at sign in the path is not userinfo",
			value: "https://cdn.example.com/logo@2x.png",
			want:  "https://cdn.example.com/logo@2x.png",
		},
		// Neither a list nor one URL: an element without a scheme is not something vmagent would
		// accept, and parsing the whole as one URL leaves the second userinfo in the path.
		{
			name:  "a list whose second element has no scheme is redacted whole",
			value: "http://a:1@h1/write,b:2@h2",
			want:  "<redacted>",
		},
		{
			name:  "an unparseable element without credentials is left alone",
			value: "http://[::1",
			want:  "http://[::1",
		},
		{
			name:  "an empty value stays empty",
			value: "",
			want:  "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, RedactURLCredentials(tc.value))
		})
	}
}
