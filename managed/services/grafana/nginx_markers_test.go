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
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The markers hold only because nginx overwrites them on every route to Grafana or vmproxy, and
// a location declaring its own proxy_set_header drops the server-level ones.
const nginxConfigPath = "../../../build/ansible/roles/nginx/files/conf.d/pmm.conf"

// markedUpstreams are the proxy_pass targets that reach a marker reader, by name or by port.
var markedUpstreams = []string{"grafana", "vmproxy", ":8430", ":3000"}

var (
	locationStart    = regexp.MustCompile(`^\s*location\s+(.+?)\s*\{`)
	proxyPassLine    = regexp.MustCompile(`^\s*proxy_pass\s+(\S+?);`)
	proxySetHeader   = regexp.MustCompile(`^\s*proxy_set_header\s+(\S+)\s+(.*?);\s*$`)
	proxyMarkerNames = []string{lbacHeaderName, adminHeaderName, writeMarkerHeaderName}
)

// writeMarkerHeaderName is the header Grafana sets on a recording rule write, which nginx clears.
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

// parseNginxLocations returns every location block, plus the server-level proxy_set_header lines.
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
				depth = strings.Count(line, "{") - strings.Count(line, "}")
				if depth <= 0 {
					require.NotContainsf(t, line, "proxy_pass",
						"%s:%d: put a proxying location on separate lines so its headers can be checked", path, i+1)

					continue
				}

				cur = &nginxLocation{name: m[1], line: i + 1, headers: make(map[string]string)}

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

func TestParseNginxLocationsClosesOneLineBlocks(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "pmm.conf")
	conf := "location = /healthz { return 200; }\nlocation /graph {\n  proxy_pass http://grafana;\n}\n"
	require.NoError(t, os.WriteFile(path, []byte(conf), 0o600))

	locations, _ := parseNginxLocations(t, path)

	require.Len(t, locations, 1)
	assert.Equal(t, "/graph", locations[0].name)
	assert.Equal(t, "http://grafana", locations[0].proxyPass)
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
			// A location declaring none of its own inherits the server-level headers.
			if loc.proxyPass == "" || len(loc.headers) == 0 || !loc.reachesMarkerReader() {
				continue
			}

			checked++

			// X-Proxy-Filter is not required: /graph/api/live/ omits it.
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

		// Nothing produces a value for it, so anything but empty is a client's or stale.
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
