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

package agents

import (
	"testing"

	"github.com/stretchr/testify/assert"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	"github.com/percona/pmm/managed/models"
)

func TestSoftwareName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		sw   Software
		name models.SoftwareName
	}{
		{&Mysqld{}, models.MysqldSoftwareName},
		{&Xtrabackup{}, models.XtrabackupSoftwareName},
		{&Xbcloud{}, models.XbcloudSoftwareName},
		{&Qpress{}, models.QpressSoftwareName},

		{&MongoDB{}, models.MongoDBSoftwareName},
		{&PBM{}, models.PBMSoftwareName},
	}

	for _, tc := range tests {
		t.Run(string(tc.name), func(t *testing.T) {
			t.Parallel()
			res := tc.sw.Name()
			assert.Equal(t, tc.name, res)
		})
	}
}

func TestGetVersionRequest(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		sw      Software
		request *agentv1.GetVersionsRequest_Software
	}{
		{"mysql", &Mysqld{}, &agentv1.GetVersionsRequest_Software{Software: &agentv1.GetVersionsRequest_Software_Mysqld{}}},
		{"xtrabackup", &Xtrabackup{}, &agentv1.GetVersionsRequest_Software{Software: &agentv1.GetVersionsRequest_Software_Xtrabackup{}}},
		{"xbcloud", &Xbcloud{}, &agentv1.GetVersionsRequest_Software{Software: &agentv1.GetVersionsRequest_Software_Xbcloud{}}},
		{"qpress", &Qpress{}, &agentv1.GetVersionsRequest_Software{Software: &agentv1.GetVersionsRequest_Software_Qpress{}}},

		{"mongodb", &MongoDB{}, &agentv1.GetVersionsRequest_Software{Software: &agentv1.GetVersionsRequest_Software_Mongod{}}},
		{"pbm", &PBM{}, &agentv1.GetVersionsRequest_Software{Software: &agentv1.GetVersionsRequest_Software_Pbm{}}},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			res := tc.sw.GetVersionRequest()
			assert.Equal(t, tc.request, res)
		})
	}
}
