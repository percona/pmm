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

// Package kubernetes provides functionality for kubernetes.
package kubernetes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	dbaasv1 "github.com/percona/dbaas-operator/api/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	dbaasv1beta1 "github.com/percona/pmm/api/managementpb/dbaas"
	"github.com/percona/pmm/managed/data"
	"github.com/percona/pmm/managed/services/dbaas/kubernetes/client"
	"github.com/percona/pmm/managed/services/dbaas/utils/convertors"
)

// ClusterType is used by Kubernetes.
type ClusterType string

const (
	ClusterTypeUnknown         ClusterType = "unknown"  //nolint:revive
	ClusterTypeMinikube        ClusterType = "minikube" //nolint:revive
	ClusterTypeEKS             ClusterType = "eks"      //nolint:revive
	ClusterTypeGeneric         ClusterType = "generic"  //nolint:revive
	pxcDeploymentName                      = "percona-xtradb-cluster-operator"
	psmdbDeploymentName                    = "percona-server-mongodb-operator"
	dbaasDeploymentName                    = "dbaas-operator-controller-manager"
	psmdbOperatorContainerName             = "percona-server-mongodb-operator"
	pxcOperatorContainerName               = "percona-xtradb-cluster-operator"
	dbaasOperatorContainerName             = "manager"
	databaseClusterKind                    = "DatabaseCluster"
	databaseClusterAPIVersion              = "dbaas.percona.com/v1"
	restartAnnotationKey                   = "dbaas.percona.com/restart"
	managedByKey                           = "dbaas.percona.com/managed-by"
	templateLabelKey                       = "dbaas.percona.com/template"
	engineLabelKey                         = "dbaas.percona.com/engine"

	// ContainerStateWaiting represents a state when container requires some
	// operations being done in order to complete start up.
	ContainerStateWaiting ContainerState = "waiting"
	// ContainerStateTerminated indicates that container began execution and
	// then either ran to completion or failed for some reason.
	ContainerStateTerminated ContainerState = "terminated"

	// Max size of volume for AWS Elastic Block Storage service is 16TiB.
	maxVolumeSizeEBS    uint64 = 16 * 1024 * 1024 * 1024 * 1024
	olmNamespace               = "olm"
	useDefaultNamespace        = ""

	// APIVersionCoreosV1 constant for some API requests.
	APIVersionCoreosV1 = "operators.coreos.com/v1"

	pollInterval = 1 * time.Second
	pollDuration = 5 * time.Minute
)

// ErrEmptyVersionTag Got an empty version tag from GitHub API.
var ErrEmptyVersionTag error = errors.New("got an empty version tag from Github")

// Kubernetes is a client for Kubernetes.
type Kubernetes struct {
	lock       *sync.RWMutex
	client     client.KubeClientConnector
	l          *logrus.Entry
	httpClient *http.Client
	kubeconfig string
}

// ContainerState describes container's state - waiting, running, terminated.
type ContainerState string

// NodeSummaryNode holds information about Node inside Node's summary.
type NodeSummaryNode struct {
	FileSystem NodeFileSystemSummary `json:"fs,omitempty"`
}

// NodeSummary holds summary of the Node.
// One gets this by requesting Kubernetes API endpoint:
// /v1/nodes/<node-name>/proxy/stats/summary.
type NodeSummary struct {
	Node NodeSummaryNode `json:"node,omitempty"`
}

// NodeFileSystemSummary holds a summary of Node's filesystem.
type NodeFileSystemSummary struct {
	UsedBytes uint64 `json:"usedBytes,omitempty"`
}

// NewIncluster returns new Kubernetes object.
func NewIncluster() (*Kubernetes, error) {
	l := logrus.WithField("component", "kubernetes")

	client, err := client.NewFromInCluster()
	if err != nil {
		return nil, err
	}

	return &Kubernetes{
		client: client,
		l:      l,
		lock:   &sync.RWMutex{},
		httpClient: &http.Client{
			Timeout: time.Second * 5,
			Transport: &http.Transport{
				MaxIdleConns:    1,
				IdleConnTimeout: 10 * time.Second,
			},
		},
	}, nil
}

// New returns new Kubernetes object.
func New(kubeconfig string) (*Kubernetes, error) {
	l := logrus.WithField("component", "kubernetes")

	client, err := client.NewFromKubeConfigString(kubeconfig)
	if err != nil {
		return nil, err
	}

	return &Kubernetes{
		client: client,
		l:      l,
		lock:   &sync.RWMutex{},
		httpClient: &http.Client{
			Timeout: time.Second * 5,
			Transport: &http.Transport{
				MaxIdleConns:    1,
				IdleConnTimeout: 10 * time.Second,
			},
		},
		kubeconfig: kubeconfig,
	}, nil
}

// NewEmpty returns new Kubernetes object.
func NewEmpty() *Kubernetes {
	return &Kubernetes{
		client: &client.Client{},
		lock:   &sync.RWMutex{},
		l:      logrus.WithField("component", "kubernetes"),
		httpClient: &http.Client{
			Timeout: time.Second * 5,
			Transport: &http.Transport{
				MaxIdleConns:    1,
				IdleConnTimeout: 10 * time.Second,
			},
		},
	}
}

// SetKubeconfig changes kubeconfig for active client.
func (k *Kubernetes) SetKubeconfig(kubeconfig string) error {
	k.lock.Lock()
	defer k.lock.Unlock()
	client, err := client.NewFromKubeConfigString(kubeconfig)
	if err != nil {
		return err
	}
	k.client = client
	k.kubeconfig = kubeconfig
	return nil
}

// GetKubeconfig generates kubeconfig compatible with kubectl for incluster created clients.
func (k *Kubernetes) GetKubeconfig(ctx context.Context) (string, error) {
	k.lock.RLock()
	defer k.lock.RUnlock()
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

// ListDatabaseClusters returns list of managed PCX clusters.
func (k *Kubernetes) ListDatabaseClusters(ctx context.Context) (*dbaasv1.DatabaseClusterList, error) {
	k.lock.RLock()
	defer k.lock.RUnlock()
	return k.client.ListDatabaseClusters(ctx)
}

// GetDatabaseCluster returns PXC clusters by provided name.
func (k *Kubernetes) GetDatabaseCluster(ctx context.Context, name string) (*dbaasv1.DatabaseCluster, error) {
	k.lock.RLock()
	defer k.lock.RUnlock()
	return k.client.GetDatabaseCluster(ctx, name)
}

// RestartDatabaseCluster restarts database cluster.
func (k *Kubernetes) RestartDatabaseCluster(ctx context.Context, name string) error {
	k.lock.Lock()
	defer k.lock.Unlock()
	cluster, err := k.client.GetDatabaseCluster(ctx, name)
	if err != nil {
		return err
	}
	cluster.TypeMeta.APIVersion = databaseClusterAPIVersion
	cluster.TypeMeta.Kind = databaseClusterKind
	if cluster.ObjectMeta.Annotations == nil {
		cluster.ObjectMeta.Annotations = make(map[string]string)
	}
	cluster.ObjectMeta.Annotations[restartAnnotationKey] = "true"
	return k.client.ApplyObject(cluster)
}

// PatchDatabaseCluster patches CR of managed Database cluster.
func (k *Kubernetes) PatchDatabaseCluster(cluster *dbaasv1.DatabaseCluster) error {
	k.lock.Lock()
	defer k.lock.Unlock()
	return k.client.ApplyObject(cluster)
}

// CreateDatabaseCluster creates database cluster.
func (k *Kubernetes) CreateDatabaseCluster(cluster *dbaasv1.DatabaseCluster) error {
	k.lock.Lock()
	defer k.lock.Unlock()
	if cluster.ObjectMeta.Annotations == nil {
		cluster.ObjectMeta.Annotations = make(map[string]string)
	}
	cluster.ObjectMeta.Annotations[managedByKey] = "pmm"
	return k.client.ApplyObject(cluster)
}

// DeleteDatabaseCluster deletes database cluster.
func (k *Kubernetes) DeleteDatabaseCluster(ctx context.Context, name string) error {
	k.lock.Lock()
	defer k.lock.Unlock()
	cluster, err := k.client.GetDatabaseCluster(ctx, name)
	if err != nil {
		return err
	}
	cluster.TypeMeta.APIVersion = databaseClusterAPIVersion
	cluster.TypeMeta.Kind = databaseClusterKind
	return k.client.DeleteObject(cluster)
}

// GetDefaultStorageClassName returns first storageClassName from kubernetes cluster.
func (k *Kubernetes) GetDefaultStorageClassName(ctx context.Context) (string, error) {
	k.lock.RLock()
	defer k.lock.RUnlock()
	storageClasses, err := k.client.GetStorageClasses(ctx)
	if err != nil {
		return "", err
	}
	if len(storageClasses.Items) != 0 {
		return storageClasses.Items[0].Name, nil
	}
	return "", errors.New("no storage classes available")
}

// GetClusterType tries to guess the underlying kubernetes cluster based on storage class.
func (k *Kubernetes) GetClusterType(ctx context.Context) (ClusterType, error) {
	k.lock.RLock()
	defer k.lock.RUnlock()
	storageClasses, err := k.client.GetStorageClasses(ctx)
	if err != nil {
		return ClusterTypeUnknown, err
	}
	for _, storageClass := range storageClasses.Items {
		if strings.Contains(storageClass.Provisioner, "aws") {
			return ClusterTypeEKS, nil
		}
		if strings.Contains(storageClass.Provisioner, "minikube") ||
			strings.Contains(storageClass.Provisioner, "kubevirt.io/hostpath-provisioner") ||
			strings.Contains(storageClass.Provisioner, "standard") {
			return ClusterTypeMinikube, nil
		}
	}
	return ClusterTypeGeneric, nil
}

// getOperatorVersion parses operator version from operator deployment.
func (k *Kubernetes) getOperatorVersion(ctx context.Context, deploymentName, containerName string) (string, error) {
	deployment, err := k.client.GetDeployment(ctx, deploymentName)
	if err != nil {
		return "", err
	}
	for _, container := range deployment.Spec.Template.Spec.Containers {
		if container.Name == containerName {
			return strings.Split(container.Image, ":")[1], nil
		}
	}
	return "", errors.New("unknown version of operator")
}

// GetPSMDBOperatorVersion parses PSMDB operator version from operator deployment.
func (k *Kubernetes) GetPSMDBOperatorVersion(ctx context.Context) (string, error) {
	k.lock.RLock()
	defer k.lock.RUnlock()
	return k.getOperatorVersion(ctx, psmdbDeploymentName, psmdbOperatorContainerName)
}

// GetPXCOperatorVersion parses PXC operator version from operator deployment.
func (k *Kubernetes) GetPXCOperatorVersion(ctx context.Context) (string, error) {
	k.lock.RLock()
	defer k.lock.RUnlock()
	return k.getOperatorVersion(ctx, pxcDeploymentName, pxcOperatorContainerName)
}

// GetDBaaSOperatorVersion parses DBaaS operator version from operator deployment.
func (k *Kubernetes) GetDBaaSOperatorVersion(ctx context.Context) (string, error) {
	k.lock.RLock()
	defer k.lock.RUnlock()
	return k.getOperatorVersion(ctx, dbaasDeploymentName, dbaasOperatorContainerName)
}

// GetSecret returns secret by name.
func (k *Kubernetes) GetSecret(ctx context.Context, name string) (*corev1.Secret, error) {
	k.lock.RLock()
	defer k.lock.RUnlock()
	return k.client.GetSecret(ctx, name)
}

// ListSecrets returns secret by name.
func (k *Kubernetes) ListSecrets(ctx context.Context) (*corev1.SecretList, error) {
	k.lock.RLock()
	defer k.lock.RUnlock()
	return k.client.ListSecrets(ctx)
}

// CreatePMMSecret creates pmm secret in kubernetes.
func (k *Kubernetes) CreatePMMSecret(secretName string, secrets map[string][]byte) error {
	k.lock.Lock()
	defer k.lock.Unlock()
	secret := &corev1.Secret{ //nolint: exhaustruct
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: secretName,
		},
		Type: corev1.SecretTypeOpaque,
		Data: secrets,
	}
	return k.client.ApplyObject(secret)
}

// CreateRestore will apply restore.
func (k *Kubernetes) CreateRestore(restore *dbaasv1.DatabaseClusterRestore) error {
	k.lock.Lock()
	defer k.lock.Unlock()
	return k.client.ApplyObject(restore)
}

// GetPods returns list of pods.
func (k *Kubernetes) GetPods(ctx context.Context, namespace string, labelSelector *metav1.LabelSelector) (*corev1.PodList, error) {
	return k.client.GetPods(ctx, namespace, labelSelector)
}

// GetLogs returns logs as slice of log lines - strings - for given pod's container.
func (k *Kubernetes) GetLogs(
	ctx context.Context,
	containerStatuses []corev1.ContainerStatus,
	pod,
	container string,
) ([]string, error) {
	if IsContainerInState(containerStatuses, ContainerStateWaiting) {
		return []string{}, nil
	}

	stdout, err := k.client.GetLogs(ctx, pod, container)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get logs")
	}

	if stdout == "" {
		return []string{}, nil
	}

	return strings.Split(stdout, "\n"), nil
}

// GetEvents returns pod's events as a slice of strings.
func (k *Kubernetes) GetEvents(ctx context.Context, pod string) ([]string, error) {
	stdout, err := k.client.GetEvents(ctx, pod)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't describe pod")
	}

	lines := strings.Split(stdout, "\n")

	return lines, nil
}

// IsContainerInState returns true if container is in give state, otherwise false.
func IsContainerInState(containerStatuses []corev1.ContainerStatus, state ContainerState) bool {
	containerState := make(map[string]interface{})
	for _, status := range containerStatuses {
		data, err := json.Marshal(status.State)
		if err != nil {
			return false
		}

		if err := json.Unmarshal(data, &containerState); err != nil {
			return false
		}

		if _, ok := containerState[string(state)]; ok {
			return true
		}
	}

	return false
}

// IsNodeInCondition returns true if node's condition given as an argument has
// status "True". Otherwise it returns false.
func IsNodeInCondition(node corev1.Node, conditionType corev1.NodeConditionType) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Status == corev1.ConditionTrue && condition.Type == conditionType {
			return true
		}
	}
	return false
}

// GetWorkerNodes returns list of cluster workers nodes.
func (k *Kubernetes) GetWorkerNodes(ctx context.Context) ([]corev1.Node, error) {
	nodes, err := k.client.GetNodes(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not get nodes of Kubernetes cluster")
	}
	forbidenTaints := map[string]corev1.TaintEffect{
		"node.cloudprovider.kubernetes.io/uninitialized": corev1.TaintEffectNoSchedule,
		"node.kubernetes.io/unschedulable":               corev1.TaintEffectNoSchedule,
		"node-role.kubernetes.io/master":                 corev1.TaintEffectNoSchedule,
	}
	workers := make([]corev1.Node, 0, len(nodes.Items))
	for _, node := range nodes.Items {
		if len(node.Spec.Taints) == 0 {
			workers = append(workers, node)
			continue
		}
		for _, taint := range node.Spec.Taints {
			effect, keyFound := forbidenTaints[taint.Key]
			if !keyFound || effect != taint.Effect {
				workers = append(workers, node)
			}
		}
	}
	return workers, nil
}

// GetAllClusterResources goes through all cluster nodes and sums their allocatable resources.
func (k *Kubernetes) GetAllClusterResources(ctx context.Context, clusterType ClusterType, volumes *corev1.PersistentVolumeList) ( //nolint:nonamedreturns
	cpuMillis uint64, memoryBytes uint64, diskSizeBytes uint64, err error,
) {
	nodes, err := k.GetWorkerNodes(ctx)
	if err != nil {
		return 0, 0, 0, errors.Wrap(err, "could not get a list of nodes")
	}
	var volumeCountEKS uint64
	for _, node := range nodes {
		cpu, memory, err := getResources(node.Status.Allocatable)
		if err != nil {
			return 0, 0, 0, errors.Wrap(err, "could not get allocatable resources of the node")
		}
		cpuMillis += cpu
		memoryBytes += memory

		switch clusterType {
		case ClusterTypeUnknown:
			return 0, 0, 0, errors.Errorf("unknown cluster type")
		case ClusterTypeGeneric:
			// TODO support other cluster types
			continue
		case ClusterTypeMinikube:
			storage, ok := node.Status.Allocatable[corev1.ResourceEphemeralStorage]
			if !ok {
				return 0, 0, 0, errors.Errorf("could not get storage size of the node")
			}
			bytes, err := convertors.StrToBytes(storage.String())
			if err != nil {
				return 0, 0, 0, errors.Wrapf(err, "could not convert storage size '%s' to bytes", storage.String())
			}
			diskSizeBytes += bytes
		case ClusterTypeEKS:
			// See https://kubernetes.io/docs/tasks/administer-cluster/out-of-resource/#scheduler.
			if IsNodeInCondition(node, corev1.NodeDiskPressure) {
				continue
			}

			// Get nodes's type.
			nodeType, ok := node.Labels["beta.kubernetes.io/instance-type"]
			if !ok {
				return 0, 0, 0, errors.New("dealing with AWS EKS cluster but the node does not have label 'beta.kubernetes.io/instance-type'")
			}
			// 39 is a default limit for EKS cluster nodes ...
			volumeLimitPerNode := uint64(39)
			typeAndSize := strings.Split(strings.ToLower(nodeType), ".")
			if len(typeAndSize) < 2 {
				return 0, 0, 0, errors.Errorf("failed to parse EKS node type '%s', it's not in expected format 'type.size'", nodeType)
			}
			// ... however, if the node type is one of M5, C5, R5, T3, Z1D it's 25.
			limitedVolumesSet := map[string]struct{}{
				"m5": {}, "c5": {}, "r5": {}, "t3": {}, "t1d": {},
			}
			if _, ok := limitedVolumesSet[typeAndSize[0]]; ok {
				volumeLimitPerNode = 25
			}
			volumeCountEKS += volumeLimitPerNode
		}
	}
	if clusterType == ClusterTypeEKS {
		volumeCountEKSBackup := volumeCountEKS
		volumeCountEKS -= uint64(len(volumes.Items))
		if volumeCountEKS > volumeCountEKSBackup {
			// handle uint underflow
			volumeCountEKS = 0
		}

		consumedBytes, err := sumVolumesSize(volumes)
		if err != nil {
			return 0, 0, 0, errors.Wrap(err, "failed to sum persistent volumes storage sizes")
		}
		diskSizeBytes = (volumeCountEKS * maxVolumeSizeEBS) + consumedBytes
	}
	return cpuMillis, memoryBytes, diskSizeBytes, nil
}

// getResources extracts resources out of corev1.ResourceList and converts them to int64 values.
// Millicpus are used for CPU values and bytes for memory.
func getResources(resources corev1.ResourceList) (cpuMillis uint64, memoryBytes uint64, err error) { //nolint:nonamedreturns
	cpu, ok := resources[corev1.ResourceCPU]
	if ok {
		cpuMillis, err = convertors.StrToMilliCPU(cpu.String())
		if err != nil {
			return 0, 0, errors.Wrapf(err, "failed to convert '%s' to millicpus", cpu.String())
		}
	}
	memory, ok := resources[corev1.ResourceMemory]
	if ok {
		memoryBytes, err = convertors.StrToBytes(memory.String())
		if err != nil {
			return 0, 0, errors.Wrapf(err, "failed to convert '%s' to bytes", memory.String())
		}
	}
	return cpuMillis, memoryBytes, nil
}

// GetConsumedCPUAndMemory returns consumed CPU and Memory in given namespace. If namespace
// is empty, it tries to get them from all namespaces.
func (k *Kubernetes) GetConsumedCPUAndMemory(ctx context.Context, namespace string) ( //nolint:nonamedreturns
	cpuMillis uint64, memoryBytes uint64, err error,
) {
	// Get CPU and Memory Requests of Pods' containers.
	pods, err := k.GetPods(ctx, namespace, nil)
	if err != nil {
		return 0, 0, errors.Wrap(err, "failed to get consumed resources")
	}
	for _, ppod := range pods.Items {
		if ppod.Status.Phase != corev1.PodRunning {
			continue
		}
		nonTerminatedInitContainers := make([]corev1.Container, 0, len(ppod.Spec.InitContainers))
		for _, container := range ppod.Spec.InitContainers {
			if !IsContainerInState(
				ppod.Status.InitContainerStatuses, ContainerStateTerminated,
			) {
				nonTerminatedInitContainers = append(nonTerminatedInitContainers, container)
			}
		}
		for _, container := range append(ppod.Spec.Containers, nonTerminatedInitContainers...) {
			cpu, memory, err := getResources(container.Resources.Requests)
			if err != nil {
				return 0, 0, errors.Wrap(err, "failed to sum all consumed resources")
			}
			cpuMillis += cpu
			memoryBytes += memory
		}
	}

	return cpuMillis, memoryBytes, nil
}

// GetConsumedDiskBytes returns consumed bytes. The strategy differs based on k8s cluster type.
func (k *Kubernetes) GetConsumedDiskBytes(ctx context.Context, clusterType ClusterType, volumes *corev1.PersistentVolumeList) (consumedBytes uint64, err error) { //nolint:lll,nonamedreturns
	switch clusterType {
	case ClusterTypeUnknown:
		return 0, errors.Errorf("unknown cluster type")
	case ClusterTypeGeneric:
		// TODO support other cluster types
		return 0, nil
	case ClusterTypeMinikube:
		nodes, err := k.GetWorkerNodes(ctx)
		if err != nil {
			return 0, errors.Wrap(err, "can't compute consumed disk size: failed to get worker nodes")
		}
		clientConfig, err := clientcmd.NewClientConfigFromBytes([]byte(k.kubeconfig))
		if err != nil {
			return 0, errors.Wrap(err, "failed to build kubeconfig out of given path")
		}
		config, err := clientConfig.ClientConfig()
		if err != nil {
			return 0, errors.Wrap(err, "failed to build kubeconfig out of given path")
		}
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			return 0, errors.Wrap(err, "failed to build client out of submited kubeconfig")
		}
		for _, node := range nodes {
			var summary NodeSummary
			request := clientset.CoreV1().RESTClient().Get().Resource("nodes").Name(node.Name).SubResource("proxy").Suffix("stats/summary")
			responseRawArrayOfBytes, err := request.DoRaw(context.Background()) //nolint:contextcheck
			if err != nil {
				return 0, errors.Wrap(err, "failed to get stats from node")
			}
			if err := json.Unmarshal(responseRawArrayOfBytes, &summary); err != nil {
				return 0, errors.Wrap(err, "failed to unmarshal response from kubernetes API")
			}
			consumedBytes += summary.Node.FileSystem.UsedBytes
		}
		return consumedBytes, nil
	case ClusterTypeEKS:
		consumedBytes, err := sumVolumesSize(volumes)
		if err != nil {
			return 0, errors.Wrap(err, "failed to sum persistent volumes storage sizes")
		}
		return consumedBytes, nil
	}

	return 0, nil
}

// sumVolumesSize returns sum of persistent volumes storage size in bytes.
func sumVolumesSize(pvs *corev1.PersistentVolumeList) (sum uint64, err error) { //nolint:nonamedreturns
	for _, pv := range pvs.Items {
		bytes, err := convertors.StrToBytes(pv.Spec.Capacity.Storage().String())
		if err != nil {
			return 0, err
		}
		sum += bytes
	}
	return
}

// GetPersistentVolumes returns list of persistent volumes.
func (k *Kubernetes) GetPersistentVolumes(ctx context.Context) (*corev1.PersistentVolumeList, error) {
	return k.client.GetPersistentVolumes(ctx)
}

// GetStorageClasses returns all storage classes available in the cluster.
func (k *Kubernetes) GetStorageClasses(ctx context.Context) (*storagev1.StorageClassList, error) {
	return k.client.GetStorageClasses(ctx)
}

// InstallOLMOperator installs the OLM in the Kubernetes cluster.
func (k *Kubernetes) InstallOLMOperator(ctx context.Context) error {
	deployment, err := k.client.GetDeployment(ctx, "olm-operator")
	if err == nil && deployment != nil && deployment.ObjectMeta.Name != "" {
		return nil // already installed
	}

	var crdFile, olmFile, perconaCatalog []byte

	crdFile, err = fs.ReadFile(data.OLMCRDs, "crds/olm/crds.yaml")
	if err != nil {
		return errors.Wrapf(err, "failed to read OLM CRDs file")
	}

	if err := k.client.ApplyFile(crdFile); err != nil {
		return errors.Wrapf(err, "cannot apply %q file", crdFile)
	}

	olmFile, err = fs.ReadFile(data.OLMCRDs, "crds/olm/olm.yaml")
	if err != nil {
		return errors.Wrapf(err, "failed to read OLM file")
	}

	if err := k.client.ApplyFile(olmFile); err != nil {
		return errors.Wrapf(err, "cannot apply %q file", crdFile)
	}

	perconaCatalog, err = fs.ReadFile(data.OLMCRDs, "crds/olm/percona-dbaas-catalog.yaml")
	if err != nil {
		return errors.Wrapf(err, "failed to read percona catalog yaml file")
	}

	if err := k.client.ApplyFile(perconaCatalog); err != nil {
		return errors.Wrapf(err, "cannot apply %q file", crdFile)
	}

	if err := k.client.DoRolloutWait(ctx, types.NamespacedName{Namespace: olmNamespace, Name: "olm-operator"}); err != nil {
		return errors.Wrap(err, "error while waiting for deployment rollout")
	}
	if err := k.client.DoRolloutWait(ctx, types.NamespacedName{Namespace: "olm", Name: "catalog-operator"}); err != nil {
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

	resources := append(crdResources, olmResources...) //nolint:gocritic

	subscriptions := filterResources(resources, func(r unstructured.Unstructured) bool {
		return r.GroupVersionKind() == schema.GroupVersionKind{
			Group:   v1alpha1.GroupName,
			Version: v1alpha1.GroupVersion,
			Kind:    v1alpha1.SubscriptionKind,
		}
	})

	for _, sub := range subscriptions {
		subscriptionKey := types.NamespacedName{Namespace: sub.GetNamespace(), Name: sub.GetName()}
		log.Printf("Waiting for subscription/%s to install CSV", subscriptionKey.Name)
		csvKey, err := k.client.GetSubscriptionCSV(ctx, subscriptionKey)
		if err != nil {
			return fmt.Errorf("subscription/%s failed to install CSV: %w", subscriptionKey.Name, err)
		}
		log.Printf("Waiting for clusterserviceversion/%s to reach 'Succeeded' phase", csvKey.Name)
		if err := k.client.DoCSVWait(ctx, csvKey); err != nil {
			return fmt.Errorf("clusterserviceversion/%s failed to reach 'Succeeded' phase", csvKey.Name)
		}
	}

	if err := k.client.DoRolloutWait(ctx, types.NamespacedName{Namespace: "olm", Name: "packageserver"}); err != nil {
		return errors.Wrap(err, "error while waiting for deployment rollout")
	}

	return nil
}

func decodeResources(f []byte) ([]unstructured.Unstructured, error) {
	dec := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(f), 8)
	var objs []unstructured.Unstructured

	for {
		var u unstructured.Unstructured
		err := dec.Decode(&u)
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, err
		}
		objs = append(objs, u)
	}

	return objs, nil
}

func filterResources( //nolint:nonamedreturns
	resources []unstructured.Unstructured,
	filter func(unstructured.Unstructured) bool,
) (filtered []unstructured.Unstructured) {
	for _, r := range resources {
		if filter(r) {
			filtered = append(filtered, r)
		}
	}
	return filtered
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
func (k *Kubernetes) InstallOperator(ctx context.Context, req InstallOperatorRequest) error {
	if err := createOperatorGroupIfNeeded(ctx, k.client, req.OperatorGroup); err != nil {
		return err
	}

	subs, err := k.client.CreateSubscriptionForCatalog(ctx, req.Namespace, req.Name, "olm", req.CatalogSource,
		req.Name, req.Channel, req.StartingCSV, v1alpha1.ApprovalManual)
	if err != nil {
		return errors.Wrap(err, "cannot create a susbcription to install the operator")
	}

	err = wait.PollUntilContextTimeout(ctx, pollInterval, pollDuration, false, func(context.Context) (bool, error) {
		k.lock.Lock()
		defer k.lock.Unlock()

		subs, err = k.client.GetSubscription(ctx, req.Namespace, req.Name)
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

	ip, err := k.client.GetInstallPlan(ctx, req.Namespace, subs.Status.Install.Name)
	if err != nil {
		return err
	}

	ip.Spec.Approved = true
	_, err = k.client.UpdateInstallPlan(ctx, req.Namespace, ip)

	return err
}

func createOperatorGroupIfNeeded(ctx context.Context, client client.KubeClientConnector, name string) error {
	_, err := client.GetOperatorGroup(ctx, useDefaultNamespace, name)
	if err == nil {
		return nil
	}

	_, err = client.CreateOperatorGroup(ctx, "default", name)

	return err
}

// ListSubscriptions all the subscriptions in the namespace.
func (k *Kubernetes) ListSubscriptions(ctx context.Context, namespace string) (*v1alpha1.SubscriptionList, error) {
	return k.client.ListSubscriptions(ctx, namespace)
}

// UpgradeOperator upgrades an operator to the next available version.
func (k *Kubernetes) UpgradeOperator(ctx context.Context, namespace, name string) error {
	var subs *v1alpha1.Subscription

	// If the subscription was recently created, the install plan might not be ready yet.
	err := wait.PollUntilContextTimeout(ctx, pollInterval, pollDuration, false, func(context.Context) (bool, error) {
		var err error
		subs, err = k.client.GetSubscription(ctx, namespace, name)
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

	ip, err := k.client.GetInstallPlan(ctx, namespace, subs.Status.Install.Name)
	if err != nil {
		return errors.Wrapf(err, "cannot get install plan to upgrade %q", name)
	}

	if ip.Spec.Approved {
		return nil // There are no upgrades.
	}

	ip.Spec.Approved = true

	_, err = k.client.UpdateInstallPlan(ctx, namespace, ip)

	return err
}

// GetServerVersion returns server version.
func (k *Kubernetes) GetServerVersion() (*version.Info, error) {
	return k.client.GetServerVersion()
}

// ListTemplates returns a list of templates.
func (k *Kubernetes) ListTemplates(ctx context.Context, engine, namespace string) ([]*dbaasv1beta1.Template, error) {
	k.lock.RLock()
	defer k.lock.RUnlock()

	labelSelector := &metav1.LabelSelector{
		MatchLabels: map[string]string{
			templateLabelKey: "yes",
			engineLabelKey:   engine,
		},
	}

	templateCRDs, err := k.client.ListCRDs(ctx, labelSelector)
	if err != nil {
		return nil, errors.Wrap(err, "failed listing template CRDs")
	}

	templates := []*dbaasv1beta1.Template{}
	for _, templateCRD := range templateCRDs.Items {
		var storedVersionName string
		for _, version := range templateCRD.Spec.Versions {
			if version.Storage {
				storedVersionName = version.Name
				break
			}
		}
		// XXX: logically we should check that storedVersionName has been set and
		// return an error otherwise but according to the
		// CustomResourceDefinitionVersion documentation
		// "There must be exactly one version with storage=true." so we are sure
		// that storedVersionName will be set. If for some reason it's not, it will
		// fail to find the CRs so an error will be returned either way.
		gvr := schema.GroupVersionResource{
			Group:    templateCRD.Spec.Group,
			Version:  storedVersionName,
			Resource: templateCRD.Spec.Names.Plural,
		}

		templateCRs, err := k.client.ListCRs(ctx, namespace, gvr, labelSelector)
		if err != nil {
			return nil, errors.Wrap(err, "failed listing template CRs")
		}

		for _, templateCR := range templateCRs.Items {
			//nolint:forcetypeassert
			templates = append(templates, &dbaasv1beta1.Template{
				Name: templateCR.Object["metadata"].(map[string]interface{})["name"].(string),
				Kind: templateCR.Object["kind"].(string),
			})
		}
	}

	return templates, nil
}
