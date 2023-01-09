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

package dbaas

import (
	"context"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"sync"
	"time"

	goversion "github.com/hashicorp/go-version"
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

const (
	catalogSourceNamespace = "olm"
	catalogSource          = "percona-dbaas-catalog"
)

var (
	operatorIsForbiddenRegexp          = regexp.MustCompile(`.*\.percona\.com is forbidden`)
	resourceDoesntExistsRegexp         = regexp.MustCompile(`the server doesn't have a resource type "(PerconaXtraDBCluster|PerconaServerMongoDB)"`)
	errKubeconfigIsEmpty               = errors.New("kubeconfig is empty")
	errMissingRequiredKubeconfigEnvVar = errors.New("required environment variable is not defined in kubeconfig")
	// errNoInstallPlanToApprove          = errors.New("there are no install plans to approve") TODO: @Carlos do we still need it?

	flagClusterName              = "--cluster-name"
	flagRegion                   = "--region"
	flagRole                     = "--role-arn"
	kubeconfigFlagsConversionMap = map[string]string{flagClusterName: "-i", flagRegion: "--region", flagRole: "-r"}
	kubeconfigFlagsList          = []string{flagClusterName, flagRegion, flagRole}
)

type kubernetesServer struct {
	l                *logrus.Entry
	db               *reform.DB
	dbaasClient      dbaasClient
	kubernetesClient kubernetesClient
	versionService   versionService
	grafanaClient    grafanaClient

	dbaasv1beta1.UnimplementedKubernetesServer
}

// NewKubernetesServer creates Kubernetes Server.
func NewKubernetesServer(db *reform.DB, dbaasClient dbaasClient,
	kubernetesClient kubernetesClient, versionService versionService,
	grafanaClient grafanaClient,
) dbaasv1beta1.KubernetesServer {
	l := logrus.WithField("component", "kubernetes_server")
	return &kubernetesServer{
		l:                l,
		db:               db,
		dbaasClient:      dbaasClient,
		kubernetesClient: kubernetesClient,
		versionService:   versionService,
		grafanaClient:    grafanaClient,
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

// getOperatorStatus exists mainly to assign appropriate status when installed operator is unsupported.
// dbaas-controller does not have a clue what's supported, so we have to do it here.
func (k kubernetesServer) convertToOperatorStatus(versionsList []string, operatorVersion string) dbaasv1beta1.OperatorsStatus {
	if operatorVersion == "" {
		return dbaasv1beta1.OperatorsStatus_OPERATORS_STATUS_NOT_INSTALLED
	}
	for _, version := range versionsList {
		if version == operatorVersion {
			return dbaasv1beta1.OperatorsStatus_OPERATORS_STATUS_OK
		}
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
			resp, e := k.dbaasClient.CheckKubernetesClusterConnection(ctx, cluster.KubeConfig)
			if e != nil {
				clusters[i].Status = dbaasv1beta1.KubernetesClusterStatus_KUBERNETES_CLUSTER_STATUS_UNAVAILABLE
				return
			}

			clusters[i].Status = dbaasv1beta1.KubernetesClusterStatus(resp.Status)
			if !cluster.IsReady {
				clusters[i].Status = dbaasv1beta1.KubernetesClusterStatus_KUBERNETES_CLUSTER_STATUS_PROVISIONING
			}
			if resp.Operators == nil {
				return
			}

			clusters[i].Operators.Pxc.Status = k.convertToOperatorStatus(operatorsVersions[pxcOperator], resp.Operators.PxcOperatorVersion)
			clusters[i].Operators.Psmdb.Status = k.convertToOperatorStatus(operatorsVersions[psmdbOperator], resp.Operators.PsmdbOperatorVersion)

			clusters[i].Operators.Pxc.Version = resp.Operators.PxcOperatorVersion
			clusters[i].Operators.Psmdb.Version = resp.Operators.PsmdbOperatorVersion

			// FIXME: Uncomment it when FE will be ready
			// kubeClient, err := kubernetes.New(cluster.KubeConfig)
			// if err != nil {
			// 	return
			// }
			//version, err := kubeClient.GetDBaaSOperatorVersion(ctx)
			//if err != nil {
			//	return
			//}
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

// TODO: @Carlos do we still need it?
// func installOLMOperator(ctx context.Context, client dbaasClient, kubeconfig, version string) error {
// 	installOLMOperatorReq := &dbaascontrollerv1beta1.InstallOLMOperatorRequest{
// 		KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
// 			Kubeconfig: kubeconfig,
// 		},
// 		Version: version,
// 	}
//
// 	if _, err := client.InstallOLMOperator(ctx, installOLMOperatorReq); err != nil {
// 		return errors.Wrap(err, "cannot install OLM operator")
// 	}
//
// 	return nil
// }

func approveInstallPlan(ctx context.Context, client dbaasClient, kubeConfig, namespace, name string) error { //nolint:unparam
	req := &dbaascontrollerv1beta1.ApproveInstallPlanRequest{
		KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
			Kubeconfig: kubeConfig,
		},
		Name:      name,
		Namespace: namespace,
	}
	_, err := client.ApproveInstallPlan(ctx, req)

	return err
}

// RegisterKubernetesCluster registers an existing Kubernetes cluster in PMM.
func (k kubernetesServer) RegisterKubernetesCluster(ctx context.Context, req *dbaasv1beta1.RegisterKubernetesClusterRequest) (*dbaasv1beta1.RegisterKubernetesClusterResponse, error) { //nolint:lll,cyclop
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

	var clusterInfo *dbaascontrollerv1beta1.CheckKubernetesClusterConnectionResponse
	err = k.db.InTransaction(func(t *reform.TX) error {
		var e error
		clusterInfo, e = k.dbaasClient.CheckKubernetesClusterConnection(ctx, req.KubeAuth.Kubeconfig)
		if e != nil {
			return e
		}

		_, err := models.CreateKubernetesCluster(t.Querier, &models.CreateKubernetesClusterParams{
			KubernetesClusterName: req.KubernetesClusterName,
			KubeConfig:            req.KubeAuth.Kubeconfig,
		})
		return err
	})
	if err != nil {
		return nil, err
	}

	pmmVersion, err := goversion.NewVersion(pmmversion.PMMVersion)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	pxcOperatorVersion, psmdbOperatorVersion, err := k.versionService.LatestOperatorVersion(ctx, pmmVersion.Core().String())
	if err != nil {
		return nil, err
	}
	operatorsToInstall := make(map[string]bool)
	if clusterInfo.Operators == nil || clusterInfo.Operators.OlmOperatorVersion == "" {
		operatorsToInstall["olm"] = true
		operatorsToInstall["vm"] = true
		operatorsToInstall["dbaas"] = true
	}
	if pxcOperatorVersion != nil && (clusterInfo.Operators == nil || clusterInfo.Operators.PxcOperatorVersion == "") {
		operatorsToInstall["pxc"] = true
	}
	if psmdbOperatorVersion != nil && (clusterInfo.Operators == nil || clusterInfo.Operators.PsmdbOperatorVersion == "") {
		operatorsToInstall["psmdb"] = true
	}
	settings, err := models.GetSettings(k.db.Querier)
	if err != nil {
		return nil, errors.Wrap(err, "cannot get PMM settings to start Victoria Metrics")
	}
	if settings.PMMPublicAddress == "" {
		return nil, errors.New("can not setup monitoring because public address is empty")
	}
	var apiKeyID int64
	var apiKey string
	apiKeyName := fmt.Sprintf("pmm-vmagent-%s-%d", req.KubernetesClusterName, rand.Int63()) //nolint:gosec
	apiKeyID, apiKey, err = k.grafanaClient.CreateAdminAPIKey(ctx, apiKeyName)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create Grafana admin API key")
	}

	go k.setupMonitoring(context.Background(), operatorsToInstall, settings.PMMPublicAddress, apiKey, apiKeyID, req)
	return &dbaasv1beta1.RegisterKubernetesClusterResponse{}, nil
}
func (k kubernetesServer) setupMonitoring(ctx context.Context, operatorsToInstall map[string]bool, pmmPublicAddress string,
	apiKey string, apiKeyID int64, req *dbaasv1beta1.RegisterKubernetesClusterRequest,
) {
	errs := k.installDefaultOperators(operatorsToInstall, req)
	if errs["vm"] != nil {
		k.l.Errorf("cannot install vm operator: %s", errs["vm"])
		return
	}

	err := k.startMonitoring(ctx, pmmPublicAddress, apiKey, apiKeyID, req.KubeAuth.Kubeconfig)
	if err != nil {
		k.l.Errorf("cannot start monitoring the clusdter: %s", err)
	}
	err = models.ChangeKubernetesClusterToReady(k.db.Querier, req.KubernetesClusterName)
	if err != nil {
		k.l.Errorf("couldn't update kubernetes cluster state: %s", err)
	}
}
func (k kubernetesServer) installDefaultOperators(operatorsToInstall map[string]bool, req *dbaasv1beta1.RegisterKubernetesClusterRequest) map[string]error {
	ctx := context.TODO()

	retval := make(map[string]error)

	if _, ok := operatorsToInstall["olm"]; ok {
		_, err := k.dbaasClient.InstallOLMOperator(ctx, &dbaascontrollerv1beta1.InstallOLMOperatorRequest{
			KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
				Kubeconfig: req.KubeAuth.Kubeconfig,
			},
			Version: "", // Use dbaas-controller default.
		})
		if err != nil {
			retval["olm"] = err
			k.l.Errorf("cannot install OLM operator to register the Kubernetes cluster: %s", err)
			return retval
		}
	}

	namespace := "default"

	if _, ok := operatorsToInstall["pxc"]; ok {
		operator := "percona-xtradb-cluster-operator"

		if err := k.installOperator(ctx, operator, namespace, "", "stable-v1", req.KubeAuth.Kubeconfig); err != nil {
			retval["pxc"] = err
			k.l.Errorf("cannot instal PXC operator in the new cluster: %s", err)
			return retval
		}

		installPlanName, err := getInstallPlanForSubscription(ctx, k.dbaasClient, req.KubeAuth.Kubeconfig, namespace, operator)
		if err != nil {
			retval["pxc"] = err
			k.l.Errorf("cannot get install plan for subscription %q: %s", operator, err)
			return retval
		}

		if err := approveInstallPlan(ctx, k.dbaasClient, req.KubeAuth.Kubeconfig, namespace, installPlanName); err != nil {
			retval["pxc"] = err
			k.l.Errorf("cannot approve the PXC install plan: %s", err)
		}
	}

	if _, ok := operatorsToInstall["psmdb"]; ok {
		operator := "percona-server-mongodb-operator"

		if err := k.installOperator(ctx, operator, namespace, "percona-server-mongodb-operator.v1.11.0", "stable-v1", req.KubeAuth.Kubeconfig); err != nil {
			retval["psmdb"] = err
			k.l.Errorf("cannot install PSMDB operator in the new cluster: %s", err)
		}

		installPlanName, err := getInstallPlanForSubscription(ctx, k.dbaasClient, req.KubeAuth.Kubeconfig, namespace, operator)
		if err != nil {
			retval["psmdb"] = err
			k.l.Errorf("cannot get install plan for subscription %q: %s", operator, err)
		}

		if err := approveInstallPlan(ctx, k.dbaasClient, req.KubeAuth.Kubeconfig, namespace, installPlanName); err != nil {
			retval["psmdb"] = err
			k.l.Errorf("cannot approve the PSMDB install plan: %s", err)
		}

	}
	if _, ok := operatorsToInstall["dbaas"]; ok {
		operator := "dbaas-operator"

		if err := k.installOperator(ctx, operator, namespace, "", "stable-v0", req.KubeAuth.Kubeconfig); err != nil {
			retval["dbaas"] = err
			k.l.Errorf("cannot install dbaas operator: %s", err)
			return retval
		}

		installPlanName, err := getInstallPlanForSubscription(ctx, k.dbaasClient, req.KubeAuth.Kubeconfig, namespace, operator)
		if err != nil {
			retval["dbaas"] = err
			k.l.Errorf("cannot get install plan for subscription %q: %s", operator, err)
		}

		if err := approveInstallPlan(ctx, k.dbaasClient, req.KubeAuth.Kubeconfig, namespace, installPlanName); err != nil {
			retval["dbaas"] = err
			k.l.Errorf("cannot approve the dbaas install plan: %s", err)
		}
	}

	if _, ok := operatorsToInstall["vm"]; ok {
		operator := "victoriametrics-operator"

		if err := k.installOperator(ctx, operator, namespace, "", "stable-v0", req.KubeAuth.Kubeconfig); err != nil {
			retval["vm"] = err
			k.l.Errorf("cannot install victoria metrics operator: %s", err)
			return retval
		}

		installPlanName, err := getInstallPlanForSubscription(ctx, k.dbaasClient, req.KubeAuth.Kubeconfig, namespace, operator)
		if err != nil {
			retval["vm"] = err
			k.l.Errorf("cannot get install plan for subscription %q: %s", operator, err)
		}

		if err := approveInstallPlan(ctx, k.dbaasClient, req.KubeAuth.Kubeconfig, namespace, installPlanName); err != nil {
			retval["vm"] = err
			k.l.Errorf("cannot approve the PSMDB install plan: %s", err)
		}
	}

	return retval
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

func (k kubernetesServer) installOperator(ctx context.Context, name, namespace, startingCSV, channel, kubeConfig string) error {
	_, err := k.dbaasClient.InstallOperator(ctx, &dbaascontrollerv1beta1.InstallOperatorRequest{
		KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
			Kubeconfig: kubeConfig,
		},
		Namespace:              namespace,
		Name:                   name,
		OperatorGroup:          "percona-operators-group",
		CatalogSource:          catalogSource,
		CatalogSourceNamespace: catalogSourceNamespace,
		Channel:                channel,
		InstallPlanApproval:    "Manual",
		StartingCsv:            startingCSV,
	})

	return err
}

func getInstallPlanForSubscription(ctx context.Context, client dbaasClient, kubeConfig, namespace, name string) (string, error) { //nolint:unparam
	var subscription *dbaascontrollerv1beta1.GetSubscriptionResponse
	var err error
	for i := 0; i < 6; i++ {
		subscription, err = client.GetSubscription(ctx, &dbaascontrollerv1beta1.GetSubscriptionRequest{
			KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
				Kubeconfig: kubeConfig,
			},
			Namespace: namespace,
			Name:      name,
		})
		if err != nil {
			return "", errors.Wrap(err, "cannot list subscriptions")
		}

		if subscription.Subscription.InstallPlanName != "" {
			break
		}

		time.Sleep(5 * time.Second)
	}

	return subscription.Subscription.InstallPlanName, nil
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

		if err := k.kubernetesClient.SetKubeconfig(kubernetesCluster.KubeConfig); err != nil {
			return errors.Wrap(err, "failed to create kubernetes client")
		}
		out, err := k.kubernetesClient.ListDatabaseClusters(ctx)

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
	kubernetesCluster, err := models.FindKubernetesClusterByName(k.db.Querier, req.KubernetesClusterName)
	if err != nil {
		return nil, err
	}

	err = k.kubernetesClient.SetKubeconfig(kubernetesCluster.KubeConfig)
	if err != nil {
		return nil, err
	}

	// Get cluster type
	clusterType, err := k.kubernetesClient.GetClusterType(ctx)
	if err != nil {
		return nil, err
	}

	var volumes *corev1.PersistentVolumeList
	if clusterType == kubernetes.ClusterTypeEKS {
		volumes, err = k.kubernetesClient.GetPersistentVolumes(ctx)
		if err != nil {
			return nil, err
		}
	}
	allCPUMillis, allMemoryBytes, allDiskBytes, err := k.kubernetesClient.GetAllClusterResources(ctx, clusterType, volumes)
	if err != nil {
		return nil, err
	}

	consumedCPUMillis, consumedMemoryBytes, err := k.kubernetesClient.GetConsumedCPUAndMemory(ctx, "")
	if err != nil {
		return nil, err
	}

	consumedDiskBytes, err := k.kubernetesClient.GetConsumedDiskBytes(ctx, clusterType, volumes)
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
	kubernetesCluster, err := models.FindKubernetesClusterByName(k.db.Querier, req.KubernetesClusterName)
	if err != nil {
		return nil, err
	}

	err = k.kubernetesClient.SetKubeconfig(kubernetesCluster.KubeConfig)
	if err != nil {
		return nil, err
	}

	storageClasses, err := k.kubernetesClient.GetStorageClasses(ctx)
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
