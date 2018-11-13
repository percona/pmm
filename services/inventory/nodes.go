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
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
)

// TODO Better errors.

// NodesService works with inventory API Nodes.
type NodesService struct {
	DB *reform.DB
}

// makeNode converts database row to Inventory API Node.
func makeNode(row *models.NodeRow) inventory.Node {
	switch row.Type {
	case models.BareMetalNodeType:
		return &inventory.BareMetalNode{
			Id:       row.ID,
			Name:     row.Name,
			Hostname: pointer.GetString(row.Hostname),
		}

	case models.VirtualMachineNodeType:
		return &inventory.VirtualMachineNode{
			Id:       row.ID,
			Name:     row.Name,
			Hostname: pointer.GetString(row.Hostname),
		}

	case models.ContainerNodeType:
		return &inventory.ContainerNode{
			Id:   row.ID,
			Name: row.Name,
		}

	case models.RemoteNodeType:
		return &inventory.RemoteNode{
			Id:   row.ID,
			Name: row.Name,
		}

	case models.RDSNodeType:
		return &inventory.RDSNode{
			Id:       row.ID,
			Name:     row.Name,
			Hostname: pointer.GetString(row.Hostname),
			Region:   pointer.GetString(row.Region),
		}

	default:
		panic(fmt.Errorf("unhandled NodeRow type %s", row.Type))
	}
}

// List selects all Nodes in a stable order.
func (ns *NodesService) List(ctx context.Context) ([]inventory.Node, error) {
	structs, err := ns.DB.SelectAllFrom(models.NodeRowTable, "ORDER BY id")
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
func (ns *NodesService) Get(ctx context.Context, id uint32) (inventory.Node, error) {
	row := &models.NodeRow{ID: id}
	if err := ns.DB.Reload(row); err != nil {
		return nil, errors.WithStack(err)
	}

	return makeNode(row), nil
}

// Add inserts Node with given parameters.
func (ns *NodesService) Add(ctx context.Context, nodeType models.NodeType, name string, hostname, region *string) (inventory.Node, error) {
	// TODO Decide about validation. https://jira.percona.com/browse/PMM-1416

	row := &models.NodeRow{
		Type:     nodeType,
		Name:     name,
		Hostname: hostname,
		Region:   region,
	}
	if err := ns.DB.Insert(row); err != nil {
		return nil, errors.WithStack(err)
	}

	return makeNode(row), nil
}

// Change updates Node by ID.
func (ns *NodesService) Change(ctx context.Context, id uint32, name string) error {
	row := &models.NodeRow{ID: id}
	if err := ns.DB.Reload(row); err != nil {
		return errors.WithStack(err)
	}

	row.Name = name
	if err := ns.DB.Update(row); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// Remove deletes Node by ID.
func (ns *NodesService) Remove(ctx context.Context, id uint32) error {
	// TODO Decide about validation. https://jira.percona.com/browse/PMM-1416

	return ns.DB.Delete(&models.NodeRow{ID: id})
}
