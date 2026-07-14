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

package dbaas

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	dbaascontrollerv1beta1 "github.com/percona-platform/dbaas-api/gen/controller"
	"github.com/percona/promconfig"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"

	dbaasv1beta1 "github.com/percona/pmm/api/managementpb/dbaas"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/dbaas/kubernetes"
	pmmversion "github.com/percona/pmm/version"
)

var (
	operatorIsForbiddenRegexp          = regexp.MustCompile(`.*\.percona\.com is forbidden`)
	resourceDoesntExistsRegexp         = regexp.MustCompile(`the server doesn't have a resource type "(PerconaXtraDBCluster|PerconaServerMongoDB)"`)
	errKubeconfigIsEmpty               = errors.New("kubeconfig is empty")
	errMissingRequiredKubeconfigEnvVar = errors.New("required environment variable is not defined in kubeconfig")

	flagClusterName              = "--cluster-name"
	flagRegion                   = "--region"
	flagRole                     = "--role-arn"
	kubeconfigFlagsConversionMap = map[string]string{flagClusterName: "-i", flagRegion: "--region", flagRole: "-r"}
	kubeconfigFlagsList          = []string{flagClusterName, flagRegion, flagRole}
)

type kubernetesServer struct {
	l              *logrus.Entry
	db             *reform.DB
	dbaasClient    dbaasClient
	versionService versionService
	grafanaClient  grafanaClient
	kubeStorage    *KubeStorage

	dbaasv1beta1.UnimplementedKubernetesServer
}

// NewKubernetesServer creates Kubernetes Server.
func NewKubernetesServer(db *reform.DB, dbaasClient dbaasClient, versionService versionService, //nolint:ireturn,nolintlint
	grafanaClient grafanaClient,
) dbaasv1beta1.KubernetesServer {
	l := logrus.WithField("component", "kubernetes_server")
	return &kubernetesServer{
		l:              l,
		db:             db,
		dbaasClient:    dbaasClient,
		versionService: versionService,
		grafanaClient:  grafanaClient,
		kubeStorage:    NewKubeStorage(db),
	}
}

// Enabled returns if service is enabled and can be used.
func (k *kubernetesServer) Enabled() bool {
	settings, err := models.GetSettings(k.db)
	if err != nil {
		k.l.WithError(err).Error("can't get settings")
		return false
	}
	return settings.DBaaS.Enabled
}

// convertToOperatorStatus exists mainly to provide an appropriate status when installed operator is unsupported.
// Dbaas-controller does not have a clue what's supported, so we have to do it here.
func (k kubernetesServer) convertToOperatorStatus(versionsList []string, operatorVersion string) dbaasv1beta1.OperatorsStatus {
	if operatorVersion == "" {
		return dbaasv1beta1.OperatorsStatus_OPERATORS_STATUS_NOT_INSTALLED
	}
	for _, version := range versionsList {
		if version == operatorVersion {
			return dbaasv1beta1.OperatorsStatus_OPERATORS_STATUS_OK
		}
	}

	allowUnsupportedOperators := os.Getenv("DBAAS_ALLOW_UNSUPPORTED_OPERATORS")
	if boolValue, _ := strconv.ParseBool(allowUnsupportedOperators); boolValue {
		return dbaasv1beta1.OperatorsStatus_OPERATORS_STATUS_OK
	}

	return dbaasv1beta1.OperatorsStatus_OPERATORS_STATUS_UNSUPPORTED
}

// ListKubernetesClusters returns a list of all registered Kubernetes clusters.
func (k kubernetesServer) ListKubernetesClusters(ctx context.Context, _ *dbaasv1beta1.ListKubernetesClustersRequest) (*dbaasv1beta1.ListKubernetesClustersResponse, error) { //nolint:lll
	kubernetesClusters, err := models.FindAllKubernetesClusters(k.db.Querier)
	if err != nil {
		return nil, err
	}
	if len(kubernetesClusters) == 0 {
		return &dbaasv1beta1.ListKubernetesClustersResponse{}, nil
	}

	operatorsVersions, err := k.versionService.SupportedOperatorVersionsList(ctx, pmmversion.PMMVersion)
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	clusters := make([]*dbaasv1beta1.ListKubernetesClustersResponse_Cluster, len(kubernetesClusters))
	for i, cluster := range kubernetesClusters {
		i := i
		cluster := cluster
		wg.Add(1)
		go func(cluster *models.KubernetesCluster) {
			defer wg.Done()
			clusters[i] = &dbaasv1beta1.ListKubernetesClustersResponse_Cluster{
				KubernetesClusterName: cluster.KubernetesClusterName,
				Operators: &dbaasv1beta1.Operators{
					Pxc:   &dbaasv1beta1.Operator{},
					Psmdb: &dbaasv1beta1.Operator{},
					Dbaas: &dbaasv1beta1.Operator{},
				},
			}
			kubeClient, err := k.kubeStorage.GetOrSetClient(cluster.KubernetesClusterName)
			if err != nil {
				clusters[i].Status = dbaasv1beta1.KubernetesClusterStatus_KUBERNETES_CLUSTER_STATUS_UNAVAILABLE
				return
			}

			clusters[i].Status = dbaasv1beta1.KubernetesClusterStatus_KUBERNETES_CLUSTER_STATUS_OK
			if !cluster.IsReady {
				clusters[i].Status = dbaasv1beta1.KubernetesClusterStatus_KUBERNETES_CLUSTER_STATUS_PROVISIONING
			}
			pxcVersion, err := kubeClient.GetPXCOperatorVersion(ctx)
			if err != nil {
				k.l.Errorf("couldn't get pxc operator version: %s", err)
			}
			psmdbVersion, err := kubeClient.GetPSMDBOperatorVersion(ctx)
			if err != nil {
				k.l.Errorf("couldn't get psmdb operator version: %s", err)
			}

			clusters[i].Operators.Pxc.Status = k.convertToOperatorStatus(operatorsVersions[pxcOperator], pxcVersion)
			clusters[i].Operators.Psmdb.Status = k.convertToOperatorStatus(operatorsVersions[psmdbOperator], psmdbVersion)

			clusters[i].Operators.Pxc.Version = pxcVersion
			clusters[i].Operators.Psmdb.Version = psmdbVersion

			// FIXME: Uncomment it when FE will be ready
			// kubeClient, err := kubernetes.New(cluster.KubeConfig)
			// if err != nil {
			// 	return
			// }
			// version, err := kubeClient.GetDBaaSOperatorVersion(ctx)
			// if err != nil {
			//   return
			// }
			// clusters[i].Operators.Dbaas.Version = version
			// clusters[i].Operators.Dbaas.Status = dbaasv1beta1.OperatorsStatus_OPERATORS_STATUS_OK
		}(cluster)
	}
	wg.Wait()
	return &dbaasv1beta1.ListKubernetesClustersResponse{KubernetesClusters: clusters}, nil
}

type envVar struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

type kubectlUserExec struct {
	APIVersion         string   `yaml:"apiVersion,omitempty"`
	Args               []string `yaml:"args,omitempty"`
	Command            string   `yaml:"command,omitempty"`
	Env                []envVar `yaml:"env,omitempty"`
	ProvideClusterInfo bool     `yaml:"provideClusterInfo"`
}

type kubectlUser struct {
	ClientCertificateData string          `yaml:"client-certificate-data,omitempty"`
	ClientKeyData         string          `yaml:"client-key-data,omitempty"`
	Exec                  kubectlUserExec `yaml:"exec,omitempty"`
}

type kubectlUserWithName struct {
	Name string       `yaml:"name,omitempty"`
	User *kubectlUser `yaml:"user,omitempty"`
}

type kubectlConfig struct {
	Kind           string                 `yaml:"kind,omitempty"`
	APIVersion     string                 `yaml:"apiVersion,omitempty"`
	CurrentContext string                 `yaml:"current-context,omitempty"`
	Clusters       []interface{}          `yaml:"clusters,omitempty"`
	Contexts       []interface{}          `yaml:"contexts,omitempty"`
	Preferences    map[string]interface{} `yaml:"preferences"`
	Users          []*kubectlUserWithName `yaml:"users,omitempty"`
}

func (k *kubectlConfig) maskSecrets() (*kubectlConfig, error) {
	nk, ok := promconfig.Copy(k).(*kubectlConfig)
	if !ok {
		return nil, errors.New("failed to copy config")
	}

	for i, user := range nk.Users {
		for j, env := range user.User.Exec.Env {
			if env.Name == "AWS_ACCESS_KEY_ID" || env.Name == "AWS_SECRET_ACCESS_KEY" {
				nk.Users[i].User.Exec.Env[j].Value = "<secret>"
			}
		}
	}
	return nk, nil
}

func getFlagValue(args []string, flagName string) string {
	for i, arg := range args {
		if arg == flagName && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

func getKubeconfigUserExecEnvValue(envs []envVar, variableName string) string {
	for _, env := range envs {
		if name := env.Name; name == variableName {
			return env.Value
		}
	}
	return ""
}

// replaceAWSAuthIfPresent replaces use of aws binary with aws-iam-authenticator if use of aws binary is found.
// If such use is not found, it returns passed kubeconfig without any change.
func replaceAWSAuthIfPresent(kubeconfig string, keyID, key string) (string, error) {
	if strings.TrimSpace(kubeconfig) == "" {
		return "", errKubeconfigIsEmpty
	}
	var config kubectlConfig
	err := yaml.Unmarshal([]byte(kubeconfig), &config)
	if err != nil {
		return "", err
	}
	var changed bool
	for _, user := range config.Users {
		if user.User.Exec.Command == "aws" {
			user.User.Exec.Command = "aws-iam-authenticator"
			// check and set flags
			converted := []string{"token"}
			for _, oldFlag := range kubeconfigFlagsList {
				if flag := getFlagValue(user.User.Exec.Args, oldFlag); flag != "" {
					converted = append(converted, kubeconfigFlagsConversionMap[oldFlag], flag)
				}
			}
			user.User.Exec.Args = converted
			changed = true
		}

		// check and set authentication environment variables
		for _, envVar := range []envVar{{"AWS_ACCESS_KEY_ID", keyID}, {"AWS_SECRET_ACCESS_KEY", key}} {
			if value := getKubeconfigUserExecEnvValue(user.User.Exec.Env, envVar.Name); value == "" && envVar.Value != "" {
				user.User.Exec.Env = append(user.User.Exec.Env, envVar)
				changed = true
			}
		}
	}
	if !changed {
		return kubeconfig, nil
	}
	c, err := yaml.Marshal(config)
	return string(c), err
}

// RegisterKubernetesCluster registers an existing Kubernetes cluster in PMM.
func (k kubernetesServer) RegisterKubernetesCluster(ctx context.Context, req *dbaasv1beta1.RegisterKubernetesClusterRequest) (*dbaasv1beta1.RegisterKubernetesClusterResponse, error) { //nolint:lll
	var err error
	req.KubeAuth.Kubeconfig, err = replaceAWSAuthIfPresent(req.KubeAuth.Kubeconfig, req.AwsAccessKeyId, req.AwsSecretAccessKey)
	if err != nil {
		if errors.Is(err, errKubeconfigIsEmpty) {
			return nil, status.Error(codes.InvalidArgument, "Kubeconfig can't be empty")
		} else if errors.Is(err, errMissingRequiredKubeconfigEnvVar) {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("Failed to transform kubeconfig to work with aws-iam-authenticator: %s", err))
		}
		k.l.Errorf("Replacing `aws` with `aim-authenticator` failed: %s", err)
		return nil, status.Error(codes.Internal, "Internal server error")
	}

	err = k.db.InTransaction(func(t *reform.TX) error {
		_, err := models.CreateKubernetesCluster(t.Querier, &models.CreateKubernetesClusterParams{
			KubernetesClusterName: req.KubernetesClusterName,
			KubeConfig:            req.KubeAuth.Kubeconfig,
		})
		return err
	})
	if err != nil {
		return nil, err
	}
	kubeClient, err := k.kubeStorage.GetOrSetClient(req.KubernetesClusterName)
	if err != nil {
		return nil, err
	}

	operatorsToInstall := make(map[string]bool)
	operatorsToInstall["olm"] = true
	operatorsToInstall["vm"] = true
	operatorsToInstall["dbaas"] = true
	if pxcVersion, err := kubeClient.GetPXCOperatorVersion(ctx); pxcVersion == "" || err != nil {
		operatorsToInstall["pxc"] = true
	}
	if psmdbVersion, err := kubeClient.GetPSMDBOperatorVersion(ctx); psmdbVersion == "" || err != nil {
		operatorsToInstall["psmdb"] = true
	}
	settings, err := models.GetSettings(k.db.Querier)
	if err != nil {
		return nil, errors.Wrap(err, "cannot get PMM settings to start Victoria Metrics")
	}
	var apiKeyID int64
	var apiKey string
	apiKeyName := fmt.Sprintf("pmm-vmagent-%s-%d", req.KubernetesClusterName, rand.Int63()) //nolint:gosec
	apiKeyID, apiKey, err = k.grafanaClient.CreateAdminAPIKey(ctx, apiKeyName)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create Grafana admin API key")
	}

	go k.setupMonitoring( //nolint:contextcheck
		context.TODO(),
		operatorsToInstall,
		req.KubernetesClusterName,
		req.KubeAuth.Kubeconfig,
		settings.PMMPublicAddress,
		apiKey,
		apiKeyID)

	return &dbaasv1beta1.RegisterKubernetesClusterResponse{}, nil
}

func (k kubernetesServer) setupMonitoring(ctx context.Context, operatorsToInstall map[string]bool, clusterName, kubeConfig, pmmPublicAddress string,
	apiKey string, apiKeyID int64,
) {
	kubeClient, err := k.kubeStorage.GetOrSetClient(clusterName)
	if err != nil {
		return
	}
	errs := k.installDefaultOperators(operatorsToInstall, kubeClient) //nolint:contextcheck
	if errs["vm"] != nil {
		k.l.Errorf("cannot install vm operator: %s", errs["vm"])
		return
	}

	err = k.startMonitoring(ctx, pmmPublicAddress, apiKey, apiKeyID, kubeConfig)
	if err != nil {
		k.l.Errorf("cannot start monitoring the clusdter: %s", err)
	}

	err = models.ChangeKubernetesClusterToReady(k.db.Querier, clusterName)
	if err != nil {
		k.l.Errorf("couldn't update kubernetes cluster state: %s", err)
	}
}

func (k kubernetesServer) startMonitoring(ctx context.Context, pmmPublicAddress string, apiKey string,
	apiKeyID int64, kubeConfig string,
) error {
	pmmParams := &dbaascontrollerv1beta1.PMMParams{
		PublicAddress: fmt.Sprintf("https://%s", pmmPublicAddress),
		Login:         "api_key",
		Password:      apiKey,
	}

	_, err := k.dbaasClient.StartMonitoring(ctx, &dbaascontrollerv1beta1.StartMonitoringRequest{
		KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
			Kubeconfig: kubeConfig,
		},
		Pmm: pmmParams,
	})
	if err != nil {
		e := k.grafanaClient.DeleteAPIKeyByID(ctx, apiKeyID)
		if e != nil {
			k.l.Warnf("couldn't delete created API Key %v: %s", apiKeyID, e)
		}
		k.l.Errorf("couldn't start monitoring of the kubernetes cluster: %s", err)
		return errors.Wrap(err, "couldn't start monitoring of the kubernetes cluster")
	}

	return nil
}

func (k kubernetesServer) installDefaultOperators(operatorsToInstall map[string]bool, kubeClient kubernetesClient) map[string]error {
	ctx := context.TODO()

	retval := make(map[string]error)

	if _, ok := operatorsToInstall["olm"]; ok {
		err := kubeClient.InstallOLMOperator(ctx)
		if err != nil {
			retval["olm"] = err
			k.l.Errorf("cannot install OLM operator to register the Kubernetes cluster: %s", err)
		}
	}

	namespace := "default"
	catalogSourceNamespace := "olm"
	operatorGroup := "percona-operators-group"
	catalogSource := "percona-dbaas-catalog"

	if _, ok := operatorsToInstall["vm"]; ok {
		channel, ok := os.LookupEnv("DBAAS_VM_OP_CHANNEL")
		if !ok || channel == "" {
			channel = "stable-v0"
		}
		operatorName := "victoriametrics-operator"
		params := kubernetes.InstallOperatorRequest{
			Namespace:              namespace,
			Name:                   operatorName,
			OperatorGroup:          operatorGroup,
			CatalogSource:          catalogSource,
			CatalogSourceNamespace: catalogSourceNamespace,
			Channel:                channel,
			InstallPlanApproval:    v1alpha1.ApprovalManual,
		}

		if err := kubeClient.InstallOperator(ctx, params); err != nil {
			retval["vm"] = err
			k.l.Errorf("cannot instal PXC operator in the new cluster: %s", err)
		}
	}

	if _, ok := operatorsToInstall["pxc"]; ok {
		channel, ok := os.LookupEnv("DBAAS_PXC_OP_CHANNEL")
		if !ok || channel == "" {
			channel = "stable-v1"
		}
		operatorName := "percona-xtradb-cluster-operator"
		params := kubernetes.InstallOperatorRequest{
			Namespace:              namespace,
			Name:                   operatorName,
			OperatorGroup:          operatorGroup,
			CatalogSource:          catalogSource,
			CatalogSourceNamespace: catalogSourceNamespace,
			Channel:                channel,
			InstallPlanApproval:    v1alpha1.ApprovalManual,
		}

		if err := kubeClient.InstallOperator(ctx, params); err != nil {
			retval["pxc"] = err
			k.l.Errorf("cannot instal PXC operator in the new cluster: %s", err)
		}
	}

	if _, ok := operatorsToInstall["psmdb"]; ok {
		operatorName := "percona-server-mongodb-operator"
		channel, ok := os.LookupEnv("DBAAS_PSMDB_OP_CHANNEL")
		if !ok || channel == "" {
			channel = "stable-v1"
		}
		params := kubernetes.InstallOperatorRequest{
			Namespace:              namespace,
			Name:                   operatorName,
			OperatorGroup:          operatorGroup,
			CatalogSource:          catalogSource,
			CatalogSourceNamespace: catalogSourceNamespace,
			Channel:                channel,
			InstallPlanApproval:    v1alpha1.ApprovalManual,
		}

		if err := kubeClient.InstallOperator(ctx, params); err != nil {
			retval["psmdb"] = err
			k.l.Errorf("cannot instal PXC operator in the new cluster: %s", err)
		}
	}

	if _, ok := operatorsToInstall["dbaas"]; ok {
		operatorName := "dbaas-operator"
		channel, ok := os.LookupEnv("DBAAS_DBAAS_OP_CHANNEL")
		if !ok || channel == "" {
			channel = "stable-v0"
		}
		params := kubernetes.InstallOperatorRequest{
			Namespace:              namespace,
			Name:                   operatorName,
			OperatorGroup:          operatorGroup,
			CatalogSource:          "percona-dbaas-catalog",
			CatalogSourceNamespace: catalogSourceNamespace,
			Channel:                channel,
			InstallPlanApproval:    v1alpha1.ApprovalManual,
		}

		if err := kubeClient.InstallOperator(ctx, params); err != nil {
			retval["vm"] = err
			k.l.Errorf("cannot instal PXC operator in the new cluster: %s", err)
		}
	}

	return retval
}

// UnregisterKubernetesCluster removes a registered Kubernetes cluster from PMM.
func (k kubernetesServer) UnregisterKubernetesCluster(ctx context.Context, req *dbaasv1beta1.UnregisterKubernetesClusterRequest) (*dbaasv1beta1.UnregisterKubernetesClusterResponse, error) { //nolint:lll
	err := k.db.InTransaction(func(t *reform.TX) error {
		kubernetesCluster, err := models.FindKubernetesClusterByName(t.Querier, req.KubernetesClusterName)
		if err != nil {
			return err
		}

		_, err = k.dbaasClient.StopMonitoring(ctx, &dbaascontrollerv1beta1.StopMonitoringRequest{
			KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
				Kubeconfig: kubernetesCluster.KubeConfig,
			},
		})
		if err != nil {
			k.l.Warnf("cannot stop monitoring: %s", err)
		}
		if req.Force {
			return models.RemoveKubernetesCluster(t.Querier, req.KubernetesClusterName)
		}

		kubeClient, err := k.kubeStorage.GetOrSetClient(req.KubernetesClusterName)
		if err != nil {
			return err
		}

		out, err := kubeClient.ListDatabaseClusters(ctx)

		switch {
		case err != nil && accessError(err):
			k.l.Warn(err)
		case err != nil:
			return err
		case len(out.Items) != 0:
			return status.Errorf(codes.FailedPrecondition, "Kubernetes cluster %s has database clusters", req.KubernetesClusterName)
		}

		return models.RemoveKubernetesCluster(t.Querier, req.KubernetesClusterName)
	})
	if err != nil {
		return nil, err
	}

	return &dbaasv1beta1.UnregisterKubernetesClusterResponse{}, nil
}

// GetKubernetesCluster return KubeAuth with Kubernetes config.
func (k kubernetesServer) GetKubernetesCluster(_ context.Context, req *dbaasv1beta1.GetKubernetesClusterRequest) (*dbaasv1beta1.GetKubernetesClusterResponse, error) {
	kubernetesCluster, err := models.FindKubernetesClusterByName(k.db.Querier, req.KubernetesClusterName)
	if err != nil {
		return nil, err
	}
	config := &kubectlConfig{}
	err = yaml.Unmarshal([]byte(kubernetesCluster.KubeConfig), config)
	if err != nil {
		return nil, err
	}
	safeKubeConfig, err := config.maskSecrets()
	if err != nil {
		return nil, err
	}
	kubeConfig, err := yaml.Marshal(safeKubeConfig)
	if err != nil {
		return nil, err
	}

	return &dbaasv1beta1.GetKubernetesClusterResponse{
		KubeAuth: &dbaasv1beta1.KubeAuth{
			Kubeconfig: string(kubeConfig),
		},
	}, nil
}

func accessError(err error) bool {
	if err == nil {
		return false
	}
	accessErrors := []*regexp.Regexp{
		operatorIsForbiddenRegexp,
		resourceDoesntExistsRegexp,
	}

	for _, regex := range accessErrors {
		if regex.MatchString(err.Error()) {
			logrus.Warn(err.Error())
			return true
		}
	}
	return false
}

// GetResources returns all and available resources of a Kubernetes cluster.
func (k kubernetesServer) GetResources(ctx context.Context, req *dbaasv1beta1.GetResourcesRequest) (*dbaasv1beta1.GetResourcesResponse, error) {
	kubeClient, err := k.kubeStorage.GetOrSetClient(req.KubernetesClusterName)
	if err != nil {
		return nil, err
	}

	// Get cluster type
	clusterType, err := kubeClient.GetClusterType(ctx)
	if err != nil {
		return nil, err
	}

	var volumes *corev1.PersistentVolumeList
	if clusterType == kubernetes.ClusterTypeEKS {
		volumes, err = kubeClient.GetPersistentVolumes(ctx)
		if err != nil {
			return nil, err
		}
	}
	allCPUMillis, allMemoryBytes, allDiskBytes, err := kubeClient.GetAllClusterResources(ctx, clusterType, volumes)
	if err != nil {
		return nil, err
	}

	consumedCPUMillis, consumedMemoryBytes, err := kubeClient.GetConsumedCPUAndMemory(ctx, "")
	if err != nil {
		return nil, err
	}

	consumedDiskBytes, err := kubeClient.GetConsumedDiskBytes(ctx, clusterType, volumes)
	if err != nil {
		return nil, err
	}

	availableCPUMillis := allCPUMillis - consumedCPUMillis
	// handle underflow
	if availableCPUMillis > allCPUMillis {
		availableCPUMillis = 0
	}
	availableMemoryBytes := allMemoryBytes - consumedMemoryBytes
	// handle underflow
	if availableMemoryBytes > allMemoryBytes {
		availableMemoryBytes = 0
	}
	availableDiskBytes := allDiskBytes - consumedDiskBytes
	// handle underflow
	if availableDiskBytes > allDiskBytes {
		availableDiskBytes = 0
	}

	return &dbaasv1beta1.GetResourcesResponse{
		All: &dbaasv1beta1.Resources{
			CpuM:        allCPUMillis,
			MemoryBytes: allMemoryBytes,
			DiskSize:    allDiskBytes,
		},
		Available: &dbaasv1beta1.Resources{
			CpuM:        availableCPUMillis,
			MemoryBytes: availableMemoryBytes,
			DiskSize:    availableDiskBytes,
		},
	}, nil
}

// ListStorageClasses returns the names of all storage classes available in a Kubernetes cluster.
func (k kubernetesServer) ListStorageClasses(ctx context.Context, req *dbaasv1beta1.ListStorageClassesRequest) (*dbaasv1beta1.ListStorageClassesResponse, error) {
	kubeClient, err := k.kubeStorage.GetOrSetClient(req.KubernetesClusterName)
	if err != nil {
		return nil, err
	}

	storageClasses, err := kubeClient.GetStorageClasses(ctx)
	if err != nil {
		return nil, err
	}

	storageClassesNames := make([]string, 0, len(storageClasses.Items))
	for _, storageClass := range storageClasses.Items {
		storageClassesNames = append(storageClassesNames, storageClass.Name)
	}

	return &dbaasv1beta1.ListStorageClassesResponse{
		StorageClasses: storageClassesNames,
	}, nil
}
