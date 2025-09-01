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
	err := q.Reload(node)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil
		}
		return errors.WithStack(err)
	}

	return status.Errorf(codes.AlreadyExists, "Node with ID %q already exists.", id)
}

func checkUniqueNodeName(q *reform.Querier, name string) error {
	if name == "" {
		return status.Error(codes.InvalidArgument, "Empty Node name.")
	}

	_, err := q.FindOneFrom(NodeTable, "node_name", name)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil
		}
		return errors.WithStack(err)
	}

	return status.Errorf(codes.AlreadyExists, "Node with name %q already exists.", name)
}

// CheckUniqueNodeAddressRegion checks for uniqueness of instance address and region.
// This function not only returns an error in case it finds an existing node with the same address and region, but
// also returns the Node itself if there is any, because if we are recreating the instance (--force in pmm-admin)
// we need to know the Node.ID to remove it along with its dependencies.
// This check is performed only if the region is not empty.
func CheckUniqueNodeAddressRegion(q *reform.Querier, address string, region *string) (*Node, error) {
	if pointer.GetString(region) == "" {
		return nil, nil //nolint:nilnil
	}

	if address == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty Node address.")
	}

	var node Node
	err := q.SelectOneTo(&node, "WHERE address = $1 AND region = $2 LIMIT 1", address, region)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, nil //nolint:nilnil
		}
		return nil, errors.WithStack(err)
	}

	return &node, status.Errorf(codes.AlreadyExists, "Node with address %q and region %q already exists.", address, *region)
}

// NodeFilters represents filters for nodes list.
type NodeFilters struct {
	// Return Nodes with provided type.
	NodeType *NodeType
}

// FindNodes returns Nodes by filters.
func FindNodes(q *reform.Querier, filters NodeFilters) ([]*Node, error) {
	var whereClause string
	var args []any
	if filters.NodeType != nil {
		whereClause = "WHERE node_type = $1"
		args = append(args, *filters.NodeType)
	}
	structs, err := q.SelectAllFrom(NodeTable, fmt.Sprintf("%s ORDER BY node_id", whereClause), args...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	nodes := make([]*Node, len(structs))
	for i, s := range structs {
		nodes[i] = s.(*Node) //nolint:forcetypeassert
	}

	return nodes, nil
}

// FindNodeByID finds a Node by ID.
func FindNodeByID(q *reform.Querier, id string) (*Node, error) {
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty Node ID.")
	}

	node := &Node{NodeID: id}
	err := q.Reload(node)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "Node with ID %q not found.", id)
		}
		return nil, errors.WithStack(err)
	}
	return node, nil
}

// FindNodesByIDs finds Nodes by IDs.
func FindNodesByIDs(q *reform.Querier, ids []string) ([]*Node, error) {
	if len(ids) == 0 {
		return []*Node{}, nil
	}

	p := strings.Join(q.Placeholders(1, len(ids)), ", ")
	tail := fmt.Sprintf("WHERE node_id IN (%s) ORDER BY node_id", p)
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	structs, err := q.SelectAllFrom(NodeTable, tail, args...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	res := make([]*Node, len(structs))
	for i, s := range structs {
		res[i] = s.(*Node) //nolint:forcetypeassert
	}
	return res, nil
}

// FindNodeByName finds a Node by name.
func FindNodeByName(q *reform.Querier, name string) (*Node, error) {
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty Node name.")
	}

	var node Node
	err := q.FindOneTo(&node, "node_name", name)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "Node with name %q not found.", name)
		}
		return nil, errors.WithStack(err)
	}

	return &node, nil
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
	InstanceID    string
	Region        *string
	Password      *string
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
		if strings.Contains(params.InstanceID, ".") {
			return nil, status.Error(codes.InvalidArgument, "DB instance identifier should not contain dots.")
		}
	}

	if _, err := CheckUniqueNodeAddressRegion(q, params.Address, params.Region); err != nil {
		return nil, err
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
		InstanceID:    params.InstanceID,
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
	id := uuid.New().String()
	return createNodeWithID(q, id, nodeType, params)
}

// RemoveNode removes single Node.
func RemoveNode(q *reform.Querier, id string, mode RemoveMode) error {
	n, err := FindNodeByID(q, id)
	if err != nil {
		return err
	}

	if id == PMMServerNodeID {
		return status.Error(codes.PermissionDenied, "PMM Server node can't be removed.")
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
				agentID := str.(*Agent).AgentID //nolint:forcetypeassert
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
				agentID := str.(*Agent).AgentID //nolint:forcetypeassert
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
				serviceID := str.(*Service).ServiceID //nolint:forcetypeassert
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
