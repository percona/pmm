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

// Package dbaas contains logic related to communication with dbaas-controller.
//
//nolint:lll
package dbaas

import (
	"context"
	"sync"
	"time"

	controllerv1beta1 "github.com/percona-platform/dbaas-api/gen/controller"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/percona/pmm/managed/services/dbaas/kubernetes"
	"github.com/percona/pmm/version"
)

// Client is a client for dbaas-controller.
type Client struct {
	l                         *logrus.Entry
	kubernetesClient          controllerv1beta1.KubernetesClusterAPIClient
	pxcClusterClient          controllerv1beta1.PXCClusterAPIClient
	psmdbClusterClient        controllerv1beta1.PSMDBClusterAPIClient
	logsClient                controllerv1beta1.LogsAPIClient
	pxcOperatorClient         controllerv1beta1.PXCOperatorAPIClient
	psmdbOperatorClient       controllerv1beta1.PSMDBOperatorAPIClient
	connM                     sync.RWMutex
	conn                      *grpc.ClientConn
	dbaasControllerAPIAddress string
}

// NewClient creates new Client object.
func NewClient(dbaasControllerAPIAddress string) *Client {
	c := &Client{
		l:                         logrus.WithField("component", "dbaas.Client"),
		dbaasControllerAPIAddress: dbaasControllerAPIAddress,
	}
	return c
}

// Connect connects the client to dbaas-controller API.
func (c *Client) Connect(ctx context.Context) error {
	c.connM.Lock()
	defer c.connM.Unlock()
	c.l.Infof("Connecting to dbaas-controller API on %s.", c.dbaasControllerAPIAddress)
	if c.conn != nil {
		c.l.Warnf("Trying to connect to dbaas-controller API but connection is already up.")
		return nil
	}
	backoffConfig := backoff.DefaultConfig
	backoffConfig.MaxDelay = 10 * time.Second
	opts := []grpc.DialOption{
		grpc.WithBlock(), // Dial blocks, we do not connect in background.
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithConnectParams(grpc.ConnectParams{Backoff: backoffConfig, MinConnectTimeout: 10 * time.Second}),
		grpc.WithUserAgent("pmm-managed/" + version.Version),
	}

	conn, err := grpc.DialContext(ctx, c.dbaasControllerAPIAddress, opts...)
	if err != nil {
		return errors.Errorf("failed to connect to dbaas-controller API: %v", err)
	}
	c.conn = conn

	c.kubernetesClient = controllerv1beta1.NewKubernetesClusterAPIClient(conn)
	c.pxcClusterClient = controllerv1beta1.NewPXCClusterAPIClient(conn)
	c.psmdbClusterClient = controllerv1beta1.NewPSMDBClusterAPIClient(conn)
	c.logsClient = controllerv1beta1.NewLogsAPIClient(conn)
	c.psmdbOperatorClient = controllerv1beta1.NewPSMDBOperatorAPIClient(conn)
	c.pxcOperatorClient = controllerv1beta1.NewPXCOperatorAPIClient(conn)

	c.l.Info("Connected to dbaas-controller API.")
	return nil
}

// Disconnect disconnects the client from dbaas-controller API.
func (c *Client) Disconnect() error {
	c.connM.Lock()
	defer c.connM.Unlock()
	c.l.Info("Disconnecting from dbaas-controller API.")

	if c.conn == nil {
		c.l.Warnf("Trying to disconnect from dbaas-controller API but the connection is not up.")
		return nil
	}

	if err := c.conn.Close(); err != nil {
		return errors.Errorf("failed to close conn to dbaas-controller API: %v", err)
	}
	c.conn = nil
	c.l.Info("Disconected from dbaas-controller API.")
	return nil
}

// GetLogs gets logs out of cluster containers and events out of pods.
func (c *Client) GetLogs(ctx context.Context, in *controllerv1beta1.GetLogsRequest, opts ...grpc.CallOption) (*controllerv1beta1.GetLogsResponse, error) {
	c.connM.RLock()
	defer c.connM.RUnlock()
	return c.logsClient.GetLogs(ctx, in, opts...)
}

// GetResources returns all and available resources of a Kubernetes cluster.
func (c *Client) GetResources(ctx context.Context, in *controllerv1beta1.GetResourcesRequest, opts ...grpc.CallOption) (*controllerv1beta1.GetResourcesResponse, error) {
	c.connM.RLock()
	defer c.connM.RUnlock()
	return c.kubernetesClient.GetResources(ctx, in, opts...)
}

// InstallPXCOperator installs kubernetes pxc operator.
func (c *Client) InstallPXCOperator(ctx context.Context, in *controllerv1beta1.InstallPXCOperatorRequest, opts ...grpc.CallOption) (*controllerv1beta1.InstallPXCOperatorResponse, error) {
	c.connM.RLock()
	defer c.connM.RUnlock()
	return c.pxcOperatorClient.InstallPXCOperator(ctx, in, opts...)
}

// InstallPSMDBOperator installs kubernetes PSMDB operator.
func (c *Client) InstallPSMDBOperator(ctx context.Context, in *controllerv1beta1.InstallPSMDBOperatorRequest, opts ...grpc.CallOption) (*controllerv1beta1.InstallPSMDBOperatorResponse, error) {
	c.connM.RLock()
	defer c.connM.RUnlock()
	return c.psmdbOperatorClient.InstallPSMDBOperator(ctx, in, opts...)
}

// StartMonitoring sets up victoria metrics operator to monitor kubernetes cluster.
func (c *Client) StartMonitoring(ctx context.Context, in *controllerv1beta1.StartMonitoringRequest, opts ...grpc.CallOption) (*controllerv1beta1.StartMonitoringResponse, error) {
	c.connM.RLock()
	defer c.connM.RUnlock()
	return c.kubernetesClient.StartMonitoring(ctx, in, opts...)
}

// StopMonitoring removes victoria metrics operator from the kubernetes cluster.
func (c *Client) StopMonitoring(ctx context.Context, in *controllerv1beta1.StopMonitoringRequest, opts ...grpc.CallOption) (*controllerv1beta1.StopMonitoringResponse, error) {
	c.connM.RLock()
	defer c.connM.RUnlock()
	return c.kubernetesClient.StopMonitoring(ctx, in, opts...)
}

// GetKubeConfig returns Kubernetes config.
func (c *Client) GetKubeConfig(ctx context.Context, _ *controllerv1beta1.GetKubeconfigRequest, _ ...grpc.CallOption) (*controllerv1beta1.GetKubeconfigResponse, error) {
	c.connM.RLock()
	defer c.connM.RUnlock()

	kClient, err := kubernetes.NewIncluster()
	if err != nil {
		c.l.Errorf("failed creating kubernetes client: %v", err)
		return nil, err
	}

	kubeConfig, err := kClient.GetKubeconfig(ctx)
	return &controllerv1beta1.GetKubeconfigResponse{
		Kubeconfig: kubeConfig,
	}, err
}
