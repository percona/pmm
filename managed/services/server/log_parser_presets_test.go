// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	serverv1 "github.com/percona/pmm/api/server/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

func TestChangeLogParserPreset(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)

	var supervisord mockSupervisordService
	supervisord.Test(t)
	supervisord.On("UpdateConfiguration", mock.Anything).Return(nil).Once()
	var nomad mockNomadService
	nomad.Test(t)
	nomad.On("UpdateConfiguration", mock.Anything).Return(nil)
	var vmdb mockPrometheusService
	vmdb.Test(t)
	vmdb.On("RequestConfigurationUpdate").Return(nil)
	var vmalert mockPrometheusService
	vmalert.Test(t)
	vmalert.On("RequestConfigurationUpdate").Return(nil)
	var agentsState mockAgentsStateUpdater
	agentsState.Test(t)
	agentsState.On("UpdateAgentsState", mock.Anything).Return(nil).Once()
	t.Cleanup(func() {
		supervisord.AssertExpectations(t)
		agentsState.AssertExpectations(t)
	})

	s, err := NewServer(&Params{
		DB:                 reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf)),
		VMDB:               &vmdb,
		VMAlert:            &vmalert,
		AgentsStateUpdater: &agentsState,
		Supervisord:        &supervisord,
		Nomad:              &nomad,
	})
	require.NoError(t, err)

	// The server collector reads PMM Server's own logs with the presets, so a change re-renders its config too.
	res, err := s.ChangeLogParserPreset(t.Context(), &serverv1.ChangeLogParserPresetRequest{
		Id:          "postgres",
		Description: new("PostgreSQL log"),
	})
	require.NoError(t, err)
	assert.Equal(t, "PostgreSQL log", res.Preset.Description)
}
