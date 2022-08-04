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

// Package dbaas contains logic related to communication with dbaas-controller.
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

// CheckKubernetesClusterConnection checks connection with kubernetes cluster.
func (c *Client) CheckKubernetesClusterConnection(ctx context.Context, kubeConfig string) (*controllerv1beta1.CheckKubernetesClusterConnectionResponse, error) {
	c.connM.RLock()
	defer c.connM.RUnlock()
	in := &controllerv1beta1.CheckKubernetesClusterConnectionRequest{
		KubeAuth: &controllerv1beta1.KubeAuth{
			Kubeconfig: kubeConfig,
		},
	}
	return c.kubernetesClient.CheckKubernetesClusterConnection(ctx, in)
}

func (c *Client) ListPXCClusters(ctx context.Context, in *controllerv1beta1.ListPXCClustersRequest, opts ...grpc.CallOption) (*controllerv1beta1.ListPXCClustersResponse, error) {
	c.connM.RLock()
	defer c.connM.RUnlock()
	return c.pxcClusterClient.ListPXCClusters(ctx, in, opts...)
}

// GetPXCCluster gets a PXC cluster.
func (c *Client) GetPXCCluster(ctx context.Context, kubeconfig, name string) (*controllerv1beta1.GetPXCClusterResponse, error) {
	c.connM.RLock()
	defer c.connM.RUnlock()
	in := &controllerv1beta1.GetPXCClusterRequest{
		KubeAuth: &controllerv1beta1.KubeAuth{
			Kubeconfig: kubeconfig,
		},
		Name: name,
	}
	return c.pxcClusterClient.GetPXCCluster(ctx, in)
}

// CreatePXCCluster creates a new PXC cluster.
func (c *Client) CreatePXCCluster(ctx context.Context, in *controllerv1beta1.CreatePXCClusterRequest, opts ...grpc.CallOption) (*controllerv1beta1.CreatePXCClusterResponse, error) {
	c.connM.RLock()
	defer c.connM.RUnlock()
	return c.pxcClusterClient.CreatePXCCluster(ctx, in, opts...)
}

// UpdatePXCCluster updates existing PXC cluster.
func (c *Client) UpdatePXCCluster(ctx context.Context, in *controllerv1beta1.UpdatePXCClusterRequest, opts ...grpc.CallOption) (*controllerv1beta1.UpdatePXCClusterResponse, error) {
	c.connM.RLock()
	defer c.connM.RUnlock()
	return c.pxcClusterClient.UpdatePXCCluster(ctx, in, opts...)
}

// DeletePXCCluster deletes PXC cluster.
func (c *Client) DeletePXCCluster(ctx context.Context, in *controllerv1beta1.DeletePXCClusterRequest, opts ...grpc.CallOption) (*controllerv1beta1.DeletePXCClusterResponse, error) {
	c.connM.RLock()
	defer c.connM.RUnlock()
	return c.pxcClusterClient.DeletePXCCluster(ctx, in, opts...)
}

// RestartPXCCluster restarts PXC cluster.
func (c *Client) RestartPXCCluster(ctx context.Context, in *controllerv1beta1.RestartPXCClusterRequest, opts ...grpc.CallOption) (*controllerv1beta1.RestartPXCClusterResponse, error) {
	c.connM.RLock()
	defer c.connM.RUnlock()
	return c.pxcClusterClient.RestartPXCCluster(ctx, in, opts...)
}

// GetPXCClusterCredentials gets PXC cluster credentials.
func (c *Client) GetPXCClusterCredentials(ctx context.Context, in *controllerv1beta1.GetPXCClusterCredentialsRequest, opts ...grpc.CallOption) (*controllerv1beta1.GetPXCClusterCredentialsResponse, error) {
	c.connM.RLock()
	defer c.connM.RUnlock()
	return c.pxcClusterClient.GetPXCClusterCredentials(ctx, in, opts...)
}

// ListPSMDBClusters returns a list of PSMDB clusters.
func (c *Client) ListPSMDBClusters(ctx context.Context, in *controllerv1beta1.ListPSMDBClustersRequest, opts ...grpc.CallOption) (*controllerv1beta1.ListPSMDBClustersResponse, error) {
	c.connM.RLock()
	defer c.connM.RUnlock()
	return c.psmdbClusterClient.ListPSMDBClusters(ctx, in, opts...)
}

// CreatePSMDBCluster creates a new PSMDB cluster.
func (c *Client) CreatePSMDBCluster(ctx context.Context, in *controllerv1beta1.CreatePSMDBClusterRequest, opts ...grpc.CallOption) (*controllerv1beta1.CreatePSMDBClusterResponse, error) {
	c.connM.RLock()
	defer c.connM.RUnlock()
	return c.psmdbClusterClient.CreatePSMDBCluster(ctx, in, opts...)
}

// UpdatePSMDBCluster updates existing PSMDB cluster.
func (c *Client) UpdatePSMDBCluster(ctx context.Context, in *controllerv1beta1.UpdatePSMDBClusterRequest, opts ...grpc.CallOption) (*controllerv1beta1.UpdatePSMDBClusterResponse, error) {
	c.connM.RLock()
	defer c.connM.RUnlock()
	return c.psmdbClusterClient.UpdatePSMDBCluster(ctx, in, opts...)
}

// DeletePSMDBCluster deletes PSMDB cluster.
func (c *Client) DeletePSMDBCluster(ctx context.Context, in *controllerv1beta1.DeletePSMDBClusterRequest, opts ...grpc.CallOption) (*controllerv1beta1.DeletePSMDBClusterResponse, error) {
	c.connM.RLock()
	defer c.connM.RUnlock()
	return c.psmdbClusterClient.DeletePSMDBCluster(ctx, in, opts...)
}

// RestartPSMDBCluster restarts PSMDB cluster.
func (c *Client) RestartPSMDBCluster(ctx context.Context, in *controllerv1beta1.RestartPSMDBClusterRequest, opts ...grpc.CallOption) (*controllerv1beta1.RestartPSMDBClusterResponse, error) {
	c.connM.RLock()
	defer c.connM.RUnlock()
	return c.psmdbClusterClient.RestartPSMDBCluster(ctx, in, opts...)
}

// GetPSMDBCluster gets a PSMDB cluster.
func (c *Client) GetPSMDBCluster(ctx context.Context, kubeconfig, name string) (*controllerv1beta1.GetPSMDBClusterResponse, error) {
	c.connM.RLock()
	defer c.connM.RUnlock()
	in := &controllerv1beta1.GetPSMDBClusterRequest{
		KubeAuth: &controllerv1beta1.KubeAuth{
			Kubeconfig: kubeconfig,
		},
		Name: name,
	}
	return c.psmdbClusterClient.GetPSMDBCluster(ctx, in)
}

// GetPSMDBClusterCredentials gets PSMDB cluster credentials.
func (c *Client) GetPSMDBClusterCredentials(ctx context.Context, in *controllerv1beta1.GetPSMDBClusterCredentialsRequest, opts ...grpc.CallOption) (*controllerv1beta1.GetPSMDBClusterCredentialsResponse, error) {
	c.connM.RLock()
	defer c.connM.RUnlock()
	return c.psmdbClusterClient.GetPSMDBClusterCredentials(ctx, in, opts...)
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
