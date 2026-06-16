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
	connectFunc func(server agentv1.AgentService_ConnectServer) error

	agentv1.UnimplementedAgentServiceServer
}

func (s *testServer) Connect(stream agentv1.AgentService_ConnectServer) error {
	return s.connectFunc(stream)
}

var _ agentv1.AgentServiceServer = (*testServer)(nil)

func setup(t *testing.T, connect func(*Channel) error, expected error) (agentv1.AgentService_ConnectClient, *grpc.ClientConn) {
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
			return connect(channel)
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
	connect := func(ch *Channel) error {
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

	connect := func(ch *Channel) error {
		for i := uint32(1); i <= count; i++ {
			resp, err := ch.SendAndWaitResponse(&agentv1.Ping{})
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

func TestServerExitsWithGRPCError(t *testing.T) {
	t.Parallel()

	errUnimplemented := status.Error(codes.Unimplemented, "Test error")
	connect := func(ch *Channel) error {
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

	connect := func(ch *Channel) error {
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

	connect := func(ch *Channel) error {
		resp, err := ch.SendAndWaitResponse(&agentv1.Ping{})
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

	connect := func(ch *Channel) error {
		resp, err := ch.SendAndWaitResponse(&agentv1.Ping{})
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
	connect := func(ch *Channel) error {
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
	connect := func(_ *Channel) error {
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
