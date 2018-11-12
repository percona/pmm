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

type NodesService struct {
	DB *reform.DB
}

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
			Id:     row.ID,
			Name:   row.Name,
			Region: row.Region,
		}
	default:
		panic(fmt.Errorf("unhandled NodeRow type %s", row.Type))
	}
}

func (ns *NodesService) List(ctx context.Context) (*inventory.ListNodesResponse, error) {
	structs, err := ns.DB.SelectAllFrom(models.NodeRowTable, "ORDER BY id")
	if err != nil {
		return nil, errors.WithStack(err)
	}

	res := new(inventory.ListNodesResponse)
	for _, str := range structs {
		row := str.(*models.NodeRow)
		node := makeNode(row)
		switch node := node.(type) {
		case *inventory.BareMetalNode:
			res.BareMetal = append(res.BareMetal, node)
		case *inventory.VirtualMachineNode:
			res.VirtualMachine = append(res.VirtualMachine, node)
		case *inventory.ContainerNode:
			res.Container = append(res.Container, node)
		case *inventory.RemoteNode:
			res.Remote = append(res.Remote, node)
		case *inventory.RDSNode:
			res.Rds = append(res.Rds, node)
		default:
			panic(fmt.Errorf("unhandled inventory Node type %T", node))
		}
	}
	return res, nil
}

func (ns *NodesService) Get(ctx context.Context, id uint32) (inventory.Node, error) {
	record, err := ns.DB.FindByPrimaryKeyFrom(models.NodeRowTable, id)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	row := record.(*models.NodeRow)
	return makeNode(row), nil
}
