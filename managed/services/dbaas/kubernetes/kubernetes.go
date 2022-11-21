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
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/managed/services/dbaas/kubernetes/client"
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
