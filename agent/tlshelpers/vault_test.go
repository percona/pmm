// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tlshelpers

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/agent/utils/secrets"
	agentv1 "github.com/percona/pmm/api/agent/v1"
	"github.com/percona/pmm/utils/vault"
)

type keyVault map[string]string

func (v keyVault) Get(_ context.Context, path, key string) (string, error) {
	s, ok := v[path+"#"+key]
	if !ok {
		return "", errors.New("no such secret")
	}
	return s, nil
}

func (v keyVault) Write(context.Context, string, map[string]string) error {
	return errors.New("not used")
}

// Not parallel: it configures the process-wide Vault client.
func TestTLSFilesFromVault(t *testing.T) {
	cert, key := generateCertPair(t)
	secrets.Configure(keyVault{"secret/data/pmm#tls_key": key})
	t.Cleanup(func() { secrets.Configure(nil) })

	files := map[string]string{
		agentv1.TLSCertFileName: cert,
		agentv1.TLSKeyFileName:  vault.Token("secret/data/pmm", "tls_key", vault.EncodingRaw),
	}

	t.Run("MySQL", func(t *testing.T) {
		t.Cleanup(DeregisterMySQLCerts)

		require.NoError(t, RegisterMySQLCerts(t.Context(), files, false))
		assert.Len(t, registeredMySQLTLSConfig(t).Certificates, 1)
		assert.Equal(t, vault.Token("secret/data/pmm", "tls_key", vault.EncodingRaw), files[agentv1.TLSKeyFileName], "the given files are kept")
	})

	t.Run("Valkey", func(t *testing.T) {
		opts, err := GetValkeyTLSConfig(t.Context(), &agentv1.TextFiles{Files: files}, true, false)
		require.NoError(t, err)
		assert.NotEmpty(t, opts)
	})

	t.Run("Unreadable", func(t *testing.T) {
		secrets.Configure(nil)
		t.Cleanup(DeregisterMySQLCerts)

		require.ErrorIs(t, RegisterMySQLCerts(t.Context(), files, false), secrets.ErrUnavailable)
		_, err := GetValkeyTLSConfig(t.Context(), &agentv1.TextFiles{Files: files}, true, false)
		require.ErrorIs(t, err, secrets.ErrUnavailable)
	})
}
