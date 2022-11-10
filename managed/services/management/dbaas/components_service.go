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
	"fmt"
	"regexp"
	"sync"

	goversion "github.com/hashicorp/go-version"
	dbaascontrollerv1beta1 "github.com/percona-platform/dbaas-api/gen/controller"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	dbaasv1beta1 "github.com/percona/pmm/api/managementpb/dbaas"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/stringset"
	pmmversion "github.com/percona/pmm/version"
)

type ComponentsService struct {
	l                    *logrus.Entry
	db                   *reform.DB
	dbaasClient          dbaasClient
	versionServiceClient versionService

	dbaasv1beta1.UnimplementedComponentsServer
}

type installedComponentsVersion struct {
	kuberentesClusterName string
	pxcOperatorVersion    string
	psmdbOperatorVersion  string
}

// NewComponentsService creates Components Service.
func NewComponentsService(db *reform.DB, dbaasClient dbaasClient, versionServiceClient versionService) *ComponentsService {
	l := logrus.WithField("component", "components_service")
	return &ComponentsService{
		l:                    l,
		db:                   db,
		dbaasClient:          dbaasClient,
		versionServiceClient: versionServiceClient,
	}
}

// Enabled returns if service is enabled and can be used.
func (c *ComponentsService) Enabled() bool {
	settings, err := models.GetSettings(c.db)
	if err != nil {
		c.l.WithError(err).Error("can't get settings")
		return false
	}
	return settings.DBaaS.Enabled
}

func (c ComponentsService) GetPSMDBComponents(ctx context.Context, req *dbaasv1beta1.GetPSMDBComponentsRequest) (*dbaasv1beta1.GetPSMDBComponentsResponse, error) {
	var kubernetesCluster *models.KubernetesCluster
	params := componentsParams{
		product:   psmdbOperator,
		dbVersion: req.DbVersion,
	}
	if req.KubernetesClusterName != "" {
		var err error
		kubernetesCluster, err = models.FindKubernetesClusterByName(c.db.Querier, req.KubernetesClusterName)
		if err != nil {
			return nil, err
		}

		checkResponse, e := c.dbaasClient.CheckKubernetesClusterConnection(ctx, kubernetesCluster.KubeConfig)
		if e != nil {
			return nil, e
		}

		params.productVersion = checkResponse.Operators.PsmdbOperatorVersion
	}

	versions, err := c.versions(ctx, params, kubernetesCluster)
	if err != nil {
		return nil, err
	}
	return &dbaasv1beta1.GetPSMDBComponentsResponse{Versions: versions}, nil
}

func (c ComponentsService) GetPXCComponents(ctx context.Context, req *dbaasv1beta1.GetPXCComponentsRequest) (*dbaasv1beta1.GetPXCComponentsResponse, error) {
	var kubernetesCluster *models.KubernetesCluster
	params := componentsParams{
		product:   pxcOperator,
		dbVersion: req.DbVersion,
	}
	if req.KubernetesClusterName != "" {
		var err error
		kubernetesCluster, err = models.FindKubernetesClusterByName(c.db.Querier, req.KubernetesClusterName)
		if err != nil {
			return nil, err
		}

		checkResponse, e := c.dbaasClient.CheckKubernetesClusterConnection(ctx, kubernetesCluster.KubeConfig)
		if e != nil {
			return nil, e
		}

		params.productVersion = checkResponse.Operators.PxcOperatorVersion
	}

	versions, err := c.versions(ctx, params, kubernetesCluster)
	if err != nil {
		return nil, err
	}
	return &dbaasv1beta1.GetPXCComponentsResponse{Versions: versions}, nil
}

func (c ComponentsService) ChangePSMDBComponents(ctx context.Context, req *dbaasv1beta1.ChangePSMDBComponentsRequest) (*dbaasv1beta1.ChangePSMDBComponentsResponse, error) {
	err := c.db.InTransaction(func(tx *reform.TX) error {
		kubernetesCluster, e := models.FindKubernetesClusterByName(tx.Querier, req.KubernetesClusterName)
		if e != nil {
			return e
		}

		if req.Mongod != nil {
			kubernetesCluster.Mongod, e = setComponent(kubernetesCluster.Mongod, req.Mongod)
			if e != nil {
				message := fmt.Sprintf("%s, cluster: %s, component: mongod", e.Error(), kubernetesCluster.KubernetesClusterName)
				return status.Errorf(codes.InvalidArgument, message)
			}
		}

		e = tx.Save(kubernetesCluster)
		if e != nil {
			return e
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &dbaasv1beta1.ChangePSMDBComponentsResponse{}, nil
}

func (c ComponentsService) ChangePXCComponents(ctx context.Context, req *dbaasv1beta1.ChangePXCComponentsRequest) (*dbaasv1beta1.ChangePXCComponentsResponse, error) {
	err := c.db.InTransaction(func(tx *reform.TX) error {
		kubernetesCluster, e := models.FindKubernetesClusterByName(tx.Querier, req.KubernetesClusterName)
		if e != nil {
			return e
		}

		if req.Pxc != nil {
			kubernetesCluster.PXC, e = setComponent(kubernetesCluster.PXC, req.Pxc)
			if e != nil {
				message := fmt.Sprintf("%s, cluster: %s, component: pxc", e.Error(), kubernetesCluster.KubernetesClusterName)
				return status.Errorf(codes.InvalidArgument, message)
			}
		}

		if req.Proxysql != nil {
			kubernetesCluster.ProxySQL, e = setComponent(kubernetesCluster.ProxySQL, req.Proxysql)
			if e != nil {
				message := fmt.Sprintf("%s, cluster: %s, component: proxySQL", e.Error(), kubernetesCluster.KubernetesClusterName)
				return status.Errorf(codes.InvalidArgument, message)
			}
		}

		if req.Haproxy != nil {
			kubernetesCluster.HAProxy, e = setComponent(kubernetesCluster.HAProxy, req.Haproxy)
			if e != nil {
				message := fmt.Sprintf("%s, cluster: %s, component: HAProxy", e.Error(), kubernetesCluster.KubernetesClusterName)
				return status.Errorf(codes.InvalidArgument, message)
			}
		}
		e = tx.Save(kubernetesCluster)
		if e != nil {
			return e
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &dbaasv1beta1.ChangePXCComponentsResponse{}, nil
}

func (c ComponentsService) installedOperatorsVersion(ctx context.Context, wg *sync.WaitGroup, responseCh chan installedComponentsVersion, kuberentesCluster *models.KubernetesCluster) {
	defer wg.Done()
	resp, err := c.dbaasClient.CheckKubernetesClusterConnection(ctx, kuberentesCluster.KubeConfig)
	if err != nil {
		c.l.Errorf("failed to check kubernetes cluster connection: %v", err)
		responseCh <- installedComponentsVersion{
			kuberentesClusterName: kuberentesCluster.KubernetesClusterName,
		}
		return
	}
	responseCh <- installedComponentsVersion{
		kuberentesClusterName: kuberentesCluster.KubernetesClusterName,
		pxcOperatorVersion:    resp.Operators.PxcOperatorVersion,
		psmdbOperatorVersion:  resp.Operators.PsmdbOperatorVersion,
	}
}

func (c ComponentsService) CheckForOperatorUpdate(ctx context.Context, req *dbaasv1beta1.CheckForOperatorUpdateRequest) (*dbaasv1beta1.CheckForOperatorUpdateResponse, error) {
	if pmmversion.PMMVersion == "" {
		return nil, status.Error(codes.Internal, "failed to get current PMM version")
	}

	// List all kuberenetes clusters.
	clusters, err := models.FindAllKubernetesClusters(c.db.Querier)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	// And get operators version from all of them.
	responseCh := make(chan installedComponentsVersion, len(clusters))
	go func() {
		wg := &sync.WaitGroup{}
		wg.Add(len(clusters))
		for _, cluster := range clusters {
			k8sCluster := cluster
			go c.installedOperatorsVersion(ctx, wg, responseCh, k8sCluster)
		}
		wg.Wait()
		close(responseCh)
	}()

	resp := &dbaasv1beta1.CheckForOperatorUpdateResponse{
		ClusterToComponents: make(map[string]*dbaasv1beta1.ComponentsUpdateInformation),
	}

	for _, cluster := range clusters {
		subscriptions, err := c.dbaasClient.ListSubscriptions(ctx, &dbaascontrollerv1beta1.ListSubscriptionsRequest{
			KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
				Kubeconfig: cluster.KubeConfig,
			},
		})
		if err != nil {
			continue
		}
		resp.ClusterToComponents[cluster.KubernetesClusterName] = &dbaasv1beta1.ComponentsUpdateInformation{
			ComponentToUpdateInformation: make(map[string]*dbaasv1beta1.ComponentUpdateInformation),
		}

		for _, item := range subscriptions.Items {
			if item.CurrentCsv != item.InstalledCsv {
				re := regexp.MustCompile(`v(\d+\.\d+\.\d+)$`)
				matches := re.FindStringSubmatch(item.CurrentCsv)
				if len(matches) == 2 {
					switch item.Package {
					case "percona-server-mongodb-operator":
						resp.ClusterToComponents[cluster.KubernetesClusterName].ComponentToUpdateInformation[psmdbOperator] = &dbaasv1beta1.ComponentUpdateInformation{
							AvailableVersion: matches[1],
						}
					case "percona-xtradb-cluster-operator":
						resp.ClusterToComponents[cluster.KubernetesClusterName].ComponentToUpdateInformation[psmdbOperator] = &dbaasv1beta1.ComponentUpdateInformation{
							AvailableVersion: matches[1],
						}
					}
				}
			}
		}
	}

	return resp, nil
}

func (c ComponentsService) versions(ctx context.Context, params componentsParams, cluster *models.KubernetesCluster) ([]*dbaasv1beta1.OperatorVersion, error) {
	components, err := c.versionServiceClient.Matrix(ctx, params)
	if err != nil {
		return nil, err
	}

	var mongod, pxc, proxySQL, haproxy *models.Component
	if cluster != nil {
		mongod = cluster.Mongod
		pxc = cluster.PXC
		proxySQL = cluster.ProxySQL
		haproxy = cluster.HAProxy
	}

	versions := make([]*dbaasv1beta1.OperatorVersion, 0, len(components.Versions))
	mongodMinimalVersion, _ := goversion.NewVersion("4.2.0")
	pxcMinimalVersion, _ := goversion.NewVersion("8.0.0")
	for _, v := range components.Versions {
		respVersion := &dbaasv1beta1.OperatorVersion{
			Product:  v.Product,
			Operator: v.ProductVersion,
			Matrix: &dbaasv1beta1.Matrix{
				Mongod:       c.matrix(v.Matrix.Mongod, mongodMinimalVersion, mongod),
				Pxc:          c.matrix(v.Matrix.Pxc, pxcMinimalVersion, pxc),
				Pmm:          c.matrix(v.Matrix.Pmm, nil, nil),
				Proxysql:     c.matrix(v.Matrix.Proxysql, nil, proxySQL),
				Haproxy:      c.matrix(v.Matrix.Haproxy, nil, haproxy),
				Backup:       c.matrix(v.Matrix.Backup, nil, nil),
				Operator:     c.matrix(v.Matrix.Operator, nil, nil),
				LogCollector: c.matrix(v.Matrix.LogCollector, nil, nil),
			},
		}
		versions = append(versions, respVersion)
	}

	return versions, nil
}

func (c ComponentsService) matrix(m map[string]componentVersion, minimalVersion *goversion.Version, kc *models.Component) map[string]*dbaasv1beta1.Component {
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

	defaultVersionSet := false
	if kc != nil {
		if _, ok := result[kc.DefaultVersion]; ok {
			result[kc.DefaultVersion].Default = true
			defaultVersionSet = true
		}
		for _, v := range kc.DisabledVersions {
			if _, ok := result[v]; ok {
				result[v].Disabled = true
			}
		}
	}
	if lastVersion != "" && !defaultVersionSet {
		result[lastVersion].Default = true
	}
	return result
}

func setComponent(kc *models.Component, rc *dbaasv1beta1.ChangeComponent) (*models.Component, error) {
	if kc == nil {
		kc = &models.Component{}
	}
	if rc.DefaultVersion != "" {
		kc.DefaultVersion = rc.DefaultVersion
	}

	disabledVersions := make(map[string]struct{})
	for _, v := range kc.DisabledVersions {
		disabledVersions[v] = struct{}{}
	}
	for _, v := range rc.Versions {
		if v.Enable && v.Disable {
			return nil, fmt.Errorf("enable and disable for version %s can't be passed together", v.Version)
		}
		if v.Enable {
			delete(disabledVersions, v.Version)
		}
		if v.Disable {
			disabledVersions[v.Version] = struct{}{}
		}
	}
	if _, ok := disabledVersions[kc.DefaultVersion]; ok {
		return nil, fmt.Errorf("default version can't be disabled")
	}
	kc.DisabledVersions = stringset.ToSlice(disabledVersions)
	return kc, nil
}

func (c ComponentsService) InstallOperator(ctx context.Context, req *dbaasv1beta1.InstallOperatorRequest) (*dbaasv1beta1.InstallOperatorResponse, error) {
	kubernetesCluster, err := models.FindKubernetesClusterByName(c.db.Querier, req.KubernetesClusterName)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	var component *models.Component
	var installFunc func() error
	switch req.OperatorType {
	case pxcOperator:
		installFunc = func() error {
			err := approveInstallPlan(ctx, c.dbaasClient, kubernetesCluster.KubeConfig, "percona-xtradb-cluster-operator")
			return err
		}
		component = kubernetesCluster.PXC
	case psmdbOperator:
		installFunc = func() error {
			err := approveInstallPlan(ctx, c.dbaasClient, kubernetesCluster.KubeConfig, "percona-server-mongodb-operator")
			return err
		}
		component = kubernetesCluster.Mongod
	default:
		return nil, errors.Errorf("%q is not supported operator", req.OperatorType)
	}

	if component != nil {
		// Default version of database could be unsupported be a new operator version.
		supported, err := c.versionServiceClient.IsDatabaseVersionSupportedByOperator(ctx, req.OperatorType, req.Version, component.DefaultVersion)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to check if default database version is supported by the operator version: %v", err)
		}
		if !supported {
			return nil, status.Errorf(codes.Internal, "default database version %s is unsupported by the operator version %s, please change default version.", component.DefaultVersion, req.Version)
		}
	}

	// Install operator.
	if err := installFunc(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to install operator: %v", err)
	}

	return &dbaasv1beta1.InstallOperatorResponse{Status: dbaasv1beta1.OperatorsStatus_OPERATORS_STATUS_OK}, nil
}

// DefaultComponent returns the component marked as default in the components list.
func DefaultComponent(m map[string]*dbaasv1beta1.Component) (*dbaasv1beta1.Component, error) {
	if len(m) == 0 {
		return nil, errNoVersionsFound
	}

	for _, component := range m {
		if component.Default {
			return &dbaasv1beta1.Component{
					ImagePath: component.ImagePath,
					ImageHash: component.ImageHash,
					Status:    component.Status,
					Critical:  component.Critical,
				},
				nil
		}
	}

	return nil, errors.New("cannot find a default version in the components list")
}
