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
	agentv1 "github.com/percona/pmm/api/agent/v1"
	agentlocal "github.com/percona/pmm/api/agentlocal/v1"
)

type testServer struct {
	connectFunc func(server agentv1.AgentService_ConnectServer) error
	agentv1.UnimplementedAgentServiceServer
}

func (s *testServer) Connect(stream agentv1.AgentService_ConnectServer) error {
	return s.connectFunc(stream)
}

var _ agentv1.AgentServiceServer = (*testServer)(nil)

func setup(t *testing.T, connect func(server agentv1.AgentService_ConnectServer) error) (port uint16, teardown func()) {
	t.Helper()

	// logrus.SetLevel(logrus.DebugLevel)

	// start server with given connect handler
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port = uint16(lis.Addr().(*net.TCPAddr).Port) //nolint:gosec // port is uint16
	server := grpc.NewServer()
	agentv1.RegisterAgentServiceServer(server, &testServer{
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
			serverMD := &agentv1.ServerConnectMetadata{
				ServerVersion: t.Name(),
			}

			connect := func(stream agentv1.AgentService_ConnectServer) error {
				md, err := agentv1.ReceiveAgentConnectMetadata(stream)
				require.NoError(t, err)
				assert.Equal(t, &agentv1.AgentConnectMetadata{ID: "agent_id"}, md)
				err = agentv1.SendServerConnectMetadata(stream, serverMD)
				require.NoError(t, err)

				msg, err := stream.Recv()
				require.NoError(t, err)
				ping := msg.GetPing()
				require.NotNil(t, ping)
				err = stream.Send(&agentv1.ServerMessage{
					Id:      msg.Id,
					Payload: (&agentv1.Pong{CurrentTime: timestamppb.Now()}).ServerMessageResponsePayload(),
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
			s.On("Changes").Return(make(<-chan *agentv1.StateChangedRequest))
			s.On("QANRequests").Return(make(<-chan *agentv1.QANCollectRequest))
			s.On("AgentsList").Return([]*agentlocal.AgentInfo{})
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

			connect := func(_ agentv1.AgentService_ConnectServer) error {
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
	serverMD := &agentv1.ServerConnectMetadata{
		ServerVersion: t.Name(),
	}
	connect := func(stream agentv1.AgentService_ConnectServer) error {
		// establish the connection
		md, err := agentv1.ReceiveAgentConnectMetadata(stream)
		require.NoError(t, err)
		assert.Equal(t, &agentv1.AgentConnectMetadata{ID: "agent_id"}, md)
		err = agentv1.SendServerConnectMetadata(stream, serverMD)
		require.NoError(t, err)
		msg, err := stream.Recv()
		require.NoError(t, err)
		ping := msg.GetPing()
		require.NotNil(t, ping)
		err = stream.Send(&agentv1.ServerMessage{
			Id:      msg.Id,
			Payload: (&agentv1.Pong{CurrentTime: timestamppb.Now()}).ServerMessageResponsePayload(),
		})
		require.NoError(t, err)

		// actual test
		cases := []struct {
			name         string
			id           uint32
			payload      *agentv1.ServerMessage_StartAction
			expectedCode codes.Code
		}{
			{
				name: "invalid action type",
				id:   4242,
				payload: &agentv1.ServerMessage_StartAction{
					StartAction: &agentv1.StartActionRequest{},
				},
				expectedCode: codes.Unimplemented,
			},
			{
				name: "mongodb restart invalid system service",
				id:   4243,
				payload: &agentv1.ServerMessage_StartAction{
					StartAction: &agentv1.StartActionRequest{
						Params: &agentv1.StartActionRequest_RestartSysServiceParams{
							RestartSysServiceParams: &agentv1.StartActionRequest_RestartSystemServiceParams{
								SystemService: agentv1.StartActionRequest_RestartSystemServiceParams_SYSTEM_SERVICE_UNSPECIFIED,
							},
						},
					},
				},
				expectedCode: codes.InvalidArgument,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				err = stream.Send(&agentv1.ServerMessage{Id: tc.id, Payload: tc.payload})
				require.NoError(t, err)

				msg, err = stream.Recv()
				require.NoError(t, err)
				assert.Equal(t, int32(tc.expectedCode), msg.GetStatus().GetCode()) //nolint:gosec // grpc code is int32
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
	s.On("Changes").Return(make(<-chan *agentv1.StateChangedRequest))
	s.On("QANRequests").Return(make(<-chan *agentv1.QANCollectRequest))
	s.On("AgentsList").Return([]*agentlocal.AgentInfo{})
	s.On("ClearChangesChannel").Return()

	r := runner.New(cfgStorage.Get().RunnerCapacity, cfgStorage.Get().RunnerMaxConnectionsPerService)
	client := New(cfgStorage, s, r, nil, nil, nil, connectionuptime.NewService(time.Hour), nil)
	err := client.Run(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, serverMD, client.GetServerConnectMetadata())
}

func TestArgListFromPgParams(t *testing.T) {
	type testParams struct {
		req      *agentv1.StartActionRequest_PTPgSummaryParams
		expected []string
	}
	testCases := []*testParams{
		{
			&agentv1.StartActionRequest_PTPgSummaryParams{Host: "10.20.30.40", Port: 555, Username: "person", Password: "secret"},
			[]string{"--host", "10.20.30.40", "--port", "555", "--username", "person", "--password", "secret"},
		}, {
			&agentv1.StartActionRequest_PTPgSummaryParams{Host: "10.20.30.40", Port: 555, Username: "person", Password: ""},
			[]string{"--host", "10.20.30.40", "--port", "555", "--username", "person"},
		}, {
			&agentv1.StartActionRequest_PTPgSummaryParams{Host: "10.20.30.40", Port: 555, Username: "", Password: "secret"},
			[]string{"--host", "10.20.30.40", "--port", "555", "--password", "secret"},
		}, {
			&agentv1.StartActionRequest_PTPgSummaryParams{Host: "10.20.30.40", Port: 65536, Username: "", Password: "secret"},
			[]string{"--host", "10.20.30.40", "--password", "secret"},
		}, {
			&agentv1.StartActionRequest_PTPgSummaryParams{Host: "", Port: 555, Username: "", Password: "secret"},
			[]string{"--port", "555", "--password", "secret"},
		}, {
			&agentv1.StartActionRequest_PTPgSummaryParams{Host: "", Port: 0, Username: "", Password: ""},
			[]string{},
		}, {
			&agentv1.StartActionRequest_PTPgSummaryParams{Host: "", Port: 0, Username: "王华", Password: `"`},
			[]string{"--username", "王华", "--password", `"`},
		}, {
			&agentv1.StartActionRequest_PTPgSummaryParams{Host: "10.20.30.40", Port: 555, Username: "person", Password: "   "},
			[]string{"--username", "person", "--port", "555", "--host", "10.20.30.40"},
		},
	}

	for _, tc := range testCases {
		t.Run(prototext.Format(tc.req), func(t *testing.T) {
			actual := argListFromPgParams(tc.req)
			t.Logf("\n%+v\n", actual)
			assert.ElementsMatch(t, tc.expected, actual)
		})
	}
}

func TestArgListFromMongoDBParams(t *testing.T) {
	type testParams struct {
		req      *agentv1.StartActionRequest_PTMongoDBSummaryParams
		expected []string
	}
	testCases := []*testParams{
		{
			&agentv1.StartActionRequest_PTMongoDBSummaryParams{Host: "10.20.30.40", Port: 555, Username: "person", Password: "secret"},
			[]string{"--username", "person", "--password=secret", "10.20.30.40:555"},
		},
		{
			&agentv1.StartActionRequest_PTMongoDBSummaryParams{Host: "10.20.30.40", Port: 555, Username: "person", Password: ""},
			[]string{"--username", "person", "10.20.30.40:555"},
		},
		{
			&agentv1.StartActionRequest_PTMongoDBSummaryParams{Host: "10.20.30.40", Port: 555, Username: "", Password: "secret"},
			[]string{"--password=secret", "10.20.30.40:555"},
		},
		{
			&agentv1.StartActionRequest_PTMongoDBSummaryParams{Host: "10.20.30.40", Port: 65536, Username: "", Password: "secret"},
			[]string{"--password=secret", "10.20.30.40"},
		},
		{
			&agentv1.StartActionRequest_PTMongoDBSummaryParams{Host: "", Port: 555, Username: "", Password: "secret"},
			[]string{"--password=secret"},
		},
		{
			&agentv1.StartActionRequest_PTMongoDBSummaryParams{Host: "", Port: 0, Username: "", Password: ""},
			[]string{},
		},
		{
			&agentv1.StartActionRequest_PTMongoDBSummaryParams{Host: "", Port: 0, Username: "王华", Password: `"`},
			[]string{"--username", "王华", `--password="`},
		},
		{
			&agentv1.StartActionRequest_PTMongoDBSummaryParams{Host: "10.20.30.40", Port: 555, Username: "person", Password: "   "},
			[]string{"--username", "person", "--password=   ", "10.20.30.40:555"},
		},
	}

	for _, tc := range testCases {
		t.Run(prototext.Format(tc.req), func(t *testing.T) {
			actual := argListFromMongoDBParams(tc.req)
			assert.ElementsMatch(t, tc.expected, actual)
		})
	}
}
