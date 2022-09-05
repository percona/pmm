package dbaas

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestMaskSecrets(t *testing.T) {
	t.Parallel()
	kubeConfig := &kubectlConfig{}
	err := yaml.Unmarshal([]byte(awsIAMAuthenticatorKubeconfigTransformed), kubeConfig)
	require.NoError(t, err)
	safeConfig, err := kubeConfig.maskSecrets()
	require.NoError(t, err)
	assert.Equal(t, "<secret>", safeConfig.Users[0].User.Exec.Env[1].Value)
	assert.Equal(t, "<secret>", safeConfig.Users[0].User.Exec.Env[2].Value)
}
