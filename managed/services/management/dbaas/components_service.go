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
	"fmt"
	"os"
	"regexp"
	"sync"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	goversion "github.com/hashicorp/go-version"
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

const (
	psmdbOperatorName = "percona-server-mongodb-operator"
	pxcOperatorName   = "percona-xtradb-cluster-operator"
	defaultNamespace  = "default"
	// Dev-latest docker image.
	devLatest = "perconalab/pmm-client:dev-latest"
)

// ComponentsService holds unexported fields and public methods to handle Components Service.
type ComponentsService struct {
	l           *logrus.Entry
	db          *reform.DB
	dbaasClient dbaasClient
	// kubeStorage          *KubeStorage
	versionServiceClient versionService
	kubeStorage          kubeStorageManager

	dbaasv1beta1.UnimplementedComponentsServer
}

type installedComponentsVersion struct {
	kuberentesClusterName string
	pxcOperatorVersion    string
	psmdbOperatorVersion  string
}

// NewComponentsService creates Components Service.
func NewComponentsService(db *reform.DB, dbaasClient dbaasClient, versionServiceClient versionService, kubeStorage kubeStorageManager) *ComponentsService {
	l := logrus.WithField("component", "components_service")
	return &ComponentsService{
		l:           l,
		db:          db,
		dbaasClient: dbaasClient,
		//   kubeStorage:          NewKubeStorage(db),
		versionServiceClient: versionServiceClient,
		kubeStorage:          kubeStorage,
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

// GetPSMDBComponents retrieves all PSMDB components for a specific cluster.
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
		kubeClient, err := c.kubeStorage.GetOrSetClient(req.KubernetesClusterName)
		if err != nil {
			return nil, err
		}
		psmdbVersion, err := kubeClient.GetPSMDBOperatorVersion(ctx)
		if err != nil {
			return nil, err
		}

		params.productVersion = psmdbVersion
	}

	versions, err := c.versions(ctx, params, kubernetesCluster)
	if err != nil {
		return nil, err
	}
	return &dbaasv1beta1.GetPSMDBComponentsResponse{Versions: versions}, nil
}

// GetPXCComponents returns versions details.
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
		kubeClient, err := c.kubeStorage.GetOrSetClient(req.KubernetesClusterName)
		if err != nil {
			return nil, err
		}
		pxcVersion, err := kubeClient.GetPXCOperatorVersion(ctx)
		if err != nil {
			return nil, err
		}

		params.productVersion = pxcVersion
	}

	versions, err := c.versions(ctx, params, kubernetesCluster)
	if err != nil {
		return nil, err
	}
	return &dbaasv1beta1.GetPXCComponentsResponse{Versions: versions}, nil
}

// ChangePSMDBComponents will apply changes on cluster.
func (c ComponentsService) ChangePSMDBComponents(_ context.Context, req *dbaasv1beta1.ChangePSMDBComponentsRequest) (*dbaasv1beta1.ChangePSMDBComponentsResponse, error) {
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

// ChangePXCComponents apply new values on PXC components.
func (c ComponentsService) ChangePXCComponents(_ context.Context, req *dbaasv1beta1.ChangePXCComponentsRequest) (*dbaasv1beta1.ChangePXCComponentsResponse, error) {
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

func (c ComponentsService) installedOperatorsVersion(ctx context.Context, wg *sync.WaitGroup, responseCh chan installedComponentsVersion, kuberentesCluster *models.KubernetesCluster) { //nolint:lll
	defer wg.Done()
	kubeClient, err := c.kubeStorage.GetOrSetClient(kuberentesCluster.KubernetesClusterName)
	if err != nil {
		c.l.Errorf("failed to check get kubernetes client: %v", err)
		responseCh <- installedComponentsVersion{
			kuberentesClusterName: kuberentesCluster.KubernetesClusterName,
		}
		return
	}
	psmdbVersion, err := kubeClient.GetPSMDBOperatorVersion(ctx)
	if err != nil {
		c.l.Errorf("failed to get psmdb operator version: %v", err)
	}
	pxcVersion, err := kubeClient.GetPXCOperatorVersion(ctx)
	if err != nil {
		c.l.Errorf("failed to get pxc operator version: %v", err)
	}

	responseCh <- installedComponentsVersion{
		kuberentesClusterName: kuberentesCluster.KubernetesClusterName,
		pxcOperatorVersion:    psmdbVersion,
		psmdbOperatorVersion:  pxcVersion,
	}
}

// CheckForOperatorUpdate check if update for operator is available.
func (c ComponentsService) CheckForOperatorUpdate(ctx context.Context, _ *dbaasv1beta1.CheckForOperatorUpdateRequest) (*dbaasv1beta1.CheckForOperatorUpdateResponse, error) { //nolint:lll
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
		kubeClient, err := c.kubeStorage.GetOrSetClient(cluster.KubernetesClusterName)
		if err != nil {
			c.l.Errorf("Cannot list the subscriptions for the cluster %q: %s", cluster.KubernetesClusterName, err)
			continue
		}

		subscriptions, err := kubeClient.ListSubscriptions(ctx, "default")
		if err != nil {
			c.l.Errorf("Cannot list the subscriptions for the cluster %q: %s", cluster.KubernetesClusterName, err)
			continue
		}
		resp.ClusterToComponents[cluster.KubernetesClusterName] = &dbaasv1beta1.ComponentsUpdateInformation{
			ComponentToUpdateInformation: map[string]*dbaasv1beta1.ComponentUpdateInformation{
				psmdbOperator: {},
				pxcOperator:   {},
			},
		}

		for _, item := range subscriptions.Items {
			if item.Status.CurrentCSV != item.Status.InstalledCSV {
				re := regexp.MustCompile(`v(\d+\.\d+\.\d+)$`)
				matches := re.FindStringSubmatch(item.Status.CurrentCSV)
				if len(matches) == 2 {
					switch item.Spec.Package {
					case psmdbOperatorName:
						resp.ClusterToComponents[cluster.KubernetesClusterName].ComponentToUpdateInformation[psmdbOperator] = &dbaasv1beta1.ComponentUpdateInformation{
							AvailableVersion: matches[1],
						}
					case pxcOperatorName:
						resp.ClusterToComponents[cluster.KubernetesClusterName].ComponentToUpdateInformation[pxcOperator] = &dbaasv1beta1.ComponentUpdateInformation{
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

// InstallOperator upgrade current operator.
func (c ComponentsService) InstallOperator(ctx context.Context, req *dbaasv1beta1.InstallOperatorRequest) (*dbaasv1beta1.InstallOperatorResponse, error) {
	kubernetesCluster, err := models.FindKubernetesClusterByName(c.db.Querier, req.KubernetesClusterName)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	kubeClient, err := c.kubeStorage.GetOrSetClient(req.KubernetesClusterName)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var component *models.Component
	var installFunc func() error

	switch req.OperatorType {
	case pxcOperator:
		installFunc = func() error {
			return kubeClient.UpgradeOperator(ctx, defaultNamespace, pxcOperatorName)
		}
		component = kubernetesCluster.PXC
	case psmdbOperator:
		installFunc = func() error {
			return kubeClient.UpgradeOperator(ctx, defaultNamespace, psmdbOperatorName)
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
			return nil, status.Errorf(codes.Internal,
				"default database version %s is unsupported by the operator version %s, please change default version.", component.DefaultVersion, req.Version)
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

func getPMMClientImage() string {
	pmmClientImage := "perconalab/pmm-client:dev-latest"

	pmmClientImageEnv, ok := os.LookupEnv("PERCONA_TEST_DBAAS_PMM_CLIENT")
	if ok {
		pmmClientImage = pmmClientImageEnv
		return pmmClientImage
	}

	if pmmversion.PMMVersion == "" { // No version set, use dev-latest.
		return pmmClientImage
	}

	v, err := goversion.NewVersion(pmmversion.PMMVersion) //nolint: varnamelen
	if err != nil {
		return pmmClientImage
	}
	// if version has a suffix like 1.2.0-dev or 3.4.1-HEAD-something it is an unreleased version.
	// Docker image won't exist in the repo so use latest stable.
	if v.Core().String() != v.String() {
		pmmClientImage = "percona/pmm-client:2"
		return pmmClientImage
	}

	exists, err := imageExists(context.Background(), pmmClientImage)
	// if !exists or there was an error while checking if the image exists, use dev-latest as default.
	if !exists || err != nil {
		return devLatest
	}

	pmmClientImage = "percona/pmm-client:" + v.Core().String()
	return pmmClientImage
}

func imageExists(ctx context.Context, name string) (bool, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	defer cli.Close() //nolint:errcheck

	reader, err := cli.ImagePull(ctx, name, image.PullOptions{})
	if err != nil {
		if client.IsErrNotFound(err) {
			return false, nil
		}

		return false, err
	}

	reader.Close() //nolint:errcheck

	return true, nil
}
