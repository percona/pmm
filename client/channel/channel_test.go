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

func (s *testServer) Connect(stream agentpb.Agent_ConnectServer) error {
	return s.connectFunc(stream)
}

var _ agentpb.AgentServer = (*testServer)(nil)

//nolint:nakedret
func setup(t *testing.T, connect func(agentpb.Agent_ConnectServer) error, expected ...error) (channel *Channel, cc *grpc.ClientConn, teardown func()) {
	// logrus.SetLevel(logrus.DebugLevel)

	t.Parallel()

	// start server with given connect handler
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer()
	agentpb.RegisterAgentServer(server, &testServer{
		connectFunc: connect,
	})

	// all assertions must happen in the main goroutine to avoid "panic: Fail in goroutine after XXX has completed"
	serveError := make(chan error)
	go func() {
		serveError <- server.Serve(lis)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

	// make client and channel
	opts := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithInsecure(),
	}
	cc, err = grpc.DialContext(ctx, lis.Addr().String(), opts...)
	require.NoError(t, err, "failed to dial server")
	stream, err := agentpb.NewAgentClient(cc).Connect(ctx)
	require.NoError(t, err, "failed to create stream")
	channel = New(stream)

	teardown = func() {
		err := channel.Wait()
		assert.Contains(t, expected, errors.Cause(err), "%+v", err)

		server.GracefulStop()
		cancel()
		require.NoError(t, <-serveError)
	}

	return
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
				Id:      i,
				Payload: new(agentpb.QANCollectResponse).ServerMessageResponsePayload(),
			})
			assert.NoError(t, err)
		}

		return nil
	}

	channel, _, teardown := setup(t, connect, io.EOF) // EOF = server exits from handler
	defer teardown()

	for i := uint32(1); i <= count; i++ {
		resp := channel.SendRequest(new(agentpb.QANCollectRequest))
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
				Id:      i,
				Payload: new(agentpb.Ping).ServerMessageRequestPayload(),
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
	defer teardown()

	for req := range channel.Requests() {
		assert.IsType(t, new(agentpb.Ping), req.Payload)

		channel.SendResponse(&AgentResponse{
			ID: req.ID,
			Payload: &agentpb.Pong{
				CurrentTime: ptypes.TimestampNow(),
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
	defer teardown()

	resp := channel.SendRequest(new(agentpb.QANCollectRequest))
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
	defer teardown()

	resp := channel.SendRequest(new(agentpb.QANCollectRequest))
	assert.Nil(t, resp)
}

func TestAgentClosesStream(t *testing.T) {
	connect := func(stream agentpb.Agent_ConnectServer) error { //nolint:unparam
		err := stream.Send(&agentpb.ServerMessage{
			Id:      1,
			Payload: new(agentpb.Ping).ServerMessageRequestPayload(),
		})
		assert.NoError(t, err)

		msg, err := stream.Recv()
		assert.Equal(t, io.EOF, err)
		assert.Nil(t, msg)

		return nil
	}

	channel, _, teardown := setup(t, connect, io.EOF)
	defer teardown()

	req := <-channel.Requests()
	require.NotNil(t, req)
	assert.IsType(t, new(agentpb.Ping), req.Payload)

	err := channel.s.CloseSend()
	assert.NoError(t, err)
}

func TestAgentClosesConnection(t *testing.T) {
	connect := func(stream agentpb.Agent_ConnectServer) error { //nolint:unparam
		err := stream.Send(&agentpb.ServerMessage{
			Id:      1,
			Payload: new(agentpb.Ping).ServerMessageRequestPayload(),
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
	defer teardown()

	req := <-channel.Requests()
	require.NotNil(t, req)
	assert.IsType(t, new(agentpb.Ping), req.Payload)

	err := cc.Close()
	assert.NoError(t, err)
}

func TestUnexpectedResponseFromServer(t *testing.T) {
	connect := func(stream agentpb.Agent_ConnectServer) error { //nolint:unparam
		// this message triggers "no subscriber for ID" error
		err := stream.Send(&agentpb.ServerMessage{
			Id:      111,
			Payload: new(agentpb.QANCollectResponse).ServerMessageResponsePayload(),
		})
		assert.NoError(t, err)

		// this message should not trigger new error
		err = stream.Send(&agentpb.ServerMessage{
			Id:      222,
			Payload: new(agentpb.QANCollectResponse).ServerMessageResponsePayload(),
		})
		assert.NoError(t, err)

		return nil
	}

	// TODO https://jira.percona.com/browse/PMM-3825
	channel, _, teardown := setup(t, connect, fmt.Errorf("no subscriber for ID 111"), io.EOF)
	defer teardown()

	// after receiving unexpected response, channel is closed
	resp := channel.SendRequest(new(agentpb.QANCollectRequest))
	assert.Nil(t, resp)
	msg := <-channel.Requests()
	assert.Nil(t, msg)

	// future requests are ignored
	resp = channel.SendRequest(new(agentpb.QANCollectRequest))
	assert.Nil(t, resp)
	msg = <-channel.Requests()
	assert.Nil(t, msg)
}
