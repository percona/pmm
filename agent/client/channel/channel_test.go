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
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/percona/exporter_shared/helpers"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
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
	agentv1.UnimplementedAgentServiceServer

	connectFunc func(server agentv1.AgentService_ConnectServer) error
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
		grpc.WithDefaultCallOptions(
			// Wait for connection to be ready before sending RPC calls
			grpc.WaitForReady(true),
		),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	cc, err = grpc.NewClient(lis.Addr().String(), opts...)
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
		require.NoError(t, err)
		assert.Equal(t, "SELECT * FROM contacts t0 WHERE t0.person_id = '\u0241\ufffd\\uD83D\ufffdÃ¼\ufffd'", msg.GetQanCollect().MetricsBucket[0].Common.Example)

		_, err = stream.Recv()
		require.EqualError(t, err, "rpc error: code = Canceled desc = context canceled")
		return nil
	}
	channel, _, teardown := setup(t, connect, status.Error(codes.Internal, `grpc: error while marshaling: string field contains invalid UTF-8`))
	t.Cleanup(teardown)

	var request agentv1.QANCollectRequest
	request.MetricsBucket = []*agentv1.MetricsBucket{{
		Common: &agentv1.MetricsBucket_Common{
			Fingerprint: fingerprint,
			Example:     query,
		},
		Mysql: &agentv1.MetricsBucket_MySQL{},
	}}
	resp, err := channel.SendAndWaitResponse(t.Context(), &request)
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
	resp, err = channel.SendAndWaitResponse(t.Context(), &request)
	require.NoError(t, err)
	assert.Nil(t, resp)
}

func TestAgentRequest(t *testing.T) {
	const count = 50
	require.Greater(t, count, serverRequestsCap)

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
			require.NoError(t, err)
		}

		return nil
	}

	channel, _, teardown := setup(t, connect, io.EOF) // EOF = server exits from handler
	t.Cleanup(teardown)

	for i := uint32(1); i <= count; i++ {
		resp, err := channel.SendAndWaitResponse(t.Context(), &agentv1.QANCollectRequest{})
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
	require.Greater(t, count, serverRequestsCap)

	connect := func(stream agentv1.AgentService_ConnectServer) error {
		for i := uint32(1); i <= count; i++ {
			err := stream.Send(&agentv1.ServerMessage{
				Id:      i,
				Payload: (&agentv1.SetStateRequest{}).ServerMessageRequestPayload(),
			})
			require.NoError(t, err)
		}

		for i := uint32(1); i <= count; i++ {
			msg, err := stream.Recv()
			require.NoError(t, err)
			assert.Equal(t, i, msg.Id)
			require.NotNil(t, msg.GetSetState())
		}

		return nil
	}

	channel, _, teardown := setup(t, connect, io.EOF) // EOF = server exits from handler
	t.Cleanup(teardown)

	for req := range channel.Requests() {
		assert.IsType(t, &agentv1.SetStateRequest{}, req.Payload)

		channel.Send(&AgentResponse{
			ID:      req.ID,
			Payload: &agentv1.SetStateResponse{},
		})
	}
}

func TestServerPing(t *testing.T) {
	const count = 50

	connect := func(stream agentv1.AgentService_ConnectServer) error {
		for i := uint32(1); i <= count; i++ {
			err := stream.Send(&agentv1.ServerMessage{
				Id:      i,
				Payload: (&agentv1.Ping{}).ServerMessageRequestPayload(),
			})
			require.NoError(t, err)

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
	t.Cleanup(teardown)

	for req := range channel.Pings() {
		assert.IsType(t, &agentv1.Ping{}, req.Payload)

		channel.Send(&AgentResponse{
			ID: req.ID,
			Payload: &agentv1.Pong{
				CurrentTime: timestamppb.Now(),
			},
		})
	}
}

// TestServerPingWhileRequestsPileUp covers half of PMM-15431: a busy consumer of Requests() used
// to delay every pong behind the requests it had not got to yet.
func TestServerPingWhileRequestsPileUp(t *testing.T) {
	const count = serverRequestsCap - 1

	pongReceived := make(chan struct{})
	connect := func(stream agentv1.AgentService_ConnectServer) error {
		// Nobody reads Requests() in this test, so these just pile up in the queue.
		for i := uint32(1); i <= count; i++ {
			err := stream.Send(&agentv1.ServerMessage{
				Id:      i,
				Payload: (&agentv1.SetStateRequest{}).ServerMessageRequestPayload(),
			})
			require.NoError(t, err)
		}

		err := stream.Send(&agentv1.ServerMessage{
			Id:      count + 1,
			Payload: (&agentv1.Ping{}).ServerMessageRequestPayload(),
		})
		require.NoError(t, err)

		msg, err := stream.Recv()
		require.NoError(t, err)
		assert.EqualValues(t, count+1, msg.Id)
		require.NotNil(t, msg.GetPong())
		close(pongReceived)

		return nil
	}

	channel, _, teardown := setup(t, connect, io.EOF) // EOF = server exits from handler
	t.Cleanup(teardown)

	req := <-channel.Pings()
	require.NotNil(t, req)
	assert.EqualValues(t, count+1, req.ID)
	channel.Send(&AgentResponse{
		ID: req.ID,
		Payload: &agentv1.Pong{
			CurrentTime: timestamppb.Now(),
		},
	})

	<-pongReceived
}

// TestFullRequestQueueDoesNotWedgeReceiver covers the other half of PMM-15431: once the queue was
// full, runReceiver parked on a bare channel send that nothing could interrupt, so closing the
// channel left it - and the only goroutine able to deliver a response - stuck for good.
func TestFullRequestQueueDoesNotWedgeReceiver(t *testing.T) {
	const count = serverRequestsCap + 5

	errClosed := errors.New("closed while the request queue was full")
	queueFilled := make(chan struct{})
	connect := func(stream agentv1.AgentService_ConnectServer) error {
		// Nobody reads Requests() in this test, so these overflow the queue and park
		// runReceiver on the send of the first one that does not fit.
		for i := uint32(1); i <= count; i++ {
			err := stream.Send(&agentv1.ServerMessage{
				Id:      i,
				Payload: (&agentv1.SetStateRequest{}).ServerMessageRequestPayload(),
			})
			require.NoError(t, err)
		}
		close(queueFilled)

		_, err := stream.Recv()
		require.Error(t, err)

		return nil
	}

	channel, _, teardown := setup(t, connect, errClosed)
	t.Cleanup(teardown)

	<-queueFilled
	requireQueueFull(t, channel)

	channel.close(errClosed)

	assertReceiverStopped(t, channel)
}

// TestFullRequestQueueGivesUpOnConnection covers the recovery path: a consumer that never resumes
// draining leaves runReceiver parked, and only giving up on the connection gets the agent back -
// nothing else can, since Run does not return while its goroutines are all still parked.
func TestFullRequestQueueGivesUpOnConnection(t *testing.T) {
	const count = serverRequestsCap + 5

	restore := requestQueueStuckTimeout
	requestQueueStuckTimeout = 200 * time.Millisecond
	t.Cleanup(func() { requestQueueStuckTimeout = restore })

	connect := func(stream agentv1.AgentService_ConnectServer) error {
		// Nobody reads Requests() in this test, so these overflow the queue.
		for i := uint32(1); i <= count; i++ {
			err := stream.Send(&agentv1.ServerMessage{
				Id:      i,
				Payload: (&agentv1.SetStateRequest{}).ServerMessageRequestPayload(),
			})
			require.NoError(t, err)
		}

		_, err := stream.Recv()
		require.Error(t, err)

		return nil
	}

	channel, _, teardown := setup(t, connect, errors.New("request queue full for"))
	t.Cleanup(teardown)

	requireQueueFull(t, channel)

	// Nothing is drained and nothing is closed from this side, so the only way Wait returns is
	// the receiver giving up on the connection by itself.
	require.ErrorContains(t, channel.Wait(), "request queue full for")
	assertReceiverStopped(t, channel)
}

func requireQueueFull(t *testing.T, channel *Channel) {
	t.Helper()

	require.Eventually(t, func() bool {
		return len(channel.requests) == serverRequestsCap
	}, 3*time.Second, 10*time.Millisecond)
}

// assertReceiverStopped drains the queue runReceiver owns and asserts that it closed it, which it
// does only on its way out.
func assertReceiverStopped(t *testing.T, channel *Channel) {
	t.Helper()

	for range serverRequestsCap {
		require.NotNil(t, <-channel.Requests())
	}
	_, more := <-channel.Requests()
	assert.False(t, more)
	_, more = <-channel.Pings()
	assert.False(t, more)
}

func TestSendAndWaitResponseCanceled(t *testing.T) {
	asserted := make(chan struct{})
	connect := func(stream agentv1.AgentService_ConnectServer) error {
		// Receive the request, but never answer it.
		msg, err := stream.Recv()
		require.NoError(t, err)
		require.NotNil(t, msg.GetQanCollect())

		// Keep the channel open until the caller has given up on its own, so that giving up
		// is the only thing that can unblock it.
		<-asserted

		return nil
	}

	channel, _, teardown := setup(t, connect, io.EOF) // EOF = server exits from handler
	t.Cleanup(teardown)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	resp, err := channel.SendAndWaitResponse(ctx, &agentv1.QANCollectRequest{})
	assert.Nil(t, resp)
	require.ErrorIs(t, err, context.Canceled)
	close(asserted)
}

func TestAbandon(t *testing.T) {
	t.Parallel()

	// The interleaving that matters cannot be forced through the gRPC harness, so drive the
	// primitives directly: only `responses` and `l` are touched by the paths under test.
	newChannel := func() *Channel {
		return &Channel{
			responses: make(map[uint32]chan Response),
			l:         logrus.WithField("test", t.Name()),
		}
	}

	t.Run("marks a request that is still tracked", func(t *testing.T) {
		t.Parallel()

		c := newChannel()
		c.subscribe(1)

		assert.True(t, c.abandon(1))
		// Tracked as abandoned rather than dropped, so a late response is recognized.
		assert.Len(t, c.responses, 1)
		assert.Nil(t, c.responses[1])
	})

	t.Run("does not resurrect an entry the publisher already took", func(t *testing.T) {
		t.Parallel()

		// The waiter sees ctx expire, then the publisher removes the entry and delivers
		// before the waiter marks it. Marking it anyway would leave an entry that no future
		// response can ever clear, because the response was already published.
		c := newChannel()
		ch := c.subscribe(1)

		c.publish(1, nil, &agentv1.Pong{CurrentTime: timestamppb.Now()})

		assert.False(t, c.abandon(1))
		assert.Empty(t, c.responses)

		// The response the publisher delivered is still there to be collected, which is
		// what SendAndWaitResponse falls back to rather than reporting a timeout.
		resp := <-ch
		require.NoError(t, resp.Error)
		assert.IsType(t, &agentv1.Pong{}, resp.Payload)
	})

	t.Run("is a no-op once the channel is closed", func(t *testing.T) {
		t.Parallel()

		c := newChannel()
		c.subscribe(1)
		c.responses = nil

		assert.False(t, c.abandon(1))
	})

	t.Run("reports a response that arrives after the sender gave up", func(t *testing.T) {
		t.Parallel()

		l, hook := logrustest.NewNullLogger()
		l.SetLevel(logrus.DebugLevel)
		c := newChannel()
		c.l = l.WithField("test", t.Name())

		c.subscribe(1)
		require.True(t, c.abandon(1))

		// No subscriber left, so this must neither block nor be reported as a response
		// to an ID the agent never sent.
		c.publish(1, nil, &agentv1.Pong{CurrentTime: timestamppb.Now()})

		assert.Empty(t, c.responses)
		entries := hook.AllEntries()
		require.Len(t, entries, 1)
		assert.Equal(t, logrus.DebugLevel, entries[0].Level)
	})
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
	t.Cleanup(teardown)

	resp, err := channel.SendAndWaitResponse(t.Context(), &agentv1.QANCollectRequest{})
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
	t.Cleanup(teardown)

	resp, err := channel.SendAndWaitResponse(t.Context(), &agentv1.QANCollectRequest{})
	require.NoError(t, err)
	assert.Nil(t, resp)
}

func TestAgentClosesStream(t *testing.T) {
	connect := func(stream agentv1.AgentService_ConnectServer) error {
		err := stream.Send(&agentv1.ServerMessage{
			Id:      1,
			Payload: (&agentv1.Ping{}).ServerMessageRequestPayload(),
		})
		require.NoError(t, err)

		msg, err := stream.Recv()
		assert.Equal(t, io.EOF, err)
		assert.Nil(t, msg)

		return nil
	}

	channel, _, teardown := setup(t, connect, io.EOF)
	t.Cleanup(teardown)

	req := <-channel.Pings()
	require.NotNil(t, req)
	assert.IsType(t, &agentv1.Ping{}, req.Payload)

	err := channel.s.CloseSend()
	require.NoError(t, err)
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
		require.NoError(t, err)

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
	t.Cleanup(teardown)

	req := <-channel.Pings()
	require.NotNil(t, req)
	assert.IsType(t, &agentv1.Ping{}, req.Payload)

	err := cc.Close()
	require.NoError(t, err)
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
		require.NoError(t, err)
		close(unexpectedIDSent)

		// Check that channel is still open.
		err = stream.Send(&agentv1.ServerMessage{
			Id:      1,
			Payload: (&agentv1.Ping{}).ServerMessageRequestPayload(),
		})
		require.NoError(t, err)
		pong, err := stream.Recv()
		require.NoError(t, err)
		assert.NotNil(t, pong)
		return nil
	}
	channel, _, teardown := setup(t, connect, io.EOF)
	t.Cleanup(teardown)

	<-unexpectedIDSent
	// Get the ping message and send pong response, channel stays open after message with unexpected id.
	msg := <-channel.Pings()
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
		require.NoError(t, err)
		_, _ = stream.Recv()

		// test unexpected payload
		err = stream.Send(&agentv1.ServerMessage{
			Id: 4242,
		})
		require.NoError(t, err)

		msg, err := stream.Recv()
		require.NoError(t, err)
		assert.Equal(t, int32(codes.Unimplemented), msg.GetStatus().GetCode())
		return nil
	}
	channel, _, teardown := setup(t, connect, io.EOF)
	t.Cleanup(teardown)
	req := <-channel.Pings()
	channel.Send(&AgentResponse{
		ID: req.ID,
		Payload: &agentv1.Pong{
			CurrentTime: timestamppb.Now(),
		},
	})
}
