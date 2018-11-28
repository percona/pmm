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
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/percona/pmm/api/agent"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type testServer struct {
	connect func(agent.Agent_ConnectServer) error
}

func (s *testServer) Register(context.Context, *agent.RegisterRequest) (*agent.RegisterResponse, error) {
	panic("should not be called")
}

func (s *testServer) Connect(stream agent.Agent_ConnectServer) error {
	return s.connect(stream)
}

var _ agent.AgentServer = (*testServer)(nil)

func setup(t *testing.T, connect func(agent.Agent_ConnectServer) error, expected ...error) (*Channel, *grpc.ClientConn, func(*testing.T)) {
	t.Parallel()

	// start server with given connect handler
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer()
	agent.RegisterAgentServer(server, &testServer{
		connect: connect,
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
	stream, err := agent.NewAgentClient(cc).Connect(ctx)
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

	connect := func(stream agent.Agent_ConnectServer) error { //nolint:unparam
		for i := uint32(1); i <= count; i++ {
			msg, err := stream.Recv()
			require.NoError(t, err)
			assert.Equal(t, i, msg.Id)
			require.NotNil(t, msg.GetQanData())

			err = stream.Send(&agent.ServerMessage{
				Id: i,
				Payload: &agent.ServerMessage_QanData{
					QanData: new(agent.QANDataResponse),
				},
			})
			assert.NoError(t, err)
		}

		return nil
	}

	channel, _, teardown := setup(t, connect, io.EOF)
	defer teardown(t)

	for i := uint32(1); i <= count; i++ {
		resp := channel.SendRequest(&agent.AgentMessage_QanData{
			QanData: new(agent.QANDataRequest),
		})
		assert.NotNil(t, resp)
	}
}

func TestServerRequest(t *testing.T) {
	const count = 50
	require.True(t, count > serverRequestsCap)

	connect := func(stream agent.Agent_ConnectServer) error { //nolint:unparam
		for i := uint32(1); i <= count; i++ {
			err := stream.Send(&agent.ServerMessage{
				Id: i,
				Payload: &agent.ServerMessage_Ping{
					Ping: new(agent.PingRequest),
				},
			})
			assert.NoError(t, err)
		}

		for i := uint32(1); i <= count; i++ {
			msg, err := stream.Recv()
			require.NoError(t, err)
			assert.Equal(t, i, msg.Id)
			pingResponse := msg.GetPing()
			require.NotNil(t, pingResponse)
			ts, err := ptypes.Timestamp(pingResponse.CurrentTime)
			assert.NoError(t, err)
			assert.InDelta(t, time.Now().Unix(), ts.Unix(), 1)
		}

		return nil
	}

	channel, _, teardown := setup(t, connect, io.EOF)
	defer teardown(t)

	for req := range channel.Requests() {
		ping := req.GetPing()
		require.NotNil(t, ping)

		channel.SendResponse(&agent.AgentMessage{
			Id: req.Id,
			Payload: &agent.AgentMessage_Ping{
				Ping: &agent.PingResponse{
					CurrentTime: ptypes.TimestampNow(),
				},
			},
		})
	}
}

func TestServerClosesStream(t *testing.T) {
	connect := func(stream agent.Agent_ConnectServer) error {
		msg, err := stream.Recv()
		require.NoError(t, err)
		assert.EqualValues(t, 1, msg.Id)
		require.NotNil(t, msg.GetQanData())

		return status.Error(codes.Unimplemented, "Test error")
	}

	channel, _, teardown := setup(t, connect, status.Error(codes.Unimplemented, "Test error"))
	defer teardown(t)

	resp := channel.SendRequest(&agent.AgentMessage_QanData{
		QanData: new(agent.QANDataRequest),
	})
	assert.Nil(t, resp)
}

func TestAgentClosesConnection(t *testing.T) {
	connect := func(stream agent.Agent_ConnectServer) error { // nolint:unparam
		err := stream.Send(&agent.ServerMessage{
			Id: 1,
			Payload: &agent.ServerMessage_Ping{
				Ping: new(agent.PingRequest),
			},
		})
		assert.NoError(t, err)

		msg, err := stream.Recv()
		assert.Equal(t, status.Error(codes.Canceled, "context canceled"), err)
		assert.Nil(t, msg)

		return nil
	}

	errClientConnClosing := status.Error(codes.Canceled, "grpc: the client connection is closing") // == grpc.ErrClientConnClosing
	errConnClosing := status.Error(codes.Unavailable, "transport is closing")
	channel, cc, teardown := setup(t, connect, errClientConnClosing, errConnClosing)
	defer teardown(t)

	req := <-channel.Requests()
	ping := req.GetPing()
	assert.NotNil(t, ping)

	assert.NoError(t, cc.Close())
}

func TestUnexpectedMessageFromServer(t *testing.T) {
	connect := func(stream agent.Agent_ConnectServer) error { // nolint:unparam
		// this message triggers "no subscriber for ID" error
		err := stream.Send(&agent.ServerMessage{
			Id: 111,
			Payload: &agent.ServerMessage_QanData{
				QanData: new(agent.QANDataResponse),
			},
		})
		assert.NoError(t, err)

		// this message should not trigger new error
		err = stream.Send(&agent.ServerMessage{
			Id: 222,
			Payload: &agent.ServerMessage_QanData{
				QanData: new(agent.QANDataResponse),
			},
		})
		assert.NoError(t, err)

		return nil
	}

	channel, _, teardown := setup(t, connect, fmt.Errorf("no subscriber for ID 111"), io.EOF)
	defer teardown(t)

	// after receiving unexpected response, channel is closed
	resp := channel.SendRequest(&agent.AgentMessage_QanData{
		QanData: new(agent.QANDataRequest),
	})
	assert.Nil(t, resp)
	msg := <-channel.Requests()
	assert.Nil(t, msg)

	// future requests are ignored
	resp = channel.SendRequest(&agent.AgentMessage_QanData{
		QanData: new(agent.QANDataRequest),
	})
	assert.Nil(t, resp)
	msg = <-channel.Requests()
	assert.Nil(t, msg)
}
