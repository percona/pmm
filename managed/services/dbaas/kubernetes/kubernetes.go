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
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	dbaasv1 "github.com/percona/dbaas-operator/api/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"

	"github.com/percona/pmm/managed/services/dbaas/kubernetes/client"
)

const (
	pxcDeploymentName         = "percona-xtradb-cluster-operator"
	psmdbDeploymentName       = "percona-server-mongodb-operator"
	databaseClusterKind       = "DatabaseCluster"
	databaseClusterAPIVersion = "dbaas.percona.com/v1"

	// ContainerStateWaiting represents a state when container requires some
	// operations being done in order to complete start up.
	ContainerStateWaiting ContainerState = "waiting"
	// ContainerStateTerminated indicates that container began execution and
	// then either ran to completion or failed for some reason.
	ContainerStateTerminated ContainerState = "terminated"
)

// Kubernetes is a client for Kubernetes.
type Kubernetes struct {
	lock       sync.RWMutex
	client     *client.Client
	l          *logrus.Entry
	httpClient *http.Client
}

// ContainerState describes container's state - waiting, running, terminated.
type ContainerState string

// NewIncluster returns new Kubernetes object.
func NewIncluster() (*Kubernetes, error) {
	l := logrus.WithField("component", "kubernetes")

	client, err := client.NewFromInCluster()
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

// NewEmpty returns new Kubernetes object.
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

// SetKubeconfig changes kubeconfig for active client
func (k *Kubernetes) SetKubeconfig(ctx context.Context, kubeconfig string) error {
	k.lock.Lock()
	defer k.lock.Unlock()
	client, err := client.NewFromKubeConfigString(kubeconfig)
	if err != nil {
		return err
	}
	k.client = client
	return nil
}

// GetKubeconfig generates kubeconfig compatible with kubectl for incluster created clients.
func (k *Kubernetes) GetKubeconfig(ctx context.Context) (string, error) {
	k.lock.RLock()
	defer k.lock.RUnlock()
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
func (k *Kubernetes) ListDatabaseClusters(ctx context.Context) (*dbaasv1.DatabaseClusterList, error) {
	k.lock.RLock()
	defer k.lock.RUnlock()
	return k.client.ListDatabaseClusters(ctx)
}

// GetDatabaseCluster returns PXC clusters by provided name.
func (k *Kubernetes) GetDatabaseCluster(ctx context.Context, name string) (*dbaasv1.DatabaseCluster, error) {
	k.lock.RLock()
	defer k.lock.RUnlock()
	return k.client.GetDatabaseCluster(ctx, name)
}

// RestartDatabaseCluster restarts database cluster
func (k *Kubernetes) RestartDatabaseCluster(ctx context.Context, name string) error {
	k.lock.Lock()
	defer k.lock.Unlock()
	cluster, err := k.client.GetDatabaseCluster(ctx, name)
	if err != nil {
		return err
	}
	cluster.TypeMeta.APIVersion = databaseClusterAPIVersion
	cluster.TypeMeta.Kind = databaseClusterKind
	cluster.Spec.Restart = true
	return k.client.ApplyObject(ctx, cluster)
}

// PatchDatabaseCluster patches CR of managed Database cluster.
func (k *Kubernetes) PatchDatabaseCluster(ctx context.Context, cluster *dbaasv1.DatabaseCluster) error {
	k.lock.Lock()
	defer k.lock.Unlock()
	return k.client.ApplyObject(ctx, cluster)
}

// CreateDatabase cluster creates database cluster
func (k *Kubernetes) CreateDatabaseCluster(ctx context.Context, cluster *dbaasv1.DatabaseCluster) error {
	k.lock.Lock()
	defer k.lock.Unlock()
	return k.client.ApplyObject(ctx, cluster)
}

// DeleteDatabaseCluster deletes database cluster
func (k *Kubernetes) DeleteDatabaseCluster(ctx context.Context, name string) error {
	k.lock.Lock()
	defer k.lock.Unlock()
	cluster, err := k.client.GetDatabaseCluster(ctx, name)
	if err != nil {
		return err
	}
	cluster.TypeMeta.APIVersion = databaseClusterAPIVersion
	cluster.TypeMeta.Kind = databaseClusterKind
	return k.client.DeleteObject(ctx, cluster)
}

// GetDefaultStorageClassName returns first storageClassName from kubernetes cluster
func (k *Kubernetes) GetDefaultStorageClassName(ctx context.Context) (string, error) {
	k.lock.RLock()
	defer k.lock.RUnlock()
	storageClasses, err := k.client.GetStorageClasses(ctx)
	if err != nil {
		return "", err
	}
	if len(storageClasses.Items) != 0 {
		return storageClasses.Items[0].Name, nil
	}
	return "", errors.New("no storage classes available")
}

// GetOperatorVersion parses operator version from operator deployment
func (k *Kubernetes) GetOperatorVersion(ctx context.Context, name string) (string, error) {
	k.lock.RLock()
	defer k.lock.RUnlock()
	deployment, err := k.client.GetDeployment(ctx, name)
	if err != nil {
		return "", err
	}
	return strings.Split(deployment.Spec.Template.Spec.Containers[0].Image, ":")[1], nil
}

// GetPSMDBOperatorVersion parses PSMDB operator version from operator deployment
func (k *Kubernetes) GetPSMDBOperatorVersion(ctx context.Context) (string, error) {
	k.lock.RLock()
	defer k.lock.RUnlock()
	return k.GetOperatorVersion(ctx, psmdbDeploymentName)
}

// GetPXCOperatorVersion parses PXC operator version from operator deployment
func (k *Kubernetes) GetPXCOperatorVersion(ctx context.Context) (string, error) {
	k.lock.RLock()
	defer k.lock.RUnlock()
	return k.GetOperatorVersion(ctx, pxcDeploymentName)
}

// GetSecret returns secret by name
func (k *Kubernetes) GetSecret(ctx context.Context, name string) (*corev1.Secret, error) {
	k.lock.RLock()
	defer k.lock.RUnlock()
	return k.client.GetSecret(ctx, name)
}

// GetPods returns list of pods based on given filters. Filters are args to
// kubectl command. For example "-lyour-label=value,next-label=value", "-ntest-namespace".
func (k *Kubernetes) GetPods(ctx context.Context, namespace string, filters ...string) (*corev1.PodList, error) {
	podList, err := k.client.GetPods(ctx, namespace, strings.Join(filters, ""))
	return podList, err
}

// GetLogs returns logs as slice of log lines - strings - for given pod's container.
func (k *Kubernetes) GetLogs(
	ctx context.Context,
	containerStatuses []corev1.ContainerStatus,
	pod,
	container string,
) ([]string, error) {
	if IsContainerInState(containerStatuses, ContainerStateWaiting) {
		return []string{}, nil
	}

	stdout, err := k.client.GetLogs(ctx, pod, container)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get logs")
	}

	if stdout == "" {
		return []string{}, nil
	}

	return strings.Split(stdout, "\n"), nil
}

// GetEvents returns pod's events as a slice of strings.
func (k *Kubernetes) GetEvents(ctx context.Context, pod string) ([]string, error) {
	stdout, err := k.client.GetEvents(ctx, pod)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't describe pod")
	}

	lines := strings.Split(stdout, "\n")

	return lines, nil
}

// IsContainerInState returns true if container is in give state, otherwise false.
func IsContainerInState(containerStatuses []corev1.ContainerStatus, state ContainerState) bool {
	containerState := make(map[string]interface{})
	for _, status := range containerStatuses {
		data, err := json.Marshal(status.State)
		if err != nil {
			return false
		}

		if err := json.Unmarshal(data, &containerState); err != nil {
			return false
		}

		if _, ok := containerState[string(state)]; ok {
			return true
		}
	}

	return false
}
