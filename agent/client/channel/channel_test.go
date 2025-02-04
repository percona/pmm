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

package channel

import (
	"context"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/percona/exporter_shared/helpers"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/percona/pmm/agent/utils/truncate"
	agentv1 "github.com/percona/pmm/api/agent/v1"
)

type testServer struct {
	connectFunc func(server agentv1.AgentService_ConnectServer) error
	agentv1.UnimplementedAgentServiceServer
}

func (s *testServer) Connect(stream agentv1.AgentService_ConnectServer) error {
	return s.connectFunc(stream)
}

var _ agentv1.AgentServiceServer = (*testServer)(nil)

func setup(t *testing.T, connect func(agentv1.AgentService_ConnectServer) error, expected ...error) (*Channel, *grpc.ClientConn, func()) {
	t.Helper()

	var channel *Channel
	var cc *grpc.ClientConn
	var teardown func()
	// logrus.SetLevel(logrus.DebugLevel)

	// start server with given connect handler
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer()
	agentv1.RegisterAgentServiceServer(server, &testServer{
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
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	cc, err = grpc.DialContext(ctx, lis.Addr().String(), opts...)
	require.NoError(t, err, "failed to dial server")
	stream, err := agentv1.NewAgentServiceClient(cc).Connect(ctx)
	require.NoError(t, err, "failed to create stream")
	channel = New(stream)

	teardown = func() {
		err := channel.Wait()

		assert.Conditionf(t, func() (success bool) {
			for _, e := range expected {
				// have to use strings.Contains because grpc returns error with random ports in message.
				if errors.Is(err, e) || strings.Contains(err.Error(), e.Error()) {
					return true
				}
			}
			return false
		}, "%+v", err)
		// assert.Contains(t, expected, errors.Cause(err), "%+v", err)

		server.Stop()
		cancel()
		require.NoError(t, <-serveError)
	}

	return channel, cc, teardown
}

func TestAgentRequestWithTruncatedInvalidUTF8(t *testing.T) {
	defaultMaxQueryLength := truncate.GetDefaultMaxQueryLength()
	fingerprint, _ := truncate.Query("SELECT * FROM contacts t0 WHERE t0.person_id = '?';", defaultMaxQueryLength, truncate.GetDefaultMaxQueryLength())
	invalidQuery := "SELECT * FROM contacts t0 WHERE t0.person_id = '\u0241\xff\\uD83D\xddÃ¼\xf1'"
	query, _ := truncate.Query(invalidQuery, defaultMaxQueryLength, truncate.GetDefaultMaxQueryLength())

	connect := func(stream agentv1.AgentService_ConnectServer) error {
		msg, err := stream.Recv()
		require.NoError(t, err)
		assert.Equal(t, uint32(1), msg.Id)
		require.NotNil(t, msg.GetQanCollect())
		err = stream.Send(&agentv1.ServerMessage{
			Id:      uint32(1),
			Payload: (&agentv1.QANCollectResponse{}).ServerMessageResponsePayload(),
		})
		assert.NoError(t, err)
		assert.Equal(t, "SELECT * FROM contacts t0 WHERE t0.person_id = '\u0241\ufffd\\uD83D\ufffdÃ¼\ufffd'", msg.GetQanCollect().MetricsBucket[0].Common.Example)

		_, err = stream.Recv()
		require.EqualError(t, err, "rpc error: code = Canceled desc = context canceled")
		return nil
	}
	channel, _, teardown := setup(t, connect, status.Error(codes.Internal, `grpc: error while marshaling: string field contains invalid UTF-8`))
	defer teardown()
	var request agentv1.QANCollectRequest
	request.MetricsBucket = []*agentv1.MetricsBucket{{
		Common: &agentv1.MetricsBucket_Common{
			Fingerprint: fingerprint,
			Example:     query,
		},
		Mysql: &agentv1.MetricsBucket_MySQL{},
	}}
	resp, err := channel.SendAndWaitResponse(&request)
	require.NoError(t, err)
	assert.NotNil(t, resp)

	// Testing that it was failing with invalid query
	request.MetricsBucket = []*agentv1.MetricsBucket{{
		Common: &agentv1.MetricsBucket_Common{
			Fingerprint: fingerprint,
			Example:     invalidQuery,
		},
		Mysql: &agentv1.MetricsBucket_MySQL{},
	}}
	resp, err = channel.SendAndWaitResponse(&request)
	require.NoError(t, err)
	assert.Nil(t, resp)
}

func TestAgentRequest(t *testing.T) {
	const count = 50
	require.True(t, count > serverRequestsCap)

	connect := func(stream agentv1.AgentService_ConnectServer) error {
		for i := uint32(1); i <= count; i++ {
			msg, err := stream.Recv()
			require.NoError(t, err)
			assert.Equal(t, i, msg.Id)
			require.NotNil(t, msg.GetQanCollect())

			err = stream.Send(&agentv1.ServerMessage{
				Id:      i,
				Payload: (&agentv1.QANCollectResponse{}).ServerMessageResponsePayload(),
			})
			assert.NoError(t, err)
		}

		return nil
	}

	channel, _, teardown := setup(t, connect, io.EOF) // EOF = server exits from handler
	defer teardown()

	for i := uint32(1); i <= count; i++ {
		resp, err := channel.SendAndWaitResponse(&agentv1.QANCollectRequest{})
		require.NoError(t, err)
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

	connect := func(stream agentv1.AgentService_ConnectServer) error {
		for i := uint32(1); i <= count; i++ {
			err := stream.Send(&agentv1.ServerMessage{
				Id:      i,
				Payload: (&agentv1.Ping{}).ServerMessageRequestPayload(),
			})
			assert.NoError(t, err)
		}

		for i := uint32(1); i <= count; i++ {
			msg, err := stream.Recv()
			require.NoError(t, err)
			assert.Equal(t, i, msg.Id)
			pong := msg.GetPong()
			require.NotNil(t, pong)
			assert.InDelta(t, time.Now().Unix(), pong.CurrentTime.AsTime().Unix(), 1)
		}

		return nil
	}

	channel, _, teardown := setup(t, connect, io.EOF) // EOF = server exits from handler
	defer teardown()

	for req := range channel.Requests() {
		assert.IsType(t, &agentv1.Ping{}, req.Payload)

		channel.Send(&AgentResponse{
			ID: req.ID,
			Payload: &agentv1.Pong{
				CurrentTime: timestamppb.Now(),
			},
		})
	}
}

func TestServerExitsWithGRPCError(t *testing.T) {
	errUnimplemented := status.Error(codes.Unimplemented, "Test error")
	connect := func(stream agentv1.AgentService_ConnectServer) error {
		msg, err := stream.Recv()
		require.NoError(t, err)
		assert.EqualValues(t, 1, msg.Id)
		require.NotNil(t, msg.GetQanCollect())

		return errUnimplemented
	}

	channel, _, teardown := setup(t, connect, errUnimplemented)
	defer teardown()

	resp, err := channel.SendAndWaitResponse(&agentv1.QANCollectRequest{})
	require.NoError(t, err)
	assert.Nil(t, resp)
}

func TestServerExitsWithUnknownError(t *testing.T) {
	connect := func(stream agentv1.AgentService_ConnectServer) error {
		msg, err := stream.Recv()
		require.NoError(t, err)
		assert.EqualValues(t, 1, msg.Id)
		require.NotNil(t, msg.GetQanCollect())

		return io.EOF // any error without GRPCStatus() method
	}

	channel, _, teardown := setup(t, connect, status.Error(codes.Unknown, "EOF"))
	defer teardown()

	resp, err := channel.SendAndWaitResponse(&agentv1.QANCollectRequest{})
	require.NoError(t, err)
	assert.Nil(t, resp)
}

func TestAgentClosesStream(t *testing.T) {
	connect := func(stream agentv1.AgentService_ConnectServer) error {
		err := stream.Send(&agentv1.ServerMessage{
			Id:      1,
			Payload: (&agentv1.Ping{}).ServerMessageRequestPayload(),
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
	assert.IsType(t, &agentv1.Ping{}, req.Payload)

	err := channel.s.CloseSend()
	assert.NoError(t, err)
}

func TestAgentClosesConnection(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	connect := func(stream agentv1.AgentService_ConnectServer) error {
		defer wg.Done()
		err := stream.Send(&agentv1.ServerMessage{
			Id:      1,
			Payload: (&agentv1.Ping{}).ServerMessageRequestPayload(),
		})
		assert.NoError(t, err)

		msg, err := stream.Recv()
		assert.Equal(t, status.Error(codes.Canceled, context.Canceled.Error()).Error(), err.Error())
		assert.Nil(t, msg)

		return nil
	}

	// gRPC library has a race in that case, so we can get three errors
	errClientConnClosing := status.Error(codes.Canceled, "grpc: the client connection is closing") // == grpc.ErrClientConnClosing
	errConnClosing := status.Error(codes.Unavailable, "transport is closing")
	// For an explanation of why we are using a dynamic error here, and why we are comparing the string representation of this error, see:
	// https://github.com/golang/go/issues/4373
	// https://github.com/golang/go/blob/master/src/internal/poll/fd.go#L20
	errConnClosed := errors.New("use of closed network connection")
	channel, cc, teardown := setup(t, connect, errClientConnClosing, errConnClosing, errConnClosed) //nolint:varnamelen
	defer teardown()

	req := <-channel.Requests()
	require.NotNil(t, req)
	assert.IsType(t, &agentv1.Ping{}, req.Payload)

	err := cc.Close()
	assert.NoError(t, err)
	wg.Wait()
}

func TestUnexpectedResponseIDFromServer(t *testing.T) {
	unexpectedIDSent := make(chan struct{})
	connect := func(stream agentv1.AgentService_ConnectServer) error {
		// This message triggers no error, we ignore message ids that have no subscriber.
		err := stream.Send(&agentv1.ServerMessage{
			Id:      111,
			Payload: (&agentv1.QANCollectResponse{}).ServerMessageResponsePayload(),
		})
		assert.NoError(t, err)
		close(unexpectedIDSent)

		// Check that channel is still open.
		err = stream.Send(&agentv1.ServerMessage{
			Id:      1,
			Payload: (&agentv1.Ping{}).ServerMessageRequestPayload(),
		})
		assert.NoError(t, err)
		pong, err := stream.Recv()
		assert.NoError(t, err)
		assert.NotNil(t, pong)
		return nil
	}
	channel, _, teardown := setup(t, connect, io.EOF)
	defer teardown()

	<-unexpectedIDSent
	// Get the ping message and send pong response, channel stays open after message with unexpected id.
	msg := <-channel.Requests()
	assert.NotNil(t, msg)
	channel.send(&agentv1.AgentMessage{
		Id:      1,
		Payload: (&agentv1.Pong{}).AgentMessageResponsePayload(),
	})
}

func TestUnexpectedResponsePayloadFromServer(t *testing.T) {
	connect := func(stream agentv1.AgentService_ConnectServer) error {
		// establish the connection
		err := stream.Send(&agentv1.ServerMessage{
			Id:      1,
			Payload: (&agentv1.Ping{}).ServerMessageRequestPayload(),
		})
		assert.NoError(t, err)
		_, _ = stream.Recv()

		// test unexpected payload
		err = stream.Send(&agentv1.ServerMessage{
			Id: 4242,
		})
		require.NoError(t, err)

		msg, err := stream.Recv()
		assert.NoError(t, err)
		assert.Equal(t, int32(codes.Unimplemented), msg.GetStatus().GetCode())
		return nil
	}
	channel, _, teardown := setup(t, connect, io.EOF)
	defer teardown()
	req := <-channel.Requests()
	channel.Send(&AgentResponse{
		ID: req.ID,
		Payload: &agentv1.Pong{
			CurrentTime: timestamppb.Now(),
		},
	})
}
