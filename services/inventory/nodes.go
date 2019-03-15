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

package inventory

import (
	"context"
	"fmt"

	"github.com/AlekSi/pointer"
	"github.com/google/uuid"
	inventorypb "github.com/percona/pmm/api/inventory"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
)

// NodesService works with inventory API Nodes.
type NodesService struct {
	q *reform.Querier
	r registry
}

func NewNodesService(q *reform.Querier, r registry) *NodesService {
	return &NodesService{
		q: q,
		r: r,
	}
}

// makeNode converts database row to Inventory API Node.
func makeNode(row *models.Node) (inventorypb.Node, error) {
	labels, err := row.GetCustomLabels()
	if err != nil {
		return nil, err
	}

	switch row.NodeType {
	case models.GenericNodeType:
		return &inventorypb.GenericNode{
			NodeId:        row.NodeID,
			NodeName:      row.NodeName,
			MachineId:     pointer.GetString(row.MachineID),
			Distro:        pointer.GetString(row.Distro),
			DistroVersion: pointer.GetString(row.DistroVersion),
			CustomLabels:  labels,
			Address:       pointer.GetString(row.Address),
		}, nil

	case models.ContainerNodeType:
		return &inventorypb.ContainerNode{
			NodeId:              row.NodeID,
			NodeName:            row.NodeName,
			MachineId:           pointer.GetString(row.MachineID),
			DockerContainerId:   pointer.GetString(row.DockerContainerID),
			DockerContainerName: pointer.GetString(row.DockerContainerName),
			CustomLabels:        labels,
		}, nil

	case models.RemoteNodeType:
		return &inventorypb.RemoteNode{
			NodeId:       row.NodeID,
			NodeName:     row.NodeName,
			CustomLabels: labels,
		}, nil

	case models.RemoteAmazonRDSNodeType:
		return &inventorypb.RemoteAmazonRDSNode{
			NodeId:       row.NodeID,
			NodeName:     row.NodeName,
			Instance:     pointer.GetString(row.Address),
			Region:       pointer.GetString(row.Region),
			CustomLabels: labels,
		}, nil

	default:
		panic(fmt.Errorf("unhandled Node type %s", row.NodeType))
	}
}

func (ns *NodesService) get(ctx context.Context, id string) (*models.Node, error) {
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty Node ID.")
	}

	row := &models.Node{NodeID: id}
	switch err := ns.q.Reload(row); err {
	case nil:
		return row, nil
	case reform.ErrNoRows:
		return nil, status.Errorf(codes.NotFound, "Node with ID %q not found.", id)
	default:
		return nil, errors.WithStack(err)
	}
}

func (ns *NodesService) checkUniqueID(ctx context.Context, id string) error {
	if id == "" {
		panic("empty Node ID")
	}

	row := &models.Node{NodeID: id}
	switch err := ns.q.Reload(row); err {
	case nil:
		return status.Errorf(codes.AlreadyExists, "Node with ID %q already exists.", id)
	case reform.ErrNoRows:
		return nil
	default:
		return errors.WithStack(err)
	}
}

func (ns *NodesService) checkUniqueName(ctx context.Context, name string) error {
	if name == "" {
		return status.Error(codes.InvalidArgument, "Empty Node name.")
	}

	_, err := ns.q.FindOneFrom(models.NodeTable, "node_name", name)
	switch err {
	case nil:
		return status.Errorf(codes.AlreadyExists, "Node with name %q already exists.", name)
	case reform.ErrNoRows:
		return nil
	default:
		return errors.WithStack(err)
	}
}

func (ns *NodesService) checkUniqueInstanceRegion(ctx context.Context, instance, region string) error {
	if instance == "" {
		return status.Error(codes.InvalidArgument, "Empty Node instance.")
	}
	if region == "" {
		return status.Error(codes.InvalidArgument, "Empty Node region.")
	}

	tail := fmt.Sprintf("WHERE address = %s AND region = %s LIMIT 1", ns.q.Placeholder(1), ns.q.Placeholder(2))
	_, err := ns.q.SelectOneFrom(models.NodeTable, tail, instance, region)
	switch err {
	case nil:
		return status.Errorf(codes.AlreadyExists, "Node with instance %q and region %q already exists.", instance, region)
	case reform.ErrNoRows:
		return nil
	default:
		return errors.WithStack(err)
	}
}

// List selects all Nodes in a stable order.
func (ns *NodesService) List(ctx context.Context) ([]inventorypb.Node, error) {
	structs, err := ns.q.SelectAllFrom(models.NodeTable, "ORDER BY node_id")
	if err != nil {
		return nil, errors.WithStack(err)
	}

	res := make([]inventorypb.Node, len(structs))
	for i, str := range structs {
		row := str.(*models.Node)
		res[i], err = makeNode(row)
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

// Get selects a single Node by ID.
func (ns *NodesService) Get(ctx context.Context, id string) (inventorypb.Node, error) {
	row, err := ns.get(ctx, id)
	if err != nil {
		return nil, err
	}
	return makeNode(row)
}

// Add inserts Node with given parameters. ID will be generated.
func (ns *NodesService) Add(ctx context.Context, nodeType models.NodeType, name string, address, region *string) (inventorypb.Node, error) {
	// TODO Decide about validation. https://jira.percona.com/browse/PMM-1416
	// No hostname for Container, etc.

	id := "/node_id/" + uuid.New().String()
	if err := ns.checkUniqueID(ctx, id); err != nil {
		return nil, err
	}

	if err := ns.checkUniqueName(ctx, name); err != nil {
		return nil, err
	}
	if address != nil && region != nil {
		if err := ns.checkUniqueInstanceRegion(ctx, *address, *region); err != nil {
			return nil, err
		}
	}

	row := &models.Node{
		NodeID:   id,
		NodeType: nodeType,
		NodeName: name,
		Address:  address,
		Region:   region,
	}
	if err := ns.q.Insert(row); err != nil {
		return nil, errors.WithStack(err)
	}
	return makeNode(row)
}

// Change updates Node by ID.
func (ns *NodesService) Change(ctx context.Context, id string, name string) (inventorypb.Node, error) {
	// TODO Decide about validation. https://jira.percona.com/browse/PMM-1416
	// ID is not 0, name is not empty and valid.

	if err := ns.checkUniqueName(ctx, name); err != nil {
		return nil, err
	}

	row, err := ns.get(ctx, id)
	if err != nil {
		return nil, err
	}

	row.NodeName = name
	if err = ns.q.Update(row); err != nil {
		return nil, errors.WithStack(err)
	}
	return makeNode(row)
}

// Remove deletes Node by ID.
func (ns *NodesService) Remove(ctx context.Context, id string) error {
	// TODO Decide about validation. https://jira.percona.com/browse/PMM-1416
	// ID is not 0.

	// TODO check absence of Services and Agents

	err := ns.q.Delete(&models.Node{NodeID: id})
	if err == reform.ErrNoRows {
		return status.Errorf(codes.NotFound, "Node with ID %q not found.", id)
	}
	return errors.WithStack(err)
}
