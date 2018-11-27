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

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm/api/inventory"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
)

// AgentsService works with inventory API Agents.
type AgentsService struct {
	Q *reform.Querier
}

// makeAgent converts database row to Inventory API Agent.
func makeAgent(row *models.AgentRow) inventory.Agent {
	switch row.Type {
	case models.NodeExporterAgentType:
		return &inventory.NodeExporter{}

	case models.MySQLdExporterAgentType:
		return &inventory.MySQLdExporter{}

	default:
		panic(fmt.Errorf("unhandled AgentRow type %s", row.Type))
	}
}

func (as *AgentsService) get(ctx context.Context, id uint32) (*models.AgentRow, error) {
	row := &models.AgentRow{ID: id}
	if err := as.Q.Reload(row); err != nil {
		if err == reform.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "Agent with ID %d not found.", id)
		}
		return nil, errors.WithStack(err)
	}
	return row, nil
}

// List selects all Agents in a stable order.
func (as *AgentsService) List(ctx context.Context) ([]inventory.Agent, error) {
	structs, err := as.Q.SelectAllFrom(models.AgentRowTable, "ORDER BY id")
	if err != nil {
		return nil, errors.WithStack(err)
	}

	res := make([]inventory.Agent, len(structs))
	for i, str := range structs {
		row := str.(*models.AgentRow)
		res[i] = makeAgent(row)
	}
	return res, nil
}

// Get selects a single Agent by ID.
func (as *AgentsService) Get(ctx context.Context, id uint32) (inventory.Agent, error) {
	row, err := as.get(ctx, id)
	if err != nil {
		return nil, err
	}
	return makeAgent(row), nil
}

// Remove deletes Agent by ID.
func (as *AgentsService) Remove(ctx context.Context, id uint32) error {
	// TODO Decide about validation. https://jira.percona.com/browse/PMM-1416
	// ID is not 0.

	err := as.Q.Delete(&models.AgentRow{ID: id})
	if err == reform.ErrNoRows {
		return status.Errorf(codes.NotFound, "Agent with ID %d not found.", id)
	}
	return errors.WithStack(err)
}
