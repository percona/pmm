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

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/util/yaml"

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
	if err == nil && deployment != nil && deployment.ObjectMeta.Name != "" {
		return nil // already installed
	}

	var crdFile, olmFile []byte

	crdFile, err = fs.ReadFile(data.OLMCRDs, "crds/olm/crds.yaml")
	if err != nil {
		return errors.Wrapf(err, "failed to read OLM CRDs file")
	}

	if err := k8sclient.ApplyFile(ctx, crdFile); err != nil {
		// TODO: revert applied files before return
		return errors.Wrapf(err, "cannot apply %q file", crdFile)
	}

	// client.WaitForCondition(ctx, "Established", crdFile)

	olmFile, err = fs.ReadFile(data.OLMCRDs, "crds/olm/olm.yaml")
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
	Unstructured) bool,
) (filtered []unstructured.Unstructured) {
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

		return true, nil
	})

	if err != nil {
		return err
	}
	if subs == nil {
		return fmt.Errorf("cannot get an install plan for the operator subscription: %q", req.Name)
	}

	ip, err := client.GetInstallPlan(ctx, req.Namespace, subs.Status.Install.Name)
	if err != nil {
		return err
	}

	ip.Spec.Approved = true
	_, err = client.UpdateInstallPlan(ctx, req.Namespace, ip)

	return err
}

func createOperatorGroupIfNeeded(ctx context.Context, client *client.Client, name string) error {
	_, err := client.GetOperatorGroup(ctx, useDefaultNamespace, name)
	if err == nil {
		return nil
	}

	_, err = client.CreateOperatorGroup(ctx, "default", name)

	return err
}

// GetSubscription list all available subscriptions.
func (o *OperatorService) UpgradeOperator(ctx context.Context, namespace, name string) error {
	k8sclient, err := client.NewFromKubeConfigString(o.kubeConfig)
	if err != nil {
		return errors.Wrap(err, "cannot initialize the kubernetes client to read a subscription")
	}

	var subs *v1alpha1.Subscription

	// If the subscription was recently created, the install plan might not be ready yet.
	err = wait.Poll(pollInterval, pollDuration, func() (bool, error) {
		var err error
		subs, err = k8sclient.GetSubscription(ctx, namespace, name)
		if err != nil {
			return false, err
		}
		if subs == nil || subs.Status.Install == nil || subs.Status.Install.Name == "" {
			return false, nil
		}

		return true, nil
	})

	if err != nil {
		return err
	}
	if subs == nil || subs.Status.Install == nil || subs.Status.Install.Name == "" {
		return fmt.Errorf("cannot get subscription for %q operator", name)
	}

	ip, err := k8sclient.GetInstallPlan(ctx, namespace, subs.Status.Install.Name)
	if err != nil {
		return errors.Wrapf(err, "cannot get install plan to upgrade %q", name)
	}

	if ip.Spec.Approved == true {
		return nil // There are no upgrades.
	}

	ip.Spec.Approved = true

	_, err = k8sclient.UpdateInstallPlan(ctx, namespace, ip)

	return err
}
