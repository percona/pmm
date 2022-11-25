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
	"encoding/json"
	"fmt"
	"os"

	dbaasv1 "github.com/percona/dbaas-operator/api/v1"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // load all auth plugins
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/percona/pmm/managed/services/dbaas/kubernetes/client/database"
)

const (
	configKind  = "Config"
	apiVersion  = "v1"
	defaultName = "default"

	dbaasToolPath = "/opt/dbaas-tools/bin"

	defaultQPSLimit   = 100
	defaultBurstLimit = 150

	defaultAPIURIPath  = "/api"
	defaultAPIsURIPath = "/apis"
)

// Client is the internal client for Kubernetes.
type Client struct {
	clientset       *kubernetes.Clientset
	dbClusterClient *database.DatabaseClusterClient
	restConfig      *rest.Config
	namespace       string
}

// NewFromIncluster returns a client object which uses the service account
// kubernetes gives to pods. It's intended for clients that expect to be
// running inside a pod running on kubernetes. It will return ErrNotInCluster
// if called from a process not running in a kubernetes environment.
func NewFromIncluster() (*Client, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	config.QPS = defaultQPSLimit
	config.Burst = defaultBurstLimit
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	c := &Client{
		clientset:  clientset,
		restConfig: config,
	}
	err = c.setup()
	return c, err
}

// NewFromKubeConfigString creates a new client for the given config string.
// It's intended for clients that expect to be running outside of a cluster
func NewFromKubeConfigString(kubeconfig string) (*Client, error) {
	config, err := clientcmd.BuildConfigFromKubeconfigGetter("", NewConfigGetter(kubeconfig).loadFromString)
	if err != nil {
		return nil, err
	}
	config.QPS = defaultQPSLimit
	config.Burst = defaultBurstLimit
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	c := &Client{
		clientset:  clientset,
		restConfig: config,
	}
	err = c.setup()
	return c, err
}

func (c *Client) setup() error {
	namespace := "default"
	if space := os.Getenv("NAMESPACE"); space != "" {
		namespace = space
	}
	// Set PATH variable to make aws-iam-authenticator executable
	path := fmt.Sprintf("%s:%s", os.Getenv("PATH"), dbaasToolPath)
	os.Setenv("PATH", path)
	c.namespace = namespace
	return c.initOperatorClients()
}

func (c *Client) initOperatorClients() error {
	dbClusterClient, err := database.NewForConfig(c.restConfig)
	if err != nil {
		return err
	}
	c.dbClusterClient = dbClusterClient
	_, err = c.GetServerVersion(context.Background())
	return err
}

// GetSecretsForServiceAccount returns secret by given service account name
func (c *Client) GetSecretsForServiceAccount(ctx context.Context, accountName string) (*corev1.Secret, error) {
	serviceAccount, err := c.clientset.CoreV1().ServiceAccounts(c.namespace).Get(ctx, accountName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	if len(serviceAccount.Secrets) == 0 {
		return nil, errors.Errorf("no secrets available for namespace %s", c.namespace)
	}

	return c.clientset.CoreV1().Secrets(c.namespace).Get(
		ctx,
		serviceAccount.Secrets[0].Name,
		metav1.GetOptions{})
}

// GenerateKubeConfig generates kubeconfig
func (c *Client) GenerateKubeConfig(secret *corev1.Secret) ([]byte, error) {
	conf := &Config{
		Kind:           configKind,
		APIVersion:     apiVersion,
		CurrentContext: defaultName,
	}
	conf.Clusters = []ClusterInfo{
		{
			Name: defaultName,
			Cluster: Cluster{
				CertificateAuthorityData: secret.Data["ca.crt"],
				Server:                   c.restConfig.Host,
			},
		},
	}
	conf.Contexts = []ContextInfo{
		{
			Name: defaultName,
			Context: Context{
				Cluster:   defaultName,
				User:      "pmm-service-account",
				Namespace: defaultName,
			},
		},
	}
	conf.Users = []UserInfo{
		{
			Name: "pmm-service-account",
			User: User{
				Token: string(secret.Data["token"]),
			},
		},
	}

	return c.marshalKubeConfig(conf)
}

func (c *Client) GetServerVersion(ctx context.Context) (*version.Info, error) {
	return c.clientset.Discovery().ServerVersion()
}

// ListDatabaseClusters returns list of managed PCX clusters.
func (c *Client) ListDatabaseClusters(ctx context.Context) (*dbaasv1.DatabaseClusterList, error) {
	return c.dbClusterClient.DBClusters(c.namespace).List(ctx, metav1.ListOptions{})
}

// GetDatabaseCluster returns PXC clusters by provided name.
func (c *Client) GetDatabaseCluster(ctx context.Context, name string) (*dbaasv1.DatabaseCluster, error) {
	cluster, err := c.dbClusterClient.DBClusters(c.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

// GetStorageClasses returns all storage classes available in the cluster
func (c *Client) GetStorageClasses(ctx context.Context) (*storagev1.StorageClassList, error) {
	return c.clientset.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
}

func (c *Client) GetDeployment(ctx context.Context, name string) (*appsv1.Deployment, error) {
	return c.clientset.AppsV1().Deployments(c.namespace).Get(ctx, name, metav1.GetOptions{})
}

func (c *Client) GetSecret(ctx context.Context, name string) (*corev1.Secret, error) {
	return c.clientset.CoreV1().Secrets(c.namespace).Get(ctx, name, metav1.GetOptions{})
}

// Delete deletes object from the k8s cluster
func (c *Client) DeleteObject(ctx context.Context, obj runtime.Object) error {
	groupResources, err := restmapper.GetAPIGroupResources(c.clientset.Discovery())
	if err != nil {
		return err
	}
	mapper := restmapper.NewDiscoveryRESTMapper(groupResources)

	gvk := obj.GetObjectKind().GroupVersionKind()
	gk := schema.GroupKind{Group: gvk.Group, Kind: gvk.Kind}
	mapping, err := mapper.RESTMapping(gk, gvk.Version)
	if err != nil {
		return err
	}
	namespace, name, err := c.retrieveMetaFromObject(obj)
	if err != nil {
		return err
	}
	cli, err := c.resourceClient(mapping.GroupVersionKind.GroupVersion())
	if err != nil {
		return err
	}
	helper := resource.NewHelper(cli, mapping)
	err = deleteObject(helper, namespace, name)
	return err
}

// TODO Refactor it
func deleteObject(helper *resource.Helper, namespace, name string) error {
	if _, err := helper.Get(namespace, name); err == nil {
		_, err = helper.Delete(namespace, name)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) ApplyObject(ctx context.Context, obj runtime.Object) error {
	groupResources, err := restmapper.GetAPIGroupResources(c.clientset.Discovery())
	if err != nil {
		return err
	}
	mapper := restmapper.NewDiscoveryRESTMapper(groupResources)

	gvk := obj.GetObjectKind().GroupVersionKind()
	gk := schema.GroupKind{Group: gvk.Group, Kind: gvk.Kind}
	mapping, err := mapper.RESTMapping(gk, gvk.Version)
	if err != nil {
		return err
	}
	namespace, name, err := c.retrieveMetaFromObject(obj)
	if err != nil {
		return err
	}
	cli, err := c.resourceClient(mapping.GroupVersionKind.GroupVersion())
	if err != nil {
		return err
	}
	helper := resource.NewHelper(cli, mapping)
	return c.applyObject(helper, namespace, name, obj)
}

func (c *Client) applyObject(helper *resource.Helper, namespace, name string, obj runtime.Object) error {
	if _, err := helper.Get(namespace, name); err != nil {
		_, err = helper.Create(namespace, false, obj)
		if err != nil {
			return err
		}
	} else {
		_, err = helper.Replace(namespace, name, true, obj)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) retrieveMetaFromObject(obj runtime.Object) (namespace, name string, err error) {
	name, err = meta.NewAccessor().Name(obj)
	if err != nil {
		return
	}
	namespace, err = meta.NewAccessor().Namespace(obj)
	if err != nil {
		return
	}
	if namespace == "" {
		namespace = c.namespace
	}
	return
}

func (c *Client) resourceClient(gv schema.GroupVersion) (rest.Interface, error) {
	cfg := c.restConfig
	cfg.ContentConfig = resource.UnstructuredPlusDefaultContentConfig()
	cfg.GroupVersion = &gv
	if len(gv.Group) == 0 {
		cfg.APIPath = defaultAPIURIPath
	} else {
		cfg.APIPath = defaultAPIsURIPath
	}
	return rest.RESTClientFor(cfg)
}

func (c *Client) marshalKubeConfig(conf *Config) ([]byte, error) {
	config, err := json.Marshal(&conf)
	if err != nil {
		return nil, err
	}

	var jsonObj interface{}
	err = yaml.Unmarshal(config, &jsonObj)
	if err != nil {
		return nil, err
	}

	return yaml.Marshal(jsonObj)
}
