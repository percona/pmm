// pmm-managed
// Copyright (C) 2017 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package rds

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRDSExporterMarshal(t *testing.T) {
	cfg := &rdsExporterConfig{
		Instances: []rdsExporterInstance{
			{
				Region:        "us-east-1",
				Instance:      "rds-aurora1",
				DBIResourceID: pointer.ToString("db-P5QCHK64NWDD5BLLBVT5NPQS2Q"),
				Type:          auroraMySQL,
			},
			{
				Region:        "us-east-1",
				Instance:      "rds-aurora57",
				DBIResourceID: pointer.ToString("db-QYBVFI3OW6VME5FQNM2URY4ZOY"),
				Type:          auroraMySQL,
				AWSAccessKey:  pointer.ToString("AKIAIOSFODNN7EXAMPLE"),
				AWSSecretKey:  pointer.ToString("wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"),
			},
			{
				Region:        "us-east-1",
				Instance:      "rds-mysql56",
				DBIResourceID: pointer.ToString("db-J6JH3LJAWBZ6MXDDWYRG4RRJ6A"),
				Type:          mySQL,
			},
			{
				Region:        "us-east-1",
				Instance:      "rds-mysql57",
				DBIResourceID: pointer.ToString("db-FE4Y2GIJU6UADBOXKULV3DBATY"),
				Type:          mySQL,
				AWSAccessKey:  pointer.ToString("AKIAIOSFODNN7EXAMPLE"),
				AWSSecretKey:  pointer.ToString("wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"),
			},
		},
	}

	expected, err := ioutil.ReadFile("../../testdata/rds_exporter/rds_exporter.yml")
	require.NoError(t, err)
	actual, err := cfg.Marshal()
	require.NoError(t, err)
	assert.Equal(t, strings.Split(string(expected), "\n"), strings.Split(string(actual), "\n"))
}
