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
	"errors"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/percona/pmm/agent/config"
	"github.com/percona/pmm/agent/connectionuptime"
	"github.com/percona/pmm/agent/runner"
	agentv1 "github.com/percona/pmm/api/agent/v1"
	agentlocal "github.com/percona/pmm/api/agentlocal/v1"
	rtav1 "github.com/percona/pmm/api/realtimeanalytics/v1"
)

type testServer struct {
	agentv1.UnimplementedAgentServiceServer

	connectFunc func(server agentv1.AgentService_ConnectServer) error
}

func (s *testServer) Connect(stream agentv1.AgentService_ConnectServer) error {
	return s.connectFunc(stream)
}

var _ agentv1.AgentServiceServer = (*testServer)(nil)

// awaitClosed waits for ch to be closed, and fails instead of hanging.
//
// The tests below park a mock on a channel the server handler closes, or the other way round, and
// every assertion in that handler runs on gRPC's goroutine - where a failing require calls Goexit,
// so the close it was heading for never happens. Waiting unbounded on that turns one reported
// failure into a package-wide -timeout panic covering every test in the file.
func awaitClosed(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()

	select {
	case <-ch:
	case <-time.After(20 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
	}
}

// awaitClosedInHandler is awaitClosed for the server handler's goroutine, which must not call
// t.Fatalf: it ends the stream with an error instead, which unblocks the client and the test.
func awaitClosedInHandler(ch <-chan struct{}, what string) error {
	select {
	case <-ch:
		return nil
	case <-time.After(20 * time.Second):
		return fmt.Errorf("timed out waiting for %s", what)
	}
}

func setup(t *testing.T, connect func(server agentv1.AgentService_ConnectServer) error) (port uint16, teardown func()) {
	t.Helper()

	// logrus.SetLevel(logrus.DebugLevel)

	// start server with given connect handler
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port = uint16(lis.Addr().(*net.TCPAddr).Port)
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

	return port, teardown
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
		require.EqualError(t, err, "missing PMM Server address: context canceled")
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
		require.EqualError(t, err, "missing Agent ID: context canceled")
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
		assert.Equal(t, codes.Canceled, status.Convert(err).Code())
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
			s.On("RTARequests").Return(make(<-chan *rtav1.CollectRequest))
			s.On("AgentsList").Return([]*agentlocal.AgentInfo{})
			s.On("ClearChangesChannel").Return()

			r := runner.New(cfgStorage.Get().RunnerCapacity, cfgStorage.Get().RunnerMaxConnectionsPerService)
			client := New(cfgStorage, &s, r, nil, nil, nil, connectionuptime.NewService(time.Hour), nil)
			err := client.Run(context.Background())
			require.NoError(t, err)
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
			require.EqualError(t, err, "failed to get server metadata: rpc error: code = Canceled desc = context canceled", "%+v", err)
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
	s.On("Changes").Return(make(<-chan *agentv1.StateChangedRequest))
	s.On("QANRequests").Return(make(<-chan *agentv1.QANCollectRequest))
	s.On("RTARequests").Return(make(<-chan *rtav1.CollectRequest))
	s.On("AgentsList").Return([]*agentlocal.AgentInfo{})
	s.On("ClearChangesChannel").Return()

	r := runner.New(cfgStorage.Get().RunnerCapacity, cfgStorage.Get().RunnerMaxConnectionsPerService)
	client := New(cfgStorage, s, r, nil, nil, nil, connectionuptime.NewService(time.Hour), nil)
	err := client.Run(context.Background())
	require.NoError(t, err)
	assert.Equal(t, serverMD, client.GetServerConnectMetadata())
}

// TestSetStateDoesNotBlockRequestLoop is the regression test for PMM-15431: SetState used to be
// applied inline on the request loop, so one that could not finish stopped the agent answering
// anything at all, pings included, with no way back except restarting pmm-agent.
func TestSetStateDoesNotBlockRequestLoop(t *testing.T) {
	serverMD := &agentv1.ServerConnectMetadata{
		ServerVersion: t.Name(),
	}

	applyingState := make(chan struct{})
	releaseState := make(chan struct{})

	connect := func(stream agentv1.AgentService_ConnectServer) error {
		// Deferred: the assertions below run on the server's goroutine, and a failing
		// require there would otherwise leave the mock parked on releaseState and this
		// test's own wait for it hanging the whole package until -timeout.
		defer close(releaseState)

		// establish the connection
		md, err := agentv1.ReceiveAgentConnectMetadata(stream)
		require.NoError(t, err)
		assert.Equal(t, &agentv1.AgentConnectMetadata{ID: "agent_id"}, md)
		err = agentv1.SendServerConnectMetadata(stream, serverMD)
		require.NoError(t, err)
		msg, err := stream.Recv()
		require.NoError(t, err)
		require.NotNil(t, msg.GetPing())
		err = stream.Send(&agentv1.ServerMessage{
			Id:      msg.Id,
			Payload: (&agentv1.Pong{CurrentTime: timestamppb.Now()}).ServerMessageResponsePayload(),
		})
		require.NoError(t, err)

		// a state that never finishes applying is still answered
		err = stream.Send(&agentv1.ServerMessage{
			Id:      4242,
			Payload: (&agentv1.SetStateRequest{}).ServerMessageRequestPayload(),
		})
		require.NoError(t, err)
		msg, err = stream.Recv()
		require.NoError(t, err)
		assert.EqualValues(t, 4242, msg.Id)
		require.NotNil(t, msg.GetSetState())

		// and while it is being applied, everything else is answered too
		err = awaitClosedInHandler(applyingState, "SetState to start being applied")
		if err != nil {
			return err
		}
		err = stream.Send(&agentv1.ServerMessage{
			Id:      4243,
			Payload: (&agentv1.Ping{}).ServerMessageRequestPayload(),
		})
		require.NoError(t, err)
		msg, err = stream.Recv()
		require.NoError(t, err)
		assert.EqualValues(t, 4243, msg.Id)
		require.NotNil(t, msg.GetPong())

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
	s.On("RTARequests").Return(make(<-chan *rtav1.CollectRequest))
	s.On("AgentsList").Return([]*agentlocal.AgentInfo{})
	s.On("ClearChangesChannel").Return()
	s.On("SetState", mock.Anything, mock.Anything).Run(func(mock.Arguments) {
		close(applyingState)
		<-releaseState
	}).Return()

	r := runner.New(cfgStorage.Get().RunnerCapacity, cfgStorage.Get().RunnerMaxConnectionsPerService)
	client := New(cfgStorage, s, r, nil, nil, nil, connectionuptime.NewService(time.Hour), nil)
	require.NoError(t, client.Run(t.Context()))
	awaitClosed(t, releaseState, "the server handler to finish")
	s.AssertExpectations(t)
}

// TestActualStatusesDoNotBlockPings covers the other half of PMM-15431: reporting the actual
// statuses starts by asking the supervisor for its Agents list, and a SetState from the connection
// before this one can still be holding that lock for as long as the Agents it replaces take to
// stop. A freshly connected agent that answers nothing meanwhile is dropped as stale.
func TestActualStatusesDoNotBlockPings(t *testing.T) {
	serverMD := &agentv1.ServerConnectMetadata{
		ServerVersion: t.Name(),
	}

	listing := make(chan struct{})
	release := make(chan struct{})

	connect := func(stream agentv1.AgentService_ConnectServer) error {
		// Deferred: the assertions below run on the server's goroutine, and a failing
		// require there would otherwise leave the mock parked on release and this
		// test's own wait for it hanging the whole package until -timeout.
		defer close(release)

		// establish the connection
		md, err := agentv1.ReceiveAgentConnectMetadata(stream)
		require.NoError(t, err)
		assert.Equal(t, &agentv1.AgentConnectMetadata{ID: "agent_id"}, md)
		err = agentv1.SendServerConnectMetadata(stream, serverMD)
		require.NoError(t, err)
		msg, err := stream.Recv()
		require.NoError(t, err)
		require.NotNil(t, msg.GetPing())
		err = stream.Send(&agentv1.ServerMessage{
			Id:      msg.Id,
			Payload: (&agentv1.Pong{CurrentTime: timestamppb.Now()}).ServerMessageResponsePayload(),
		})
		require.NoError(t, err)

		// while the supervisor cannot even be asked what it is running
		err = awaitClosedInHandler(listing, "the Agents list to be asked for")
		if err != nil {
			return err
		}
		err = stream.Send(&agentv1.ServerMessage{
			Id:      4242,
			Payload: (&agentv1.Ping{}).ServerMessageRequestPayload(),
		})
		require.NoError(t, err)
		msg, err = stream.Recv()
		require.NoError(t, err)
		assert.EqualValues(t, 4242, msg.Id)
		require.NotNil(t, msg.GetPong())

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
	s.On("RTARequests").Return(make(<-chan *rtav1.CollectRequest))
	s.On("ClearChangesChannel").Return()
	listed := sync.OnceFunc(func() { close(listing) })
	s.On("AgentsList").Run(func(mock.Arguments) {
		listed()
		<-release
	}).Return([]*agentlocal.AgentInfo{})

	r := runner.New(cfgStorage.Get().RunnerCapacity, cfgStorage.Get().RunnerMaxConnectionsPerService)
	client := New(cfgStorage, s, r, nil, nil, nil, connectionuptime.NewService(time.Hour), nil)
	require.NoError(t, client.Run(t.Context()))
	awaitClosed(t, release, "the server handler to finish")
	s.AssertExpectations(t)
}

// TestDoneClosesAfterCancel covers the handshake pmm-agent's reconnect depends on: run.go cancels
// the client context and then waits on Done() before redialing, so a processor that does not exit
// leaves the agent connected to nothing for good - which is how PMM-15431 presented.
// TestDoneClosesWhileActualStatusesAreBlocked covers the reconnect path when reporting the actual
// statuses cannot finish: it starts by asking the supervisor for its Agents list, and a SetState
// from the connection before this one can still be holding that lock for as long as the Agents it
// replaces take to stop. Acquiring a lock cannot be cancelled, so counting that goroutine towards
// Done() would leave run.go waiting on it - no redial and no shutdown - which is the PMM-15431
// outage in a new place. See runProcessors.
func TestDoneClosesWhileActualStatusesAreBlocked(t *testing.T) {
	serverMD := &agentv1.ServerConnectMetadata{
		ServerVersion: t.Name(),
	}

	listing := make(chan struct{})
	release := make(chan struct{})
	// Released unconditionally, so a failing assertion below is reported instead of leaving the
	// mock parked and hanging the package. See TestSetStateDoesNotBlockRequestLoop.
	defer close(release)

	connect := func(stream agentv1.AgentService_ConnectServer) error {
		md, err := agentv1.ReceiveAgentConnectMetadata(stream)
		require.NoError(t, err)
		assert.Equal(t, &agentv1.AgentConnectMetadata{ID: "agent_id"}, md)
		err = agentv1.SendServerConnectMetadata(stream, serverMD)
		require.NoError(t, err)
		msg, err := stream.Recv()
		require.NoError(t, err)
		require.NotNil(t, msg.GetPing())
		err = stream.Send(&agentv1.ServerMessage{
			Id:      msg.Id,
			Payload: (&agentv1.Pong{CurrentTime: timestamppb.Now()}).ServerMessageResponsePayload(),
		})
		require.NoError(t, err)

		// Idle, with the stream healthy, until the client tears it down itself.
		_, err = stream.Recv()
		require.Error(t, err)

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
	s.On("RTARequests").Return(make(<-chan *rtav1.CollectRequest))
	s.On("ClearChangesChannel").Return()

	// Stands in for the supervisor lock a previous connection's SetState is holding.
	listed := sync.OnceFunc(func() { close(listing) })
	s.On("AgentsList").Run(func(mock.Arguments) {
		listed()
		<-release
	}).Return([]*agentlocal.AgentInfo{})

	r := runner.New(cfgStorage.Get().RunnerCapacity, cfgStorage.Get().RunnerMaxConnectionsPerService)
	client := New(cfgStorage, s, r, nil, nil, nil, connectionuptime.NewService(time.Hour), nil)

	ctx, cancel := context.WithCancel(t.Context())
	runErr := make(chan error, 1)
	go func() { runErr <- client.Run(ctx) }()

	awaitClosed(t, listing, "the Agents list to be asked for")
	cancel()

	select {
	case err := <-runErr:
		require.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return while the Agents list was blocked")
	}

	select {
	case <-client.Done():
	case <-time.After(10 * time.Second):
		t.Fatal("Done() waited for the Agents list: pmm-agent would never reconnect")
	}
}

func TestDoneClosesAfterCancel(t *testing.T) {
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
		require.NotNil(t, msg.GetPing())
		err = stream.Send(&agentv1.ServerMessage{
			Id:      msg.Id,
			Payload: (&agentv1.Pong{CurrentTime: timestamppb.Now()}).ServerMessageResponsePayload(),
		})
		require.NoError(t, err)

		// Idle, with the stream healthy, until the client tears it down itself.
		_, err = stream.Recv()
		require.Error(t, err)

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
	s.On("RTARequests").Return(make(<-chan *rtav1.CollectRequest))
	s.On("AgentsList").Return([]*agentlocal.AgentInfo{})
	s.On("ClearChangesChannel").Return()

	r := runner.New(cfgStorage.Get().RunnerCapacity, cfgStorage.Get().RunnerMaxConnectionsPerService)
	client := New(cfgStorage, s, r, nil, nil, nil, connectionuptime.NewService(time.Hour), nil)

	ctx, cancel := context.WithCancel(t.Context())
	runErr := make(chan error, 1)
	go func() { runErr <- client.Run(ctx) }()

	require.Eventually(t, func() bool {
		return client.GetServerConnectMetadata() != nil
	}, 10*time.Second, 50*time.Millisecond, "client never connected")

	cancel()

	select {
	case err := <-runErr:
		require.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return after its context was cancelled")
	}

	select {
	case <-client.Done():
	case <-time.After(10 * time.Second):
		t.Fatal("Done() did not close after cancellation: pmm-agent would never reconnect")
	}
}

func TestRequestSetState(t *testing.T) {
	c := &Client{l: logrus.WithField("component", "client")}
	setStates := make(chan *agentv1.SetStateRequest, setStatesCap)

	first := &agentv1.SetStateRequest{}
	second := &agentv1.SetStateRequest{}

	c.requestSetState(setStates, first)
	c.requestSetState(setStates, second)

	// A SetStateRequest carries the whole desired state, so the first one is simply gone.
	assert.Len(t, setStates, 1)
	assert.Same(t, second, <-setStates)
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
