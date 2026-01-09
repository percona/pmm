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

// Package grpc contains functionality for agent <-> server connection.
package grpc

import (
	agentv1 "github.com/percona/pmm/api/agent/v1"
	"github.com/percona/pmm/managed/services/agents"
)

// mongoDBRealtimeServer implements agentv1.MongoDBRealtimeServer for streaming MongoDB real-time queries.
type mongoDBRealtimeServer struct {
	handler *agents.Handler

	agentv1.UnimplementedMongoDBRealtimeServer
}

// NewMongoDBRealtimeServer creates a new MongoDBRealtime gRPC server.
func NewMongoDBRealtimeServer(handler *agents.Handler) agentv1.MongoDBRealtimeServer {
	return &mongoDBRealtimeServer{handler: handler}
}

// StreamRealtimeQueries handles bidirectional streaming of MongoDB real-time query data.
func (s *mongoDBRealtimeServer) StreamRealtimeQueries(stream agentv1.MongoDBRealtime_StreamRealtimeQueriesServer) error {
	for {
		_, err := stream.Recv()
		if err != nil {
			return err // EOF or stream error
		}
		resp := &agentv1.MongoDBRealtimeResponse{}
		if err := stream.Send(resp); err != nil {
			return err
		}
	}
}

// check interface
var _ agentv1.MongoDBRealtimeServer = (*mongoDBRealtimeServer)(nil)
