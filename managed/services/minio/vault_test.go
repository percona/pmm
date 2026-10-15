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

package minio

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/utils/secrets"
)

type mapSecrets map[string]string

func (m mapSecrets) Get(_ context.Context, path, key string) (string, error) {
	v, ok := m[path+"#"+key]
	if !ok {
		return "", errors.New("no such secret")
	}
	return v, nil
}

// Not parallel: it configures the process-wide Vault client of PMM Server.
func TestClientWithKeysFromVault(t *testing.T) {
	t.Cleanup(func() { secrets.Configure(nil, 0) })

	_, err := createMinioClient(t.Context(), "https://s3.example.com", "vault:secret/data/s3#access_key", "secret")
	require.ErrorContains(t, err, "Failed to read the keys of the storage: cannot read vault:secret/data/s3#access_key: PMM Server has no Vault configured")

	secrets.Configure(mapSecrets{"secret/data/s3#access_key": "AKIA", "secret/data/s3#secret_key": "s3cr3t"}, time.Minute)
	mc, err := createMinioClient(t.Context(), "https://s3.example.com", "vault:secret/data/s3#access_key", "vault:secret/data/s3#secret_key")
	require.NoError(t, err)

	creds, err := mc.GetCreds()
	require.NoError(t, err)
	assert.Equal(t, "AKIA", creds.AccessKeyID)
	assert.Equal(t, "s3cr3t", creds.SecretAccessKey)
}
