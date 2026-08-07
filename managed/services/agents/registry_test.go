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
	"context"
	"net/url"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/utils/logger"
)

func TestRegistryIsConnectedUsesInMemoryStateWhenHAIsDisabled(t *testing.T) {
	t.Parallel()

	r := NewRegistry(nil, fakeVictoriaMetricsParams{}, &fakeHAService{params: &models.HAParams{Enabled: false}})
	r.agentsCache.Set("agent-connected", pmmAgentInfo{id: "agent-connected"})

	assert.True(t, r.IsConnected("agent-connected"))
	assert.False(t, r.IsConnected("agent-missing"))
}

func TestRegistryIsConnectedUsesFreshHACacheWithoutDatabaseLookup(t *testing.T) {
	t.Parallel()

	r := NewRegistry(nil, fakeVictoriaMetricsParams{}, &fakeHAService{params: &models.HAParams{Enabled: true}})
	r.connectionCache["agent-connected"] = struct{}{}
	r.connectionCacheTTL = time.Now().Add(time.Minute)

	assert.True(t, r.IsConnected("agent-connected"))
}

func TestRegistryIsConnectedRebuildsCacheFromDatabaseInHAMode(t *testing.T) {
	t.Parallel()

	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = mock.ExpectClose()
		assert.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	r := NewRegistry(db, fakeVictoriaMetricsParams{}, &fakeHAService{params: &models.HAParams{Enabled: true}})
	r.connectionCacheTTL = time.Now().Add(-time.Second)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT .+ FROM "agents"`).WillReturnRows(newAgentRows(
		agentRow{id: "agent-connected", connected: true},
		agentRow{id: "agent-disconnected", connected: false},
	))
	mock.ExpectCommit()

	assert.True(t, r.IsConnected("agent-connected"))
	assert.False(t, r.IsConnected("agent-disconnected"))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRegistryIsConnectedReturnsFalseWhenHACacheRebuildFails(t *testing.T) {
	t.Parallel()

	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = mock.ExpectClose()
		assert.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	r := NewRegistry(db, fakeVictoriaMetricsParams{}, &fakeHAService{params: &models.HAParams{Enabled: true}})
	r.connectionCache["agent-stale"] = struct{}{}
	r.connectionCacheTTL = time.Now().Add(-time.Second)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT .+ FROM "agents"`).WillReturnError(assert.AnError)
	mock.ExpectRollback()

	assert.False(t, r.IsConnected("agent-stale"))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRegistryGetReturnsConnectedAgentAndMissingAgentError(t *testing.T) {
	t.Parallel()

	r := NewRegistry(nil, fakeVictoriaMetricsParams{}, &fakeHAService{params: &models.HAParams{Enabled: false}})
	r.agentsCache.Set("agent-connected", pmmAgentInfo{id: "agent-connected"})

	agent, err := r.get("agent-connected")
	require.NoError(t, err)
	assert.Equal(t, "agent-connected", agent.id)

	_, err = r.get("agent-missing")
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestRegistryKickRemovesAgentAndClosesKickChannel(t *testing.T) {
	t.Parallel()

	r := NewRegistry(nil, fakeVictoriaMetricsParams{}, &fakeHAService{params: &models.HAParams{Enabled: false}})
	kickCh := make(chan struct{})
	r.agentsCache.Set("agent-1", pmmAgentInfo{id: "agent-1", kickChan: kickCh})
	ctx := logger.Set(context.Background(), "test-request")

	r.Kick(ctx, "agent-1")

	_, exists := r.agentsCache.Get("agent-1")
	assert.False(t, exists)

	select {
	case <-kickCh:
	default:
		t.Fatal("kick channel should be closed")
	}
}

func TestRegistryKickAllDisconnectsEveryRegisteredAgent(t *testing.T) {
	t.Parallel()

	r := NewRegistry(nil, fakeVictoriaMetricsParams{}, &fakeHAService{params: &models.HAParams{Enabled: false}})
	kickCh1 := make(chan struct{})
	kickCh2 := make(chan struct{})
	r.agentsCache.Set("agent-1", pmmAgentInfo{id: "agent-1", kickChan: kickCh1})
	r.agentsCache.Set("agent-2", pmmAgentInfo{id: "agent-2", kickChan: kickCh2})
	ctx := logger.Set(context.Background(), "test-request")

	r.KickAll(ctx)

	assert.EqualValues(t, 0, r.agentsCache.Size())

	select {
	case <-kickCh1:
	default:
		t.Fatal("first kick channel should be closed")
	}

	select {
	case <-kickCh2:
	default:
		t.Fatal("second kick channel should be closed")
	}
}

type fakeHAService struct {
	params *models.HAParams
}

func (s *fakeHAService) Params() *models.HAParams {
	return s.params
}

type fakeVictoriaMetricsParams struct{}

func (fakeVictoriaMetricsParams) ExternalVM() bool {
	return false
}

func (fakeVictoriaMetricsParams) URLFor(_ string) (*url.URL, error) {
	return new(url.URL), nil
}

func (fakeVictoriaMetricsParams) URL() string {
	return ""
}

func (fakeVictoriaMetricsParams) VMAgentArgs() []string {
	return nil
}

type agentRow struct {
	id        string
	connected bool
}

func newAgentRows(agents ...agentRow) *sqlmock.Rows {
	rows := sqlmock.NewRows([]string{
		"agent_id", "agent_type", "runs_on_node_id", "service_id", "node_id",
		"pmm_agent_id", "custom_labels", "environment_variables", "created_at", "updated_at",
		"disabled", "status", "listen_port", "version", "process_exec_path", "is_connected",
		"username", "password", "agent_password", "tls", "tls_skip_verify",
		"log_level", "exporter_options", "qan_options", "rta_options",
		"aws_options", "azure_options", "mongo_options", "mysql_options", "postgresql_options", "valkey_options",
	})

	now := time.Now()
	for _, a := range agents {
		rows.AddRow(
			a.id,
			string(models.PMMAgentType),
			"node-1",
			nil,
			nil,
			nil,
			nil,
			nil,
			now,
			now,
			false,
			"",
			nil,
			nil,
			nil,
			a.connected,
			nil,
			nil,
			nil,
			false,
			false,
			nil,
			`{}`,
			`{}`,
			`{}`,
			`{}`,
			`{}`,
			`{}`,
			`{}`,
			`{}`,
			`{}`,
		)
	}

	return rows
}
