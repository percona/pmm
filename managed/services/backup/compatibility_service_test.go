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
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/agents"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/stretchr/testify/mock"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"
	"testing"
)

func TestCheckCompatibility(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	versioner := &mockVersioner{}
	cSvc := NewCompatibilityService(db, versioner)

	for _, test := range []struct {
		versions      []agents.Version
		expectedError error
	}{
		{
			versions: []agents.Version{
				{Version: "8.0.25"},
				{Version: ""},
				{Version: ""},
				{Version: "1.1"},
			},
			expectedError: ErrXtrabackupNotInstalled,
		},
		{
			versions: []agents.Version{
				{Version: "8.0.25"},
				{Version: "8.0.24"},
				{Version: "8.0.25"},
				{Version: "1.1"},
			},
			expectedError: ErrInvalidXtrabackup,
		},
		{
			versions: []agents.Version{
				{Version: "8.0.25"},
				{Version: "8.0.24"},
				{Version: "8.0.24"},
				{Version: "1.1"},
			},
			expectedError: ErrIncompatibleXtrabackup,
		},
	} {
		versioner.On("GetVersions", mock.Anything, mock.Anything).Return(test, nil).Once()
		dbVersion, err := cSvc.checkCompatibility(&models.Service{ServiceType: models.MySQLServiceType}, &models.Agent{})
	}

}
