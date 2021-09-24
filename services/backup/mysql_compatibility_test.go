// pmm-managed
// Copyright (C) 2017 Percona LLC
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

package backup

import (
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMySQLAndXtrabackupCompatibility(t *testing.T) {
	xtrabackupVersions := []string{
		"8.0.26-18.0",
		"8.0.25-17.0",
		"8.0.23-16.0",
		"8.0.22-15.0",
		"8.0.14",
		"8.0.13",
		"8.0.12",
		"8.0.11",
		"8.0.10",
		"8.0.9",
		"8.0.8",
		"8.0.7",
		"8.0.6",
		"8.0.5",
		"8.0.4",

		"2.4.24",
		"2.4.23",
		"2.4.22",
		"2.4.21",
		"2.4.20",
		"2.4.19",
		"2.4.18",
		"2.4.17",
		"2.4.16",
	}

	mysqlVersions := []string{
		"8.0.26",
		"8.0.25",
		"8.0.24",
		"8.0.23",
		"8.0.22",
		"8.0.21",
		"8.0.20",
		"8.0.19",
		"8.0.18",
		"8.0.17",
		"8.0.16",
		"8.0.15",
		"8.0.14",
		"8.0.13",
		"8.0.12",
		"8.0.11",

		"5.5",
		"5.6",
		"5.7",
	}

	type compatibleRange struct {
		from string // inclusively
		to   string // exclusively
	}
	supportedMatrix := map[string]compatibleRange{
		"8.0.26-18.0": {from: "8.0", to: "8.0.27"},
		"8.0.25-17.0": {from: "8.0", to: "8.0.26"},
		"8.0.23-16.0": {from: "8.0", to: "8.0.24"},
		"8.0.22-15.0": {from: "8.0", to: "8.0.23"},
		"8.0.14":      {from: "8.0", to: "8.0.22"},
		"8.0.13":      {from: "8.0", to: "8.0.21"},
		"8.0.12":      {from: "8.0", to: "8.0.21"},
		"8.0.11":      {from: "8.0", to: "8.0.20"},
		"8.0.10":      {from: "8.0", to: "8.0.20"},
		"8.0.9":       {from: "8.0", to: "8.0.20"},
		"8.0.8":       {from: "8.0", to: "8.0.20"},
		"8.0.7":       {from: "8.0", to: "8.0.20"},
		"8.0.6":       {from: "8.0", to: "8.0.20"},

		"2.4.24": {from: "5.5", to: "5.8"},
		"2.4.23": {from: "5.5", to: "5.8"},
		"2.4.22": {from: "5.5", to: "5.8"},
		"2.4.21": {from: "5.5", to: "5.8"},
		"2.4.20": {from: "5.5", to: "5.8"},
		"2.4.19": {from: "5.5", to: "5.8"},
		"2.4.18": {from: "5.5", to: "5.8"},
	}

	for _, xtrabackupVersion := range xtrabackupVersions {
		for _, mysqlVersion := range mysqlVersions {
			var supported bool
			if r, ok := supportedMatrix[xtrabackupVersion]; ok {
				mysqlMinVersion, err := version.NewVersion(r.from)
				require.NoError(t, err)

				mysqlMaxVersion, err := version.NewVersion(r.to)
				require.NoError(t, err)

				mysqlVersion, err := version.NewVersion(mysqlVersion)
				require.NoError(t, err)

				if mysqlVersion.GreaterThanOrEqual(mysqlMinVersion) && mysqlVersion.LessThan(mysqlMaxVersion) {
					supported = true
				}
			}

			actualSupported, err := mysqlAndXtrabackupCompatible(mysqlVersion, xtrabackupVersion)
			require.NoError(t, err)
			assert.Equal(t, supported, actualSupported, "xtrabackup version %q, mysql version %q", xtrabackupVersion, mysqlVersion)
		}
	}
}
