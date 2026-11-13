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

package services

import (
	"context"

	"github.com/AlekSi/pointer"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// SecretStorer stores credentials in Vault through a pmm-agent, see agents.Registry.StoreSecret.
type SecretStorer interface {
	StoreSecret(ctx context.Context, q *reform.Querier, pmmAgentID, path string, data map[string]string) error
}

// ServiceConnectionChecker checks that an Agent connects to its Service, see agents.ConnectionChecker.
type ServiceConnectionChecker interface {
	CheckConnectionToService(ctx context.Context, q *reform.Querier, service *models.Service, agent *models.Agent) error
}

// StoreCredentialsInVault stores the given secrets, by key, in the Vault secret of the Agent, through its
// pmm-agent. Unless skipCheck is set, it first checks that the Agent connects with them: they replace the
// secrets of every Agent that reads the secret, which wrong ones would break. It reports whether it
// stored anything.
func StoreCredentialsInVault(
	ctx context.Context,
	q *reform.Querier,
	storer SecretStorer,
	checker ServiceConnectionChecker,
	agent *models.Agent,
	secrets map[string]string,
	skipCheck bool,
) (bool, error) {
	if len(secrets) == 0 {
		return false, nil
	}

	if !skipCheck && agent.ServiceID != nil {
		service, err := models.FindServiceByID(q, *agent.ServiceID)
		if err != nil {
			return false, err
		}

		err = checker.CheckConnectionToService(ctx, q, service, agent.WithCheckSecrets(secrets))
		if err != nil {
			return false, err
		}
	}

	err := storer.StoreSecret(ctx, q, pointer.GetString(agent.PMMAgentID), pointer.GetString(agent.VaultPath), secrets)
	if err != nil {
		return false, err
	}
	return true, nil
}
