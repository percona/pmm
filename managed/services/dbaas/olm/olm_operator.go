// dbaas-controller
// Copyright (C) 2020 Percona LLC
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

// Package olm contains logic related to kubernetes operators.
package olm

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"log"
	"time"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/percona/pmm/managed/data"
	"github.com/percona/pmm/managed/services/dbaas/kubernetes/client"
)

const (
	olmRepo              = "operator-lifecycle-manager"
	githubAPIURLTemplate = "https://api.github.com/repos/operator-framework/%s/releases/latest"
	baseDownloadURL      = "github.com/operator-framework/operator-lifecycle-manager/releases/download"
	olmNamespace         = "olm"

	// If version is not set, DBaaS controller will choose the latest from the repo.
	// It doesn't work for offline installation.
	latestOLMVersion    = "latest"
	defaultOLMVersion   = ""
	useDefaultNamespace = ""

	// APIVersionCoreosV1 constant for some API requests.
	APIVersionCoreosV1 = "operators.coreos.com/v1"

	pollInterval = 1 * time.Second
	pollDuration = 5 * time.Minute
)

// ErrEmptyVersionTag Got an empty version tag from GitHub API.
var ErrEmptyVersionTag error = errors.New("got an empty version tag from Github")

// OperatorService holds methods to handle the OLM operator.
type OperatorService struct {
	kubeConfig string
}

// NewOperatorServiceFromConfig returns new OperatorService instance and intializes the config.
func New(kubeConfig string) *OperatorService {
	return &OperatorService{ //nolint:exhaustruct
		kubeConfig: kubeConfig,
	}
}

// InstallOLMOperator installs the OLM in the Kubernetes cluster.
func (o *OperatorService) InstallOLMOperator(ctx context.Context) error {
	k8sclient, err := client.NewFromKubeConfigString(o.kubeConfig)
	if err != nil {
		return errors.Wrap(err, "cannot initialize the kubernetes client")
	}

	deployment, err := k8sclient.GetDeployment(ctx, "olm-operator")
	if err == nil {
		return nil
		// return errors.Wrap(err, "cannot check if the OLM operator is already installed or not")
	}

	if deployment.ObjectMeta.Name != "" {
		return nil
	}

	var crdFile, olmFile []byte

	crdFile, err = fs.ReadFile(data.OLMCRDs, "crds/crds.yaml")
	if err != nil {
		return errors.Wrapf(err, "failed to read OLM CRDs file")
	}

	if err := k8sclient.ApplyFile(ctx, crdFile); err != nil {
		// TODO: revert applied files before return
		return errors.Wrapf(err, "cannot apply %q file", crdFile)
	}

	// client.WaitForCondition(ctx, "Established", crdFile)

	olmFile, err = fs.ReadFile(data.OLMCRDs, "crds/olm.yaml")
	if err != nil {
		return errors.Wrapf(err, "failed to read OLM file")
	}

	if err := k8sclient.ApplyFile(ctx, olmFile); err != nil {
		// TODO: revert applied files before return
		return errors.Wrapf(err, "cannot apply %q file", crdFile)
	}

	if err := k8sclient.DoRolloutWait(ctx, types.NamespacedName{Namespace: "olm", Name: "olm-operator"}); err != nil {
		return errors.Wrap(err, "error while waiting for deployment rollout")
	}
	if err := k8sclient.DoRolloutWait(ctx, types.NamespacedName{Namespace: "olm", Name: "catalog-operator"}); err != nil {
		return errors.Wrap(err, "error while waiting for deployment rollout")
	}

	crdResources, err := decodeResources(crdFile)
	if err != nil {
		return errors.Wrap(err, "cannot decode crd resources")
	}

	olmResources, err := decodeResources(olmFile)
	if err != nil {
		return errors.Wrap(err, "cannot decode olm resources")
	}

	resources := append(crdResources, olmResources...)

	subscriptions := filterResources(resources, func(r unstructured.Unstructured) bool {
		return r.GroupVersionKind() == schema.GroupVersionKind{
			Group:   operatorsv1alpha1.GroupName,
			Version: operatorsv1alpha1.GroupVersion,
			Kind:    operatorsv1alpha1.SubscriptionKind,
		}
	})

	for _, sub := range subscriptions {
		subscriptionKey := types.NamespacedName{Namespace: sub.GetNamespace(), Name: sub.GetName()}
		log.Printf("Waiting for subscription/%s to install CSV", subscriptionKey.Name)
		csvKey, err := k8sclient.GetSubscriptionCSV(ctx, subscriptionKey)
		if err != nil {
			return fmt.Errorf("subscription/%s failed to install CSV: %v", subscriptionKey.Name, err)
		}
		log.Printf("Waiting for clusterserviceversion/%s to reach 'Succeeded' phase", csvKey.Name)
		if err := k8sclient.DoCSVWait(ctx, csvKey); err != nil {
			return fmt.Errorf("clusterserviceversion/%s failed to reach 'Succeeded' phase", csvKey.Name)
		}
	}

	if err := k8sclient.DoRolloutWait(ctx, types.NamespacedName{Namespace: "olm", Name: "packageserver"}); err != nil {
		return errors.Wrap(err, "error while waiting for deployment rollout")
	}

	return nil
}

func decodeResources(f []byte) (objs []unstructured.Unstructured, err error) {
	dec := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(f), 8)
	for {
		var u unstructured.Unstructured
		err = dec.Decode(&u)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		objs = append(objs, u)
	}

	return objs, nil
}

func filterResources(resources []unstructured.Unstructured, filter func(unstructured.
	Unstructured) bool) (filtered []unstructured.Unstructured) {
	for _, r := range resources {
		if filter(r) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func isInstalled(ctx context.Context, client *client.Client, namespace string) bool {
	if _, err := client.GetDeployment(ctx, "olm-operator"); err == nil {
		return true
	}
	return false
}

//  // ListInstallPlans gets the list of all available install plans.
//  func (o *OperatorService) ListInstallPlans(ctx context.Context, req *controllerv1beta1.ListInstallPlansRequest) (*controllerv1beta1.ListInstallPlansResponse, error) {
//  	client, err := k8sclient.New(ctx, req.KubeAuth.Kubeconfig)
//  	if err != nil {
//  		return nil, status.Error(codes.Internal, err.Error())
//  	}
//  	defer client.Cleanup() //nolint:errcheck
//
//  	plans, err := getInstallPlans(ctx, client, req.Namespace)
//  	if err != nil {
//  		return nil, errors.Wrap(err, "cannot get install plans")
//  	}
//
//  	response := &controllerv1beta1.ListInstallPlansResponse{
//  		Items: []*controllerv1beta1.ListInstallPlansResponse_InstallPlan{}, //nolint:nosnakecase
//  	}
//
//  	for _, item := range plans.Items {
//  		csv := ""
//  		if len(item.Spec.ClusterServiceVersionNames) > 0 {
//  			csv = item.Spec.ClusterServiceVersionNames[0]
//  		}
//
//  		if req.NotApprovedOnly && item.Spec.Approved {
//  			continue
//  		}
//
//  		if req.Name != "" {
//  			if !strings.HasPrefix(strings.ToLower(csv), strings.ToLower(req.Name)) {
//  				continue
//  			}
//  		}
//
//  		installPlan := &controllerv1beta1.ListInstallPlansResponse_InstallPlan{ //nolint:nosnakecase
//  			Namespace: item.ObjectMeta.Namespace,
//  			Name:      item.ObjectMeta.Name,
//  			Csv:       csv,
//  			Approval:  string(item.Spec.Approval),
//  			Approved:  item.Spec.Approved,
//  		}
//
//  		response.Items = append(response.Items, installPlan)
//  	}
//
//  	return response, nil
//  }
//
//  // ListSubscriptions list all available subscriptions.
//  func (o *OperatorService) ListSubscriptions(ctx context.Context, req *controllerv1beta1.ListSubscriptionsRequest) (*controllerv1beta1.ListSubscriptionsResponse, error) {
//  	resp := &controllerv1beta1.ListSubscriptionsResponse{
//  		Items: []*controllerv1beta1.Subscription{},
//  	}
//
//  	client, err := k8sclient.New(ctx, req.KubeAuth.Kubeconfig)
//  	if err != nil {
//  		return nil, status.Error(codes.Internal, err.Error())
//  	}
//  	defer client.Cleanup() //nolint:errcheck
//
//  	cmd := []string{"get", "subscriptions", "-A", "-ojson"}
//  	data, err := client.Run(ctx, cmd)
//  	if err != nil {
//  		return nil, errors.Wrap(err, "cannot get subscriptions list")
//  	}
//
//  	var susbcriptions v1alpha1.SubscriptionList
//  	err = json.Unmarshal(data, &susbcriptions)
//  	if err != nil {
//  		return nil, errors.Wrap(err, "cannot decode install plans from the response")
//  	}
//
//  	for _, item := range susbcriptions.Items {
//  		resp.Items = append(resp.Items, &controllerv1beta1.Subscription{
//  			Namespace:       item.ObjectMeta.Namespace,
//  			Name:            item.ObjectMeta.Name,
//  			Package:         item.Spec.Package,
//  			Source:          item.Spec.CatalogSource,
//  			Channel:         item.Spec.Channel,
//  			CurrentCsv:      item.Status.CurrentCSV,
//  			InstalledCsv:    item.Status.InstalledCSV,
//  			InstallPlanName: item.Status.Install.Name,
//  		})
//  	}
//
//  	return resp, nil
//  }
//
//  // GetSubscription list all available subscriptions.
//  func (o *OperatorService) GetSubscription(ctx context.Context, req *controllerv1beta1.GetSubscriptionRequest) (*controllerv1beta1.GetSubscriptionResponse, error) {
//  	client, err := k8sclient.New(ctx, req.KubeAuth.Kubeconfig)
//  	if err != nil {
//  		return nil, status.Error(codes.Internal, err.Error())
//  	}
//  	defer client.Cleanup() //nolint:errcheck
//
//  	var subscription v1alpha1.Subscription
//
//  	for i := 0; i < 6; i++ {
//  		cmd := []string{"get", "subscription", req.Name, "--namespace", req.Namespace, "-ojson"}
//  		data, err := client.Run(ctx, cmd)
//  		if err != nil {
//  			return nil, errors.Wrapf(err, "cannot get subscription %q in namespace %q", req.Name, req.Namespace)
//  		}
//
//  		err = json.Unmarshal(data, &subscription)
//  		if err != nil {
//  			return nil, errors.Wrap(err, "cannot decode install plans from the response")
//  		}
//
//  		// Retry until we have an install plan.
//  		if subscription.Status.Install != nil {
//  			break
//  		}
//
//  		time.Sleep(5 * time.Second)
//  	}
//
//  	resp := &controllerv1beta1.GetSubscriptionResponse{
//  		Subscription: &controllerv1beta1.Subscription{
//  			Namespace:    subscription.ObjectMeta.Namespace,
//  			Name:         subscription.ObjectMeta.Name,
//  			Package:      subscription.Spec.Package,
//  			Source:       subscription.Spec.CatalogSource,
//  			Channel:      subscription.Spec.Channel,
//  			CurrentCsv:   subscription.Status.CurrentCSV,
//  			InstalledCsv: subscription.Status.InstalledCSV,
//  		},
//  	}
//
//  	if subscription.Status.Install != nil {
//  		resp.Subscription.InstallPlanName = subscription.Status.Install.Name
//  	}
//
//  	return resp, nil
//  }
//
//  func getInstallPlans(ctx context.Context, client *k8sclient.K8sClient, namespace string) (*v1alpha1.InstallPlanList, error) {
//  	cmd := []string{"get", "installplans", "-ojson"}
//  	if namespace != "" {
//  		cmd = append(cmd, []string{"-n", namespace}...)
//  	} else {
//  		cmd = append(cmd, "-A")
//  	}
//
//  	data, err := client.Run(ctx, cmd)
//  	if err != nil {
//  		return nil, errors.Wrap(err, "cannot get operators group list")
//  	}
//
//  	var installPlans v1alpha1.InstallPlanList
//  	err = json.Unmarshal(data, &installPlans)
//  	if err != nil {
//  		return nil, errors.Wrap(err, "cannot decode install plans from the response")
//  	}
//
//  	return &installPlans, nil
//  }
//
//  type approveInstallPlanSpec struct {
//  	Approved bool `json:"approved"`
//  }
//
//  type approveInstallPlanReq struct {
//  	Spec approveInstallPlanSpec `json:"spec"`
//  }
//
//  // ApproveInstallPlan patches an existing install plan to set it to approved.
//  func (o *OperatorService) ApproveInstallPlan(ctx context.Context, req *controllerv1beta1.ApproveInstallPlanRequest) (*controllerv1beta1.ApproveInstallPlanResponse, error) {
//  	client, err := k8sclient.New(ctx, req.KubeAuth.Kubeconfig)
//  	if err != nil {
//  		return nil, status.Error(codes.Internal, err.Error())
//  	}
//  	defer client.Cleanup() //nolint:errcheck
//
//  	res := approveInstallPlanReq{Spec: approveInstallPlanSpec{Approved: true}}
//
//  	if err := client.Patch(ctx, "merge", "installplan", req.Name, req.Namespace, res); err != nil {
//  		return nil, errors.Wrap(err, "cannot approve the install plan")
//  	}
//
//  	resp := new(controllerv1beta1.ApproveInstallPlanResponse)
//
//  	return resp, nil
//  }
//
//  // AvailableOperators resturns the list of available operators for a given namespace and filter.
//  func (o *OperatorService) AvailableOperators(ctx context.Context, client *k8sclient.K8sClient, namespace, name string) (*operators.PackageManifestList, error) {
//  	cmd := []string{"get", "packagemanifest", "-ojson"}
//
//  	if namespace != "" {
//  		cmd = append(cmd, []string{"-n", namespace}...)
//  	}
//  	if name != "" {
//  		cmd = append(cmd, []string{"--field-selector", "metadata.name=" + name}...)
//  	}
//
//  	data, err := client.Run(ctx, cmd)
//  	if err != nil {
//  		return nil, errors.Wrap(err, "cannot get operators group list")
//  	}
//
//  	var manifestList operators.PackageManifestList
//  	err = json.Unmarshal(data, &manifestList)
//  	if err != nil {
//  		return nil, errors.Wrap(err, "cannot decode package manifest list response")
//  	}
//
//  	return &manifestList, nil
//  }
//
// InstallOperatorRequest holds the fields to make an operator install request.
type InstallOperatorRequest struct {
	Namespace              string
	Name                   string
	OperatorGroup          string
	CatalogSource          string
	CatalogSourceNamespace string
	Channel                string
	InstallPlanApproval    v1alpha1.Approval
	StartingCSV            string
}

// InstallOperator installs an operator via OLM.
func (o *OperatorService) InstallOperator(ctx context.Context, req InstallOperatorRequest) error {
	client, err := client.NewFromKubeConfigString(o.kubeConfig)
	if err != nil {
		return errors.Wrap(err, "cannot initialize the kubernetes client")
	}

	if err := createOperatorGroupIfNeeded(ctx, client, "percona-operatorgroup"); err != nil {
		return err
	}

	// func (c *Client) CreateSubscriptionForCatalog(ctx context.Context, namespace, name, catalogNamespace, catalog,
	// packageName, channel, startingCSV string, approval operatorsv1alpha1.Approval) (*v1alpha1.Subscription, error) {
	subs, err := client.CreateSubscriptionForCatalog(ctx,
		req.Namespace,
		req.Name,
		"olm",
		req.CatalogSource,
		req.Name,
		req.Channel,
		req.StartingCSV,
		operatorsv1alpha1.ApprovalManual,
	)
	if err != nil {
		return errors.Wrap(err, "cannot create a susbcription to install the operator")
	}

	err = wait.Poll(pollInterval, pollDuration, func() (bool, error) {
		subs, err = client.GetSubscription(ctx, req.Namespace, req.Name)
		if err != nil || subs == nil || (subs != nil && subs.Status.Install == nil) {
			return false, err
		}

		return false, nil
	})

	return new(controllerv1beta1.InstallOperatorResponse), nil
}

func createOperatorGroupIfNeeded(ctx context.Context, client *client.Client, name string) error {
	_, err := client.GetOperatorGroup(ctx, useDefaultNamespace, name)
	if err == nil {
		return nil
	}

	_, err = client.CreateOperatorGroup(ctx, useDefaultNamespace, name)

	return err
}

//
//
//  func (o *OperatorService) operatorGroupExists(ctx context.Context, client *k8sclient.K8sClient, namespace, name string) (bool, error) {
//  	var operatorGroupList operatorsv1.OperatorGroupList
//
//  	data, err := client.Run(ctx, []string{"get", "operatorgroups", "-ojson", "--field-selector", "metadata.name=" + name})
//  	if err != nil {
//  		return false, errors.Wrap(err, "cannot get operators group list")
//  	}
//
//  	if err := json.Unmarshal(data, &operatorGroupList); err != nil {
//  		return false, errors.Wrap(err, "cannot decode operator group list response")
//  	}
//
//  	if len(operatorGroupList.Items) > 0 {
//  		return true, nil
//  	}
//
//  	return false, nil
//  }
//
//  func (o *OperatorService) createOperatorGroup(ctx context.Context, client *k8sclient.K8sClient, namespace, name string) error {
//  	op := &operatorsv1.OperatorGroup{
//  		TypeMeta: metav1.TypeMeta{
//  			APIVersion: APIVersionCoreosV1,
//  			Kind:       operatorsv1.OperatorGroupKind,
//  		},
//
//  		ObjectMeta: metav1.ObjectMeta{
//  			Name:      name,
//  			Namespace: namespace,
//  		},
//  		Spec: operatorsv1.OperatorGroupSpec{
//  			TargetNamespaces: []string{namespace},
//  		},
//  		Status: operatorsv1.OperatorGroupStatus{
//  			LastUpdated: &metav1.Time{
//  				Time: time.Now(),
//  			},
//  		},
//  	}
//
//  	payload, err := json.Marshal(op)
//  	if err != nil {
//  		return errors.Wrap(err, "cannot encode subscription payload as yaml")
//  	}
//
//  	return client.Create(ctx, payload)
//  }
//
//  func (o *OperatorService) createSubscription(ctx context.Context, client *k8sclient.K8sClient, req *controllerv1beta1.InstallOperatorRequest) error {
//  	subscription := &v1alpha1.Subscription{
//  		TypeMeta: metav1.TypeMeta{
//  			Kind:       v1alpha1.SubscriptionKind,
//  			APIVersion: v1alpha1.SubscriptionCRDAPIVersion,
//  		},
//  		ObjectMeta: metav1.ObjectMeta{
//  			Namespace: req.Namespace,
//  			Name:      req.Name,
//  		},
//  		Spec: &v1alpha1.SubscriptionSpec{
//  			CatalogSource:          req.CatalogSource,
//  			CatalogSourceNamespace: req.CatalogSourceNamespace,
//  			Channel:                req.Channel,
//  			Package:                req.Name,
//  			InstallPlanApproval:    v1alpha1.Approval(req.InstallPlanApproval),
//  			StartingCSV:            req.StartingCsv,
//  		},
//  		Status: v1alpha1.SubscriptionStatus{
//  			LastUpdated: metav1.Time{
//  				Time: time.Now(),
//  			},
//  		},
//  	}
//
//  	payload, err := json.Marshal(subscription)
//  	if err != nil {
//  		return errors.Wrap(err, "cannot encode subscription payload as yaml")
//  	}
//
//  	return client.Create(ctx, payload)
//  }
//
//  func namespaceExists(ctx context.Context, client *k8sclient.K8sClient, namespace string) (bool, error) {
//  	var namespaceList corev1.NamespaceList
//
//  	data, err := client.Run(ctx, []string{"get", "namespaces", "--field-selector", "metadata.name=" + namespace, "-ojson"})
//  	if err != nil {
//  		return false, errors.Wrap(err, "cannot get namespaces list")
//  	}
//
//  	if err := json.Unmarshal(data, &namespaceList); err != nil {
//  		return false, errors.Wrap(err, "cannot decode namespaces list response")
//  	}
//
//  	if len(namespaceList.Items) > 0 {
//  		return true, nil
//  	}
//
//  	return false, nil
//  }
//
//  /*
//  Helpers
//  */
//  func getLatestVersion(ctx context.Context, repo string) (string, error) {
//  	url := fmt.Sprintf(githubAPIURLTemplate, repo)
//  	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
//  	if err != nil {
//  		return "", errors.Wrap(err, "cannot prepare OLM lastest version request")
//  	}
//
//  	response, err := http.DefaultClient.Do(req)
//  	if err != nil {
//  		return "", errors.Wrap(err, "cannot get OLM latest version")
//  	}
//
//  	defer response.Body.Close() //nolint:errcheck
//
//  	type jsonResponse struct {
//  		TagName string `json:"tag_name"` //nolint:tagliatelle
//  	}
//  	var resp *jsonResponse
//
//  	body, err := io.ReadAll(response.Body)
//  	if err != nil {
//  		return "", errors.Wrap(err, "cannot read OLM latest version response")
//  	}
//
//  	if err := json.Unmarshal(body, &resp); err != nil {
//  		return "", errors.Wrap(err, "cannot decode OLM latest version response")
//  	}
//
//  	if resp.TagName != "" {
//  		return resp.TagName, nil
//  	}
//
//  	return "", ErrEmptyVersionTag
//  }
//
//  func waitForDeployments(ctx context.Context, client *k8sclient.K8sClient, namespace string) error {
//  	if _, err := client.Run(ctx, []string{"rollout", "status", "-w", "deployment/olm-operator", "--namespace", namespace}); err != nil {
//  		return errors.Wrap(err, "error while waiting for olm component deployment")
//  	}
//
//  	if _, err := client.Run(ctx, []string{"rollout", "status", "-w", "deployment/catalog-operator", "--namespace", namespace}); err != nil {
//  		return errors.Wrap(err, "error while waiting for olm component deployment")
//  	}
//
//  	return nil
//  }
//
// func waitForPackageServer(ctx context.Context, client *k8sclient.K8sClient, namespace string) error {
// 	var err error
// 	var data []byte
//
// 	for i := 0; i < 15; i++ {
// 		data, err = client.Run(ctx, []string{"get", "csv", "-n", namespace, "packageserver", "-o", "jsonpath='{.status.phase}'"})
// 		if err == nil && string(data) == "Succeeded" {
// 			break
// 		}
// 		time.Sleep(5 * time.Second)
// 	}
//
// 	return err
// }
