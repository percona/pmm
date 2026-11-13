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

package om

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInstallSupportMongoDBVersions(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		osID      string
		versionID string
		expected  []string
	}{
		{"ubuntu", "22.04", []string{"7.0", "8.0"}},
		{"ubuntu", "24.04", []string{"7.0", "8.0"}},
		{"ubuntu", "20.04", nil},
		// Ubuntu matches VERSION_ID exactly.
		{"ubuntu", "22", nil},
		{"rocky", "8.10", []string{"7.0", "8.0"}},
		{"rocky", "9.4", []string{"7.0", "8.0"}},
		{"rocky", "10.0", []string{"8.0"}},
		{"rocky", "10", []string{"8.0"}},
		{"rocky", "7.9", nil},
		{"debian", "12", nil},
		{"rocky", "", nil},
	} {
		t.Run(tc.osID+" "+tc.versionID, func(t *testing.T) {
			t.Parallel()

			observed := map[string]any{"os_id": tc.osID, "os_version_id": tc.versionID}
			assert.Equal(t, tc.expected, testSupport.release(observed).versions)
		})
	}
}

func TestUnsupportedMongoDBVersion(t *testing.T) {
	t.Parallel()

	host := func(name, osID, versionID string) extensionsHost {
		return extensionsHost{Name: name, Observed: map[string]any{"os_id": osID, "os_version_id": versionID}}
	}
	rocky := []extensionsHost{host("db-01", "rocky", "9.4"), host("db-02", "rocky", "10.0"), host("db-03", "rocky", "8.10")}
	names := []string{"db-01", "db-02", "db-03"}

	t.Run("a version every release supports passes, patch ignored", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, testSupport.unsupportedMongoDBVersion("8.0.4", names, rocky))
	})

	t.Run("Rocky Linux 10 refuses 7.0, naming the node and what it supports", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t,
			"MongoDB 7.0 cannot be installed onto 1 of the selected node(s) -- db-02 runs Rocky Linux 10.0, "+
				"which supports MongoDB 8.0. Choose a version every selected node supports.",
			testSupport.unsupportedMongoDBVersion("7.0", names, rocky))
	})

	t.Run("a version no release supports names every node", func(t *testing.T) {
		t.Parallel()

		problem := testSupport.unsupportedMongoDBVersion("9.0", names, rocky)

		assert.Contains(t, problem, "cannot be installed onto 3 of the selected node(s)")
		assert.Contains(t, problem, "db-01 runs Rocky Linux 9.4, which supports MongoDB 7.0, 8.0")
	})

	t.Run("a release that gets no version is left to the readiness check", func(t *testing.T) {
		t.Parallel()

		hosts := []extensionsHost{host("db-01", "rocky", "9.4"), host("db-02", "ubuntu", "20.04")}

		assert.Empty(t, testSupport.unsupportedMongoDBVersion("7.0", []string{"db-01", "db-02"}, hosts))
	})
}

func TestMajorMinor(t *testing.T) {
	t.Parallel()

	for version, expected := range map[string]string{"8.0": "8.0", "7.0.14": "7.0", "10.1.2": "10.1", "7": "7"} {
		assert.Equal(t, expected, majorMinor(version), version)
	}
}
