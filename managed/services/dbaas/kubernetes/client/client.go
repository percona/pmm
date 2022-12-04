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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	operatorsv1 "github.com/operator-framework/api/pkg/operators/v1"
	v1 "github.com/operator-framework/api/pkg/operators/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned"
	dbaasv1 "github.com/percona/dbaas-operator/api/v1"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	yamlSerializer "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // load all auth plugins
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	deploymentutil "k8s.io/kubectl/pkg/util/deployment"
	"sigs.k8s.io/controller-runtime/pkg/client"
	kubeClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

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

var (
	inClusterConfig = rest.InClusterConfig
	newForConfig    = func(c *rest.Config) (kubernetes.Interface, error) {
		return kubernetes.NewForConfig(c)
	}
)

// Client is the internal client for Kubernetes.
type Client struct {
	clientset       kubernetes.Interface
	dbClusterClient *database.DatabaseClusterClient
	restConfig      *rest.Config
	namespace       string
}

type resourceError struct {
	name  string
	issue string
}

type podError struct {
	resourceError
}

type deploymentError struct {
	resourceError
	podErrs podErrors
}

type (
	deploymentErrors []deploymentError
	podErrors        []podError
)

func (e deploymentErrors) Error() string {
	var sb strings.Builder
	for _, i := range e {
		sb.WriteString(fmt.Sprintf("deployment %s has error: %s\n%s", i.name, i.issue, i.podErrs.Error()))
	}
	return sb.String()
}

func (e podErrors) Error() string {
	var sb strings.Builder
	for _, i := range e {
		sb.WriteString(fmt.Sprintf("\tpod %s has error: %s\n", i.name, i.issue))
	}
	return sb.String()
}

// NewFromInCluster returns a client object which uses the service account
// kubernetes gives to pods. It's intended for clients that expect to be
// running inside a pod running on kubernetes. It will return ErrNotInCluster
// if called from a process not running in a kubernetes environment.
func NewFromInCluster() (*Client, error) {
	config, err := inClusterConfig()
	if err != nil {
		return nil, err
	}
	config.QPS = defaultQPSLimit
	config.Burst = defaultBurstLimit
	clientset, err := newForConfig(config)
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
	clientset, err := newForConfig(config)
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

// Initializes clients for operators
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

// GetServerVersion returns server version
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

// GetDeployment returns deployment by name
func (c *Client) GetDeployment(ctx context.Context, name string) (*appsv1.Deployment, error) {
	return c.clientset.AppsV1().Deployments(c.namespace).Get(ctx, name, metav1.GetOptions{})
}

// GetSecret returns secret by name
func (c *Client) GetSecret(ctx context.Context, name string) (*corev1.Secret, error) {
	return c.clientset.CoreV1().Secrets(c.namespace).Get(ctx, name, metav1.GetOptions{})
}

// // NamespaceExists checks if the specified namespace exists.
// func (c *Client) NamespaceExists(ctx context.Context, namespace string) bool {
// 	_, err := c.clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
// 	if err == nil {
// 		return true
// 	}
//
// 	obj := client.Object{}
//
// 	return false
// }

func (c *Client) GetOperatorGroup(ctx context.Context, namespace, name string) (*v1.OperatorGroup, error) {
	operatorClient, err := versioned.NewForConfig(c.restConfig)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create an operator client instance")
	}

	if namespace == "" {
		namespace = c.namespace
	}

	return operatorClient.OperatorsV1().OperatorGroups(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (c *Client) CreateOperatorGroup(ctx context.Context, namespace, name string) (*v1.OperatorGroup, error) {
	operatorClient, err := versioned.NewForConfig(c.restConfig)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create an operator client instance")
	}

	if namespace == "" {
		namespace = c.namespace
	}
	og := &operatorsv1.OperatorGroup{}
	og.SetName(name)

	return operatorClient.OperatorsV1().OperatorGroups(namespace).Create(ctx, og, metav1.CreateOptions{})
}

/*

	err = wait.Poll(pollInterval, pollDuration, func() (bool, error) {
		fetchedInstallPlan, err = c.OperatorsV1alpha1().InstallPlans(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil || fetchedInstallPlan == nil {
			return false, err
		}

		return checkPhase(fetchedInstallPlan), nil
	})

		// Approve the upgrade installplan and wait for
		upgradeInstallPlan.Spec.Approved = true
		_, err = crc.OperatorsV1alpha1().InstallPlans(generatedNamespace.GetName()).Update(context.Background(), upgradeInstallPlan, metav1.UpdateOptions{})
		require.NoError(GinkgoT(), err)
*/

func (c *Client) CreateSubscriptionForCatalog(ctx context.Context, namespace, name, catalogNamespace, catalog,
	packageName, channel, startingCSV string, approval operatorsv1alpha1.Approval) (*v1alpha1.Subscription, error) {
	operatorClient, err := versioned.NewForConfig(c.restConfig)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create an operator client instance")
	}

	subscription := &operatorsv1alpha1.Subscription{
		TypeMeta: metav1.TypeMeta{
			Kind:       operatorsv1alpha1.SubscriptionKind,
			APIVersion: operatorsv1alpha1.SubscriptionCRDAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: &operatorsv1alpha1.SubscriptionSpec{
			CatalogSource:          catalog,
			CatalogSourceNamespace: catalogNamespace,
			Package:                packageName,
			Channel:                channel,
			StartingCSV:            startingCSV,
			InstallPlanApproval:    approval,
		},
	}

	return operatorClient.OperatorsV1alpha1().Subscriptions(namespace).Create(ctx, subscription, metav1.CreateOptions{})
}

func (c *Client) GetSubscription(ctx context.Context, namespace, name string) (*v1alpha1.Subscription, error) {
	operatorClient, err := versioned.NewForConfig(c.restConfig)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create an operator client instance")
	}

	return operatorClient.OperatorsV1alpha1().Subscriptions(namespace).Get(ctx, name, metav1.GetOptions{})
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

// ApplyFile accepts manifest file contents, parses into []runtime.Object
// and applies them against the cluster
func (c *Client) ApplyFile(ctx context.Context, fileBytes []byte) error {
	objs, err := c.getObjects(fileBytes)
	if err != nil {
		return err
	}
	for i := range objs {
		err := c.ApplyObject(ctx, objs[i])
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) getObjects(f []byte) ([]runtime.Object, error) {
	objs := []runtime.Object{}
	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(f), 100)
	var err error
	for {
		var rawObj runtime.RawExtension
		if err = decoder.Decode(&rawObj); err != nil {
			break
		}

		obj, _, err := yamlSerializer.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObj.Raw, nil, nil)
		if err != nil {
			return nil, err
		}

		unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return nil, err
		}

		objs = append(objs, &unstructured.Unstructured{Object: unstructuredMap})
	}

	return objs, nil
}

func (c Client) DoCSVWait(ctx context.Context, key types.NamespacedName) error {
	var (
		curPhase operatorsv1alpha1.ClusterServiceVersionPhase
		newPhase operatorsv1alpha1.ClusterServiceVersionPhase
	)

	kubeclient, err := c.getKubeclient()
	if err != nil {
		return err
	}

	csv := operatorsv1alpha1.ClusterServiceVersion{}
	csvPhaseSucceeded := func() (bool, error) {
		err := kubeclient.Get(ctx, key, &csv)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		newPhase = csv.Status.Phase
		if newPhase != curPhase {
			curPhase = newPhase
		}

		switch curPhase {
		case operatorsv1alpha1.CSVPhaseFailed:
			return false, fmt.Errorf("csv failed: reason: %q, message: %q", csv.Status.Reason, csv.Status.Message)
		case operatorsv1alpha1.CSVPhaseSucceeded:
			return true, nil
		default:
			return false, nil
		}
	}

	err = wait.PollImmediateUntil(time.Second, csvPhaseSucceeded, ctx.Done())
	if err != nil && errors.Is(err, context.DeadlineExceeded) {
		depCheckErr := c.checkDeploymentErrors(ctx, key, csv)
		if depCheckErr != nil {
			return depCheckErr
		}
	}
	return err
}

func (c Client) GetSubscriptionCSV(ctx context.Context, subKey types.NamespacedName) (types.NamespacedName, error) {
	var csvKey types.NamespacedName

	kubeclient, err := c.getKubeclient()
	if err != nil {
		return csvKey, err
	}

	subscriptionInstalledCSV := func() (bool, error) {
		sub := operatorsv1alpha1.Subscription{}
		err := kubeclient.Get(ctx, subKey, &sub)
		if err != nil {
			return false, err
		}
		installedCSV := sub.Status.InstalledCSV
		if installedCSV == "" {
			return false, nil
		}
		csvKey = types.NamespacedName{
			Namespace: subKey.Namespace,
			Name:      installedCSV,
		}
		log.Printf("  Found installed CSV %q", installedCSV)
		return true, nil
	}
	return csvKey, wait.PollImmediateUntil(time.Second, subscriptionInstalledCSV, ctx.Done())
}

func (c *Client) getKubeclient() (kubeClient.Client, error) {
	rm, err := apiutil.NewDynamicRESTMapper(c.restConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create dynamic rest mapper")
	}

	cl, err := kubeClient.New(c.restConfig, client.Options{
		Scheme: scheme.Scheme,
		Mapper: rm,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create client")
	}

	return cl, nil
}

// checkDeploymentErrors function loops through deployment specs of a given CSV, and prints reason
// in case of failures, based on deployment condition.
func (c Client) checkDeploymentErrors(ctx context.Context, key types.NamespacedName, csv operatorsv1alpha1.ClusterServiceVersion) error {
	depErrs := deploymentErrors{}
	if key.Namespace == "" {
		return fmt.Errorf("no namespace provided to get deployment failures")
	}

	kubeclient, err := c.getKubeclient()
	if err != nil {
		return err
	}

	dep := &appsv1.Deployment{}
	for _, ds := range csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs {
		depKey := types.NamespacedName{
			Namespace: key.Namespace,
			Name:      ds.Name,
		}
		depSelectors := ds.Spec.Selector
		if err := kubeclient.Get(ctx, depKey, dep); err != nil {
			depErrs = append(depErrs, deploymentError{
				resourceError: resourceError{
					name:  ds.Name,
					issue: err.Error(),
				},
			})
			continue
		}
		for _, s := range dep.Status.Conditions {
			if s.Type == appsv1.DeploymentAvailable && s.Status != corev1.ConditionTrue {
				depErr := deploymentError{
					resourceError: resourceError{
						name:  ds.Name,
						issue: s.Reason,
					},
				}
				podErr := c.checkPodErrors(ctx, kubeclient, depSelectors, key)
				podErrs := podErrors{}
				if errors.As(podErr, &podErrs) {
					depErr.podErrs = append(depErr.podErrs, podErrs...)
				} else {
					return podErr
				}
				depErrs = append(depErrs, depErr)
			}
		}
	}

	return depErrs
}

// checkPodErrors loops through pods, and returns pod errors if any.
func (c Client) checkPodErrors(ctx context.Context, kubeclient kubeClient.Client, depSelectors *metav1.LabelSelector, key types.NamespacedName) error {
	// loop through pods and return specific error message.
	podErr := podErrors{}
	podList := &corev1.PodList{}
	podLabelSelectors, err := metav1.LabelSelectorAsSelector(depSelectors)
	if err != nil {
		return err
	}

	options := client.ListOptions{
		LabelSelector: podLabelSelectors,
		Namespace:     key.Namespace,
	}

	if err := kubeclient.List(ctx, podList, &options); err != nil {
		return errors.Wrap(err, "error getting Pods")
	}

	for _, p := range podList.Items {
		for _, cs := range p.Status.ContainerStatuses {
			if !cs.Ready {
				if cs.State.Waiting != nil {
					containerName := p.Name + ":" + cs.Name
					podErr = append(podErr, podError{resourceError{name: containerName, issue: cs.State.Waiting.Message}})
				}
			}
		}
	}

	return podErr
}

func (c Client) DoRolloutWait(ctx context.Context, key types.NamespacedName) error {
	kubeclient, err := c.getKubeclient()
	if err != nil {
		return err
	}

	rolloutComplete := func() (bool, error) {
		deployment := appsv1.Deployment{}
		err := kubeclient.Get(ctx, key, &deployment)
		if err != nil {
			if apierrors.IsNotFound(err) {
				// Waiting for Deployment to appear
				return false, nil
			}
			return false, err
		}
		if deployment.Generation <= deployment.Status.ObservedGeneration {
			cond := deploymentutil.GetDeploymentCondition(deployment.Status, appsv1.DeploymentProgressing)
			if cond != nil && cond.Reason == deploymentutil.TimedOutReason {
				return false, errors.New("progress deadline exceeded")
			}
			if deployment.Spec.Replicas != nil && deployment.Status.UpdatedReplicas < *deployment.Spec.Replicas {
				// Waiting for Deployment to rollout. Not all replicas have been updated
				return false, nil
			}
			if deployment.Status.Replicas > deployment.Status.UpdatedReplicas {
				// Waiting for Deployment to rollout. Old replicas are pending termination
				return false, nil
			}
			if deployment.Status.AvailableReplicas < deployment.Status.UpdatedReplicas {
				// Waiting for Deployment to rollout. Not all updated replicas are available
				return false, nil
			}
			// Deployment successfully rolled out
			return true, nil
		}
		// Waiting for Deployment to rollout: waiting for deployment spec update to be observed
		return false, nil
	}
	return wait.PollImmediateUntil(time.Second, rolloutComplete, ctx.Done())
}
