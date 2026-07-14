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

package dbaas

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	dbaasv1beta1 "github.com/percona/pmm/api/managementpb/dbaas"
	"github.com/percona/pmm/managed/services/dbaas/kubernetes"
)

type templateService struct {
	db          *reform.DB
	l           *logrus.Entry
	kubeStorage *KubeStorage

	dbaasv1beta1.UnimplementedTemplatesServer
}

// NewTemplateService creates DB Clusters Service.
func NewTemplateService(db *reform.DB) dbaasv1beta1.TemplatesServer { //nolint:ireturn
	l := logrus.WithField("component", "dbaas_db_cluster")
	return &templateService{
		db:          db,
		l:           l,
		kubeStorage: NewKubeStorage(db),
	}
}

// ListTemplates returns a list of templates.
func (s templateService) ListTemplates(ctx context.Context, req *dbaasv1beta1.ListTemplatesRequest) (*dbaasv1beta1.ListTemplatesResponse, error) {
	var clusterType string
	switch req.ClusterType {
	case dbaasv1beta1.DBClusterType_DB_CLUSTER_TYPE_PXC:
		clusterType = string(kubernetes.DatabaseTypePXC)
	case dbaasv1beta1.DBClusterType_DB_CLUSTER_TYPE_PSMDB:
		clusterType = string(kubernetes.DatabaseTypePSMDB)
	default:
		return nil, status.Error(codes.InvalidArgument, "unexpected DB cluster type")
	}

	kubeClient, err := s.kubeStorage.GetOrSetClient(req.KubernetesClusterName)
	if err != nil {
		return nil, err
	}

	// XXX: using defaultNamespace because that's where the operator is
	// installed, must be the same as defined in kubernetes_server.go
	templates, err := kubeClient.ListTemplates(ctx, clusterType, defaultNamespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed listing templates")
	}

	return &dbaasv1beta1.ListTemplatesResponse{
		Templates: templates,
	}, nil
}
