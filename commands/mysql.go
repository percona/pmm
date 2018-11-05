// pmm-admin
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

package commands

import (
	"context"

	"github.com/percona/pmm/api/json/client"
	"github.com/percona/pmm/api/json/client/agents"
	"github.com/percona/pmm/api/json/models"
	"github.com/sirupsen/logrus"
)

// AddMySQLCmd implements `pmm-admin add mysql` command.
type AddMySQLCmd struct {
	Username string
	Password string
}

// Run implements Command interface.
func (cmd *AddMySQLCmd) Run() {
	// TODO get NodeID from local pmm-agent

	// TODO get or create MySQL service for this Node via pmm-managed

	params := &agents.AddMySqldExporterAgentParams{
		Body: &models.InventoryAddMySqldExporterAgentRequest{
			// TODO RunsOnNodeID
			// TODO ServiceID
			Username: cmd.Username,
			Password: cmd.Password,
		},

		// FIXME remove this from every request
		Context: context.Background(),
	}
	resp, err := client.Default.Agents.AddMySqldExporterAgent(params)
	logrus.Info(resp)
	logrus.Error(err)
}

// check interfaces
var (
	_ Command = (*AddMySQLCmd)(nil)
)
