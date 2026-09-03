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

package alerting

import (
	"context"

	"github.com/grafana/grafana-openapi-client-go/models"

	"github.com/percona/pmm/managed/services"
)

type grafanaClient interface {
	CreateAlertRule(ctx context.Context, folderUID, groupName, interval string, rule *services.Rule) error
	GetDatasourceUIDByName(ctx context.Context, name string) (string, error)
	GetFolderByUID(ctx context.Context, uid string) (*models.Folder, error)
}

// grafanaProvisioningClient is the part of the Grafana client the rule provisioner needs. It is
// kept apart from grafanaClient because the two are used by unrelated code paths: one serves user
// requests, this one runs in the background and has no credentials, which is why readiness is all
// it can ask for.
type grafanaProvisioningClient interface {
	IsReady(ctx context.Context) error
}

// supervisordService is a subset of methods of supervisord.Service used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type supervisordService interface {
	StartSupervisedService(serviceName string) error
	RestartSupervisedService(serviceName string) error
	IsSupervisedServiceRunning(serviceName string) (*bool, error)
}

// leaderService reports whether this node may act on behalf of the cluster. It reports true on a
// standalone server, where every node is trivially the leader.
type leaderService interface {
	IsLeader() bool
}
