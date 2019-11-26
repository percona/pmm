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

package models

import (
	"fmt"
	"strings"

	"github.com/AlekSi/pointer"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
)

func checkUniqueNodeID(q *reform.Querier, id string) error {
	if id == "" {
		panic("empty Node ID")
	}

	node := &Node{NodeID: id}
	switch err := q.Reload(node); err {
	case nil:
		return status.Errorf(codes.AlreadyExists, "Node with ID %q already exists.", id)
	case reform.ErrNoRows:
		return nil
	default:
		return errors.WithStack(err)
	}
}

func checkUniqueNodeName(q *reform.Querier, name string) error {
	if name == "" {
		return status.Error(codes.InvalidArgument, "Empty Node name.")
	}

	_, err := q.FindOneFrom(NodeTable, "node_name", name)
	switch err {
	case nil:
		return status.Errorf(codes.AlreadyExists, "Node with name %q already exists.", name)
	case reform.ErrNoRows:
		return nil
	default:
		return errors.WithStack(err)
	}
}

func checkUniqueNodeInstanceRegion(q *reform.Querier, instance, region string) error {
	if instance == "" {
		return status.Error(codes.InvalidArgument, "Empty Node instance.")
	}
	if region == "" {
		return status.Error(codes.InvalidArgument, "Empty Node region.")
	}

	tail := fmt.Sprintf("WHERE address = %s AND region = %s LIMIT 1", q.Placeholder(1), q.Placeholder(2)) //nolint:gosec
	_, err := q.SelectOneFrom(NodeTable, tail, instance, region)
	switch err {
	case nil:
		return status.Errorf(codes.AlreadyExists, "Node with instance %q and region %q already exists.", instance, region)
	case reform.ErrNoRows:
		return nil
	default:
		return errors.WithStack(err)
	}
}

// FindAllNodes returns all Nodes.
func FindAllNodes(q *reform.Querier) ([]*Node, error) {
	structs, err := q.SelectAllFrom(NodeTable, "ORDER BY node_id")
	if err != nil {
		return nil, errors.WithStack(err)
	}

	nodes := make([]*Node, len(structs))
	for i, s := range structs {
		nodes[i] = s.(*Node)
	}

	return nodes, nil
}

// FindNodeByID finds a Node by ID.
func FindNodeByID(q *reform.Querier, id string) (*Node, error) {
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty Node ID.")
	}

	node := &Node{NodeID: id}
	switch err := q.Reload(node); err {
	case nil:
		return node, nil
	case reform.ErrNoRows:
		return nil, status.Errorf(codes.NotFound, "Node with ID %q not found.", id)
	default:
		return nil, errors.WithStack(err)
	}
}

// FindNodeByName finds a Node by name.
func FindNodeByName(q *reform.Querier, name string) (*Node, error) {
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty Node name.")
	}

	node := new(Node)
	switch err := q.FindOneTo(node, "node_name", name); err {
	case nil:
		return node, nil
	case reform.ErrNoRows:
		return nil, status.Errorf(codes.NotFound, "Node with name %q not found.", name)
	default:
		return nil, errors.WithStack(err)
	}
}

// CreateNodeParams contains parameters for creating Nodes.
type CreateNodeParams struct {
	NodeName      string
	MachineID     *string
	Distro        string
	NodeModel     string
	AZ            string
	ContainerID   *string
	ContainerName *string
	CustomLabels  map[string]string
	Address       string
	Region        *string
}

// createNodeWithID creates a Node with given ID.
func createNodeWithID(q *reform.Querier, id string, nodeType NodeType, params *CreateNodeParams) (*Node, error) {
	if err := checkUniqueNodeID(q, id); err != nil {
		return nil, err
	}

	if err := checkUniqueNodeName(q, params.NodeName); err != nil {
		return nil, err
	}

	// do not check that machine-id is unique: https://jira.percona.com/browse/PMM-4196

	if nodeType == RemoteRDSNodeType {
		if strings.Contains(params.Address, ".") {
			return nil, status.Error(codes.InvalidArgument, "DB instance identifier should not contain dot.")
		}
	}

	if params.Region != nil {
		if err := checkUniqueNodeInstanceRegion(q, params.Address, *params.Region); err != nil {
			return nil, err
		}
	}

	// Trim trailing \n received from broken 2.0.0 clients.
	// See https://jira.percona.com/browse/PMM-4720
	machineID := pointer.ToStringOrNil(strings.TrimSpace(pointer.GetString(params.MachineID)))

	node := &Node{
		NodeID:        id,
		NodeType:      nodeType,
		NodeName:      params.NodeName,
		MachineID:     machineID,
		Distro:        params.Distro,
		NodeModel:     params.NodeModel,
		AZ:            params.AZ,
		ContainerID:   params.ContainerID,
		ContainerName: params.ContainerName,
		Address:       params.Address,
		Region:        params.Region,
	}
	if err := node.SetCustomLabels(params.CustomLabels); err != nil {
		return nil, err
	}
	if err := q.Insert(node); err != nil {
		return nil, errors.WithStack(err)
	}

	return node, nil
}

// CreateNode creates a Node.
func CreateNode(q *reform.Querier, nodeType NodeType, params *CreateNodeParams) (*Node, error) {
	id := "/node_id/" + uuid.New().String()
	return createNodeWithID(q, id, nodeType, params)
}

// RemoveNode removes single Node.
func RemoveNode(q *reform.Querier, id string, mode RemoveMode) error {
	n, err := FindNodeByID(q, id)
	if err != nil {
		return err
	}

	// check/remove Agents
	structs, err := q.FindAllFrom(AgentTable, "node_id", id)
	if err != nil {
		return errors.Wrap(err, "failed to select Agent IDs")
	}
	if len(structs) != 0 {
		switch mode {
		case RemoveRestrict:
			return status.Errorf(codes.FailedPrecondition, "Node with ID %q has agents.", id)
		case RemoveCascade:
			for _, str := range structs {
				agentID := str.(*Agent).AgentID
				if _, err = RemoveAgent(q, agentID, RemoveCascade); err != nil {
					return err
				}
			}
		default:
			panic(fmt.Errorf("unhandled RemoveMode %v", mode))
		}
	}

	// check/remove pmm-agents
	structs, err = q.FindAllFrom(AgentTable, "runs_on_node_id", id)
	if err != nil {
		return errors.Wrap(err, "failed to select Agents")
	}
	if len(structs) != 0 {
		switch mode {
		case RemoveRestrict:
			return status.Errorf(codes.FailedPrecondition, "Node with ID %q has pmm-agent.", id)
		case RemoveCascade:
			for _, str := range structs {
				agentID := str.(*Agent).AgentID
				if _, err = RemoveAgent(q, agentID, RemoveCascade); err != nil {
					return err
				}
			}
		default:
			panic(fmt.Errorf("unhandled RemoveMode %v", mode))
		}
	}

	// check/remove Services
	structs, err = q.FindAllFrom(ServiceTable, "node_id", id)
	if err != nil {
		return errors.Wrap(err, "failed to select Service IDs")
	}
	if len(structs) != 0 {
		switch mode {
		case RemoveRestrict:
			return status.Errorf(codes.FailedPrecondition, "Node with ID %q has services.", id)
		case RemoveCascade:
			for _, str := range structs {
				serviceID := str.(*Service).ServiceID
				if err = RemoveService(q, serviceID, RemoveCascade); err != nil {
					return err
				}
			}
		default:
			panic(fmt.Errorf("unhandled RemoveMode %v", mode))
		}
	}

	return errors.Wrap(q.Delete(n), "failed to delete Node")
}
