// pmm-managed
// Copyright (C) 2017 Percona LLC
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
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/percona/exporter_shared/helpers"
	api "github.com/percona/pmm/api/agent"
	"github.com/pkg/errors"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm-managed/utils/interceptors"
)

type testServer struct {
	connectFunc func(api.Agent_ConnectServer) error
}

func (s *testServer) Register(context.Context, *api.RegisterRequest) (*api.RegisterResponse, error) {
	panic("not implemented")
}

func (s *testServer) Connect(stream api.Agent_ConnectServer) error {
	return s.connectFunc(stream)
}

var _ api.AgentServer = (*testServer)(nil)

func setup(t *testing.T, connect func(*Channel) error, expected ...error) (api.Agent_ConnectClient, *grpc.ClientConn, func(*testing.T)) {
	// logrus.SetLevel(logrus.DebugLevel)

	t.Parallel()

	// start server with given connect handler
	var channel *Channel
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := grpc.NewServer(
		grpc.UnaryInterceptor(interceptors.Unary),
		grpc.StreamInterceptor(interceptors.Stream),
	)
	api.RegisterAgentServer(server, &testServer{
		connectFunc: func(stream api.Agent_ConnectServer) error {
			channel = NewChannel(stream, newSharedMetrics())
			return connect(channel)
		},
	})
	go func() {
		err = server.Serve(lis)
		require.NoError(t, err)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

	// make client and channel
	opts := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithWaitForHandshake(),
		grpc.WithInsecure(),
	}
	cc, err := grpc.DialContext(ctx, lis.Addr().String(), opts...)
	require.NoError(t, err, "failed to dial server")
	stream, err := api.NewAgentClient(cc).Connect(ctx)
	require.NoError(t, err, "failed to create stream")

	teardown := func(t *testing.T) {
		require.NotNil(t, channel, "Test exited before first message reached connect handler.")

		err := channel.Wait()
		assert.Contains(t, expected, errors.Cause(err), "%+v", err)

		server.GracefulStop()
		cancel()
	}

	return stream, cc, teardown
}

func TestAgentRequest(t *testing.T) {
	const count = 50
	require.True(t, count > agentRequestsCap)

	var channel *Channel
	connect := func(ch *Channel) error { //nolint:unparam
		channel = ch // store to check metrics below

		for i := uint32(1); i <= count; i++ {
			msg := <-ch.Requests()
			require.NotNil(t, msg)
			assert.Equal(t, i, msg.Id)
			assert.NotNil(t, msg.GetQanData())

			ch.SendResponse(&api.ServerMessage{
				Id: i,
				Payload: &api.ServerMessage_QanData{
					QanData: new(api.QANDataResponse),
				},
			})
		}

		assert.Nil(t, <-ch.Requests())
		return nil
	}

	stream, _, teardown := setup(t, connect, io.EOF) // EOF = server exits from handler
	defer teardown(t)

	for i := uint32(1); i <= count; i++ {
		err := stream.Send(&api.AgentMessage{
			Id: i,
			Payload: &api.AgentMessage_QanData{
				QanData: new(api.QANDataRequest),
			},
		})
		require.NoError(t, err)

		msg, err := stream.Recv()
		require.NoError(t, err)
		assert.Equal(t, i, msg.Id)
		assert.NotNil(t, msg.GetQanData())
	}

	err := stream.CloseSend()
	assert.NoError(t, err)

	// check metrics
	metrics := make([]prom.Metric, 0, 100)
	metricsCh := make(chan prom.Metric)
	go func() {
		channel.metrics.Collect(metricsCh)
		close(metricsCh)
	}()
	for m := range metricsCh {
		metrics = append(metrics, m)
	}
	expectedMetrics := strings.Split(strings.TrimSpace(`
# HELP pmm_managed_agents_messages_received_total A total number of messages received from pmm-agents.
# TYPE pmm_managed_agents_messages_received_total counter
pmm_managed_agents_messages_received_total 50
# HELP pmm_managed_agents_messages_sent_total A total number of messages sent to pmm-agents.
# TYPE pmm_managed_agents_messages_sent_total counter
pmm_managed_agents_messages_sent_total 50
`), "\n")
	assert.Equal(t, expectedMetrics, helpers.Format(metrics))

	// check that descriptions match metrics: same number, same order
	descCh := make(chan *prom.Desc)
	go func() {
		channel.metrics.Describe(descCh)
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
	require.True(t, count > agentRequestsCap)

	connect := func(ch *Channel) error { //nolint:unparam
		for i := uint32(1); i <= count; i++ {
			msg := ch.SendRequest(&api.ServerMessage_Ping{
				Ping: new(api.Ping),
			})
			pong := msg.(*api.AgentMessage_Pong)
			ts, err := ptypes.Timestamp(pong.Pong.CurrentTime)
			assert.NoError(t, err)
			assert.InDelta(t, time.Now().Unix(), ts.Unix(), 1)
		}

		assert.Nil(t, <-ch.Requests())
		return nil
	}

	stream, _, teardown := setup(t, connect, io.EOF) // EOF = server exits from handler
	defer teardown(t)

	for i := uint32(1); i <= count; i++ {
		msg, err := stream.Recv()
		require.NoError(t, err)
		assert.Equal(t, i, msg.Id)
		assert.NotNil(t, msg.GetPing())

		err = stream.Send(&api.AgentMessage{
			Id: i,
			Payload: &api.AgentMessage_Pong{
				Pong: &api.Pong{
					CurrentTime: ptypes.TimestampNow(),
				},
			},
		})
		assert.NoError(t, err)
	}

	err := stream.CloseSend()
	assert.NoError(t, err)
}

func TestServerExitsWithGRPCError(t *testing.T) {
	errUnimplemented := status.Error(codes.Unimplemented, "Test error")
	connect := func(ch *Channel) error { //nolint:unparam
		msg := <-ch.Requests()
		require.NotNil(t, msg)
		assert.EqualValues(t, 1, msg.Id)
		assert.NotNil(t, msg.GetQanData())

		return errUnimplemented
	}

	stream, _, teardown := setup(t, connect, status.Error(codes.Canceled, context.Canceled.Error()))
	defer teardown(t)

	err := stream.Send(&api.AgentMessage{
		Id: 1,
		Payload: &api.AgentMessage_QanData{
			QanData: new(api.QANDataRequest),
		},
	})
	assert.NoError(t, err)

	_, err = stream.Recv()
	assert.Equal(t, errUnimplemented, err)
}

func TestServerExitsWithUnknownErrorIntercepted(t *testing.T) {
	connect := func(ch *Channel) error { //nolint:unparam
		msg := <-ch.Requests()
		require.NotNil(t, msg)
		assert.EqualValues(t, 1, msg.Id)
		assert.NotNil(t, msg.GetQanData())

		return io.EOF // any error without GRPCStatus() method
	}

	stream, _, teardown := setup(t, connect, status.Error(codes.Canceled, context.Canceled.Error()))
	defer teardown(t)

	err := stream.Send(&api.AgentMessage{
		Id: 1,
		Payload: &api.AgentMessage_QanData{
			QanData: new(api.QANDataRequest),
		},
	})
	assert.NoError(t, err)

	_, err = stream.Recv()
	assert.Equal(t, status.Error(codes.Internal, "Internal server error."), err)
}

func TestAgentClosesStream(t *testing.T) {
	connect := func(ch *Channel) error { //nolint:unparam
		msg := ch.SendRequest(&api.ServerMessage_Ping{
			Ping: new(api.Ping),
		})
		assert.Nil(t, msg)

		assert.Nil(t, <-ch.Requests())
		return nil
	}

	stream, _, teardown := setup(t, connect, io.EOF)
	defer teardown(t)

	msg, err := stream.Recv()
	require.NoError(t, err)
	assert.NotNil(t, msg)

	err = stream.CloseSend()
	assert.NoError(t, err)
}

func TestAgentClosesConnection(t *testing.T) {
	connect := func(ch *Channel) error { //nolint:unparam
		msg := ch.SendRequest(&api.ServerMessage_Ping{
			Ping: new(api.Ping),
		})
		assert.Nil(t, msg)

		assert.Nil(t, <-ch.Requests())
		return nil
	}

	stream, cc, teardown := setup(t, connect, status.Error(codes.Canceled, context.Canceled.Error()))
	defer teardown(t)

	msg, err := stream.Recv()
	require.NoError(t, err)
	assert.NotNil(t, msg)

	err = cc.Close()
	assert.NoError(t, err)
}

func TestUnexpectedResponseFromAgent(t *testing.T) {
	connect := func(ch *Channel) error { //nolint:unparam
		// after receiving unexpected response, channel is closed
		msg := ch.SendRequest(&api.ServerMessage_Ping{
			Ping: new(api.Ping),
		})
		assert.Nil(t, msg)

		// future requests are ignored
		msg = ch.SendRequest(&api.ServerMessage_Ping{
			Ping: new(api.Ping),
		})
		assert.Nil(t, msg)

		return nil
	}

	stream, _, teardown := setup(t, connect, fmt.Errorf("no subscriber for ID 111"))
	defer teardown(t)

	// this message triggers "no subscriber for ID" error
	err := stream.Send(&api.AgentMessage{
		Id: 111,
		Payload: &api.AgentMessage_Pong{
			Pong: new(api.Pong),
		},
	})
	assert.NoError(t, err)

	// this message should not trigger new error
	err = stream.Send(&api.AgentMessage{
		Id: 222,
		Payload: &api.AgentMessage_Pong{
			Pong: new(api.Pong),
		},
	})
	assert.NoError(t, err)

	_, err = stream.Recv()
	assert.NoError(t, err)
}
