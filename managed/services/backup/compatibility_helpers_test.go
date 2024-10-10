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

package backup

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/percona/pmm/managed/models"
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

func TestVendorToServiceType(t *testing.T) {
	for _, test := range []struct {
		name      string
		input     string
		output    models.ServiceType
		errString string
	}{
		{
			name:      "supported type",
			input:     "mysql",
			output:    models.MySQLServiceType,
			errString: "",
		},
		{
			name:      "unsupported type",
			input:     "haproxy",
			output:    "",
			errString: "unimplemented service type",
		},
		{
			name:      "unknown type",
			input:     "some_service_type",
			output:    "",
			errString: "unknown service type",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			res, err := vendorToServiceType(test.input)

			assert.Equal(t, test.output, res)
			if test.errString != "" {
				assert.Contains(t, err.Error(), test.errString)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSoftwareVersionsToMap(t *testing.T) {
	input := models.SoftwareVersions{
		{Name: "mysqld", Version: "8.0.25"},
		{Name: "xtrabackup", Version: "8.0.25"},
		{Name: "qpress", Version: "1.1"},
		{Name: "random_software", Version: "99"},
	}
	expected := map[models.SoftwareName]string{
		models.SoftwareName("mysqld"):          "8.0.25",
		models.SoftwareName("xtrabackup"):      "8.0.25",
		models.SoftwareName("qpress"):          "1.1",
		models.SoftwareName("random_software"): "99",
	}

	res := softwareVersionsToMap(models.SoftwareVersions{})
	assert.Empty(t, res)

	res = softwareVersionsToMap(input)
	assert.Equal(t, expected, res)
}

func TestMySQLSoftwaresInstalledAndCompatible(t *testing.T) {
	for _, test := range []struct {
		name  string
		input map[models.SoftwareName]string
		err   error
	}{
		// mysql cases
		{
			name: "successful",
			input: map[models.SoftwareName]string{
				models.SoftwareName("mysqld"):     "8.0.25",
				models.SoftwareName("xtrabackup"): "8.0.25",
				models.SoftwareName("xbcloud"):    "8.0.25",
				models.SoftwareName("qpress"):     "1.1",
			},
			err: nil,
		},
		{
			name: "no xtrabackup",
			input: map[models.SoftwareName]string{
				models.SoftwareName("mysqld"):  "8.0.25",
				models.SoftwareName("xbcloud"): "8.0.25",
				models.SoftwareName("qpress"):  "1.1",
			},
			err: ErrXtrabackupNotInstalled,
		},
		{
			name: "no xbcloud",
			input: map[models.SoftwareName]string{
				models.SoftwareName("mysqld"):     "8.0.25",
				models.SoftwareName("xtrabackup"): "8.0.25",
				models.SoftwareName("qpress"):     "1.1",
			},
			err: ErrXtrabackupNotInstalled,
		},
		{
			name: "no mysqld",
			input: map[models.SoftwareName]string{
				models.SoftwareName("xtrabackup"): "8.0.25",
				models.SoftwareName("xbcloud"):    "8.0.25",
				models.SoftwareName("qpress"):     "1.1",
			},
			err: ErrIncompatibleService,
		},
		{
			name: "invalid xtrabackup",
			input: map[models.SoftwareName]string{
				models.SoftwareName("mysqld"):     "8.0.25",
				models.SoftwareName("xtrabackup"): "8.0.26",
				models.SoftwareName("xbcloud"):    "8.0.25",
				models.SoftwareName("qpress"):     "1.1",
			},
			err: ErrInvalidXtrabackup,
		},
		{
			name: "incompatible xtrabackup",
			input: map[models.SoftwareName]string{
				models.SoftwareName("mysqld"):     "8.0.25",
				models.SoftwareName("xtrabackup"): "8.0.24",
				models.SoftwareName("xbcloud"):    "8.0.24",
				models.SoftwareName("qpress"):     "1.1",
			},
			err: ErrIncompatibleXtrabackup,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := mySQLBackupSoftwareInstalledAndCompatible(test.input)
			if test.err != nil {
				assert.ErrorIs(t, err, test.err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMongoDBBackupSoftwareInstalledAndCompatible(t *testing.T) {
	for _, test := range []struct {
		name  string
		input map[models.SoftwareName]string
		err   error
	}{
		{
			name: "successful",
			input: map[models.SoftwareName]string{
				models.SoftwareName("mongodb"): "6.0.2",
				models.SoftwareName("pbm"):     "2.0.1",
			},
			err: nil,
		},
		{
			name: "incompatible pbm",
			input: map[models.SoftwareName]string{
				models.SoftwareName("mongodb"): "6.0.2",
				models.SoftwareName("pbm"):     "1.8.0",
			},
			err: ErrIncompatiblePBM,
		},
		{
			name: "pbm not installed",
			input: map[models.SoftwareName]string{
				models.SoftwareName("mongodb"): "6.0.2",
				models.SoftwareName("pbm"):     "",
			},
			err: ErrIncompatibleService,
		},
		{
			name: "mongod not installed",
			input: map[models.SoftwareName]string{
				models.SoftwareName("mongodb"): "",
				models.SoftwareName("pbm"):     "2.0.1",
			},
			err: ErrIncompatibleService,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := mongoDBBackupSoftwareInstalledAndCompatible(test.input)
			if test.err != nil {
				assert.ErrorIs(t, err, test.err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
