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

package kubernetes

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	dbaasv1 "github.com/percona/dbaas-operator/api/v1"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"

	"github.com/percona/pmm/managed/services/dbaas/kubernetes/client"
)

const (
	pxcDeploymentName   = "percona-xtradb-cluster-operator"
	psmdbDeploymentName = "percona-server-mongodb-operator"
)

// Kubernetes is a client for Kubernetes.
type Kubernetes struct {
	client     *client.Client
	l          *logrus.Entry
	httpClient *http.Client
}

// NewIncluster returns new Kubernetes object.
func NewIncluster(ctx context.Context) (*Kubernetes, error) {
	l := logrus.WithField("component", "kubernetes")

	client, err := client.NewFromIncluster()
	if err != nil {
		return nil, err
	}

	return &Kubernetes{
		client: client,
		l:      l,
		httpClient: &http.Client{
			Timeout: time.Second * 5,
			Transport: &http.Transport{
				MaxIdleConns:    1,
				IdleConnTimeout: 10 * time.Second,
			},
		},
	}, nil
}

// New returns new Kubernetes object.
func New(ctx context.Context, kubeconfig string) (*Kubernetes, error) {
	l := logrus.WithField("component", "kubernetes")

	client, err := client.NewFromKubeConfigString(kubeconfig)
	if err != nil {
		return nil, err
	}

	return &Kubernetes{
		client: client,
		l:      l,
		httpClient: &http.Client{
			Timeout: time.Second * 5,
			Transport: &http.Transport{
				MaxIdleConns:    1,
				IdleConnTimeout: 10 * time.Second,
			},
		},
	}, nil
}

// New returns new Kubernetes object.
func NewEmpty() *Kubernetes {
	return &Kubernetes{
		client: &client.Client{},
		l:      logrus.WithField("component", "kubernetes"),
		httpClient: &http.Client{
			Timeout: time.Second * 5,
			Transport: &http.Transport{
				MaxIdleConns:    1,
				IdleConnTimeout: 10 * time.Second,
			},
		},
	}
}

// New returns new Kubernetes object.
func (k *Kubernetes) ChangeKubeconfig(ctx context.Context, kubeconfig string) error {
	client, err := client.NewFromKubeConfigString(kubeconfig)
	if err != nil {
		return err
	}
	k.client = client
	return nil
}

// GetKubeconfig generates kubeconfig compatible with kubectl for incluster created clients.
func (k *Kubernetes) GetKubeconfig(ctx context.Context) (string, error) {
	secret, err := k.client.GetSecretsForServiceAccount(ctx, "pmm-service-account")
	if err != nil {
		k.l.Errorf("failed getting service account: %v", err)
		return "", err
	}

	kubeConfig, err := k.client.GenerateKubeConfig(secret)
	if err != nil {
		k.l.Errorf("failed generating kubeconfig: %v", err)
		return "", err
	}

	return string(kubeConfig), nil
}

// ListDatabaseClusters returns list of managed PCX clusters.
func (c *Kubernetes) ListDatabaseClusters(ctx context.Context) (*dbaasv1.DatabaseClusterList, error) {
	return c.client.ListDatabaseClusters(ctx)
}

// GetDatabaseCluster returns PXC clusters by provided name.
func (c *Kubernetes) GetDatabaseCluster(ctx context.Context, name string) (*dbaasv1.DatabaseCluster, error) {
	return c.client.GetDatabaseCluster(ctx, name)
}

// PatchDatabaseCluster patches CR of managed PXC cluster.
func (c *Kubernetes) PatchDatabaseCluster(ctx context.Context, cluster *dbaasv1.DatabaseCluster) error {
	return c.client.ApplyObject(ctx, cluster)
}

func (c *Kubernetes) CreateDatabaseCluster(ctx context.Context, cluster *dbaasv1.DatabaseCluster) error {
	return c.client.ApplyObject(ctx, cluster)
}

func (c *Kubernetes) DeleteDatabaseCluster(ctx context.Context, cluster *dbaasv1.DatabaseCluster) error {
	return c.client.DeleteObject(ctx, cluster)
}

func (c *Kubernetes) GetDefaultStorageClassName(ctx context.Context) (string, error) {
	storageClasses, err := c.client.GetStorageClasses(ctx)
	if err != nil {
		return "", err
	}
	if len(storageClasses.Items) != 0 {
		return storageClasses.Items[0].Name, nil
	}
	return "", errors.New("no storage classes available")
}

func (c *Kubernetes) GetOperatorVersion(ctx context.Context, name string) (string, error) {
	deployment, err := c.client.GetDeployment(ctx, name)
	if err != nil {
		return "", err
	}
	return strings.Split(deployment.Spec.Template.Spec.Containers[0].Image, ":")[1], nil
}

func (c *Kubernetes) GetPSMDBOperatorVersion(ctx context.Context) (string, error) {
	return c.GetOperatorVersion(ctx, psmdbDeploymentName)
}

func (c *Kubernetes) GetPXCOperatorVersion(ctx context.Context) (string, error) {
	return c.GetOperatorVersion(ctx, pxcDeploymentName)
}

func (c *Kubernetes) GetSecret(ctx context.Context, name string) (*corev1.Secret, error) {
	return c.client.GetSecret(ctx, name)
}
