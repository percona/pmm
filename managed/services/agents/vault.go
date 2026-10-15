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

package agents

import (
	"context"
	"maps"
	"slices"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/utils/logger"
)

// StoreSecret asks the pmm-agent to store data in the Vault secret at path, keeping the keys of the
// secret that data does not have. The pmm-agent writes it with its own Vault identity, the one it reads
// the credentials of its Agents with, so PMM Server needs no access to Vault.
func (r *Registry) StoreSecret(ctx context.Context, q *reform.Querier, pmmAgentID, path string, data map[string]string) error {
	pmmAgent, err := models.FindAgentByID(q, pmmAgentID)
	if err != nil {
		return err
	}
	err = models.IsAgentSupported(pmmAgent, models.VaultFunctionality, models.PMMAgentMinVersionForVault)
	if err != nil {
		return status.Error(codes.FailedPrecondition, err.Error())
	}

	agent, err := r.get(pmmAgentID)
	if err != nil {
		return status.Errorf(codes.FailedPrecondition,
			"The credentials are stored in Vault by the pmm-agent, and pmm-agent %s is not connected.", pmmAgentID)
	}

	// The keys only: the values are the credentials.
	logger.Get(ctx).WithField("component", "agents/vault").
		Infof("Storing keys %v of Vault secret %s through pmm-agent %s.", slices.Sorted(maps.Keys(data)), path, pmmAgentID)

	resp, err := agent.channel.SendAndWaitResponse(ctx, &agentv1.StoreSecretRequest{Path: path, Data: data})
	if err != nil {
		return err
	}

	res, ok := resp.(*agentv1.StoreSecretResponse)
	if !ok {
		return status.Errorf(codes.Internal, "unexpected response type %T", resp)
	}
	if res.Error != "" {
		return status.Errorf(codes.FailedPrecondition, "Failed to store the credentials in Vault: %s.", res.Error)
	}
	return nil
}
