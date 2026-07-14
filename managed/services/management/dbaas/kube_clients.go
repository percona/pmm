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
	"sync"

	"github.com/pkg/errors"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/dbaas/kubernetes"
)

// KubeStorage stores kuberenetes clients for DBaaS.
type KubeStorage struct {
	mu      sync.Mutex
	db      *reform.DB
	clients map[string]kubernetesClient
}

var errDatabaseNotSet = errors.New("Database connection not set")

// NewKubeStorage returns a created KubeStorage.
func NewKubeStorage(db *reform.DB) *KubeStorage {
	return &KubeStorage{
		db:      db,
		clients: make(map[string]kubernetesClient),
	}
}

// GetOrSetClient gets client from map or sets a new client to the map.
func (k *KubeStorage) GetOrSetClient(name string) (kubernetesClient, error) { //nolint:ireturn,nolintlint
	k.mu.Lock()
	defer k.mu.Unlock()
	kubeClient, ok := k.clients[name]
	if ok {
		_, err := kubeClient.GetServerVersion()
		return kubeClient, err
	}

	if k.db == nil {
		return nil, errDatabaseNotSet
	}

	kubernetesCluster, err := models.FindKubernetesClusterByName(k.db.Querier, name)
	if err != nil {
		return nil, err
	}
	kubeClient, err = kubernetes.New(kubernetesCluster.KubeConfig)
	if err != nil {
		return nil, err
	}
	k.clients[name] = kubeClient
	return kubeClient, nil
}

// DeleteClient deletes client from storage.
func (k *KubeStorage) DeleteClient(name string) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	delete(k.clients, name)
	return models.RemoveKubernetesCluster(k.db.Querier, name)
}
