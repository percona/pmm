package dbaas

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
)

func TestKubeClient(t *testing.T) {
	t.Parallel()
	expectedConf := `apiVersion: v1
clusters:
    - cluster:
        certificate-authority-data: aGVsbG8=
        server: https://localhost
      name: default-cluster
contexts:
    - context:
        cluster: default-cluster
        namespace: default
        user: pmm-service-account
      name: default
current-context: default
kind: Config
users:
    - name: pmm-service-account
      user:
        token: world
`
	_, err := NewK8sInclusterClient()
	require.Error(t, err)
	k := &k8sClient{
		conf: &rest.Config{Host: "https://localhost"},
	}
	conf, err := k.GenerateKubeConfig(&v1.Secret{
		Data: map[string][]byte{
			"ca.crt": []byte("hello"),
			"token":  []byte("world"),
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, expectedConf, string(conf))

}
