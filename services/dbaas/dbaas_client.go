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

// Package dbaas contains logic related to communication with dbaas-controller.
package dbaas

import (
	"context"

	controllerv1beta1 "github.com/percona-platform/dbaas-api/gen/controller"
	"google.golang.org/grpc"
)

// Client is a client for dbaas-controller.
type Client struct {
	kubernetesClient    controllerv1beta1.KubernetesClusterAPIClient
	xtradbClusterClient controllerv1beta1.XtraDBClusterAPIClient
	psmdbClusterClient  controllerv1beta1.PSMDBClusterAPIClient
}

// NewClient creates new Client object.
func NewClient(con grpc.ClientConnInterface) *Client {
	return &Client{
		kubernetesClient:    controllerv1beta1.NewKubernetesClusterAPIClient(con),
		xtradbClusterClient: controllerv1beta1.NewXtraDBClusterAPIClient(con),
		psmdbClusterClient:  controllerv1beta1.NewPSMDBClusterAPIClient(con),
	}
}

// CheckKubernetesClusterConnection checks connection with kubernetes cluster.
func (c *Client) CheckKubernetesClusterConnection(ctx context.Context, kubeConfig string) error {
	_, err := c.kubernetesClient.CheckKubernetesClusterConnection(ctx, &controllerv1beta1.CheckKubernetesClusterConnectionRequest{
		KubeAuth: &controllerv1beta1.KubeAuth{Kubeconfig: kubeConfig},
	})
	return err
}

// ListXtraDBClusters returns a list of XtraDB clusters.
func (c *Client) ListXtraDBClusters(ctx context.Context, in *controllerv1beta1.ListXtraDBClustersRequest, opts ...grpc.CallOption) (*controllerv1beta1.ListXtraDBClustersResponse, error) {
	return c.xtradbClusterClient.ListXtraDBClusters(ctx, in, opts...)
}

// CreateXtraDBCluster creates a new XtraDB cluster.
func (c *Client) CreateXtraDBCluster(ctx context.Context, in *controllerv1beta1.CreateXtraDBClusterRequest, opts ...grpc.CallOption) (*controllerv1beta1.CreateXtraDBClusterResponse, error) {
	return c.xtradbClusterClient.CreateXtraDBCluster(ctx, in, opts...)
}

// UpdateXtraDBCluster updates existing XtraDB cluster.
func (c *Client) UpdateXtraDBCluster(ctx context.Context, in *controllerv1beta1.UpdateXtraDBClusterRequest, opts ...grpc.CallOption) (*controllerv1beta1.UpdateXtraDBClusterResponse, error) {
	return c.xtradbClusterClient.UpdateXtraDBCluster(ctx, in, opts...)
}

// DeleteXtraDBCluster deletes XtraDB cluster.
func (c *Client) DeleteXtraDBCluster(ctx context.Context, in *controllerv1beta1.DeleteXtraDBClusterRequest, opts ...grpc.CallOption) (*controllerv1beta1.DeleteXtraDBClusterResponse, error) {
	return c.xtradbClusterClient.DeleteXtraDBCluster(ctx, in, opts...)
}

// ListPSMDBClusters returns a list of PSMDB clusters.
func (c *Client) ListPSMDBClusters(ctx context.Context, in *controllerv1beta1.ListPSMDBClustersRequest, opts ...grpc.CallOption) (*controllerv1beta1.ListPSMDBClustersResponse, error) {
	return c.psmdbClusterClient.ListPSMDBClusters(ctx, in, opts...)
}

// CreatePSMDBCluster creates a new PSMDB cluster.
func (c *Client) CreatePSMDBCluster(ctx context.Context, in *controllerv1beta1.CreatePSMDBClusterRequest, opts ...grpc.CallOption) (*controllerv1beta1.CreatePSMDBClusterResponse, error) {
	return c.psmdbClusterClient.CreatePSMDBCluster(ctx, in, opts...)
}

// UpdatePSMDBCluster updates existing PSMDB cluster.
func (c *Client) UpdatePSMDBCluster(ctx context.Context, in *controllerv1beta1.UpdatePSMDBClusterRequest, opts ...grpc.CallOption) (*controllerv1beta1.UpdatePSMDBClusterResponse, error) {
	return c.psmdbClusterClient.UpdatePSMDBCluster(ctx, in, opts...)
}

// DeletePSMDBCluster deletes PSMDB cluster.
func (c *Client) DeletePSMDBCluster(ctx context.Context, in *controllerv1beta1.DeletePSMDBClusterRequest, opts ...grpc.CallOption) (*controllerv1beta1.DeletePSMDBClusterResponse, error) {
	return c.psmdbClusterClient.DeletePSMDBCluster(ctx, in, opts...)
}
