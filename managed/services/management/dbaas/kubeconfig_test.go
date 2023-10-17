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
