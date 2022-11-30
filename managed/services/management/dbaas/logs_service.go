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

package dbaas

import (
	"context"

	dbaascontrollerv1beta1 "github.com/percona-platform/dbaas-api/gen/controller"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	dbaasv1beta1 "github.com/percona/pmm/api/managementpb/dbaas"
	"github.com/percona/pmm/managed/models"
)

// LogsService implements dbaasv1beta1.LogsAPIServer methods.
type LogsService struct {
	l                *logrus.Entry
	db               *reform.DB
	controllerClient dbaasClient

	dbaasv1beta1.UnimplementedLogsAPIServer
}

// NewLogsService creates new LogsService.
func NewLogsService(db *reform.DB, client dbaasClient) dbaasv1beta1.LogsAPIServer {
	l := logrus.WithField("component", "logs_api")
	return &LogsService{db: db, l: l, controllerClient: client}
}

// Enabled returns if service is enabled and can be used.
func (s *LogsService) Enabled() bool {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		s.l.WithError(err).Error("can't get settings")
		return false
	}
	return settings.DBaaS.Enabled
}

// GetLogs returns container's logs of a database cluster and its pods events.
func (s LogsService) GetLogs(ctx context.Context, in *dbaasv1beta1.GetLogsRequest) (*dbaasv1beta1.GetLogsResponse, error) {
	kubernetesCluster, err := models.FindKubernetesClusterByName(s.db.Querier, in.KubernetesClusterName)
	if err != nil {
		return nil, err
	}

	req := &dbaascontrollerv1beta1.GetLogsRequest{
		KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
			Kubeconfig: kubernetesCluster.KubeConfig,
		},
		ClusterName: in.ClusterName,
	}
	out, err := s.controllerClient.GetLogs(ctx, req)
	if err != nil {
		return nil, err
	}

	logs := make([]*dbaasv1beta1.Logs, 0, len(out.Logs))
	for _, l := range out.Logs {
		logs = append(logs, &dbaasv1beta1.Logs{
			Pod:       l.Pod,
			Container: l.Container,
			Logs:      l.Logs,
		})
	}
	return &dbaasv1beta1.GetLogsResponse{
		Logs: logs,
	}, nil
}
