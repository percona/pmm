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

package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/percona/exporter_shared/helpers"
	"github.com/percona/pmm/api/agentpb"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type testServer struct {
	connectFunc func(agentpb.Agent_ConnectServer) error
}

func (s *testServer) Register(context.Context, *agentpb.RegisterRequest) (*agentpb.RegisterResponse, error) {
	panic("not implemented")
}

func (s *testServer) Connect(stream agentpb.Agent_ConnectServer) error {
	return s.connectFunc(stream)
}

var _ agentpb.AgentServer = (*testServer)(nil)

func setup(t *testing.T, connect func(agentpb.Agent_ConnectServer) error, expected ...error) (*Channel, *grpc.ClientConn, func(*testing.T)) {
	// logrus.SetLevel(logrus.DebugLevel)

	t.Parallel()

	// start server with given connect handler
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer()
	agentpb.RegisterAgentServer(server, &testServer{
		connectFunc: connect,
	})
	go func() {
		err = server.Serve(lis)
		require.NoError(t, err)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

	// make client and channel
	opts := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithInsecure(),
	}
	cc, err := grpc.DialContext(ctx, lis.Addr().String(), opts...)
	require.NoError(t, err, "failed to dial server")
	stream, err := agentpb.NewAgentClient(cc).Connect(ctx)
	require.NoError(t, err, "failed to create stream")
	channel := NewChannel(stream)

	teardown := func(t *testing.T) {
		err := channel.Wait()
		assert.Contains(t, expected, errors.Cause(err), "%+v", err)

		server.GracefulStop()
		cancel()
	}

	return channel, cc, teardown
}

func TestAgentRequest(t *testing.T) {
	const count = 50
	require.True(t, count > serverRequestsCap)

	connect := func(stream agentpb.Agent_ConnectServer) error { //nolint:unparam
		for i := uint32(1); i <= count; i++ {
			msg, err := stream.Recv()
			require.NoError(t, err)
			assert.Equal(t, i, msg.Id)
			require.NotNil(t, msg.GetQanCollect())

			err = stream.Send(&agentpb.ServerMessage{
				Id: i,
				Payload: &agentpb.ServerMessage_QanCollect{
					QanCollect: new(agentpb.QANCollectResponse),
				},
			})
			assert.NoError(t, err)
		}

		return nil
	}

	channel, _, teardown := setup(t, connect, io.EOF) // EOF = server exits from handler
	defer teardown(t)

	for i := uint32(1); i <= count; i++ {
		resp := channel.SendRequest(&agentpb.AgentMessage_QanCollect{
			QanCollect: new(agentpb.QANCollectRequest),
		})
		assert.NotNil(t, resp)
	}

	// check metrics
	metrics := make([]prometheus.Metric, 0, 100)
	metricsCh := make(chan prometheus.Metric)
	go func() {
		channel.Collect(metricsCh)
		close(metricsCh)
	}()
	for m := range metricsCh {
		metrics = append(metrics, m)
	}
	expectedMetrics := strings.Split(strings.TrimSpace(`
# HELP pmm_agent_channel_messages_received_total A total number of received messages from pmm-managed.
# TYPE pmm_agent_channel_messages_received_total counter
pmm_agent_channel_messages_received_total 50
# HELP pmm_agent_channel_messages_sent_total A total number of sent messages to pmm-managed.
# TYPE pmm_agent_channel_messages_sent_total counter
pmm_agent_channel_messages_sent_total 50
`), "\n")
	assert.Equal(t, expectedMetrics, helpers.Format(metrics))

	// check that descriptions match metrics: same number, same order
	descCh := make(chan *prometheus.Desc)
	go func() {
		channel.Describe(descCh)
		close(descCh)
	}()
	var i int
	for d := range descCh {
		assert.Equal(t, metrics[i].Desc(), d)
		i++
	}
	assert.Len(t, metrics, i)
}

func TestServerRequest(t *testing.T) {
	const count = 50
	require.True(t, count > serverRequestsCap)

	connect := func(stream agentpb.Agent_ConnectServer) error { //nolint:unparam
		for i := uint32(1); i <= count; i++ {
			err := stream.Send(&agentpb.ServerMessage{
				Id: i,
				Payload: &agentpb.ServerMessage_Ping{
					Ping: new(agentpb.Ping),
				},
			})
			assert.NoError(t, err)
		}

		for i := uint32(1); i <= count; i++ {
			msg, err := stream.Recv()
			require.NoError(t, err)
			assert.Equal(t, i, msg.Id)
			pong := msg.GetPong()
			require.NotNil(t, pong)
			ts, err := ptypes.Timestamp(pong.CurrentTime)
			assert.NoError(t, err)
			assert.InDelta(t, time.Now().Unix(), ts.Unix(), 1)
		}

		return nil
	}

	channel, _, teardown := setup(t, connect, io.EOF) // EOF = server exits from handler
	defer teardown(t)

	for req := range channel.Requests() {
		ping := req.GetPing()
		require.NotNil(t, ping)

		channel.SendResponse(&agentpb.AgentMessage{
			Id: req.Id,
			Payload: &agentpb.AgentMessage_Pong{
				Pong: &agentpb.Pong{
					CurrentTime: ptypes.TimestampNow(),
				},
			},
		})
	}
}

func TestServerExitsWithGRPCError(t *testing.T) {
	errUnimplemented := status.Error(codes.Unimplemented, "Test error")
	connect := func(stream agentpb.Agent_ConnectServer) error {
		msg, err := stream.Recv()
		require.NoError(t, err)
		assert.EqualValues(t, 1, msg.Id)
		require.NotNil(t, msg.GetQanCollect())

		return errUnimplemented
	}

	channel, _, teardown := setup(t, connect, errUnimplemented)
	defer teardown(t)

	resp := channel.SendRequest(&agentpb.AgentMessage_QanCollect{
		QanCollect: new(agentpb.QANCollectRequest),
	})
	assert.Nil(t, resp)
}

func TestServerExitsWithUnknownError(t *testing.T) {
	connect := func(stream agentpb.Agent_ConnectServer) error {
		msg, err := stream.Recv()
		require.NoError(t, err)
		assert.EqualValues(t, 1, msg.Id)
		require.NotNil(t, msg.GetQanCollect())

		return io.EOF // any error without GRPCStatus() method
	}

	channel, _, teardown := setup(t, connect, status.Error(codes.Unknown, "EOF"))
	defer teardown(t)

	resp := channel.SendRequest(&agentpb.AgentMessage_QanCollect{
		QanCollect: new(agentpb.QANCollectRequest),
	})
	assert.Nil(t, resp)
}

func TestAgentClosesStream(t *testing.T) {
	connect := func(stream agentpb.Agent_ConnectServer) error { //nolint:unparam
		err := stream.Send(&agentpb.ServerMessage{
			Id: 1,
			Payload: &agentpb.ServerMessage_Ping{
				Ping: new(agentpb.Ping),
			},
		})
		assert.NoError(t, err)

		msg, err := stream.Recv()
		assert.Equal(t, io.EOF, err)
		assert.Nil(t, msg)

		return nil
	}

	channel, _, teardown := setup(t, connect, io.EOF)
	defer teardown(t)

	req := <-channel.Requests()
	ping := req.GetPing()
	assert.NotNil(t, ping)

	err := channel.s.CloseSend()
	assert.NoError(t, err)
}

func TestAgentClosesConnection(t *testing.T) {
	connect := func(stream agentpb.Agent_ConnectServer) error { //nolint:unparam
		err := stream.Send(&agentpb.ServerMessage{
			Id: 1,
			Payload: &agentpb.ServerMessage_Ping{
				Ping: new(agentpb.Ping),
			},
		})
		assert.NoError(t, err)

		msg, err := stream.Recv()
		assert.Equal(t, status.Error(codes.Canceled, context.Canceled.Error()), err)
		assert.Nil(t, msg)

		return nil
	}

	// gRPC library has a race in that case, so we can get two errors
	errClientConnClosing := status.Error(codes.Canceled, "grpc: the client connection is closing") // == grpc.ErrClientConnClosing
	errConnClosing := status.Error(codes.Unavailable, "transport is closing")
	channel, cc, teardown := setup(t, connect, errClientConnClosing, errConnClosing)
	defer teardown(t)

	req := <-channel.Requests()
	ping := req.GetPing()
	assert.NotNil(t, ping)

	err := cc.Close()
	assert.NoError(t, err)
}

func TestUnexpectedResponseFromServer(t *testing.T) {
	connect := func(stream agentpb.Agent_ConnectServer) error { //nolint:unparam
		// this message triggers "no subscriber for ID" error
		err := stream.Send(&agentpb.ServerMessage{
			Id: 111,
			Payload: &agentpb.ServerMessage_QanCollect{
				QanCollect: new(agentpb.QANCollectResponse),
			},
		})
		assert.NoError(t, err)

		// this message should not trigger new error
		err = stream.Send(&agentpb.ServerMessage{
			Id: 222,
			Payload: &agentpb.ServerMessage_QanCollect{
				QanCollect: new(agentpb.QANCollectResponse),
			},
		})
		assert.NoError(t, err)

		return nil
	}

	channel, _, teardown := setup(t, connect, fmt.Errorf("no subscriber for ID 111"))
	defer teardown(t)

	// after receiving unexpected response, channel is closed
	resp := channel.SendRequest(&agentpb.AgentMessage_QanCollect{
		QanCollect: new(agentpb.QANCollectRequest),
	})
	assert.Nil(t, resp)
	msg := <-channel.Requests()
	assert.Nil(t, msg)

	// future requests are ignored
	resp = channel.SendRequest(&agentpb.AgentMessage_QanCollect{
		QanCollect: new(agentpb.QANCollectRequest),
	})
	assert.Nil(t, resp)
	msg = <-channel.Requests()
	assert.Nil(t, msg)
}
