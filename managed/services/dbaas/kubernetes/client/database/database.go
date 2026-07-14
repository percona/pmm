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

// Package database TODO.
package database

import (
	"context"
	"sync"

	dbaasv1 "github.com/percona/dbaas-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

const (
	apiKind = "databaseclusters"
)

// ClusterClient contains client for database cluster.
type ClusterClient struct {
	restClient rest.Interface
}

var addToScheme sync.Once

// NewForConfig create database cluster client from given config.
func NewForConfig(c *rest.Config) (*ClusterClient, error) {
	config := *c
	config.ContentConfig.GroupVersion = &dbaasv1.GroupVersion
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	config.UserAgent = rest.DefaultKubernetesUserAgent()

	addToScheme.Do(func() {
		dbaasv1.SchemeBuilder.AddToScheme(scheme.Scheme) //nolint:errcheck
		metav1.AddToGroupVersion(scheme.Scheme, dbaasv1.GroupVersion)
	})

	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}

	return &ClusterClient{restClient: client}, nil
}

// DBClusters returns database cluster client.
func (c *ClusterClient) DBClusters(namespace string) databaseClusterInterface { //nolint:ireturn
	return &dbClusterClient{
		restClient: c.restClient,
		namespace:  namespace,
	}
}

type databaseClusterInterface interface {
	List(ctx context.Context, opts metav1.ListOptions) (*dbaasv1.DatabaseClusterList, error)
	Get(ctx context.Context, name string, options metav1.GetOptions) (*dbaasv1.DatabaseCluster, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
}

type dbClusterClient struct {
	restClient rest.Interface
	namespace  string
}

func (c *dbClusterClient) List(ctx context.Context, opts metav1.ListOptions) (*dbaasv1.DatabaseClusterList, error) {
	result := &dbaasv1.DatabaseClusterList{}
	err := c.restClient.
		Get().
		Namespace(c.namespace).
		Resource(apiKind).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return result, err
}

func (c *dbClusterClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*dbaasv1.DatabaseCluster, error) {
	result := &dbaasv1.DatabaseCluster{}
	err := c.restClient.
		Get().
		Namespace(c.namespace).
		Resource(apiKind).
		VersionedParams(&opts, scheme.ParameterCodec).
		Name(name).
		Do(ctx).
		Into(result)
	return result, err
}

func (c *dbClusterClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) { //nolint:ireturn
	opts.Watch = true
	return c.restClient.
		Get().
		Namespace(c.namespace).
		Resource(apiKind).
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch(ctx)
}
