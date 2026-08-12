// Copyright (C) 2023 Percona LLC
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
	"io"
	"net"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	"github.com/percona/pmm/managed/utils/interceptors"
	"github.com/percona/pmm/managed/utils/tests"
)

type testServer struct {
	agentv1.UnimplementedAgentServiceServer

	connectFunc func(server agentv1.AgentService_ConnectServer) error
}

func (s *testServer) Connect(stream agentv1.AgentService_ConnectServer) error {
	return s.connectFunc(stream)
}

var _ agentv1.AgentServiceServer = (*testServer)(nil)

func setup(t *testing.T, connect func(context.Context, *Channel) error, expected error) (agentv1.AgentService_ConnectClient, *grpc.ClientConn) {
	t.Helper()

	// start server with given connect handler
	var channel *Channel
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	grpcMetrics := interceptors.NewServerMetricsWithExtension(&interceptors.GRPCMetricsExtension{})
	grpcStreamInterceptor := grpcMetrics.StreamServerInterceptor()
	grpcUnaryInterceptor := grpcMetrics.UnaryServerInterceptor()

	server := grpc.NewServer(
		grpc.UnaryInterceptor(interceptors.UnaryAdd(grpcUnaryInterceptor)),
		grpc.StreamInterceptor(interceptors.Stream(grpcStreamInterceptor)),
	)

	agentv1.RegisterAgentServiceServer(server, &testServer{
		connectFunc: func(stream agentv1.AgentService_ConnectServer) error {
			channel = New(stream.Context(), stream)
			return connect(stream.Context(), channel)
		},
	})
	serveError := make(chan error)
	go func() {
		serveError <- server.Serve(lis)
	}()
	t.Cleanup(func() {
		require.NoError(t, <-serveError)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)

	// make client and channel
	opts := []grpc.DialOption{
		grpc.WithDefaultCallOptions(
			// Wait for connection to be ready before sending RPC calls
			grpc.WaitForReady(true),
		),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	cc, err := grpc.NewClient(lis.Addr().String(), opts...)
	require.NoError(t, err, "failed to dial server")
	stream, err := agentv1.NewAgentServiceClient(cc).Connect(ctx)
	require.NoError(t, err, "failed to create stream")

	t.Cleanup(func() {
		require.NotNil(t, channel, "Test exited before first message reached connect handler.")
		require.ErrorContains(t, channel.Wait(), expected.Error())
		server.GracefulStop()
	})

	return stream, cc
}

func TestAgentRequest(t *testing.T) {
	t.Parallel()

	const count = 50
	require.Greater(t, count, agentRequestsCap)

	var channel *Channel
	connect := func(_ context.Context, ch *Channel) error {
		channel = ch // store to check metrics below

		for i := uint32(1); i <= count; i++ {
			req := <-ch.Requests()
			require.NotNil(t, req)
			assert.Equal(t, i, req.ID)
			assert.IsType(t, &agentv1.QANCollectRequest{}, req.Payload)

			ch.Send(&ServerResponse{
				ID:      i,
				Payload: &agentv1.QANCollectResponse{},
			})
		}

		assert.Nil(t, <-ch.Requests())
		return nil
	}

	stream, _ := setup(t, connect, io.EOF) // EOF = server exits from handler

	for i := uint32(1); i <= count; i++ {
		collectReq := &agentv1.QANCollectRequest{}
		err := stream.Send(&agentv1.AgentMessage{
			Id:      i,
			Payload: collectReq.AgentMessageRequestPayload(),
		})
		require.NoError(t, err)

		msg, err := stream.Recv()
		require.NoError(t, err)
		assert.Equal(t, i, msg.Id)
		assert.NotNil(t, msg.GetQanCollect())
	}

	err := stream.CloseSend()
	require.NoError(t, err)

	// check metrics
	expectedMetrics := &Metrics{
		Sent: 50,
		Recv: 50,
	}
	assert.Equal(t, expectedMetrics, channel.Metrics())
}

func TestServerRequest(t *testing.T) {
	t.Parallel()

	const count = 50
	require.Greater(t, count, agentRequestsCap)

	connect := func(ctx context.Context, ch *Channel) error {
		for i := uint32(1); i <= count; i++ {
			resp, err := ch.SendAndWaitResponse(ctx, &agentv1.Ping{})
			require.NoError(t, err)
			pong := resp.(*agentv1.Pong)
			ts := pong.CurrentTime.AsTime()
			err = pong.CurrentTime.CheckValid()
			require.NoError(t, err)
			assert.InDelta(t, time.Now().Unix(), ts.Unix(), 1)
		}

		assert.Nil(t, <-ch.Requests())
		return nil
	}

	stream, _ := setup(t, connect, io.EOF) // EOF = server exits from handler

	for i := uint32(1); i <= count; i++ {
		msg, err := stream.Recv()
		require.NoError(t, err)
		assert.Equal(t, i, msg.Id)
		assert.NotNil(t, msg.GetPing())

		err = stream.Send(&agentv1.AgentMessage{
			Id: i,
			Payload: (&agentv1.Pong{
				CurrentTime: timestamppb.Now(),
			}).AgentMessageResponsePayload(),
		})
		require.NoError(t, err)
	}

	err := stream.CloseSend()
	require.NoError(t, err)
}

func TestServerRequestTimeout(t *testing.T) {
	t.Parallel()

	timedOut := make(chan struct{})
	connect := func(ctx context.Context, ch *Channel) error {
		// connect runs on the gRPC handler goroutine while the test goroutine waits on
		// timedOut. Closing it from a defer keeps a failed require, which exits this
		// goroutine, from deadlocking the test instead of failing it.
		func() {
			defer close(timedOut)

			ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
			defer cancel()

			// The agent never answers this ping, exactly as it would on a silently dropped
			// connection. Waiting for it must not block forever. See PMM-15310.
			resp, err := ch.SendAndWaitResponse(ctx, &agentv1.Ping{})
			assert.Nil(t, resp)
			require.ErrorIs(t, err, context.DeadlineExceeded)

			// The abandoned request must not be counted as a queued response.
			assert.Zero(t, ch.Metrics().Responses)
		}()

		assert.Nil(t, <-ch.Requests())
		return nil
	}

	// EOF = server exits from handler
	stream, _ := setup(t, connect, io.EOF)

	msg, err := stream.Recv()
	require.NoError(t, err)
	assert.NotNil(t, msg.GetPing())

	<-timedOut

	err = stream.CloseSend()
	require.NoError(t, err)
}

func TestLateResponseAfterTimeout(t *testing.T) {
	t.Parallel()

	timedOut := make(chan struct{})
	lateSent := make(chan struct{})
	connect := func(ctx context.Context, ch *Channel) error {
		// See the note in TestServerRequestTimeout on closing timedOut from a defer.
		func() {
			defer close(timedOut)

			ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
			defer cancel()

			_, err := ch.SendAndWaitResponse(ctx, &agentv1.Ping{})
			require.ErrorIs(t, err, context.DeadlineExceeded)
		}()

		<-lateSent

		// The second exchange proves the late response for ID 1 has already been handled,
		// because runReceiver processes messages in order.
		resp, err := ch.SendAndWaitResponse(ctx, &agentv1.Ping{})
		require.NoError(t, err)
		assert.NotNil(t, resp)

		// A response to an abandoned request is expected, not unsolicited, so the channel
		// recognizes it and drops the entry it was tracking. See PMM-15310.
		ch.rw.RLock()
		tracked := len(ch.responses)
		ch.rw.RUnlock()
		assert.Zero(t, tracked)

		assert.Nil(t, <-ch.Requests())
		return nil
	}

	// EOF = server exits from handler
	stream, _ := setup(t, connect, io.EOF)

	// ID 1 is answered only after the wait for it has already timed out.
	msg, err := stream.Recv()
	require.NoError(t, err)
	assert.EqualValues(t, 1, msg.Id)

	<-timedOut

	err = stream.Send(&agentv1.AgentMessage{
		Id: 1,
		Payload: (&agentv1.Pong{
			CurrentTime: timestamppb.Now(),
		}).AgentMessageResponsePayload(),
	})
	require.NoError(t, err)
	close(lateSent)

	// ID 2 is answered normally.
	msg, err = stream.Recv()
	require.NoError(t, err)
	assert.EqualValues(t, 2, msg.Id)

	err = stream.Send(&agentv1.AgentMessage{
		Id: 2,
		Payload: (&agentv1.Pong{
			CurrentTime: timestamppb.Now(),
		}).AgentMessageResponsePayload(),
	})
	require.NoError(t, err)

	err = stream.CloseSend()
	require.NoError(t, err)
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
		assert.Zero(t, c.Metrics().Responses)
	})

	t.Run("does not resurrect an entry the publisher already took", func(t *testing.T) {
		// The waiter sees an empty response channel, then the publisher removes the entry and
		// delivers before the waiter marks it. Marking it anyway would leave an entry that no
		// future response can ever clear, because the response was already published.
		// See PMM-15310.
		t.Parallel()

		c := newChannel()
		ch := c.subscribe(1)

		c.publish(1, nil, &agentv1.Pong{CurrentTime: timestamppb.Now()})

		assert.False(t, c.abandon(1))
		assert.Empty(t, c.responses)

		// The response the publisher delivered is still there to be collected.
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
}

func TestDeliver(t *testing.T) {
	t.Parallel()

	// What deliver says about a response is the whole point of the four states, so give each
	// channel its own logger to read back.
	newChannel := func() (*Channel, *logrustest.Hook) {
		l, hook := logrustest.NewNullLogger()
		l.SetLevel(logrus.DebugLevel)

		return &Channel{
			responses: make(map[uint32]chan Response),
			l:         l.WithField("test", t.Name()),
		}, hook
	}

	pong := func() Response {
		return Response{Payload: &agentv1.Pong{CurrentTime: timestamppb.Now()}}
	}

	levels := func(hook *logrustest.Hook) []logrus.Level {
		out := make([]logrus.Level, 0, len(hook.Entries))
		for _, e := range hook.Entries {
			out = append(out, e.Level)
		}

		return out
	}

	t.Run("hands the response to a waiting sender", func(t *testing.T) {
		t.Parallel()

		c, hook := newChannel()
		ch := c.subscribe(1)

		c.deliver(1, pong())

		resp := <-ch
		require.NoError(t, resp.Error)
		assert.IsType(t, &agentv1.Pong{}, resp.Payload)
		assert.Empty(t, levels(hook))
		assert.Empty(t, c.responses)
	})

	t.Run("closes the subscription after an error", func(t *testing.T) {
		t.Parallel()

		c, hook := newChannel()
		ch := c.subscribe(1)

		c.deliver(1, Response{Error: errChannelClosed})

		resp := <-ch
		require.ErrorIs(t, resp.Error, errChannelClosed)
		// Nothing can follow an error, so the sender must see the channel closed.
		_, ok := <-ch
		assert.False(t, ok)
		assert.Empty(t, levels(hook))
	})

	t.Run("reports an abandoned request at debug level", func(t *testing.T) {
		t.Parallel()

		c, hook := newChannel()
		c.subscribe(1)
		require.True(t, c.abandon(1))

		c.deliver(1, pong())

		// Expected, not unsolicited: the marker is what keeps this off the error log.
		assert.Equal(t, []logrus.Level{logrus.DebugLevel}, levels(hook))
		assert.Empty(t, c.responses)
	})

	t.Run("reports an unknown ID as an error", func(t *testing.T) {
		t.Parallel()

		c, hook := newChannel()

		c.deliver(1, pong())

		assert.Equal(t, []logrus.Level{logrus.ErrorLevel}, levels(hook))
	})

	t.Run("stays silent once the channel is closed", func(t *testing.T) {
		t.Parallel()

		c, hook := newChannel()
		c.subscribe(1)
		c.responses = nil

		// close unblocks every sender itself, so a response racing it is not worth a word -
		// least of all "no subscriber", which would show up on any rough disconnect.
		c.deliver(1, pong())

		assert.Empty(t, levels(hook))
	})
}

func TestServerExitsWithGRPCError(t *testing.T) {
	t.Parallel()

	errUnimplemented := status.Error(codes.Unimplemented, "Test error")
	connect := func(_ context.Context, ch *Channel) error {
		req := <-ch.Requests()
		require.NotNil(t, req)
		assert.EqualValues(t, 1, req.ID)
		assert.IsType(t, &agentv1.QANCollectRequest{}, req.Payload)

		return errUnimplemented
	}

	stream, _ := setup(t, connect, status.Error(codes.Canceled, context.Canceled.Error()))

	collectReq := &agentv1.QANCollectRequest{}
	err := stream.Send(&agentv1.AgentMessage{
		Id:      1,
		Payload: collectReq.AgentMessageRequestPayload(),
	})
	require.NoError(t, err)

	_, err = stream.Recv()
	require.ErrorIs(t, err, errUnimplemented)
}

func TestServerExitsWithUnknownErrorIntercepted(t *testing.T) {
	t.Parallel()

	connect := func(_ context.Context, ch *Channel) error {
		req := <-ch.Requests()
		require.NotNil(t, req)
		assert.EqualValues(t, 1, req.ID)
		assert.IsType(t, &agentv1.QANCollectRequest{}, req.Payload)

		return io.EOF // any error without GRPCStatus() method
	}

	stream, _ := setup(t, connect, status.Error(codes.Canceled, context.Canceled.Error()))

	collectReq := &agentv1.QANCollectRequest{}
	err := stream.Send(&agentv1.AgentMessage{
		Id:      1,
		Payload: collectReq.AgentMessageRequestPayload(),
	})
	require.NoError(t, err)

	_, err = stream.Recv()
	tests.AssertGRPCError(t, status.New(codes.Internal, "Internal server error."), err)
}

func TestAgentClosesStream(t *testing.T) {
	t.Parallel()

	connect := func(ctx context.Context, ch *Channel) error {
		resp, err := ch.SendAndWaitResponse(ctx, &agentv1.Ping{})
		require.Errorf(t, err, "channel is closed")
		assert.Nil(t, resp)

		assert.Nil(t, <-ch.Requests())
		return nil
	}

	stream, _ := setup(t, connect, io.EOF)

	msg, err := stream.Recv()
	require.NoError(t, err)
	assert.NotNil(t, msg)

	err = stream.CloseSend()
	require.NoError(t, err)
}

func TestAgentClosesConnection(t *testing.T) {
	t.Parallel()

	connect := func(ctx context.Context, ch *Channel) error {
		resp, err := ch.SendAndWaitResponse(ctx, &agentv1.Ping{})
		require.Errorf(t, err, "channel is closed")
		assert.Nil(t, resp)

		assert.Nil(t, <-ch.Requests())
		return nil
	}

	stream, cc := setup(t, connect, status.Error(codes.Canceled, context.Canceled.Error()))

	msg, err := stream.Recv()
	require.NoError(t, err)
	assert.NotNil(t, msg)
	require.NoError(t, cc.Close())
}

func TestUnexpectedResponseIdFromAgent(t *testing.T) {
	t.Parallel()

	invalidIDSent := make(chan struct{})
	connect := func(_ context.Context, ch *Channel) error {
		<-invalidIDSent
		select {
		case req := <-ch.Requests():
			t.Fatalf("Request with invalid id should have been ignored: %v", req)
		default:
		}
		// We can read the message with proper id.
		respCh := ch.subscribe(9898)
		ping := &agentv1.Ping{}
		ch.send(&agentv1.ServerMessage{
			Id:      9898,
			Payload: ping.ServerMessageRequestPayload(),
		})
		response := <-respCh
		require.NoError(t, response.Error)
		require.NotNil(t, response.Payload)

		return nil
	}

	stream, _ := setup(t, connect, status.Error(codes.Canceled, context.Canceled.Error()))

	// This request with unexpected id is ignored by the pmm-managed, channel stays open.
	pong := &agentv1.Pong{}
	err := stream.Send(&agentv1.AgentMessage{
		Id:      111,
		Payload: pong.AgentMessageResponsePayload(),
	})
	require.NoError(t, err)
	close(invalidIDSent)

	// This is a request with a proper id.
	pong = &agentv1.Pong{}
	err = stream.Send(&agentv1.AgentMessage{
		Id:      9898,
		Payload: pong.AgentMessageResponsePayload(),
	})
	require.NoError(t, err)

	_, err = stream.Recv()
	require.NoError(t, err)
}

func TestUnexpectedResponsePayloadFromAgent(t *testing.T) {
	t.Parallel()

	stop := make(chan struct{})
	stopServer := make(chan struct{})
	connect := func(_ context.Context, _ *Channel) error {
		<-stopServer
		close(stop)
		return nil
	}
	stream, _ := setup(t, connect, status.Error(codes.Canceled, context.Canceled.Error()))

	err := stream.Send(&agentv1.AgentMessage{
		Id: 4242,
	})
	require.NoError(t, err)

	msg, err := stream.Recv()
	require.NoError(t, err)
	assert.Equal(t, int32(codes.Unimplemented), msg.GetStatus().GetCode())
	close(stopServer)
	<-stop
}
