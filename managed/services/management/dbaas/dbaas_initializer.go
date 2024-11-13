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
	"sync"
	"time"

	dbaascontrollerv1beta1 "github.com/percona-platform/dbaas-api/gen/controller"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"
	"k8s.io/client-go/rest"

	dbaasv1beta1 "github.com/percona/pmm/api/managementpb/dbaas"
	"github.com/percona/pmm/managed/models"
)

// Initializer initializes dbaas feature.
type Initializer struct {
	db *reform.DB
	l  *logrus.Entry

	dbaasClient      dbaasClient
	kubernetesServer dbaasv1beta1.KubernetesServer

	enabled bool
	cancel  func()
	m       sync.Mutex
}

const (
	defaultClusterName  = "default-pmm-cluster"
	pxcSecretNameTmpl   = "dbaas-%s-pxc-secrets"   //nolint:gosec
	psmdbSecretNameTmpl = "dbaas-%s-psmdb-secrets" //nolint:gosec
)

var errClusterExists = errors.New("cluster already exists")

// NewInitializer returns initialized Initializer structure.
func NewInitializer(db *reform.DB, client dbaasClient) *Initializer {
	l := logrus.WithField("component", "dbaas_initializer")
	return &Initializer{
		db:          db,
		l:           l,
		dbaasClient: client,
	}
}

// RegisterKubernetesServer sets the Kubernetes server instance.
func (in *Initializer) RegisterKubernetesServer(k dbaasv1beta1.KubernetesServer) {
	in.kubernetesServer = k
}

// Update updates current dbaas settings.
func (in *Initializer) Update(ctx context.Context) error {
	settings, err := models.GetSettings(in.db)
	if err != nil {
		in.l.Errorf("Failed to get settings: %+v.", err)
		return err
	}
	if settings.DBaaS.Enabled {
		return in.Enable(ctx)
	}
	return in.Disable(ctx)
}

// Enable enables dbaas feature and connects to dbaas-controller.
func (in *Initializer) Enable(ctx context.Context) error {
	in.m.Lock()
	defer in.m.Unlock()
	if in.enabled {
		return nil
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	err := in.dbaasClient.Connect(timeoutCtx)
	if err != nil {
		return err
	}
	ctx, in.cancel = context.WithCancel(ctx)

	in.enabled = true
	return in.registerInCluster(ctx)
}

// registerIncluster automatically adds k8s cluster to dbaas when PMM is running inside k8s cluster.
func (in *Initializer) registerInCluster(ctx context.Context) error {
	kubeConfig, err := in.dbaasClient.GetKubeConfig(ctx, &dbaascontrollerv1beta1.GetKubeconfigRequest{})
	switch {
	case err == nil:
		// If err is not equal to nil, dont' register cluster and fail silently
		err := in.db.InTransaction(func(t *reform.TX) error {
			cluster, err := models.FindKubernetesClusterByName(t.Querier, defaultClusterName)
			if err != nil {
				in.l.Errorf("failed finding cluster: %v", err)
				return nil
			}
			if cluster != nil {
				return errClusterExists
			}
			return nil
		})
		if err != nil {
			if errors.Is(err, errClusterExists) {
				return nil
			}
			return err
		}
		if len(kubeConfig.Kubeconfig) != 0 {
			req := &dbaasv1beta1.RegisterKubernetesClusterRequest{
				KubernetesClusterName: defaultClusterName,
				KubeAuth: &dbaasv1beta1.KubeAuth{
					Kubeconfig: kubeConfig.Kubeconfig,
				},
			}
			_, err = in.kubernetesServer.RegisterKubernetesCluster(ctx, req)
			if err != nil {
				return err
			}
			in.l.Info("Cluster is successfully initialized")
		}
	case errors.Is(err, rest.ErrNotInCluster):
		in.l.Info("PMM is running outside a kubernetes cluster")
	default:
		in.l.Errorf("failed getting kubeconfig inside cluster: %v", err)
	}
	return nil
}

// Disable disconnects from dbaas-controller and disabled dbaas feature.
func (in *Initializer) Disable(_ context.Context) error {
	in.m.Lock()
	defer in.m.Unlock()
	if !in.enabled { // Don't disable if already disabled
		return nil
	}
	if in.cancel != nil {
		in.cancel()
	}
	err := in.dbaasClient.Disconnect()
	if err != nil {
		return err
	}
	in.enabled = false
	return nil
}
