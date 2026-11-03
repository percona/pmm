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

package grafana

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The markers this package computes are only trustworthy because nginx overwrites them on every
// route a client can reach Grafana or vmproxy by. A location that declares a proxy_set_header of
// its own drops the server-level ones, so a new location is one forgotten line away from
// forwarding a client's value. Nothing else checks that file, so this does.
const nginxConfigPath = "../../../build/ansible/roles/nginx/files/conf.d/pmm.conf"

// markedUpstreams are the proxy_pass targets whose requests reach a marker reader: vmproxy itself,
// and Grafana, which forwards a client's headers on to its data sources. Ports are matched too, in
// case a location names one instead of the upstream.
var markedUpstreams = []string{"grafana", "vmproxy", ":8430", ":3000"}

var (
	locationStart    = regexp.MustCompile(`^\s*location\s+(.+?)\s*\{`)
	proxyPassLine    = regexp.MustCompile(`^\s*proxy_pass\s+(\S+?);`)
	proxySetHeader   = regexp.MustCompile(`^\s*proxy_set_header\s+(\S+)\s+(.*?);\s*$`)
	proxyMarkerNames = []string{lbacHeaderName, adminHeaderName, writeMarkerHeaderName}
)

// writeMarkerHeaderName is set by Grafana on a recording rule write and read by vmproxy. Unlike
// the other two it is never set from the auth_request response, so nginx clears it instead. The
// name is repeated rather than imported because the other two copies are unexported, in
// managed/services/supervisord/supervisord.go and in grafana.ini.
const writeMarkerHeaderName = "X-Proxy-Write"

type nginxLocation struct {
	name      string
	line      int
	proxyPass string
	headers   map[string]string
}

// reachesMarkerReader reports whether this location's proxy_pass target reads the markers.
func (l nginxLocation) reachesMarkerReader() bool {
	for _, u := range markedUpstreams {
		if strings.Contains(l.proxyPass, u) {
			return true
		}
	}

	return false
}

// parseNginxLocations returns every location block, plus the proxy_set_header directives declared
// at server level, outside any location.
func parseNginxLocations(t *testing.T, path string) ([]nginxLocation, map[string]string) {
	t.Helper()

	buf, err := os.ReadFile(path)
	require.NoError(t, err, "the nginx config this test guards has moved; update nginxConfigPath")

	var locations []nginxLocation
	serverLevel := make(map[string]string)

	var cur *nginxLocation
	depth := 0

	for i, line := range strings.Split(string(buf), "\n") {
		if cur == nil {
			if m := locationStart.FindStringSubmatch(line); m != nil {
				cur = &nginxLocation{name: m[1], line: i + 1, headers: make(map[string]string)}
				depth = 1
				continue
			}

			if m := proxySetHeader.FindStringSubmatch(line); m != nil {
				serverLevel[m[1]] = m[2]
			}

			continue
		}

		depth += strings.Count(line, "{") - strings.Count(line, "}")
		if depth <= 0 {
			locations = append(locations, *cur)
			cur = nil
			continue
		}

		if m := proxyPassLine.FindStringSubmatch(line); m != nil {
			cur.proxyPass = m[1]
		}
		if m := proxySetHeader.FindStringSubmatch(line); m != nil {
			cur.headers[m[1]] = m[2]
		}
	}

	require.Nil(t, cur, "unbalanced braces in %s", path)
	require.NotEmpty(t, locations, "parsed no locations from %s", path)

	return locations, serverLevel
}

func TestNginxOwnsTheProxyMarkers(t *testing.T) {
	t.Parallel()

	locations, serverLevel := parseNginxLocations(t, nginxConfigPath)

	t.Run("server level sets every marker", func(t *testing.T) {
		t.Parallel()

		for _, name := range proxyMarkerNames {
			assert.Containsf(t, serverLevel, name,
				"%s must be set outside any location, or a location that declares no header of its own forwards the client's value",
				name)
		}
	})

	t.Run("a location declaring its own headers repeats them", func(t *testing.T) {
		t.Parallel()

		checked := 0

		for _, loc := range locations {
			// A location with no proxy_set_header of its own inherits the server-level
			// ones, so only those that declare any have to repeat them.
			if loc.proxyPass == "" || len(loc.headers) == 0 || !loc.reachesMarkerReader() {
				continue
			}

			checked++

			// X-Proxy-Filter is deliberately absent from /graph/api/live/, which predates
			// this test, so only the two markers vmproxy acts on are required here.
			for _, name := range []string{adminHeaderName, writeMarkerHeaderName} {
				assert.Containsf(t, loc.headers, name,
					"location %s (%s:%d) proxies to %s and declares its own proxy_set_header, so it inherits none: it must set %s",
					loc.name, nginxConfigPath, loc.line, loc.proxyPass, name)
			}
		}

		require.NotZero(t, checked, "matched no location reaching Grafana or vmproxy; the parser or the upstream names are stale")
	})

	t.Run("the write marker is only ever cleared", func(t *testing.T) {
		t.Parallel()

		// Nothing in PMM produces a value for it, so any value but the empty one would be
		// a client's own or a stale variable.
		if value, ok := serverLevel[writeMarkerHeaderName]; ok {
			assert.Equalf(t, `""`, value, "%s must be cleared, not set to %s", writeMarkerHeaderName, value)
		}

		for _, loc := range locations {
			value, ok := loc.headers[writeMarkerHeaderName]
			if !ok {
				continue
			}

			assert.Equalf(t, `""`, value,
				"location %s (%s:%d) must clear %s, not set it to %s",
				loc.name, nginxConfigPath, loc.line, writeMarkerHeaderName, value)
		}
	})
}
