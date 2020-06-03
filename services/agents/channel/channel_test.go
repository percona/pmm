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

package channel

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
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm-managed/utils/interceptors"
)

type testServer struct {
	connectFunc func(agentpb.Agent_ConnectServer) error
}

func (s *testServer) Connect(stream agentpb.Agent_ConnectServer) error {
	return s.connectFunc(stream)
}

var _ agentpb.AgentServer = (*testServer)(nil)

func setup(t *testing.T, connect func(*Channel) error, expected ...error) (agentpb.Agent_ConnectClient, *grpc.ClientConn, func(*testing.T)) {
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
	agentpb.RegisterAgentServer(server, &testServer{
		connectFunc: func(stream agentpb.Agent_ConnectServer) error {
			channel = New(stream, NewSharedMetrics())
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
		grpc.WithInsecure(),
	}
	cc, err := grpc.DialContext(ctx, lis.Addr().String(), opts...)
	require.NoError(t, err, "failed to dial server")
	stream, err := agentpb.NewAgentClient(cc).Connect(ctx)
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
	connect := func(ch *Channel) error {
		channel = ch // store to check metrics below

		for i := uint32(1); i <= count; i++ {
			req := <-ch.Requests()
			require.NotNil(t, req)
			assert.Equal(t, i, req.ID)
			assert.IsType(t, new(agentpb.QANCollectRequest), req.Payload)

			ch.SendResponse(&ServerResponse{
				ID:      i,
				Payload: new(agentpb.QANCollectResponse),
			})
		}

		assert.Nil(t, <-ch.Requests())
		return nil
	}

	stream, _, teardown := setup(t, connect, io.EOF) // EOF = server exits from handler
	defer teardown(t)

	for i := uint32(1); i <= count; i++ {
		err := stream.Send(&agentpb.AgentMessage{
			Id:      i,
			Payload: new(agentpb.QANCollectRequest).AgentMessageRequestPayload(),
		})
		require.NoError(t, err)

		msg, err := stream.Recv()
		require.NoError(t, err)
		assert.Equal(t, i, msg.Id)
		assert.NotNil(t, msg.GetQanCollect())
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
# HELP pmm_managed_channel_messages_received_total A total number of messages received from pmm-agents.
# TYPE pmm_managed_channel_messages_received_total counter
pmm_managed_channel_messages_received_total 50
# HELP pmm_managed_channel_messages_sent_total A total number of messages sent to pmm-agents.
# TYPE pmm_managed_channel_messages_sent_total counter
pmm_managed_channel_messages_sent_total 50
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

	connect := func(ch *Channel) error {
		for i := uint32(1); i <= count; i++ {
			resp := ch.SendRequest(new(agentpb.Ping))
			pong := resp.(*agentpb.Pong)
			ts, err := ptypes.Timestamp(pong.CurrentTime)
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

		err = stream.Send(&agentpb.AgentMessage{
			Id: i,
			Payload: (&agentpb.Pong{
				CurrentTime: ptypes.TimestampNow(),
			}).AgentMessageResponsePayload(),
		})
		assert.NoError(t, err)
	}

	err := stream.CloseSend()
	assert.NoError(t, err)
}

func TestServerExitsWithGRPCError(t *testing.T) {
	errUnimplemented := status.Error(codes.Unimplemented, "Test error")
	connect := func(ch *Channel) error {
		req := <-ch.Requests()
		require.NotNil(t, req)
		assert.EqualValues(t, 1, req.ID)
		assert.IsType(t, new(agentpb.QANCollectRequest), req.Payload)

		return errUnimplemented
	}

	stream, _, teardown := setup(t, connect, status.Error(codes.Canceled, context.Canceled.Error()))
	defer teardown(t)

	err := stream.Send(&agentpb.AgentMessage{
		Id:      1,
		Payload: new(agentpb.QANCollectRequest).AgentMessageRequestPayload(),
	})
	assert.NoError(t, err)

	_, err = stream.Recv()
	assert.Equal(t, errUnimplemented, err)
}

func TestServerExitsWithUnknownErrorIntercepted(t *testing.T) {
	connect := func(ch *Channel) error {
		req := <-ch.Requests()
		require.NotNil(t, req)
		assert.EqualValues(t, 1, req.ID)
		assert.IsType(t, new(agentpb.QANCollectRequest), req.Payload)

		return io.EOF // any error without GRPCStatus() method
	}

	stream, _, teardown := setup(t, connect, status.Error(codes.Canceled, context.Canceled.Error()))
	defer teardown(t)

	err := stream.Send(&agentpb.AgentMessage{
		Id:      1,
		Payload: new(agentpb.QANCollectRequest).AgentMessageRequestPayload(),
	})
	assert.NoError(t, err)

	_, err = stream.Recv()
	assert.Equal(t, status.Error(codes.Internal, "Internal server error."), err)
}

func TestAgentClosesStream(t *testing.T) {
	connect := func(ch *Channel) error {
		resp := ch.SendRequest(new(agentpb.Ping))
		assert.Nil(t, resp)

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
	connect := func(ch *Channel) error {
		resp := ch.SendRequest(new(agentpb.Ping))
		assert.Nil(t, resp)

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
	connect := func(ch *Channel) error {
		// after receiving unexpected response, channel is closed
		resp := ch.SendRequest(new(agentpb.Ping))
		assert.Nil(t, resp)

		// future requests are ignored
		resp = ch.SendRequest(new(agentpb.Ping))
		assert.Nil(t, resp)

		return nil
	}

	stream, _, teardown := setup(t, connect, fmt.Errorf("no subscriber for ID 111"))
	defer teardown(t)

	// this message triggers "no subscriber for ID" error
	err := stream.Send(&agentpb.AgentMessage{
		Id:      111,
		Payload: new(agentpb.Pong).AgentMessageResponsePayload(),
	})
	assert.NoError(t, err)

	// this message should not trigger new error
	err = stream.Send(&agentpb.AgentMessage{
		Id:      222,
		Payload: new(agentpb.Pong).AgentMessageResponsePayload(),
	})
	assert.NoError(t, err)

	_, err = stream.Recv()
	assert.NoError(t, err)
}
