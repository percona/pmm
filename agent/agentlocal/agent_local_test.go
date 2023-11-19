// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package agentlocal

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/percona/pmm/agent/config"
	"github.com/percona/pmm/agent/tailog"
	agentv1 "github.com/percona/pmm/api/agent/v1"
	agentlocalpb "github.com/percona/pmm/api/agentlocalpb/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
)

func TestServerStatus(t *testing.T) {
	setup := func(t *testing.T) ([]*agentlocalpb.AgentInfo, *mockSupervisor, *mockClient, configGetReloader) {
		t.Helper()
		agentInfo := []*agentlocalpb.AgentInfo{{
			AgentId:   "/agent_id/00000000-0000-4000-8000-000000000002",
			AgentType: inventoryv1.AgentType_AGENT_TYPE_NODE_EXPORTER,
			Status:    inventoryv1.AgentStatus_AGENT_STATUS_RUNNING,
		}}
		var supervisor mockSupervisor
		supervisor.Test(t)
		supervisor.On("AgentsList").Return(agentInfo)
		var client mockClient
		client.Test(t)
		client.On("GetServerConnectMetadata").Return(&agentv1.ServerConnectMetadata{
			AgentRunsOnNodeID: "/node_id/00000000-0000-4000-8000-000000000003",
			ServerVersion:     "2.0.0-dev",
		})
		client.On("GetConnectionUpTime").Return(float32(100.00))
		cfgStorage := config.NewStorage(&config.Config{
			ID: "/agent_id/00000000-0000-4000-8000-000000000001",
			Server: config.Server{
				Address:  "127.0.0.1:8443",
				Username: "username",
				Password: "password",
			},
		})
		return agentInfo, &supervisor, &client, cfgStorage
	}

	t.Run("without network info", func(t *testing.T) {
		agentInfo, supervisor, client, cfg := setup(t)
		defer supervisor.AssertExpectations(t)
		defer client.AssertExpectations(t)
		logStore := tailog.NewStore(500)
		s := NewServer(cfg, supervisor, client, "/some/dir/pmm-agent.yaml", logStore)

		// without network info
		actual, err := s.Status(context.Background(), &agentlocalpb.StatusRequest{GetNetworkInfo: false})
		require.NoError(t, err)
		expected := &agentlocalpb.StatusResponse{
			AgentId:      "/agent_id/00000000-0000-4000-8000-000000000001",
			RunsOnNodeId: "/node_id/00000000-0000-4000-8000-000000000003",
			ServerInfo: &agentlocalpb.ServerInfo{
				Url:       "https://username:password@127.0.0.1:8443/",
				Version:   "2.0.0-dev",
				Connected: true,
			},
			AgentsInfo:       agentInfo,
			ConnectionUptime: 100.00,
			ConfigFilepath:   "/some/dir/pmm-agent.yaml",
		}
		assert.Equal(t, expected, actual)
	})

	t.Run("with network info", func(t *testing.T) {
		agentInfo, supervisor, client, cfg := setup(t)
		latency := 5 * time.Millisecond
		clockDrift := time.Second
		client.On("GetNetworkInformation").Return(latency, clockDrift, nil)
		defer supervisor.AssertExpectations(t)
		defer client.AssertExpectations(t)
		logStore := tailog.NewStore(500)
		s := NewServer(cfg, supervisor, client, "/some/dir/pmm-agent.yaml", logStore)

		// with network info
		actual, err := s.Status(context.Background(), &agentlocalpb.StatusRequest{GetNetworkInfo: true})
		require.NoError(t, err)
		expected := &agentlocalpb.StatusResponse{
			AgentId:      "/agent_id/00000000-0000-4000-8000-000000000001",
			RunsOnNodeId: "/node_id/00000000-0000-4000-8000-000000000003",
			ServerInfo: &agentlocalpb.ServerInfo{
				Url:        "https://username:password@127.0.0.1:8443/",
				Version:    "2.0.0-dev",
				Latency:    durationpb.New(latency),
				ClockDrift: durationpb.New(clockDrift),
				Connected:  true,
			},
			ConnectionUptime: 100.00,
			AgentsInfo:       agentInfo,
			ConfigFilepath:   "/some/dir/pmm-agent.yaml",
		}
		assert.Equal(t, expected, actual)
	})
}

func TestGetZipFile(t *testing.T) {
	setup := func(t *testing.T) ([]*agentlocalpb.AgentInfo, *mockSupervisor, *mockClient, configGetReloader) {
		t.Helper()
		agentInfo := []*agentlocalpb.AgentInfo{{
			AgentId:   "/agent_id/00000000-0000-4000-8000-000000000002",
			AgentType: inventoryv1.AgentType_AGENT_TYPE_NODE_EXPORTER,
			Status:    inventoryv1.AgentStatus_AGENT_STATUS_RUNNING,
		}}
		var supervisor mockSupervisor
		supervisor.Test(t)
		supervisor.On("AgentsList").Return(agentInfo)
		agentLogs := make(map[string][]string)
		agentLogs[inventoryv1.AgentType_AGENT_TYPE_NODE_EXPORTER.String()] = []string{
			"logs1",
			"logs2",
		}
		supervisor.On("AgentsLogs").Return(agentLogs)
		var client mockClient
		client.Test(t)
		client.On("GetServerConnectMetadata").Return(&agentv1.ServerConnectMetadata{
			AgentRunsOnNodeID: "/node_id/00000000-0000-4000-8000-000000000003",
			ServerVersion:     "2.0.0-dev",
		})
		client.On("GetConnectionUpTime").Return(float32(100.00))

		cfgStorage := config.NewStorage(&config.Config{
			ID: "/agent_id/00000000-0000-4000-8000-000000000001",
			Server: config.Server{
				Address:  "127.0.0.1:8443",
				Username: "username",
				Password: "password",
			},
		})
		return agentInfo, &supervisor, &client, cfgStorage
	}

	t.Run("test zip file", func(t *testing.T) {
		_, supervisor, client, cfg := setup(t)
		defer supervisor.AssertExpectations(t)
		defer client.AssertExpectations(t)
		logStore := tailog.NewStore(10)
		s := NewServer(cfg, supervisor, client, "/some/dir/pmm-agent.yaml", logStore)
		_, err := s.Status(context.Background(), &agentlocalpb.StatusRequest{GetNetworkInfo: false})
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/logs.zip", nil)
		s.ZipLogs(rec, req)
		existFile, err := io.ReadAll(rec.Body)
		require.NoError(t, err)

		bufExs := bytes.NewReader(existFile)
		zipExs, err := zip.NewReader(bufExs, bufExs.Size())
		require.NoError(t, err)

		for _, ex := range zipExs.File {
			file, err := ex.Open()
			require.NoError(t, err)
			if contents, err := io.ReadAll(file); err == nil {
				if ex.Name == serverZipFile {
					assert.Empty(t, contents)
				} else {
					assert.NotEmpty(t, contents)
				}
			}
		}
	})
}
