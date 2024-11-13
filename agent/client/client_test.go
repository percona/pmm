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

package client

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/percona/pmm/agent/config"
	"github.com/percona/pmm/agent/connectionuptime"
	"github.com/percona/pmm/agent/runner"
	"github.com/percona/pmm/api/agentlocalpb"
	"github.com/percona/pmm/api/agentpb"
)

type testServer struct {
	connectFunc func(agentpb.Agent_ConnectServer) error
	agentpb.UnimplementedAgentServer
}

func (s *testServer) Connect(stream agentpb.Agent_ConnectServer) error {
	return s.connectFunc(stream)
}

var _ agentpb.AgentServer = (*testServer)(nil)

func setup(t *testing.T, connect func(agentpb.Agent_ConnectServer) error) (port uint16, teardown func()) {
	t.Helper()

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
	t.Parallel()

	t.Run("NoAddress", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())

		cfgStorage := config.NewStorage(&config.Config{})
		client := New(cfgStorage, nil, nil, nil, nil, nil, nil, nil)
		cancel()
		err := client.Run(ctx)
		assert.EqualError(t, err, "missing PMM Server address: context canceled")
	})

	t.Run("NoAgentID", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())

		cfgStorage := config.NewStorage(&config.Config{
			Server: config.Server{
				Address: "127.0.0.1:1",
			},
		})
		client := New(cfgStorage, nil, nil, nil, nil, nil, nil, nil)
		cancel()
		err := client.Run(ctx)
		assert.EqualError(t, err, "missing Agent ID: context canceled")
	})

	t.Run("FailedToDial", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		cfgStorage := config.NewStorage(&config.Config{
			ID: "agent_id",
			Server: config.Server{
				Address: "127.0.0.1:1",
			},
		})
		client := New(cfgStorage, nil, nil, nil, nil, nil, connectionuptime.NewService(time.Hour), nil)
		err := client.Run(ctx)
		assert.EqualError(t, err, "failed to dial: context deadline exceeded")
	})

	t.Run("WithServer", func(t *testing.T) {
		t.Run("Normal", func(t *testing.T) {
			serverMD := &agentpb.ServerConnectMetadata{
				ServerVersion: t.Name(),
			}

			connect := func(stream agentpb.Agent_ConnectServer) error {
				md, err := agentpb.ReceiveAgentConnectMetadata(stream)
				require.NoError(t, err)
				assert.Equal(t, &agentpb.AgentConnectMetadata{ID: "agent_id"}, md)
				err = agentpb.SendServerConnectMetadata(stream, serverMD)
				require.NoError(t, err)

				msg, err := stream.Recv()
				require.NoError(t, err)
				ping := msg.GetPing()
				require.NotNil(t, ping)
				err = stream.Send(&agentpb.ServerMessage{
					Id:      msg.Id,
					Payload: (&agentpb.Pong{CurrentTime: timestamppb.Now()}).ServerMessageResponsePayload(),
				})
				require.NoError(t, err)

				return errors.New("connect done")
			}

			port, teardown := setup(t, connect)
			defer teardown()

			cfgStorage := config.NewStorage(&config.Config{
				ID: "agent_id",
				Server: config.Server{
					Address:    fmt.Sprintf("127.0.0.1:%d", port),
					WithoutTLS: true,
				},
			})

			var s mockSupervisor
			s.On("Changes").Return(make(<-chan *agentpb.StateChangedRequest))
			s.On("QANRequests").Return(make(<-chan *agentpb.QANCollectRequest))
			s.On("AgentsList").Return([]*agentlocalpb.AgentInfo{})
			s.On("ClearChangesChannel").Return()

			r := runner.New(cfgStorage.Get().RunnerCapacity, cfgStorage.Get().RunnerMaxConnectionsPerService)
			client := New(cfgStorage, &s, r, nil, nil, nil, connectionuptime.NewService(time.Hour), nil)
			err := client.Run(context.Background())
			assert.NoError(t, err)
			assert.Equal(t, serverMD, client.GetServerConnectMetadata())
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

			cfgStorage := config.NewStorage(&config.Config{
				ID: "agent_id",
				Server: config.Server{
					Address:    fmt.Sprintf("127.0.0.1:%d", port),
					WithoutTLS: true,
				},
			})

			client := New(cfgStorage, nil, nil, nil, nil, nil, connectionuptime.NewService(time.Hour), nil)
			client.dialTimeout = 100 * time.Millisecond
			err := client.Run(ctx)
			assert.EqualError(t, err, "failed to get server metadata: rpc error: code = Canceled desc = context canceled", "%+v", err)
		})
	})
}

func TestUnexpectedActionType(t *testing.T) {
	serverMD := &agentpb.ServerConnectMetadata{
		ServerVersion: t.Name(),
	}
	connect := func(stream agentpb.Agent_ConnectServer) error {
		// establish the connection
		md, err := agentpb.ReceiveAgentConnectMetadata(stream)
		require.NoError(t, err)
		assert.Equal(t, &agentpb.AgentConnectMetadata{ID: "agent_id"}, md)
		err = agentpb.SendServerConnectMetadata(stream, serverMD)
		require.NoError(t, err)
		msg, err := stream.Recv()
		require.NoError(t, err)
		ping := msg.GetPing()
		require.NotNil(t, ping)
		err = stream.Send(&agentpb.ServerMessage{
			Id:      msg.Id,
			Payload: (&agentpb.Pong{CurrentTime: timestamppb.Now()}).ServerMessageResponsePayload(),
		})
		require.NoError(t, err)

		// actual test
		cases := []struct {
			name         string
			id           uint32
			payload      *agentpb.ServerMessage_StartAction
			expectedCode codes.Code
		}{
			{
				name: "invlalid action type",
				id:   4242,
				payload: &agentpb.ServerMessage_StartAction{
					StartAction: &agentpb.StartActionRequest{},
				},
				expectedCode: codes.InvalidArgument,
			},
			{
				name: "mongodb restart invalid system service",
				id:   4243,
				payload: &agentpb.ServerMessage_StartAction{
					StartAction: &agentpb.StartActionRequest{
						Params: &agentpb.StartActionRequest_RestartSysServiceParams{
							RestartSysServiceParams: &agentpb.StartActionRequest_RestartSystemServiceParams{
								SystemService: agentpb.StartActionRequest_RestartSystemServiceParams_SYSTEM_SERVICE_INVALID,
							},
						},
					},
				},
				expectedCode: codes.InvalidArgument,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				err = stream.Send(&agentpb.ServerMessage{Id: tc.id, Payload: tc.payload})
				require.NoError(t, err)

				msg, err = stream.Recv()
				require.NoError(t, err)
				assert.Equal(t, int32(tc.expectedCode), msg.GetStatus().GetCode())
			})
		}
		return nil
	}
	port, teardown := setup(t, connect)
	defer teardown()

	cfgStorage := config.NewStorage(&config.Config{
		ID: "agent_id",
		Server: config.Server{
			Address:    fmt.Sprintf("127.0.0.1:%d", port),
			WithoutTLS: true,
		},
	})

	s := &mockSupervisor{}
	s.On("Changes").Return(make(<-chan *agentpb.StateChangedRequest))
	s.On("QANRequests").Return(make(<-chan *agentpb.QANCollectRequest))
	s.On("AgentsList").Return([]*agentlocalpb.AgentInfo{})
	s.On("ClearChangesChannel").Return()

	r := runner.New(cfgStorage.Get().RunnerCapacity, cfgStorage.Get().RunnerMaxConnectionsPerService)
	client := New(cfgStorage, s, r, nil, nil, nil, connectionuptime.NewService(time.Hour), nil)
	err := client.Run(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, serverMD, client.GetServerConnectMetadata())
}

func TestArgListFromPgParams(t *testing.T) {
	type testParams struct {
		req      *agentpb.StartActionRequest_PTPgSummaryParams
		expected []string
	}
	testCases := []*testParams{
		{
			&agentpb.StartActionRequest_PTPgSummaryParams{Host: "10.20.30.40", Port: 555, Username: "person", Password: "secret"},
			[]string{"--host", "10.20.30.40", "--port", "555", "--username", "person", "--password", "secret"},
		}, {
			&agentpb.StartActionRequest_PTPgSummaryParams{Host: "10.20.30.40", Port: 555, Username: "person", Password: ""},
			[]string{"--host", "10.20.30.40", "--port", "555", "--username", "person"},
		}, {
			&agentpb.StartActionRequest_PTPgSummaryParams{Host: "10.20.30.40", Port: 555, Username: "", Password: "secret"},
			[]string{"--host", "10.20.30.40", "--port", "555", "--password", "secret"},
		}, {
			&agentpb.StartActionRequest_PTPgSummaryParams{Host: "10.20.30.40", Port: 65536, Username: "", Password: "secret"},
			[]string{"--host", "10.20.30.40", "--password", "secret"},
		}, {
			&agentpb.StartActionRequest_PTPgSummaryParams{Host: "", Port: 555, Username: "", Password: "secret"},
			[]string{"--port", "555", "--password", "secret"},
		}, {
			&agentpb.StartActionRequest_PTPgSummaryParams{Host: "", Port: 0, Username: "", Password: ""},
			[]string{},
		}, {
			&agentpb.StartActionRequest_PTPgSummaryParams{Host: "", Port: 0, Username: "王华", Password: `"`},
			[]string{"--username", "王华", "--password", `"`},
		}, {
			&agentpb.StartActionRequest_PTPgSummaryParams{Host: "10.20.30.40", Port: 555, Username: "person", Password: "   "},
			[]string{"--username", "person", "--port", "555", "--host", "10.20.30.40"},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(prototext.Format(tc.req), func(t *testing.T) {
			actual := argListFromPgParams(tc.req)
			t.Logf("\n%+v\n", actual)
			assert.ElementsMatch(t, tc.expected, actual)
		})
	}
}

func TestArgListFromMongoDBParams(t *testing.T) {
	type testParams struct {
		req      *agentpb.StartActionRequest_PTMongoDBSummaryParams
		expected []string
	}
	testCases := []*testParams{
		{
			&agentpb.StartActionRequest_PTMongoDBSummaryParams{Host: "10.20.30.40", Port: 555, Username: "person", Password: "secret"},
			[]string{"--username", "person", "--password=secret", "10.20.30.40:555"},
		},
		{
			&agentpb.StartActionRequest_PTMongoDBSummaryParams{Host: "10.20.30.40", Port: 555, Username: "person", Password: ""},
			[]string{"--username", "person", "10.20.30.40:555"},
		},
		{
			&agentpb.StartActionRequest_PTMongoDBSummaryParams{Host: "10.20.30.40", Port: 555, Username: "", Password: "secret"},
			[]string{"--password=secret", "10.20.30.40:555"},
		},
		{
			&agentpb.StartActionRequest_PTMongoDBSummaryParams{Host: "10.20.30.40", Port: 65536, Username: "", Password: "secret"},
			[]string{"--password=secret", "10.20.30.40"},
		},
		{
			&agentpb.StartActionRequest_PTMongoDBSummaryParams{Host: "", Port: 555, Username: "", Password: "secret"},
			[]string{"--password=secret"},
		},
		{
			&agentpb.StartActionRequest_PTMongoDBSummaryParams{Host: "", Port: 0, Username: "", Password: ""},
			[]string{},
		},
		{
			&agentpb.StartActionRequest_PTMongoDBSummaryParams{Host: "", Port: 0, Username: "王华", Password: `"`},
			[]string{"--username", "王华", `--password="`},
		},
		{
			&agentpb.StartActionRequest_PTMongoDBSummaryParams{Host: "10.20.30.40", Port: 555, Username: "person", Password: "   "},
			[]string{"--username", "person", "--password=   ", "10.20.30.40:555"},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(prototext.Format(tc.req), func(t *testing.T) {
			actual := argListFromMongoDBParams(tc.req)
			assert.ElementsMatch(t, tc.expected, actual)
		})
	}
}
