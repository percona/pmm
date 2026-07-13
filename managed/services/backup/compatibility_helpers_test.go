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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

type mysqlAndPXBVersions struct {
	mysql, pxb string
}

func TestMysqlAndXtrabackupCompatible(t *testing.T) {
	t.Parallel()

	compatible := []mysqlAndPXBVersions{
		// MySQL [5.5; 5.8), PXB [2.4.18; 2.5)
		{"5.5", "2.4.18"},
		{"5.5", "2.4.20"},
		{"5.5", "2.4.99"},
		{"5.6", "2.4.18"},
		{"5.6", "2.4.20"},
		{"5.6", "2.4.99"},
		{"5.7", "2.4.18"},
		{"5.7", "2.4.20"},
		{"5.7", "2.4.99"},

		// MySQL [8.0; 8.0.20), PXB [8.0.6; 8.1.0)
		{"8.0", "8.0.6"},
		{"8.0", "8.0.8"},
		{"8.0", "8.0.99"},
		{"8.0.12", "8.0.6"},
		{"8.0.12", "8.0.8"},
		{"8.0.12", "8.0.99"},
		{"8.0.19", "8.0.6"},
		{"8.0.19", "8.0.8"},
		{"8.0.19", "8.0.99"},

		// MySQL [8.0.20; 8.0.21), PXB [8.0.12; 8.1.0)
		{"8.0.20", "8.0.12"},
		{"8.0.20", "8.0.18"},
		{"8.0.20", "8.0.99"},

		// MySQL [8.0.21; 8.0.22), PXB [8.0.14; 8.1.0)
		{"8.0.21", "8.0.14"},
		{"8.0.21", "8.0.18"},
		{"8.0.21", "8.0.99"},

		// MySQL [8.0.22; 8.0.34), PXB [MySQL version; 8.1.0)
		{"8.0.22", "8.0.22"},
		{"8.0.22", "8.0.22-15.0"},
		{"8.0.22", "8.0.50"},
		{"8.0.22-13", "8.0.22"},
		{"8.0.22-13", "8.0.22-15.0"},
		{"8.0.22-13", "8.0.50"},
		{"8.0.28", "8.0.28"},
		{"8.0.28", "8.0.50"},

		// MySQL 8.0.34 and newer 8.0 releases, PXB 8.0.34 and newer 8.0 releases.
		{"8.0.34", "8.0.34"},
		{"8.0.34", "8.0.35"},
		{"8.0.44", "8.0.34"},
		{"8.0.44", "8.0.35"},
		{"8.0.44-35.1", "8.0.35-35"},

		// MySQL 8.1.x, PXB 8.1.x.
		{"8.1.0", "8.1.0"},
		{"8.1.0-1", "8.1.0-1"},

		// MySQL 8.2.x, PXB 8.2.x.
		{"8.2.0", "8.2.0"},
		{"8.2.0-1", "8.2.0-1"},

		// MySQL 8.3.x, PXB 8.3.x.
		{"8.3.0", "8.3.0"},
		{"8.3.0-1", "8.3.0-1"},

		// MySQL [8.4.0; 8.5.0), PXB [8.4.0; 8.5.0)
		{"8.4.0", "8.4.0"},
		{"8.4.0", "8.4.0-4"},
		{"8.4.5-5.1", "8.4.0-4"},
		{"8.4.6", "8.4.0-4"},
		{"8.4.6", "8.4.0-5"},
		{"8.4.6", "8.4.1"},
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

		// MySQL [8.0; 8.0.20), PXB [8.0.6; 8.1.0)
		{"7.99.99", "8.0.5"},
		{"7.99.99", "8.0.6"},
		{"7.99.99", "8.0.10"},
		{"7.99.99", "8.99.99"},
		{"7.99.99", "9.0"},
		//
		{"8.0", "8.0.5"},
		{"8.0", "8.1.0"},
		{"8.0", "8.3.0"},
		{"8.0", "8.4.0"},
		{"8.0", "9.0"},
		//
		{"8.0.10", "8.0.5"},
		{"8.0.10", "8.1.0"},
		{"8.0.10", "8.3.0"},
		{"8.0.10", "8.4.0"},
		{"8.0.10", "9.0"},
		//
		{"8.0.19", "8.0.5"},
		{"8.0.19", "8.1.0"},
		{"8.0.19", "8.3.0"},
		{"8.0.19", "8.4.0"},
		{"8.0.19", "9.0"},

		// MySQL [8.0.20; 8.0.21), PXB [8.0.12; 8.1.0)
		{"8.0.20", "8.0.11"},
		{"8.0.20", "8.1.0"},
		{"8.0.20", "8.3.0"},
		{"8.0.20", "8.4.0"},
		{"8.0.20", "9.0"},

		// MySQL [8.0.21; 8.0.22), PXB [8.0.14; 8.1.0)
		{"8.0.21", "8.0.13"},
		{"8.0.21", "8.1.0"},
		{"8.0.21", "8.3.0"},
		{"8.0.21", "8.4.0"},
		{"8.0.21", "9.0"},

		// MySQL [8.0.22; 8.0.34), PXB [MySQL version; 8.1.0)
		{"8.0.22", "8.0.21"},
		{"8.0.22", "8.1.0"},
		{"8.0.22", "8.3.0"},
		{"8.0.22", "8.4.0"},
		{"8.0.22", "9.0"},
		//
		{"8.0.28", "8.0.22-15.0"},
		{"8.0.28", "8.0.27"},
		{"8.0.28", "8.1.0"},
		{"8.0.28", "8.3.0"},
		{"8.0.28", "8.4.0"},
		{"8.0.28", "9.0"},
		//
		// MySQL 8.0.34 and newer 8.0 releases, PXB 8.0.34 and newer 8.0 releases.
		{"8.0.34", "8.0.33"},
		{"8.0.44", "8.0.33"},
		{"8.0.44", "8.1.0"},
		{"8.0.44", "8.3.0"},
		{"8.0.44", "8.4.0"},
		{"8.0.44", "9.0"},
		//
		// MySQL 8.1.x, PXB 8.1.x.
		{"8.1.0", "8.0.35"},
		{"8.1.0", "8.2.0"},
		{"8.1.0", "8.4.0"},
		{"8.1.0", "9.0"},
		//
		// MySQL 8.2.x, PXB 8.2.x.
		{"8.2.0", "8.0.35"},
		{"8.2.0", "8.1.0"},
		{"8.2.0", "8.3.0"},
		{"8.2.0", "8.4.0"},
		{"8.2.0", "9.0"},
		//
		// MySQL 8.3.x, PXB 8.3.x.
		{"8.3.0", "8.0.35"},
		{"8.3.0", "8.2.0"},
		{"8.3.0", "8.4.0"},
		{"8.3.0", "9.0"},
		//
		// MySQL [8.4.0; 8.5.0), PXB [8.4.0; 8.5.0)
		{"8.4.0", "8.0.35"},
		{"8.4.5-5.1", "8.0.35"},
		{"8.4.6", "8.3.0"},
		{"8.4.6", "8.5.0"},
		{"8.4.6", "9.0"},
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
		require.NoError(t, err)
		assert.True(t, actual, "mysql version %q, xtrabackup version %q", ver.mysql, ver.pxb)
	}

	for _, ver := range incompatible {
		actual, err := mysqlAndXtrabackupCompatible(ver.mysql, ver.pxb)
		require.NoError(t, err)
		assert.False(t, actual, "mysql version %q, xtrabackup version %q", ver.mysql, ver.pxb)
	}

	_, err := mysqlAndXtrabackupCompatible("eight", "8.0.6")
	require.Error(t, err)

	_, err = mysqlAndXtrabackupCompatible("8.0", "eight")
	require.Error(t, err)
}

func TestMysqlAndXtrabackupCompatibilityError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		mysql      string
		pxb        string
		errMessage string
	}{
		{
			name:  "compatible",
			mysql: "8.4.6",
			pxb:   "8.4.0",
		},
		{
			name:       "invalid mysql version",
			mysql:      "eight",
			pxb:        "8.0.6",
			errMessage: "malformed version",
		},
		{
			name:       "invalid xtrabackup version",
			mysql:      "8.0",
			pxb:        "eight",
			errMessage: "malformed version",
		},
		{
			name:       "mysql 8.4 requires xtrabackup 8.4",
			mysql:      "8.4.6",
			pxb:        "8.3.0",
			errMessage: "use Percona XtraBackup 8.4.x for MySQL 8.4.x",
		},
		{
			name:       "mysql 8.3 requires xtrabackup 8.3",
			mysql:      "8.3.0",
			pxb:        "8.4.0",
			errMessage: "use Percona XtraBackup 8.3.x for MySQL 8.3.x",
		},
		{
			name:       "mysql 8.2 requires xtrabackup 8.2",
			mysql:      "8.2.0",
			pxb:        "8.3.0",
			errMessage: "use Percona XtraBackup 8.2.x for MySQL 8.2.x",
		},
		{
			name:       "mysql 8.1 requires xtrabackup 8.1",
			mysql:      "8.1.0",
			pxb:        "8.2.0",
			errMessage: "use Percona XtraBackup 8.1.x for MySQL 8.1.x",
		},
		{
			name:       "mysql 8.0.34+ requires universal 8.0 xtrabackup",
			mysql:      "8.0.44",
			pxb:        "8.0.33",
			errMessage: "use Percona XtraBackup 8.0.34 or newer 8.0.x for MySQL 8.0.34+",
		},
		{
			name:       "mysql 8.0 aligned rejects older xtrabackup",
			mysql:      "8.0.28",
			pxb:        "8.0.27",
			errMessage: "older than MySQL",
		},
		{
			name:       "mysql 8.0 aligned rejects non 8.0 xtrabackup",
			mysql:      "8.0.28",
			pxb:        "8.1.0",
			errMessage: "use Percona XtraBackup 8.0.x for MySQL 8.0.x",
		},
		{
			name:       "legacy mysql rejects unsupported xtrabackup",
			mysql:      "8.0.20",
			pxb:        "8.0.11",
			errMessage: "install a Percona XtraBackup version supported for this MySQL version",
		},
		{
			name:       "unsupported mysql",
			mysql:      "9.0",
			pxb:        "9.0",
			errMessage: "PMM does not support Percona XtraBackup",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := mysqlAndXtrabackupCompatibilityError(test.mysql, test.pxb)
			if test.errMessage == "" {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			require.Contains(t, err.Error(), test.errMessage)
		})
	}
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
				require.NoError(t, err)
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
		name      string
		input     map[models.SoftwareName]string
		err       error
		errString string
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
			name: "successful with mysql 8.4 and xtrabackup 8.4",
			input: map[models.SoftwareName]string{
				models.SoftwareName("mysqld"):     "8.4.5-5.1",
				models.SoftwareName("xtrabackup"): "8.4.0-4",
				models.SoftwareName("xbcloud"):    "8.4.0-4",
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
			err:       ErrIncompatibleXtrabackup,
			errString: "older than MySQL",
		},
		{
			name: "incompatible xtrabackup family",
			input: map[models.SoftwareName]string{
				models.SoftwareName("mysqld"):     "8.4.5-5.1",
				models.SoftwareName("xtrabackup"): "8.0.35-34",
				models.SoftwareName("xbcloud"):    "8.0.35-34",
				models.SoftwareName("qpress"):     "1.1",
			},
			err:       ErrIncompatibleXtrabackup,
			errString: "use Percona XtraBackup 8.4.x for MySQL 8.4.x",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := mySQLBackupSoftwareInstalledAndCompatible(test.input)
			if test.err != nil {
				require.ErrorIs(t, err, test.err)
				if test.errString != "" {
					require.Contains(t, err.Error(), test.errString)
				}
			} else {
				require.NoError(t, err)
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
			err: fmt.Errorf("installed pbm version %q, min required pbm version %q: %w", "1.8.0", pbmMinSupportedVersion, ErrIncompatiblePBM),
		},
		{
			name: "pbm not installed",
			input: map[models.SoftwareName]string{
				models.SoftwareName("mongodb"): "6.0.2",
				models.SoftwareName("pbm"):     "",
			},
			err: fmt.Errorf("software %q is not installed: %w", "pbm", ErrIncompatibleService),
		},
		{
			name: "mongod not installed",
			input: map[models.SoftwareName]string{
				models.SoftwareName("mongodb"): "",
				models.SoftwareName("pbm"):     "2.0.1",
			},
			err: fmt.Errorf("software %q is not installed: %w", "mongodb", ErrIncompatibleService),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := mongoDBBackupSoftwareInstalledAndCompatible(test.input)
			if test.err != nil {
				require.Error(t, err)
				require.Equal(t, test.err.Error(), err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
