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

package dbaas

import (
	"context"

	goversion "github.com/hashicorp/go-version"
	dbaasv1beta1 "github.com/percona/pmm/api/managementpb/dbaas"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
)

type componentsService struct {
	l                    *logrus.Entry
	db                   *reform.DB
	dbaasClient          dbaasClient
	versionServiceClient versionService
}

// NewComponentsService creates Components Service.
func NewComponentsService(db *reform.DB, dbaasClient dbaasClient, versionServiceClient versionService) dbaasv1beta1.ComponentsServer {
	l := logrus.WithField("component", "components_service")
	return &componentsService{
		l:                    l,
		db:                   db,
		dbaasClient:          dbaasClient,
		versionServiceClient: versionServiceClient,
	}
}

func (c componentsService) GetPSMDBComponents(ctx context.Context, req *dbaasv1beta1.GetPSMDBComponentsRequest) (*dbaasv1beta1.GetPSMDBComponentsResponse, error) {
	params := componentsParams{
		operator:  psmdbOperator,
		dbVersion: req.DbVersion,
	}
	if req.KubernetesClusterName != "" {
		kubernetesCluster, err := models.FindKubernetesClusterByName(c.db.Querier, req.KubernetesClusterName)
		if err != nil {
			return nil, err
		}

		checkResponse, e := c.dbaasClient.CheckKubernetesClusterConnection(ctx, kubernetesCluster.KubeConfig)
		if e != nil {
			return nil, e
		}

		params.operatorVersion = checkResponse.Operators.Psmdb.Version
	}

	versions, err := c.versions(ctx, params)
	if err != nil {
		return nil, err
	}
	return &dbaasv1beta1.GetPSMDBComponentsResponse{Versions: versions}, nil
}

func (c componentsService) GetPXCComponents(ctx context.Context, req *dbaasv1beta1.GetPXCComponentsRequest) (*dbaasv1beta1.GetPXCComponentsResponse, error) {
	params := componentsParams{
		operator:  pxcOperator,
		dbVersion: req.DbVersion,
	}
	if req.KubernetesClusterName != "" {
		kubernetesCluster, err := models.FindKubernetesClusterByName(c.db.Querier, req.KubernetesClusterName)
		if err != nil {
			return nil, err
		}

		checkResponse, e := c.dbaasClient.CheckKubernetesClusterConnection(ctx, kubernetesCluster.KubeConfig)
		if e != nil {
			return nil, e
		}

		params.operatorVersion = checkResponse.Operators.Xtradb.Version
	}

	versions, err := c.versions(ctx, params)
	if err != nil {
		return nil, err
	}
	return &dbaasv1beta1.GetPXCComponentsResponse{Versions: versions}, nil
}

func (c componentsService) versions(ctx context.Context, params componentsParams) ([]*dbaasv1beta1.Version, error) {
	components, err := c.versionServiceClient.Matrix(ctx, params)
	if err != nil {
		return nil, err
	}

	versions := make([]*dbaasv1beta1.Version, 0, len(components.Versions))
	mongodMinimalVersion, _ := goversion.NewVersion("4.2.0")
	pxcMinimalVersion, _ := goversion.NewVersion("8.0.0")
	for _, v := range components.Versions {
		respVersion := &dbaasv1beta1.Version{
			Product:  v.Product,
			Operator: v.Operator,
			Matrix: &dbaasv1beta1.Matrix{
				Mongod:       c.matrix(v.Matrix.Mongod, mongodMinimalVersion),
				Pxc:          c.matrix(v.Matrix.Pxc, pxcMinimalVersion),
				Pmm:          c.matrix(v.Matrix.Pmm, nil),
				Proxysql:     c.matrix(v.Matrix.Proxysql, nil),
				Haproxy:      c.matrix(v.Matrix.Haproxy, nil),
				Backup:       c.matrix(v.Matrix.Backup, nil),
				Operator:     c.matrix(v.Matrix.Operator, nil),
				LogCollector: c.matrix(v.Matrix.LogCollector, nil),
			},
		}
		versions = append(versions, respVersion)
	}

	return versions, nil
}

func (c componentsService) matrix(m map[string]component, minimalVersion *goversion.Version) map[string]*dbaasv1beta1.Component {
	result := make(map[string]*dbaasv1beta1.Component)

	var lastVersion string
	lastVersionParsed, _ := goversion.NewVersion("0.0.0")
	for v, component := range m {
		parsedVersion, err := goversion.NewVersion(v)
		if err != nil {
			c.l.Warnf("couldn't parse version %s: %s", v, err.Error())
			continue
		}
		if minimalVersion != nil && parsedVersion.LessThan(minimalVersion) {
			continue
		}
		result[v] = &dbaasv1beta1.Component{
			ImagePath: component.ImagePath,
			ImageHash: component.ImageHash,
			Status:    component.Status,
			Critical:  component.Critical,
		}
		if lastVersionParsed.LessThan(parsedVersion) && component.Status == "recommended" {
			lastVersionParsed = parsedVersion
			lastVersion = v
		}
	}
	if lastVersion != "" {
		result[lastVersion].Default = true
	}
	return result
}
