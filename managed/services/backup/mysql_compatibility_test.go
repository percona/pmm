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

	"github.com/stretchr/testify/assert"
)

type mysqlAndPXBVersions struct {
	mysql, pxb string
}

func TestMysqlAndXtrabackupCompatible(t *testing.T) {
	t.Parallel()

	compatible := []mysqlAndPXBVersions{
		// MySQL [5.5; 5.8), PXB [2.4.18; 5.2)
		{"5.5", "2.4.18"},
		{"5.5", "2.4.20"},
		{"5.5", "2.4.99"},
		{"5.6", "2.4.18"},
		{"5.6", "2.4.20"},
		{"5.6", "2.4.99"},
		{"5.7", "2.4.18"},
		{"5.7", "2.4.20"},
		{"5.7", "2.4.99"},

		// MySQL [8.0; 8.0.20), PXB [8.0.6; 9.0)
		{"8.0", "8.0.6"},
		{"8.0", "8.0.8"},
		{"8.0", "8.99.99"},
		{"8.0.12", "8.0.6"},
		{"8.0.12", "8.0.8"},
		{"8.0.12", "8.99.99"},
		{"8.0.19", "8.0.6"},
		{"8.0.19", "8.0.8"},
		{"8.0.19", "8.99.99"},

		// MySQL [8.0.20; 8.0.21), PXB [8.0.12; 9.0)
		{"8.0.20", "8.0.12"},
		{"8.0.20", "8.0.18"},
		{"8.0.20", "8.99.99"},

		// MySQL [8.0.21; 8.0.22), PXB [8.0.14; 9.0)
		{"8.0.21", "8.0.14"},
		{"8.0.21", "8.0.18"},
		{"8.0.21", "8.99.99"},

		// MySQL [8.0.22; 9.0), PXB [8.0.22; 9.0)
		{"8.0.22", "8.0.22"},
		{"8.0.22", "8.0.22-15.0"},
		{"8.0.22", "8.0.50"},
		{"8.0.22", "8.99.99"},
		{"8.0.22-13", "8.0.22"},
		{"8.0.22-13", "8.0.22-15.0"},
		{"8.0.22-13", "8.0.50"},
		{"8.0.22-13", "8.99.99"},
		{"8.0.28", "8.0.28"},
		{"8.0.28", "8.0.50"},
		{"8.0.28", "8.99.99"},
		{"8.99.99", "8.99.99"},
	}

	incompatible := []mysqlAndPXBVersions{
		// MySQL [5.5; 5.8), PXB [2.4.18; 2.5)
		{"5.4", "2.4.17"},
		{"5.4", "2.4.18"},
		{"5.4", "2.4.25"},
		{"5.4", "2.4.99"},
		{"5.4", "2.5"},
		//
		{"5.5", "2.4.17"},
		{"5.5", "2.5"},
		//
		{"5.6", "2.4.17"},
		{"5.6", "2.5"},
		//
		{"5.7", "2.4.17"},
		{"5.7", "2.5"},
		//
		{"5.8", "2.4.17"},
		{"5.8", "2.4.18"},
		{"5.8", "2.4.25"},
		{"5.8", "2.4.99"},
		{"5.8", "2.5"},

		// MySQL [8.0; 8.0.20), PXB [8.0.6; 9.0)
		{"7.99.99", "8.0.5"},
		{"7.99.99", "8.0.6"},
		{"7.99.99", "8.0.10"},
		{"7.99.99", "8.99.99"},
		{"7.99.99", "9.0"},
		//
		{"8.0", "8.0.5"},
		{"8.0", "9.0"},
		//
		{"8.0.10", "8.0.5"},
		{"8.0.10", "9.0"},
		//
		{"8.0.19", "8.0.5"},
		{"8.0.19", "9.0"},

		// MySQL [8.0.20; 8.0.21), PXB [8.0.12; 9.0)
		{"8.0.20", "8.0.11"},
		{"8.0.20", "9.0"},

		// MySQL [8.0.21; 8.0.22), PXB [8.0.14; 9.0)
		{"8.0.21", "8.0.13"},
		{"8.0.21", "9.0"},

		// MySQL [8.0.22; 9.0), PXB [8.0.22; 9.0)
		{"8.0.22", "8.0.21"},
		{"8.0.22", "9.0"},
		//
		{"8.0.28", "8.0.22-15.0"},
		{"8.0.28", "8.0.27"},
		{"8.0.28", "9.0"},
		//
		{"8.99.99", "8.99.98"},
		{"8.99.99", "9.0"},
		//
		{"9.0", "8.0.21"},
		{"9.0", "8.0.22"},
		{"9.0", "8.0.30"},
		{"9.0", "8.99.99"},
		{"9.0", "9.0"},
	}

	for _, ver := range compatible {
		actual, err := mysqlAndXtrabackupCompatible(ver.mysql, ver.pxb)
		assert.NoError(t, err)
		assert.True(t, actual, "mysql version %q, xtrabackup version %q", ver.mysql, ver.pxb)
	}

	for _, ver := range incompatible {
		actual, err := mysqlAndXtrabackupCompatible(ver.mysql, ver.pxb)
		assert.NoError(t, err)
		assert.False(t, actual, "mysql version %q, xtrabackup version %q", ver.mysql, ver.pxb)
	}

	_, err := mysqlAndXtrabackupCompatible("eight", "8.0.6")
	assert.Error(t, err)

	_, err = mysqlAndXtrabackupCompatible("8.0", "eight")
	assert.Error(t, err)
}
