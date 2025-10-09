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
	agentlocal "github.com/percona/pmm/api/agentlocal/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
)

func TestServerStatus(t *testing.T) {
	setup := func(t *testing.T) ([]*agentlocal.AgentInfo, *mockSupervisor, *mockClient, configGetReloader) {
		t.Helper()
		agentInfo := []*agentlocal.AgentInfo{{
			AgentId:   "00000000-0000-4000-8000-000000000002",
			AgentType: inventoryv1.AgentType_AGENT_TYPE_NODE_EXPORTER,
			Status:    inventoryv1.AgentStatus_AGENT_STATUS_RUNNING,
		}}
		var supervisor mockSupervisor
		supervisor.Test(t)
		supervisor.On("AgentsList").Return(agentInfo)
		client := &mockClient{}
		client.Test(t)
		client.On("GetServerConnectMetadata").Return(&agentv1.ServerConnectMetadata{
			AgentRunsOnNodeID: "00000000-0000-4000-8000-000000000003",
			ServerVersion:     "2.0.0-dev",
		})
		client.On("GetConnectionUpTime").Return(float32(100.00))
		cfgStorage := config.NewStorage(&config.Config{
			ID: "00000000-0000-4000-8000-000000000001",
			Server: config.Server{
				Address:  "127.0.0.1:8443",
				Username: "username",
				Password: "password",
			},
		})
		return agentInfo, &supervisor, client, cfgStorage
	}

	t.Run("without network info", func(t *testing.T) {
		agentInfo, supervisor, client, cfg := setup(t)
		defer supervisor.AssertExpectations(t)
		defer client.AssertExpectations(t)
		logStore := tailog.NewStore(500)
		s := NewServer(cfg, supervisor, client, "/some/dir/pmm-agent.yaml", logStore)

		// without network info
		actual, err := s.Status(context.Background(), &agentlocal.StatusRequest{GetNetworkInfo: false})
		require.NoError(t, err)
		expected := &agentlocal.StatusResponse{
			AgentId:      "00000000-0000-4000-8000-000000000001",
			RunsOnNodeId: "00000000-0000-4000-8000-000000000003",
			ServerInfo: &agentlocal.ServerInfo{
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
		actual, err := s.Status(context.Background(), &agentlocal.StatusRequest{GetNetworkInfo: true})
		require.NoError(t, err)
		expected := &agentlocal.StatusResponse{
			AgentId:      "00000000-0000-4000-8000-000000000001",
			RunsOnNodeId: "00000000-0000-4000-8000-000000000003",
			ServerInfo: &agentlocal.ServerInfo{
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
	setup := func(t *testing.T) ([]*agentlocal.AgentInfo, *mockSupervisor, *mockClient, configGetReloader) {
		t.Helper()
		agentInfo := []*agentlocal.AgentInfo{{
			AgentId:   "00000000-0000-4000-8000-000000000002",
			AgentType: inventoryv1.AgentType_AGENT_TYPE_NODE_EXPORTER,
			Status:    inventoryv1.AgentStatus_AGENT_STATUS_RUNNING,
		}}
		var supervisor mockSupervisor
		supervisor.Test(t)
		supervisor.On("AgentsList").Return(agentInfo)
		agentLogs := make(map[string][]string)
		// Use agent ID as key (matches real behavior)
		agentLogs["00000000-0000-4000-8000-000000000002"] = []string{
			"logs1",
			"logs2",
		}
		// Add another agent to test filtering
		agentLogs["00000000-0000-4000-8000-000000000099"] = []string{
			"other agent logs1",
			"other agent logs2",
		}
		supervisor.On("AgentsLogs").Return(agentLogs)
		var client mockClient
		client.Test(t)
		client.On("GetServerConnectMetadata").Return(&agentv1.ServerConnectMetadata{
			AgentRunsOnNodeID: "00000000-0000-4000-8000-000000000003",
			ServerVersion:     "2.0.0-dev",
		})
		client.On("GetConnectionUpTime").Return(float32(100.00))

		cfgStorage := config.NewStorage(&config.Config{
			ID: "00000000-0000-4000-8000-000000000001",
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
		_, err := s.Status(context.Background(), &agentlocal.StatusRequest{GetNetworkInfo: false})
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
				if ex.Name == pmmAgentZipFile {
					assert.Empty(t, contents)
				} else {
					assert.NotEmpty(t, contents)
				}
			}
		}
	})

	t.Run("test zip file with agent_id filter", func(t *testing.T) {
		agentInfo, supervisor, client, cfg := setup(t)
		defer supervisor.AssertExpectations(t)
		defer client.AssertExpectations(t)
		logStore := tailog.NewStore(10)
		s := NewServer(cfg, supervisor, client, "/some/dir/pmm-agent.yaml", logStore)
		_, err := s.Status(context.Background(), &agentlocal.StatusRequest{GetNetworkInfo: false})
		require.NoError(t, err)

		// Test with agent_id query parameter using actual agent ID
		agentID := agentInfo[0].AgentId
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/logs.zip?agent_id="+agentID, nil)
		s.ZipLogs(rec, req)
		existFile, err := io.ReadAll(rec.Body)
		require.NoError(t, err)

		bufExs := bytes.NewReader(existFile)
		zipExs, err := zip.NewReader(bufExs, bufExs.Size())
		require.NoError(t, err)

		// When filtering by agent_id, we should get:
		// 1. The specific agent's log file (agent ID.log)
		// 2. pmm-agent.log (always included)
		// 3. NOT other agents' logs
		// Count of files should be exactly 2
		assert.Equal(t, 2, len(zipExs.File), "Should contain exactly 2 files: agent log + pmm-agent.log")

		foundAgentLog := false
		foundPmmAgentLog := false
		foundOtherAgentLog := false

		for _, ex := range zipExs.File {
			switch ex.Name {
			case agentID + ".log":
				foundAgentLog = true
			case pmmAgentZipFile:
				foundPmmAgentLog = true
			case "00000000-0000-4000-8000-000000000099.log":
				// This is the other agent that should NOT be included
				foundOtherAgentLog = true
			}
		}

		assert.True(t, foundAgentLog, "Should contain "+agentID+".log")
		assert.True(t, foundPmmAgentLog, "Should always contain pmm-agent.log")
		assert.False(t, foundOtherAgentLog, "Should NOT contain other agent's logs when filtering")
	})

	t.Run("test zip file without agent_id filter includes all logs", func(t *testing.T) {
		_, supervisor, client, cfg := setup(t)
		defer supervisor.AssertExpectations(t)
		defer client.AssertExpectations(t)
		logStore := tailog.NewStore(10)
		s := NewServer(cfg, supervisor, client, "/some/dir/pmm-agent.yaml", logStore)
		_, err := s.Status(context.Background(), &agentlocal.StatusRequest{GetNetworkInfo: false})
		require.NoError(t, err)

		// Test without agent_id query parameter (should get all logs)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/logs.zip", nil)
		s.ZipLogs(rec, req)
		existFile, err := io.ReadAll(rec.Body)
		require.NoError(t, err)

		bufExs := bytes.NewReader(existFile)
		zipExs, err := zip.NewReader(bufExs, bufExs.Size())
		require.NoError(t, err)

		// Without filter, we should get all agent logs + pmm-agent.log
		// We have 2 agents in mock + pmm-agent.log = 3 files
		fileCount := len(zipExs.File)
		assert.Equal(t, 3, fileCount, "Should contain all logs: 2 agents + pmm-agent.log")

		foundPmmAgentLog := false
		foundFirstAgent := false
		foundSecondAgent := false

		for _, ex := range zipExs.File {
			switch ex.Name {
			case pmmAgentZipFile:
				foundPmmAgentLog = true
			case "00000000-0000-4000-8000-000000000002.log":
				foundFirstAgent = true
			case "00000000-0000-4000-8000-000000000099.log":
				foundSecondAgent = true
			}
		}

		assert.True(t, foundPmmAgentLog, "Should always contain pmm-agent.log")
		assert.True(t, foundFirstAgent, "Should contain first agent's log")
		assert.True(t, foundSecondAgent, "Should contain second agent's log")
	})
}
