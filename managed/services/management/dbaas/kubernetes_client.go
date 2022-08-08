package dbaas

import (
	"context"
	"encoding/json"

	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
)

type (
	Cluster struct {
		CertificateAuthorityData []byte `json:"certificate-authority-data"`
		Server                   string `json:"server"`
	}
	ClusterInfo struct {
		Name    string  `json:"name"`
		Cluster Cluster `json:"cluster"`
	}
	User struct {
		Token string `json:"token"`
	}
	UserInfo struct {
		Name string `json:"name"`
		User User   `json:"user"`
	}
	Context struct {
		Cluster   string `json:"cluster"`
		User      string `json:"user"`
		Namespace string `json:"namespace"`
	}
	ContextInfo struct {
		Name    string  `json:"name"`
		Context Context `json:"context"`
	}
	Config struct {
		// Legacy field from pkg/api/types.go TypeMeta.
		// TODO(jlowdermilk): remove this after eliminating downstream dependencies.
		// +k8s:conversion-gen=false
		// +optional
		Kind string `json:"kind,omitempty"`
		// Legacy field from pkg/api/types.go TypeMeta.
		// TODO(jlowdermilk): remove this after eliminating downstream dependencies.
		// +k8s:conversion-gen=false
		// +optional
		APIVersion string `json:"apiVersion,omitempty"`
		// Preferences holds general information to be use for cli interactions
		Clusters []ClusterInfo `json:"clusters"`
		// AuthInfos is a map of referencable names to user configs
		Users []UserInfo `json:"users"`
		// Contexts is a map of referencable names to context configs
		Contexts []ContextInfo `json:"contexts"`
		// CurrentContext is the name of the context that you would like to use by default
		CurrentContext string `json:"current-context"`
	}

	K8sClient struct {
		conf      *rest.Config
		clientSet *kubernetes.Clientset
	}
)

// NewK8sInclusterClient initializes a new k8s client which
// uses the service account and automatically gets config to
// communicate within cluster if we're running inside k8s cluster
func NewK8sInclusterClient() (*K8sClient, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &K8sClient{clientSet: clientSet, conf: config}, nil
}

func (k *K8sClient) GetSecretsForServiceAccount(ctx context.Context, namespace, accountName string) (*v1.Secret, error) {
	serviceAccount, err := k.clientSet.CoreV1().ServiceAccounts(namespace).Get(context.TODO(), accountName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return k.clientSet.CoreV1().Secrets(namespace).Get(
		context.TODO(),
		serviceAccount.Secrets[0].Name,
		metav1.GetOptions{},
	)
}

func (k *K8sClient) GenerateKubeConfig(secret *v1.Secret) ([]byte, error) {
	c := &Config{
		Kind:           "Config",
		APIVersion:     "v1",
		CurrentContext: "default",
	}
	c.Clusters = []ClusterInfo{
		{
			Name: "default-cluster",
			Cluster: Cluster{
				CertificateAuthorityData: secret.Data["ca.crt"],
				Server:                   k.conf.Host,
			},
		},
	}
	c.Contexts = []ContextInfo{
		{
			Name: "default",
			Context: Context{
				Cluster:   "default-cluster",
				User:      "pmm-service-account",
				Namespace: "default",
			},
		},
	}
	c.Users = []UserInfo{
		{
			Name: "pmm-service-account",
			User: User{
				Token: string(secret.Data["token"]),
			},
		},
	}
	return k.marshalKubeConfig(c)
}
func (k *K8sClient) marshalKubeConfig(c *Config) ([]byte, error) {
	conf, err := json.Marshal(&c)
	if err != nil {
		return nil, err
	}
	var jsonObj interface{}
	err = yaml.Unmarshal(conf, &jsonObj)
	if err != nil {
		return nil, err
	}
	return yaml.Marshal(jsonObj)
}
