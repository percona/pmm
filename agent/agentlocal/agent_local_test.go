// pmm-agent
// Copyright 2019 Percona LLC
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
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/percona/pmm/agent/config"
	"github.com/percona/pmm/agent/storelogs"
	"github.com/percona/pmm/api/agentlocalpb"
	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
)

func TestServerStatus(t *testing.T) {
	setup := func(t *testing.T) ([]*agentlocalpb.AgentInfo, *mockSupervisor, *mockClient, *config.Config) {
		agentInfo := []*agentlocalpb.AgentInfo{{
			AgentId:   "/agent_id/00000000-0000-4000-8000-000000000002",
			AgentType: inventorypb.AgentType_NODE_EXPORTER,
			Status:    inventorypb.AgentStatus_RUNNING,
		}}
		var supervisor mockSupervisor
		supervisor.Test(t)
		supervisor.On("AgentsList").Return(agentInfo)
		var client mockClient
		client.Test(t)
		client.On("GetServerConnectMetadata").Return(&agentpb.ServerConnectMetadata{
			AgentRunsOnNodeID: "/node_id/00000000-0000-4000-8000-000000000003",
			ServerVersion:     "2.0.0-dev",
		})
		cfg := &config.Config{
			ID: "/agent_id/00000000-0000-4000-8000-000000000001",
			Server: config.Server{
				Address:  "127.0.0.1:8443",
				Username: "username",
				Password: "password",
			},
		}
		return agentInfo, &supervisor, &client, cfg
	}

	t.Run("without network info", func(t *testing.T) {
		agentInfo, supervisor, client, cfg := setup(t)
		defer supervisor.AssertExpectations(t)
		defer client.AssertExpectations(t)
		ringLog := storelogs.New(500)
		s := NewServer(cfg, supervisor, client, "/some/dir/pmm-agent.yaml", ringLog)

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
			AgentsInfo:     agentInfo,
			ConfigFilepath: "/some/dir/pmm-agent.yaml",
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
		ringLog := storelogs.New(500)
		s := NewServer(cfg, supervisor, client, "/some/dir/pmm-agent.yaml", ringLog)

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
			AgentsInfo:     agentInfo,
			ConfigFilepath: "/some/dir/pmm-agent.yaml",
		}
		assert.Equal(t, expected, actual)
	})
}

func TestGetZipFile(t *testing.T) {
	setup := func(t *testing.T) ([]*agentlocalpb.AgentInfo, *mockSupervisor, *mockClient, *config.Config) {
		agentInfo := []*agentlocalpb.AgentInfo{{
			AgentId:   "/agent_id/00000000-0000-4000-8000-000000000002",
			AgentType: inventorypb.AgentType_NODE_EXPORTER,
			Status:    inventorypb.AgentStatus_RUNNING,
		}}
		var supervisor mockSupervisor
		supervisor.Test(t)
		supervisor.On("AgentsList").Return(agentInfo)
		agentLogs := make(map[string][]string)
		agentLogs[inventorypb.AgentType_NODE_EXPORTER.String()] = []string{
			"logs1",
			"logs2",
		}
		supervisor.On("AgentsLogs").Return(agentLogs)
		var client mockClient
		client.Test(t)
		client.On("GetServerConnectMetadata").Return(&agentpb.ServerConnectMetadata{
			AgentRunsOnNodeID: "/node_id/00000000-0000-4000-8000-000000000003",
			ServerVersion:     "2.0.0-dev",
		})
		cfg := &config.Config{
			ID: "/agent_id/00000000-0000-4000-8000-000000000001",
			Server: config.Server{
				Address:  "127.0.0.1:8443",
				Username: "username",
				Password: "password",
			},
		}
		return agentInfo, &supervisor, &client, cfg
	}

	t.Run("test zip file", func(t *testing.T) {
		_, supervisor, client, cfg := setup(t)
		defer supervisor.AssertExpectations(t)
		defer client.AssertExpectations(t)
		ringLog := storelogs.New(10)
		s := NewServer(cfg, supervisor, client, "/some/dir/pmm-agent.yaml", ringLog)
		_, err := s.Status(context.Background(), &agentlocalpb.StatusRequest{GetNetworkInfo: false})
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/logs.zip", nil)
		s.Zip(rec, req)
		existFile, err := ioutil.ReadAll(rec.Body)
		require.NoError(t, err)

		expectedFile, err := generateTestZip(s)
		require.NoError(t, err)
		bufExp := bytes.NewReader(expectedFile)
		bufExs := bytes.NewReader(existFile)

		zipExp, err := zip.NewReader(bufExp, bufExp.Size())
		require.NoError(t, err)
		zipExs, err := zip.NewReader(bufExp, bufExs.Size())
		require.NoError(t, err)

		for i, ex := range zipExp.File {
			assert.Equal(t, ex.Name, zipExs.File[i].Name)
			deepCompare(t, ex, zipExs.File[i])
		}
		require.NoError(t, err)
		assert.Equal(t, expectedFile, existFile)
	})
}

// generateTestZip generate test zip file.
func generateTestZip(s *Server) ([]byte, error) {
	agentLogs := make(map[string][]string)
	agentLogs[inventorypb.AgentType_NODE_EXPORTER.String()] = []string{
		"logs1",
		"logs2",
	}
	buf := &bytes.Buffer{}
	writer := zip.NewWriter(buf)
	b := &bytes.Buffer{}
	for _, serverLog := range s.ringLogs.GetLogs() {
		_, err := b.WriteString(serverLog)
		if err != nil {
			return nil, err
		}
	}
	addData(writer, "pmm-agent.txt", b.Bytes())

	for id, logs := range agentLogs {
		b := &bytes.Buffer{}
		for _, l := range logs {
			_, err := b.WriteString(l + "\n")
			if err != nil {
				return nil, err
			}
		}
		addData(writer, fmt.Sprintf("%s.txt", id), b.Bytes())
	}
	err := writer.Close()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// deepCompare compare two zip files.
func deepCompare(t *testing.T, file1, file2 *zip.File) bool {
	sf, err := file1.Open()
	if err != nil {
		log.Fatal(err)
	}

	df, err := file2.Open()
	if err != nil {
		log.Fatal(err)
	}

	sscan := bufio.NewScanner(sf)
	dscan := bufio.NewScanner(df)

	for sscan.Scan() {
		dscan.Scan()
		assert.Equal(t, sscan.Bytes(), dscan.Bytes())
	}

	return false
}
