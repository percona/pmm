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

package vmretention

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

const namespaceFile = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

// fieldManager identifies PMM's writes in the resource's managedFields, so that ownership of
// spec.retentionPeriod is attributable and does not depend on the binary name.
const fieldManager = "pmm-managed"

// KubeParams identifies the VictoriaMetrics custom resource that holds the retention period.
type KubeParams struct {
	// Name of the resource. Empty disables reconciliation.
	Name string
	// Namespace of the resource. Defaults to the namespace of the running pod.
	Namespace string
	// APIVersion of the resource, for example operator.victoriametrics.com/v1beta1.
	APIVersion string
	// Kind of the resource, for example VMCluster. Required unless Resource is set.
	Kind string
	// Resource overrides the plural resource name derived from Kind.
	Resource string
}

type kubeClient struct {
	resource dynamic.ResourceInterface
	name     string
}

// NewKubeClient returns a client for the custom resource described by params, or nil if the
// resource was not named. A nil client is not an error, it is how every deployment whose
// VictoriaMetrics is not managed by an operator opts out.
//
// Naming a resource is a statement of intent, so from there on anything that stops us from
// reaching it is an error rather than a silent opt-out, pmm-managed not running inside a
// cluster included. Failing at startup is what keeps a misconfiguration from looking exactly
// like a deployment that never wanted reconciliation.
//
// The interface return is deliberate: a nil Client is how callers learn that reconciliation
// does not apply, and a nil *kubeClient stored in a Client interface would not compare equal
// to nil.
func NewKubeClient(params KubeParams) (Client, error) { //nolint:ireturn
	if params.Name == "" {
		return nil, nil //nolint:nilnil
	}

	if params.Kind == "" && params.Resource == "" {
		return nil, errors.New("the resource to apply data retention to is not identified, set PMM_VM_CLUSTER_KIND or PMM_VM_CLUSTER_RESOURCE")
	}

	gv, err := schema.ParseGroupVersion(params.APIVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %q as an API version: %w", params.APIVersion, err)
	}

	// ParseGroupVersion accepts both an empty string and a bare version, resolving each to
	// the core API group. A custom resource always has a group, so neither can be right, and
	// left unchecked they build a client that instead fails on every reconcile.
	if gv.Group == "" || gv.Version == "" {
		return nil, fmt.Errorf("%q is not a group-qualified API version, set PMM_VM_CLUSTER_API_VERSION "+
			"to a group/version such as operator.victoriametrics.com/v1beta1", params.APIVersion)
	}

	namespace := params.Namespace
	if namespace == "" {
		b, err := os.ReadFile(namespaceFile)
		if err != nil {
			return nil, fmt.Errorf("failed to detect the current namespace, set PMM_VM_CLUSTER_NAMESPACE: %w", err)
		}
		namespace = strings.TrimSpace(string(b))
	}

	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to build the in-cluster Kubernetes configuration; unset "+
			"PMM_VM_CLUSTER_NAME if VictoriaMetrics is not managed by an operator here: %w", err)
	}

	client, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create the Kubernetes client: %w", err)
	}

	return &kubeClient{
		resource: client.Resource(gv.WithResource(resourceFor(params))).Namespace(namespace),
		name:     params.Name,
	}, nil
}

// Get returns the retentionPeriod currently set on the resource.
func (c *kubeClient) Get(ctx context.Context) (Retention, error) {
	obj, err := c.resource.Get(ctx, c.name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return Retention{}, fmt.Errorf("%w; check PMM_VM_CLUSTER_NAME and PMM_VM_CLUSTER_NAMESPACE", err)
		}
		return Retention{}, err
	}

	// A missing field reads as an empty string, which never equals a formatted retention
	// period, so the caller always writes it.
	period, _, err := unstructured.NestedString(obj.Object, "spec", "retentionPeriod")
	if err != nil {
		return Retention{}, fmt.Errorf("failed to read spec.retentionPeriod: %w", err)
	}

	return Retention{Period: period, resourceVersion: obj.GetResourceVersion()}, nil
}

// Set writes the retentionPeriod to the resource, leaving the rest of the spec alone.
//
// The resourceVersion read by Get travels in the patch body, where the API server treats it
// as a precondition and answers with a conflict if anything else changed the resource in the
// meantime. Without it a concurrent write would be silently lost.
func (c *kubeClient) Set(ctx context.Context, retention Retention) error {
	patch, err := json.Marshal(map[string]any{
		"metadata": map[string]any{"resourceVersion": retention.resourceVersion},
		"spec":     map[string]any{"retentionPeriod": retention.Period},
	})
	if err != nil {
		return err
	}

	_, err = c.resource.Patch(ctx, c.name, types.MergePatchType, patch, metav1.PatchOptions{
		FieldManager: fieldManager,
	})
	return err
}

// resourceFor returns the plural, lower-case resource name for a kind, which is how the
// VictoriaMetrics operator names its CRDs (VMCluster -> vmclusters, VMSingle -> vmsingles).
// PMM_VM_CLUSTER_RESOURCE overrides it for any kind this does not pluralise correctly.
func resourceFor(params KubeParams) string {
	if params.Resource != "" {
		return params.Resource
	}
	return strings.ToLower(params.Kind) + "s"
}

// check interfaces.
var _ Client = (*kubeClient)(nil)
