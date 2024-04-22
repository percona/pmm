// Copyright (C) 2024 Percona LLC
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

	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	dbaasv1beta1 "github.com/percona/pmm/api/managementpb/dbaas"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/dbaas/kubernetes"
)

// overallLinesLimit defines how many last lines of logs we should return upon
// calling the getClusterLogs method.
const overallLinesLimit = 1000

// LogsService implements dbaasv1beta1.LogsAPIServer methods.
type LogsService struct {
	l  *logrus.Entry
	db *reform.DB

	dbaasv1beta1.UnimplementedLogsAPIServer
}

type tuple struct {
	statuses   []corev1.ContainerStatus
	containers []corev1.Container
}

// NewLogsService creates new LogsService.
func NewLogsService(db *reform.DB) dbaasv1beta1.LogsAPIServer { //nolint:ireturn
	l := logrus.WithField("component", "logs_api")
	return &LogsService{db: db, l: l}
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

	kClient, err := kubernetes.New(kubernetesCluster.KubeConfig)
	if err != nil {
		// return nil, status.Error(codes.Internal, "Cannot initialize K8s kClient: "+err.Error())
		return nil, err
	}

	labelSelector := &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"app.kubernetes.io/instance": in.ClusterName,
		},
	}

	pods, err := kClient.GetPods(ctx, "", labelSelector)
	if err != nil {
		return nil, err
	}

	// Every pod has at least one contaier, set cap to that value.
	response := make([]*dbaasv1beta1.Logs, 0, len(pods.Items))
	for _, pod := range pods.Items {
		tuples := []tuple{
			{
				statuses:   pod.Status.ContainerStatuses,
				containers: pod.Spec.Containers,
			},
			{
				statuses:   pod.Status.InitContainerStatuses,
				containers: pod.Spec.InitContainers,
			},
		}
		// Get all logs from all regular containers and all init containers.
		for _, t := range tuples {
			for _, container := range t.containers {
				logs, err := kClient.GetLogs(
					ctx, t.statuses, pod.Name, container.Name)
				if err != nil {
					return nil, err
				}

				response = append(response, &dbaasv1beta1.Logs{
					Pod:       pod.Name,
					Container: container.Name,
					Logs:      logs,
				})
			}
		}

		// Get pod's events.
		events, err := kClient.GetEvents(ctx, pod.Name)
		if err != nil {
			return nil, err
		}

		response = append(response, &dbaasv1beta1.Logs{
			Pod:       pod.Name,
			Container: "",
			Logs:      events,
		})
	}

	// Limit number of overall log lines.
	limitLines(response, overallLinesLimit)

	return &dbaasv1beta1.GetLogsResponse{
		Logs: response,
	}, nil
}

// limitLines limits each entry's logs lines count in the way the overall sum of
// all log lines is equal to given limit.
func limitLines(logs []*dbaasv1beta1.Logs, limit int) {
	counts := make([]int, len(logs))
	lastSum := -1
	var newSum int
	for newSum < limit && newSum > lastSum {
		lastSum = newSum
		for i, item := range logs {
			if counts[i] < len(item.Logs) {
				counts[i]++
				newSum++
				if newSum == limit {
					break
				}
			}
		}
	}

	// Do the actual slicing.
	for i, item := range logs {
		logs[i].Logs = item.Logs[len(item.Logs)-counts[i]:]
	}
}
