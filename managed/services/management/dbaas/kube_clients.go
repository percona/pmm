package dbaas

import (
	"sync"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/dbaas/kubernetes"
	"gopkg.in/reform.v1"
)

type KubeStorage struct {
	mu      sync.Mutex
	db      *reform.DB
	clients map[string]kubernetesClient
}

func NewKubeStorage(db *reform.DB) *KubeStorage {
	return &KubeStorage{
		db:      db,
		clients: make(map[string]kubernetesClient),
	}
}

func (k *KubeStorage) GetOrSetClient(name string) (kubernetesClient, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	kubeClient, ok := k.clients[name]
	if ok {
		return kubeClient, nil
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

func (k *KubeStorage) DeleteClient(name string) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	delete(k.clients, name)
	return models.RemoveKubernetesCluster(k.db.Querier, name)
}
