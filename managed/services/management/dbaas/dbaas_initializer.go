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
	"math/rand"
	"sync"
	"time"

	goversion "github.com/hashicorp/go-version"
	dbaascontrollerv1beta1 "github.com/percona-platform/dbaas-api/gen/controller"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	dbaasv1beta1 "github.com/percona/pmm/api/managementpb/dbaas"
	"github.com/percona/pmm/managed/models"
	pmmversion "github.com/percona/pmm/version"
)

type Initializer struct {
	db *reform.DB
	l  *logrus.Entry

	dbaasClient    dbaasClient
	grafanaClient  grafanaClient
	versionService versionService

	enabled bool
	cancel  func()
	m       sync.Mutex
}

const defaultClusterName = "default-pmm-cluster"

var errClusterExists = errors.New("cluster already exists")

func NewInitializer(db *reform.DB, client dbaasClient, grafanaClient grafanaClient, versionService versionService) *Initializer {
	l := logrus.WithField("component", "dbaas_initializer")
	return &Initializer{
		db:             db,
		l:              l,
		dbaasClient:    client,
		grafanaClient:  grafanaClient,
		versionService: versionService,
	}
}

func (in *Initializer) Update(ctx context.Context) error {
	settings, err := models.GetSettings(in.db)
	if err != nil {
		in.l.Errorf("Failed to get settings: %+v.", err)
		return err
	}
	if settings.DBaaS.Enabled {
		return in.Enable(ctx)
	} else {
		return in.Disable(ctx)
	}
}

func (in *Initializer) Enable(ctx context.Context) error {
	in.m.Lock()
	defer in.m.Unlock()
	if in.enabled {
		return nil
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	err := in.dbaasClient.Connect(timeoutCtx)
	cancel()
	if err != nil {
		return err
	}
	ctx, in.cancel = context.WithCancel(ctx)
	in.enabled = true
	kubeConfig, err := in.dbaasClient.GetKubeConfig()
	if err == nil {
		// If err is not equal to nil, dont' register cluster and fail silently
		err := in.db.InTransaction(func(t *reform.TX) error {
			cluster, err := models.FindKubernetesClusterByName(t, defaultClusterName)
			if err != nil {
				return err
			}
			if cluster != nil {
				return errClusterExists
			}
			return nil
		})
		if err != nil {
			if err == errClusterExists {
				return nil
			}
			return err
		}

		req := &dbaasv1beta1.RegisterKubernetesClusterRequest{
			KubernetesClusterName: defaultClusterName,
			KubeAuth: &dbaasv1beta1.KubeAuth{
				Kubeconfig: kubeConfig,
			},
		}
		_, err := in.RegisterCluster(context.Background(), req)
		if err != nil {
			return err
		}
		in.l.Info("Cluster is successfully initialized")
	} else {
		in.l.Errorf("failed getting kubeconfig inside cluster: %v", err)
	}

	return nil
}

func (in *Initializer) Disable(ctx context.Context) error {
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

func (in *Initializer) RegisterCluster(ctx context.Context, req *dbaasv1beta1.RegisterKubernetesClusterRequest) (*dbaasv1beta1.RegisterKubernetesClusterResponse, error) {
	var err error
	req.KubeAuth.Kubeconfig, err = replaceAWSAuthIfPresent(req.KubeAuth.Kubeconfig, req.AwsAccessKeyId, req.AwsSecretAccessKey)
	if err != nil {
		if errors.Is(err, errKubeconfigIsEmpty) {
			return nil, status.Error(codes.InvalidArgument, "Kubeconfig can't be empty")
		} else if errors.Is(err, errMissingRequiredKubeconfigEnvVar) {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("Failed to transform kubeconfig to work with aws-iam-authenticator: %s", err))
		}
		in.l.Errorf("Replacing `aws` with `aim-authenticator` failed: %s", err)
		return nil, status.Error(codes.Internal, "Internal server error")
	}
	var clusterInfo *dbaascontrollerv1beta1.CheckKubernetesClusterConnectionResponse
	err = in.db.InTransaction(func(t *reform.TX) error {
		var e error
		clusterInfo, e = in.dbaasClient.CheckKubernetesClusterConnection(ctx, req.KubeAuth.Kubeconfig)
		if e != nil {
			return e
		}

		_, err := models.CreateKubernetesCluster(t.Querier, &models.CreateKubernetesClusterParams{
			KubernetesClusterName: req.KubernetesClusterName,
			KubeConfig:            req.KubeAuth.Kubeconfig,
		})
		return err
	})
	if err != nil {
		return nil, err
	}
	pmmVersion, err := goversion.NewVersion(pmmversion.PMMVersion)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	pxcOperatorVersion, psmdbOperatorVersion, err := in.versionService.LatestOperatorVersion(ctx, pmmVersion.Core().String())
	if err != nil {
		return nil, err
	}
	if pxcOperatorVersion != nil && (clusterInfo.Operators == nil || clusterInfo.Operators.PxcOperatorVersion == "") {
		_, err = in.dbaasClient.InstallPXCOperator(ctx, &dbaascontrollerv1beta1.InstallPXCOperatorRequest{
			KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
				Kubeconfig: req.KubeAuth.Kubeconfig,
			},
			Version: pxcOperatorVersion.String(),
		})
		if err != nil {
			return nil, err
		}
	}
	if psmdbOperatorVersion != nil && (clusterInfo.Operators == nil || clusterInfo.Operators.PsmdbOperatorVersion == "") {
		_, err = in.dbaasClient.InstallPSMDBOperator(ctx, &dbaascontrollerv1beta1.InstallPSMDBOperatorRequest{
			KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
				Kubeconfig: req.KubeAuth.Kubeconfig,
			},
			Version: psmdbOperatorVersion.String(),
		})
		if err != nil {
			return nil, err
		}
	}

	settings, err := models.GetSettings(in.db.Querier)
	if err != nil {
		return nil, err
	}
	if settings.PMMPublicAddress != "" {
		var apiKeyID int64
		var apiKey string
		apiKeyName := fmt.Sprintf("pmm-vmagent-%s-%d", req.KubernetesClusterName, rand.Int63())
		apiKeyID, apiKey, err = in.grafanaClient.CreateAdminAPIKey(ctx, apiKeyName)
		if err != nil {
			return nil, err
		}
		pmmParams := &dbaascontrollerv1beta1.PMMParams{
			PublicAddress: fmt.Sprintf("https://%s", settings.PMMPublicAddress),
			Login:         "api_key",
			Password:      apiKey,
		}

		_, err := in.dbaasClient.StartMonitoring(ctx, &dbaascontrollerv1beta1.StartMonitoringRequest{
			KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
				Kubeconfig: req.KubeAuth.Kubeconfig,
			},
			Pmm: pmmParams,
		})
		if err != nil {
			e := in.grafanaClient.DeleteAPIKeyByID(ctx, apiKeyID)
			if e != nil {
				in.l.Warnf("couldn't delete created API Key %v: %s", apiKeyID, e)
			}
			in.l.Warnf("couldn't start monitoring of the kubernetes cluster: %s", err)
			return nil, status.Errorf(codes.Internal, "couldn't start monitoring of the kubernetes cluster: %s", err.Error())
		}
	}

	return &dbaasv1beta1.RegisterKubernetesClusterResponse{}, nil

}
