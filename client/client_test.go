// pmm-agent
// Copyright (C) 2018 Percona LLC
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

package client

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/percona/pmm/api/agentpb"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/percona/pmm-agent/config"
)

type testServer struct {
	connectFunc func(agentpb.Agent_ConnectServer) error
}

func (s *testServer) Connect(stream agentpb.Agent_ConnectServer) error {
	return s.connectFunc(stream)
}

var _ agentpb.AgentServer = (*testServer)(nil)

func setup(t *testing.T, connect func(agentpb.Agent_ConnectServer) error) (port uint16, teardown func()) {
	// logrus.SetLevel(logrus.DebugLevel)

	// start server with given connect handler
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port = uint16(lis.Addr().(*net.TCPAddr).Port)
	server := grpc.NewServer()
	agentpb.RegisterAgentServer(server, &testServer{
		connectFunc: connect,
	})

	// all assertions must happen in the main goroutine to avoid "panic: Fail in goroutine after XXX has completed"
	serveError := make(chan error)
	go func() {
		serveError <- server.Serve(lis)
	}()

	teardown = func() {
		server.GracefulStop()
		require.NoError(t, <-serveError)
	}

	return
}

func TestClient(t *testing.T) {
	t.Run("NoAddress", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())

		cfg := &config.Config{}
		client := New(cfg, nil)
		cancel()
		err := client.Run(ctx)
		assert.EqualError(t, err, "missing PMM Server address: context canceled")
	})

	t.Run("NoAgentID", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())

		cfg := &config.Config{
			Server: config.Server{
				Address: "127.0.0.1:1",
			},
		}
		client := New(cfg, nil)
		cancel()
		err := client.Run(ctx)
		assert.EqualError(t, err, "missing Agent ID: context canceled")
	})

	t.Run("FailedToDial", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		cfg := &config.Config{
			ID: "agent_id",
			Server: config.Server{
				Address: "127.0.0.1:1",
			},
		}
		client := New(cfg, nil)
		err := client.Run(ctx)
		assert.EqualError(t, err, "failed to dial: context deadline exceeded")
	})

	t.Run("WithServer", func(t *testing.T) {
		t.Run("Normal", func(t *testing.T) {
			serverMD := &agentpb.AgentServerMetadata{
				ServerVersion: t.Name(),
			}

			connect := func(stream agentpb.Agent_ConnectServer) error {
				md := agentpb.GetAgentConnectMetadata(stream.Context())
				assert.Equal(t, agentpb.AgentConnectMetadata{ID: "agent_id"}, md)
				err := agentpb.SendAgentServerMetadata(stream, serverMD)
				require.NoError(t, err)

				msg, err := stream.Recv()
				require.NoError(t, err)
				ping := msg.GetPing()
				require.NotNil(t, ping)
				err = stream.Send(&agentpb.ServerMessage{
					Id:      msg.Id,
					Payload: (&agentpb.Pong{CurrentTime: ptypes.TimestampNow()}).ServerMessageResponsePayload(),
				})
				require.NoError(t, err)

				return errors.New("connect done")
			}

			port, teardown := setup(t, connect)
			defer teardown()

			cfg := &config.Config{
				ID: "agent_id",
				Server: config.Server{
					Address: fmt.Sprintf("127.0.0.1:%d", port),
				},
			}

			s := new(mockSupervisor)
			s.On("Changes").Return(make(<-chan agentpb.StateChangedRequest))
			s.On("QANRequests").Return(make(<-chan agentpb.QANCollectRequest))

			client := New(cfg, s)
			client.withoutTLS = true
			err := client.Run(context.Background())
			assert.NoError(t, err)
			assert.Equal(t, serverMD, client.GetAgentServerMetadata())
		})

		t.Run("NoManaged", func(t *testing.T) {
			t.Skip("FIXME https://jira.percona.com/browse/PMM-4076")

			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()

			connect := func(stream agentpb.Agent_ConnectServer) error {
				time.Sleep(300 * time.Millisecond)
				return errors.New("connect done")
			}

			port, teardown := setup(t, connect)
			defer teardown()

			cfg := &config.Config{
				ID: "agent_id",
				Server: config.Server{
					Address: fmt.Sprintf("127.0.0.1:%d", port),
				},
			}

			client := New(cfg, nil)
			client.dialTimeout = 100 * time.Millisecond
			client.withoutTLS = true
			err := client.Run(ctx)
			assert.EqualError(t, err, "failed to get server metadata: rpc error: code = Canceled desc = context canceled", "%+v", err)
		})
	})
}
