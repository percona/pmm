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

package ha

import (
	"context"

	"github.com/hashicorp/memberlist"

	hav1beta1 "github.com/percona/pmm/api/ha/v1beta1"
)

// HAServer implements the HAService gRPC API.
type HAServer struct { //nolint:revive
	hav1beta1.UnimplementedHAServiceServer

	service *Service
}

// NewHAServer creates a new HAServer instance.
func NewHAServer(service *Service) *HAServer {
	return &HAServer{
		service: service,
	}
}

// Status returns the current HA mode status.
func (s *HAServer) Status(_ context.Context, _ *hav1beta1.StatusRequest) (*hav1beta1.StatusResponse, error) { //nolint:unparam
	status := "Disabled"
	if s.service.params.Enabled {
		status = "Enabled"
	}
	return &hav1beta1.StatusResponse{Status: status}, nil
}

// ListNodes returns a list of all nodes in the High Availability cluster.
func (s *HAServer) ListNodes(_ context.Context, _ *hav1beta1.ListNodesRequest) (*hav1beta1.ListNodesResponse, error) { //nolint:unparam
	if !s.service.params.Enabled {
		return &hav1beta1.ListNodesResponse{Nodes: []*hav1beta1.HANode{}}, nil
	}

	s.service.rw.RLock()
	memberlist := s.service.memberlist
	raftNode := s.service.raftNode
	s.service.rw.RUnlock()

	if memberlist == nil {
		return &hav1beta1.ListNodesResponse{Nodes: []*hav1beta1.HANode{}}, nil
	}

	_, leaderID := raftNode.LeaderWithID()
	members := memberlist.Members()
	nodes := []*hav1beta1.HANode{}

	for _, member := range members {
		role := hav1beta1.NodeRole_NODE_ROLE_FOLLOWER
		if member.Name == string(leaderID) {
			role = hav1beta1.NodeRole_NODE_ROLE_LEADER
		}

		status := memberlistStateToString(member.State)

		nodes = append(nodes, &hav1beta1.HANode{
			NodeName: member.Name,
			Role:     role,
			Status:   status,
		})
	}

	return &hav1beta1.ListNodesResponse{Nodes: nodes}, nil
}

// memberlistStateToString converts memberlist state to a string representation.
func memberlistStateToString(state memberlist.NodeStateType) string {
	switch state {
	case memberlist.StateAlive:
		return "alive"
	case memberlist.StateSuspect:
		return "suspect"
	case memberlist.StateDead:
		return "dead"
	case memberlist.StateLeft:
		return "left"
	default:
		return "unknown"
	}
}
