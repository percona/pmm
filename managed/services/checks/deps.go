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

package checks

import (
	"context"

	"github.com/percona/pmm/api/alertmanager/ammodels"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
)

//go:generate mockery -name=agentsRegistry -case=snake -inpkg -testonly
//go:generate mockery -name=alertmanagerService -case=snake -inpkg -testonly

// agentsRegistry is a subset of methods of agents.Registry used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type agentsRegistry interface {
	StartMySQLQueryShowAction(ctx context.Context, id, pmmAgentID, dsn, query string, files map[string]string, tdp *models.DelimiterPair, tlsSkipVerify bool) error
	StartMySQLQuerySelectAction(ctx context.Context, id, pmmAgentID, dsn, query string, files map[string]string, tdp *models.DelimiterPair, tlsSkipVerify bool) error
	StartPostgreSQLQueryShowAction(ctx context.Context, id, pmmAgentID, dsn string) error
	StartPostgreSQLQuerySelectAction(ctx context.Context, id, pmmAgentID, dsn, query string) error
	StartMongoDBQueryGetParameterAction(ctx context.Context, id, pmmAgentID, dsn string, files map[string]string, tdp *models.DelimiterPair) error
	StartMongoDBQueryBuildInfoAction(ctx context.Context, id, pmmAgentID, dsn string, files map[string]string, tdp *models.DelimiterPair) error
	StartMongoDBQueryGetCmdLineOptsAction(ctx context.Context, id, pmmAgentID, dsn string, files map[string]string, tdp *models.DelimiterPair) error
	StartMongoDBQueryReplSetGetStatusAction(ctx context.Context, id, pmmAgentID, dsn string, files map[string]string, tdp *models.DelimiterPair) error
	StartMongoDBQueryGetDiagnosticDataAction(ctx context.Context, id, pmmAgentID, dsn string, files map[string]string, tdp *models.DelimiterPair) error
}

// alertmanagerService is is a subset of methods of alertmanager.Service used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type alertmanagerService interface {
	SendAlerts(ctx context.Context, alerts ammodels.PostableAlerts)
	SilenceAlerts(ctx context.Context, alerts []*ammodels.GettableAlert) error
	UnsilenceAlerts(ctx context.Context, alerts []*ammodels.GettableAlert) error
	GetAlerts(ctx context.Context, params *services.FilterParams) ([]*ammodels.GettableAlert, error)
}
