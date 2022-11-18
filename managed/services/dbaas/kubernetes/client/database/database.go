package database

import (
	"context"
	"sync"

	dbaasv1 "github.com/percona/dbaas-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

const (
	DBClusterKind = "DatabaseCluster"
	apiKind       = "databaseclusters"
)

type DatabaseClusterClientInterface interface {
	DBClusters(namespace string) DatabaseClusterInterface
}

type DatabaseClusterClient struct {
	restClient rest.Interface
}

var addToScheme sync.Once

func NewForConfig(c *rest.Config) (*DatabaseClusterClient, error) {
	config := *c
	config.ContentConfig.GroupVersion = &dbaasv1.GroupVersion
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	config.UserAgent = rest.DefaultKubernetesUserAgent()

	addToScheme.Do(func() {
		dbaasv1.SchemeBuilder.AddToScheme(scheme.Scheme)
		metav1.AddToGroupVersion(scheme.Scheme, dbaasv1.GroupVersion)
	})

	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}

	return &DatabaseClusterClient{restClient: client}, nil
}

func (c *DatabaseClusterClient) DBClusters(namespace string) DatabaseClusterInterface {
	return &dbClusterClient{
		restClient: c.restClient,
		namespace:  namespace,
	}
}

type DatabaseClusterInterface interface {
	List(ctx context.Context, opts metav1.ListOptions) (*dbaasv1.DatabaseClusterList, error)
	Get(ctx context.Context, name string, options metav1.GetOptions) (*dbaasv1.DatabaseCluster, error)
	Patch(context.Context, string, types.PatchType, []byte, metav1.PatchOptions) (*dbaasv1.DatabaseCluster, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
}

type dbClusterClient struct {
	restClient rest.Interface
	namespace  string
}

func (c *dbClusterClient) List(ctx context.Context, opts metav1.ListOptions) (*dbaasv1.DatabaseClusterList, error) {
	result := new(dbaasv1.DatabaseClusterList)
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
	result := new(dbaasv1.DatabaseCluster)
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

func (c *dbClusterClient) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions) (*dbaasv1.DatabaseCluster, error) {
	result := new(dbaasv1.DatabaseCluster)
	err := c.restClient.
		Patch(pt).
		Namespace(c.namespace).
		Resource(apiKind).
		Name(name).
		Body(data).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return result, err
}

func (c *dbClusterClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.restClient.
		Get().
		Namespace(c.namespace).
		Resource(apiKind).
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch(ctx)
}
