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

package client

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	fake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

func TestInCluster(t *testing.T) {
	t.Parallel()
	_, err := NewFromInCluster()
	require.Error(t, err)
}

func mockInClusterConfig() (*rest.Config, error) {
	return &rest.Config{}, nil
}

func mockInClusterConfigWithError() (*rest.Config, error) {
	return nil, errors.Errorf("mock error getting in-cluster config")
}

func mockNewForConfig(c *rest.Config) (kubernetes.Interface, error) {
	return fake.NewSimpleClientset(), nil
}

func mockNewForConfigWithError(c *rest.Config) (kubernetes.Interface, error) {
	return nil, errors.Errorf("mock error getting client set for config")
}

func TestNewFromInCluster(t *testing.T) {
	origInClusterConfig := inClusterConfig
	inClusterConfig = mockInClusterConfig
	origNewForConfig := newForConfig
	newForConfig = mockNewForConfig

	defer func() {
		inClusterConfig = origInClusterConfig
		newForConfig = origNewForConfig
	}()

	_, err := NewFromInCluster()
	require.Nil(t, err, "error is not nil")
}

func TestNewFromInCluster_ConfigError(t *testing.T) {
	origInClusterConfig := inClusterConfig
	inClusterConfig = mockInClusterConfigWithError

	defer func() {
		inClusterConfig = origInClusterConfig
	}()

	client, err := NewFromInCluster()
	require.Nil(t, client, "Client is not nil")
	require.NotNil(t, err, "error is nil")
}

func TestNewFromInCluster_ClientSetError(t *testing.T) {
	origInClusterConfig := inClusterConfig
	inClusterConfig = mockInClusterConfig
	origNewForConfig := newForConfig
	newForConfig = mockNewForConfigWithError

	defer func() {
		inClusterConfig = origInClusterConfig
		newForConfig = origNewForConfig
	}()

	client, err := NewFromInCluster()
	require.Nil(t, client, "Client is not nil")
	require.NotNil(t, err, "error is nil")
}

func TestGetSecretsForServiceAccount(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pmm-service-account",
				Namespace: "default",
			},
			Secrets: []corev1.ObjectReference{
				{
					Name: "pmm-service-account-token",
				},
				{
					Name: "pmm-service-account-token-ktgqd",
				},
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pmm-service-account-token",
				Namespace: "default",
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pmm-service-account-token-ktgqd",
				Namespace: "default",
			},
		})
	client := &Client{clientset: clientset, restConfig: nil, namespace: "default"}

	ctx := context.Background()
	secret, err := client.GetSecretsForServiceAccount(ctx, "pmm-service-account")
	require.NotNil(t, secret, "secret is nil")
	require.Nil(t, err, "error is not nil")
}

func TestGetSecretsForServiceAccountNoSecrets(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pmm-service-account",
				Namespace: "default",
			},
		})
	client := &Client{clientset: clientset, restConfig: nil, namespace: "default"}

	ctx := context.Background()
	secret, err := client.GetSecretsForServiceAccount(ctx, "pmm-service-account")
	require.Nil(t, secret, "secret is not nil")
	require.NotNil(t, err, "error is nil")
}
