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
	"errors"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
)

type syncTestServer struct {
	*Server

	db          sqlmock.Sqlmock
	supervisord *mockSupervisordService
	nomad       *mockNomadService
	vmdb        *mockPrometheusService
	vmalert     *mockPrometheusService
	state       *mockAgentsStateUpdater
}

func newSyncTestServer(t *testing.T) *syncTestServer {
	t.Helper()

	sqlDB, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		dbMock.ExpectClose()
		require.NoError(t, sqlDB.Close())
	})

	ts := &syncTestServer{
		db:          dbMock,
		supervisord: &mockSupervisordService{},
		nomad:       &mockNomadService{},
		vmdb:        &mockPrometheusService{},
		vmalert:     &mockPrometheusService{},
		state:       &mockAgentsStateUpdater{},
	}
	ha := &mockHaService{}
	for _, m := range []interface{ Test(mock.TestingT) }{ts.supervisord, ts.nomad, ts.vmdb, ts.vmalert, ts.state, ha} {
		m.Test(t)
	}
	ha.On("Params").Return(&models.HAParams{Enabled: true})

	ts.Server = &Server{
		db:          reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf)),
		vmdb:        ts.vmdb,
		vmalert:     ts.vmalert,
		agentsState: ts.state,
		supervisord: ts.supervisord,
		nomad:       ts.nomad,
		haService:   ha,
		l:           logrus.WithField("component", "server"),
	}

	t.Cleanup(func() {
		ts.supervisord.AssertExpectations(t)
		ts.nomad.AssertExpectations(t)
		ts.vmdb.AssertExpectations(t)
		ts.vmalert.AssertExpectations(t)
		ts.state.AssertExpectations(t)
		require.NoError(t, dbMock.ExpectationsWereMet())
	})

	return ts
}

func (ts *syncTestServer) expectSettings(settings string) {
	ts.db.ExpectQuery("SELECT settings FROM settings").
		WillReturnRows(sqlmock.NewRows([]string{"settings"}).AddRow([]byte(settings)))
}

func (ts *syncTestServer) expectApply() {
	ts.nomad.On("UpdateConfiguration", mock.Anything).Return(nil).Once()
	ts.supervisord.On("UpdateConfiguration", mock.Anything).Return(nil).Once()
	ts.vmdb.On("RequestConfigurationUpdate").Return().Once()
	ts.vmalert.On("RequestConfigurationUpdate").Return().Once()
}

func (ts *syncTestServer) expectOwnAgentStateUpdate() {
	ts.state.On("RequestStateUpdate", mock.Anything, models.PMMServerAgentID).Return().Once()
}

const (
	hr5s  = `{"metrics_resolutions":{"hr":5000000000,"mr":10000000000,"lr":60000000000}}`
	hr10s = `{"metrics_resolutions":{"hr":10000000000,"mr":10000000000,"lr":60000000000}}`
)

func TestSyncReplica(t *testing.T) {
	t.Parallel()

	t.Run("AppliesSettingsNotYetApplied", func(t *testing.T) {
		t.Parallel()
		ts := newSyncTestServer(t)

		ts.expectSettings(hr5s)
		ts.expectSettings(hr5s)
		ts.expectApply()
		ts.expectOwnAgentStateUpdate()
		ts.syncReplica(t.Context())
	})

	t.Run("OnlyResendsOwnAgentStateWhenSettingsAreUnchanged", func(t *testing.T) {
		t.Parallel()
		ts := newSyncTestServer(t)

		ts.expectSettings(hr5s)
		ts.expectApply()
		ts.state.On("UpdateAgentsState", mock.Anything).Return(nil).Once()
		require.NoError(t, ts.UpdateConfigurations(t.Context()))

		ts.expectSettings(hr5s)
		ts.expectOwnAgentStateUpdate()
		ts.syncReplica(t.Context())
	})

	t.Run("AppliesSettingsChangedThroughAnotherReplica", func(t *testing.T) {
		t.Parallel()
		ts := newSyncTestServer(t)

		ts.expectSettings(hr5s)
		ts.expectApply()
		ts.state.On("UpdateAgentsState", mock.Anything).Return(nil).Once()
		require.NoError(t, ts.UpdateConfigurations(t.Context()))

		ts.expectSettings(hr10s)
		ts.expectSettings(hr10s)
		ts.expectApply()
		ts.expectOwnAgentStateUpdate()
		ts.syncReplica(t.Context())
	})

	t.Run("RetriesFailedApplyAndStillResendsOwnAgentState", func(t *testing.T) {
		t.Parallel()
		ts := newSyncTestServer(t)

		ts.expectSettings(hr5s)
		ts.expectSettings(hr5s)
		ts.nomad.On("UpdateConfiguration", mock.Anything).Return(nil).Once()
		ts.supervisord.On("UpdateConfiguration", mock.Anything).Return(errors.New("supervisorctl failed")).Once()
		ts.expectOwnAgentStateUpdate()
		ts.syncReplica(t.Context())

		ts.expectSettings(hr5s)
		ts.expectSettings(hr5s)
		ts.expectApply()
		ts.expectOwnAgentStateUpdate()
		ts.syncReplica(t.Context())
	})
}
