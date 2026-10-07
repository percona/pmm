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

package server

import (
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
)

func TestSyncReplica(t *testing.T) {
	sqlDB, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		dbMock.ExpectClose()
		require.NoError(t, sqlDB.Close())
	})
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	expectSettings := func(settings string) {
		dbMock.ExpectQuery("SELECT settings FROM settings").
			WillReturnRows(sqlmock.NewRows([]string{"settings"}).AddRow([]byte(settings)))
	}

	supervisord := &mockSupervisordService{}
	supervisord.Test(t)
	nomad := &mockNomadService{}
	nomad.Test(t)
	vmdb := &mockPrometheusService{}
	vmdb.Test(t)
	vmalert := &mockPrometheusService{}
	vmalert.Test(t)
	state := &mockAgentsStateUpdater{}
	state.Test(t)
	ha := &mockHaService{}
	ha.Test(t)
	ha.On("Params").Return(&models.HAParams{Enabled: true})

	s := &Server{
		db:          db,
		vmdb:        vmdb,
		vmalert:     vmalert,
		agentsState: state,
		supervisord: supervisord,
		nomad:       nomad,
		haService:   ha,
		l:           logrus.WithField("component", "server"),
	}

	expectApply := func() {
		supervisord.On("UpdateConfiguration", mock.Anything).Return(nil).Once()
		nomad.On("UpdateConfiguration", mock.Anything).Return(nil).Once()
		vmdb.On("RequestConfigurationUpdate").Return().Once()
		vmalert.On("RequestConfigurationUpdate").Return().Once()
		state.On("UpdateAgentsState", mock.Anything).Return(nil).Once()
	}
	assertAll := func(t *testing.T) {
		t.Helper()
		supervisord.AssertExpectations(t)
		nomad.AssertExpectations(t)
		vmdb.AssertExpectations(t)
		vmalert.AssertExpectations(t)
		state.AssertExpectations(t)
		require.NoError(t, dbMock.ExpectationsWereMet())
	}

	hr5s := `{"metrics_resolutions":{"hr":5000000000,"mr":10000000000,"lr":60000000000}}`
	hr10s := `{"metrics_resolutions":{"hr":10000000000,"mr":10000000000,"lr":60000000000}}`

	t.Run("AppliesSettingsNotYetApplied", func(t *testing.T) {
		expectSettings(hr5s)
		expectSettings(hr5s)
		expectApply()

		s.syncReplica(t.Context())

		assertAll(t)
	})

	t.Run("ResendsOwnAgentStateWhenSettingsAreUnchanged", func(t *testing.T) {
		expectSettings(hr5s)
		state.On("RequestStateUpdate", mock.Anything, models.PMMServerAgentID).Return().Once()

		s.syncReplica(t.Context())

		assertAll(t)
	})

	t.Run("AppliesSettingsChangedThroughAnotherReplica", func(t *testing.T) {
		expectSettings(hr10s)
		expectSettings(hr10s)
		expectApply()

		s.syncReplica(t.Context())

		assertAll(t)
	})

	t.Run("DoesNotReapplySettingsAppliedByChangeSettings", func(t *testing.T) {
		expectSettings(hr5s)
		expectApply()
		require.NoError(t, s.UpdateConfigurations(t.Context()))

		expectSettings(hr5s)
		state.On("RequestStateUpdate", mock.Anything, models.PMMServerAgentID).Return().Once()

		s.syncReplica(t.Context())

		assertAll(t)
	})
}
