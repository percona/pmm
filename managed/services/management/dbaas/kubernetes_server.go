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

	goversion "github.com/hashicorp/go-version"
	dbaascontrollerv1beta1 "github.com/percona-platform/dbaas-api/gen/controller"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/yaml.v3"

	dbaasv1beta1 "github.com/percona/pmm/api/managementpb/dbaas"
	"github.com/percona/pmm/managed/models"
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

	dbaasv1beta1.UnimplementedKubernetesServer
}

// NewKubernetesServer creates Kubernetes Server.
func NewKubernetesServer(db *reform.DB, dbaasClient dbaasClient, grafanaClient grafanaClient, versionService versionService) dbaasv1beta1.KubernetesServer {
	l := logrus.WithField("component", "kubernetes_server")
	return &kubernetesServer{
		l:              l,
		db:             db,
		dbaasClient:    dbaasClient,
		grafanaClient:  grafanaClient,
		versionService: versionService,
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
func (k kubernetesServer) convertToOperatorStatus(ctx context.Context, operatorType string, operatorVersion string) (dbaasv1beta1.OperatorsStatus, error) {
	if operatorVersion == "" {
		return dbaasv1beta1.OperatorsStatus_OPERATORS_STATUS_NOT_INSTALLED, nil
	}

	supported, err := k.versionService.IsOperatorVersionSupported(ctx, operatorType, pmmversion.PMMVersion, operatorVersion)
	if err != nil {
		return dbaasv1beta1.OperatorsStatus_OPERATORS_STATUS_INVALID, err
	}
	if supported {
		return dbaasv1beta1.OperatorsStatus_OPERATORS_STATUS_OK, nil
	}

	return dbaasv1beta1.OperatorsStatus_OPERATORS_STATUS_UNSUPPORTED, nil
}

// ListKubernetesClusters returns a list of all registered Kubernetes clusters.
func (k kubernetesServer) ListKubernetesClusters(ctx context.Context, _ *dbaasv1beta1.ListKubernetesClustersRequest) (*dbaasv1beta1.ListKubernetesClustersResponse, error) {
	kubernetesClusters, err := models.FindAllKubernetesClusters(k.db.Querier)
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	clusters := make([]*dbaasv1beta1.ListKubernetesClustersResponse_Cluster, len(kubernetesClusters))
	for i, cluster := range kubernetesClusters {
		i := i
		cluster := cluster
		wg.Add(1)
		go func() {
			defer wg.Done()
			clusters[i] = &dbaasv1beta1.ListKubernetesClustersResponse_Cluster{
				KubernetesClusterName: cluster.KubernetesClusterName,
				Operators: &dbaasv1beta1.Operators{
					Pxc:   &dbaasv1beta1.Operator{},
					Psmdb: &dbaasv1beta1.Operator{},
				},
			}
			resp, e := k.dbaasClient.CheckKubernetesClusterConnection(ctx, cluster.KubeConfig)
			if e != nil {
				clusters[i].Status = dbaasv1beta1.KubernetesClusterStatus_KUBERNETES_CLUSTER_STATUS_UNAVAILABLE
				return
			}

			clusters[i].Status = dbaasv1beta1.KubernetesClusterStatus(resp.Status)

			if resp.Operators == nil {
				return
			}

			clusters[i].Operators.Pxc.Status, err = k.convertToOperatorStatus(ctx, pxcOperator, resp.Operators.PxcOperatorVersion)
			if err != nil {
				k.l.Errorf("failed to convert dbaas-controller operator status to PMM status: %v", err)
			}
			clusters[i].Operators.Psmdb.Status, err = k.convertToOperatorStatus(ctx, psmdbOperator, resp.Operators.PsmdbOperatorVersion)
			if err != nil {
				k.l.Errorf("failed to convert dbaas-controller operator status to PMM status: %v", err)
			}

			clusters[i].Operators.Pxc.Version = resp.Operators.PxcOperatorVersion
			clusters[i].Operators.Psmdb.Version = resp.Operators.PsmdbOperatorVersion
		}()
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
	Exec kubectlUserExec `yaml:"exec,omitempty"`
}

type kubectlUserWithName struct {
	Name string       `yaml:"name,omitempty"`
	User *kubectlUser `yaml:"user,omitempty"`
}

type kubectlConfig struct {
	Kind           string                 `yaml:"kind,omitempty"`
	ApiVersion     string                 `yaml:"apiVersion,omitempty"`
	CurrentContext string                 `yaml:"current-context,omitempty"`
	Clusters       []interface{}          `yaml:"clusters,omitempty"`
	Contexts       []interface{}          `yaml:"contexts,omitempty"`
	Preferences    map[string]interface{} `yaml:"preferences"`
	Users          []*kubectlUserWithName `yaml:"users,omitempty"`
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
func (k kubernetesServer) RegisterKubernetesCluster(ctx context.Context, req *dbaasv1beta1.RegisterKubernetesClusterRequest) (*dbaasv1beta1.RegisterKubernetesClusterResponse, error) {
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

	if pxcOperatorVersion != nil && (clusterInfo.Operators == nil || clusterInfo.Operators.PxcOperatorVersion == "") {
		_, err = k.dbaasClient.InstallPXCOperator(ctx, &dbaascontrollerv1beta1.InstallPXCOperatorRequest{
			KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
				Kubeconfig: req.KubeAuth.Kubeconfig,
			},
			Version: pxcOperatorVersion.String(),
		})
		if err != nil {
			return nil, err
		}
	}
	if psmdbOperatorVersion != nil && (clusterInfo.Operators == nil || clusterInfo.Operators.PsmdbOperatorVersion == "") {
		_, err = k.dbaasClient.InstallPSMDBOperator(ctx, &dbaascontrollerv1beta1.InstallPSMDBOperatorRequest{
			KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
				Kubeconfig: req.KubeAuth.Kubeconfig,
			},
			Version: psmdbOperatorVersion.String(),
		})
		if err != nil {
			return nil, err
		}
	}

	settings, err := models.GetSettings(k.db.Querier)
	if err != nil {
		return nil, err
	}
	if settings.PMMPublicAddress != "" {
		var apiKeyID int64
		var apiKey string
		apiKeyName := fmt.Sprintf("pmm-vmagent-%s-%d", req.KubernetesClusterName, rand.Int63())
		apiKeyID, apiKey, err = k.grafanaClient.CreateAdminAPIKey(ctx, apiKeyName)
		if err != nil {
			return nil, err
		}
		pmmParams := &dbaascontrollerv1beta1.PMMParams{
			PublicAddress: fmt.Sprintf("https://%s", settings.PMMPublicAddress),
			Login:         "api_key",
			Password:      apiKey,
		}

		_, err := k.dbaasClient.StartMonitoring(ctx, &dbaascontrollerv1beta1.StartMonitoringRequest{
			KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
				Kubeconfig: req.KubeAuth.Kubeconfig,
			},
			Pmm: pmmParams,
		})
		if err != nil {
			e := k.grafanaClient.DeleteAPIKeyByID(ctx, apiKeyID)
			if e != nil {
				k.l.Warnf("couldn't delete created API Key %v: %s", apiKeyID, e)
			}
			k.l.Warnf("couldn't start monitoring of the kubernetes cluster: %s", err)
			return nil, status.Errorf(codes.Internal, "couldn't start monitoring of the kubernetes cluster: %s", err.Error())
		}
	}

	return &dbaasv1beta1.RegisterKubernetesClusterResponse{}, nil
}

// UnregisterKubernetesCluster removes a registered Kubernetes cluster from PMM.
func (k kubernetesServer) UnregisterKubernetesCluster(ctx context.Context, req *dbaasv1beta1.UnregisterKubernetesClusterRequest) (*dbaasv1beta1.UnregisterKubernetesClusterResponse, error) {
	err := k.db.InTransaction(func(t *reform.TX) error {
		kubernetesCluster, err := models.FindKubernetesClusterByName(t.Querier, req.KubernetesClusterName)
		if err != nil {
			return err
		}

		if req.Force {
			return models.RemoveKubernetesCluster(t.Querier, req.KubernetesClusterName)
		}

		_, err = k.dbaasClient.StopMonitoring(ctx, &dbaascontrollerv1beta1.StopMonitoringRequest{
			KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
				Kubeconfig: kubernetesCluster.KubeConfig,
			},
		})

		if err != nil {
			k.l.Warnf("cannot stop monitoring: %s", err)
		}

		pxcClusters, err := k.dbaasClient.ListPXCClusters(ctx,
			&dbaascontrollerv1beta1.ListPXCClustersRequest{
				KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
					Kubeconfig: kubernetesCluster.KubeConfig,
				},
			})
		switch {
		case err != nil && accessError(err):
			k.l.Warn(err)
		case err != nil:
			return err
		case len(pxcClusters.Clusters) != 0:
			return status.Errorf(codes.FailedPrecondition, "Kubernetes cluster %s has PXC clusters", req.KubernetesClusterName)
		}

		psmdbClusters, err := k.dbaasClient.ListPSMDBClusters(ctx, &dbaascontrollerv1beta1.ListPSMDBClustersRequest{
			KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
				Kubeconfig: kubernetesCluster.KubeConfig,
			},
		})
		switch {
		case err != nil && accessError(err):
			k.l.Warn(err)
		case err != nil:
			return err
		case len(psmdbClusters.Clusters) != 0:
			return status.Errorf(codes.FailedPrecondition, "Kubernetes cluster %s has PSMDB clusters", req.KubernetesClusterName)
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
	safeKubeConfig := &kubectlConfig{}
	err = yaml.Unmarshal([]byte(kubernetesCluster.KubeConfig), safeKubeConfig)
	if err != nil {
		return nil, err
	}
	for i, user := range safeKubeConfig.Users {
		for j, env := range user.User.Exec.Env {
			if env.Name == "AWS_ACCESS_KEY_ID" || env.Name == "AWS_SECRET_ACCESS_KEY" {
				safeKubeConfig.Users[i].User.Exec.Env[j].Value = "<secret>"
			}
		}
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
	in := &dbaascontrollerv1beta1.GetResourcesRequest{
		KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
			Kubeconfig: kubernetesCluster.KubeConfig,
		},
	}
	response, err := k.dbaasClient.GetResources(ctx, in)
	if err != nil {
		return nil, err
	}
	return &dbaasv1beta1.GetResourcesResponse{
		All: &dbaasv1beta1.Resources{
			CpuM:        response.All.CpuM,
			MemoryBytes: response.All.MemoryBytes,
			DiskSize:    response.All.DiskSize,
		},
		Available: &dbaasv1beta1.Resources{
			CpuM:        response.Available.CpuM,
			MemoryBytes: response.Available.MemoryBytes,
			DiskSize:    response.Available.DiskSize,
		},
	}, nil
}
