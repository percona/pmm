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
	"github.com/percona/pmm/api/inventory"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
)

// NodesService works with inventory API Nodes.
type NodesService struct {
	q *reform.Querier
	// r *agents.Registry
}

func NewNodesService(q *reform.Querier) *NodesService {
	return &NodesService{
		q: q,
		// r: r,
	}
}

// makeNode converts database row to Inventory API Node.
func makeNode(row *models.NodeRow) inventory.Node {
	switch row.Type {
	case models.PMMServerNodeType: // FIXME remove this branch
		fallthrough

	case models.GenericNodeType:
		return &inventory.GenericNode{
			Id:       row.ID,
			Name:     row.Name,
			Hostname: pointer.GetString(row.Hostname),
		}

	case models.RemoteNodeType:
		return &inventory.RemoteNode{
			Id:   row.ID,
			Name: row.Name,
		}

	case models.AmazonRDSRemoteNodeType:
		return &inventory.AmazonRDSRemoteNode{
			Id:       row.ID,
			Name:     row.Name,
			Hostname: pointer.GetString(row.Hostname),
			Region:   pointer.GetString(row.Region),
		}

	default:
		panic(fmt.Errorf("unhandled NodeRow type %s", row.Type))
	}
}

func (ns *NodesService) get(ctx context.Context, id string) (*models.NodeRow, error) {
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty Node ID.")
	}

	row := &models.NodeRow{ID: id}
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

	row := &models.NodeRow{ID: id}
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

	_, err := ns.q.FindOneFrom(models.NodeRowTable, "name", name)
	switch err {
	case nil:
		return status.Errorf(codes.AlreadyExists, "Node with name %q already exists.", name)
	case reform.ErrNoRows:
		return nil
	default:
		return errors.WithStack(err)
	}
}

func (ns *NodesService) checkUniqueHostnameRegion(ctx context.Context, hostname, region string) error {
	if hostname == "" {
		return status.Error(codes.InvalidArgument, "Empty Node hostname.")
	}
	if region == "" {
		return status.Error(codes.InvalidArgument, "Empty Node region.")
	}

	tail := fmt.Sprintf("WHERE hostname = %s AND region = %s LIMIT 1", ns.q.Placeholder(1), ns.q.Placeholder(2))
	_, err := ns.q.SelectOneFrom(models.NodeRowTable, tail, hostname, region)
	switch err {
	case nil:
		return status.Errorf(codes.AlreadyExists, "Node with hostname %q and region %q already exists.", hostname, region)
	case reform.ErrNoRows:
		return nil
	default:
		return errors.WithStack(err)
	}
}

// List selects all Nodes in a stable order.
func (ns *NodesService) List(ctx context.Context) ([]inventory.Node, error) {
	structs, err := ns.q.SelectAllFrom(models.NodeRowTable, "ORDER BY id")
	if err != nil {
		return nil, errors.WithStack(err)
	}

	res := make([]inventory.Node, len(structs))
	for i, str := range structs {
		row := str.(*models.NodeRow)
		res[i] = makeNode(row)
	}
	return res, nil
}

// Get selects a single Node by ID.
func (ns *NodesService) Get(ctx context.Context, id string) (inventory.Node, error) {
	row, err := ns.get(ctx, id)
	if err != nil {
		return nil, err
	}
	return makeNode(row), nil
}

// Add inserts Node with given parameters. ID will be generated if it is empty.
func (ns *NodesService) Add(ctx context.Context, id string, nodeType models.NodeType, name string, hostname, region *string) (inventory.Node, error) {
	// TODO Decide about validation. https://jira.percona.com/browse/PMM-1416
	// No hostname for Container, etc.

	if id == "" {
		id = makeID()
	}
	if err := ns.checkUniqueID(ctx, id); err != nil {
		return nil, err
	}

	if err := ns.checkUniqueName(ctx, name); err != nil {
		return nil, err
	}
	if hostname != nil && region != nil {
		if err := ns.checkUniqueHostnameRegion(ctx, *hostname, *region); err != nil {
			return nil, err
		}
	}

	row := &models.NodeRow{
		ID:       id,
		Type:     nodeType,
		Name:     name,
		Hostname: hostname,
		Region:   region,
	}
	if err := ns.q.Insert(row); err != nil {
		return nil, errors.WithStack(err)
	}
	return makeNode(row), nil
}

// Change updates Node by ID.
func (ns *NodesService) Change(ctx context.Context, id string, name string) (inventory.Node, error) {
	// TODO Decide about validation. https://jira.percona.com/browse/PMM-1416
	// ID is not 0, name is not empty and valid.

	if err := ns.checkUniqueName(ctx, name); err != nil {
		return nil, err
	}

	row, err := ns.get(ctx, id)
	if err != nil {
		return nil, err
	}

	row.Name = name
	if err = ns.q.Update(row); err != nil {
		return nil, errors.WithStack(err)
	}
	return makeNode(row), nil
}

// Remove deletes Node by ID.
func (ns *NodesService) Remove(ctx context.Context, id string) error {
	// TODO Decide about validation. https://jira.percona.com/browse/PMM-1416
	// ID is not 0.

	// TODO check absence of Services and Agents

	err := ns.q.Delete(&models.NodeRow{ID: id})
	if err == reform.ErrNoRows {
		return status.Errorf(codes.NotFound, "Node with ID %q not found.", id)
	}
	return errors.WithStack(err)
}
