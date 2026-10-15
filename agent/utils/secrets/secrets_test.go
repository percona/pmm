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

package secrets

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/utils/vault"
)

type mapGetter map[string]string

func (m mapGetter) Get(_ context.Context, path, key string) (string, error) {
	v, ok := m[path+"#"+key]
	if !ok {
		return "", errors.New("no such secret")
	}
	return v, nil
}

// Not parallel: the tests configure the process-wide Vault client.
func TestSecrets(t *testing.T) {
	user := vault.Token("database/creds/pmm", "username", vault.EncodingUserinfo)
	password := vault.Token("database/creds/pmm", "password", vault.EncodingUserinfo)
	dsn := "postgres://" + user + ":" + password + "@127.0.0.1:5432/postgres"

	t.Run("NotConfigured", func(t *testing.T) {
		Configure(nil)

		res, values, err := Resolve(t.Context(), "postgres://user:pass@host/")
		require.NoError(t, err)
		assert.Equal(t, "postgres://user:pass@host/", res)
		assert.Empty(t, values)

		_, _, err = Resolve(t.Context(), dsn)
		require.ErrorIs(t, err, ErrUnavailable)
		require.ErrorIs(t, err, ErrNotConfigured)

		v, err := ResolveRef(t.Context(), "glsa_token")
		require.NoError(t, err)
		assert.Equal(t, "glsa_token", v)

		_, err = ResolveRef(t.Context(), "vault:secret/data/pmm#token")
		require.ErrorIs(t, err, ErrNotConfigured)

		fp, err := Fingerprint(t.Context(), "no", "tokens")
		require.NoError(t, err)
		assert.Empty(t, fp)
	})

	t.Run("Configured", func(t *testing.T) {
		secrets := mapGetter{
			"database/creds/pmm#username": "v-pmm",
			"database/creds/pmm#password": "p@ss",
			"secret/data/pmm#token":       "glsa_token",
		}
		Configure(secrets)
		t.Cleanup(func() { Configure(nil) })

		res, values, err := Resolve(t.Context(), dsn)
		require.NoError(t, err)
		assert.Equal(t, "postgres://v-pmm:p%40ss@127.0.0.1:5432/postgres", res)
		assert.Equal(t, []string{"v-pmm", "p@ss", "p%40ss"}, values)

		v, err := ResolveRef(t.Context(), "vault:secret/data/pmm#token")
		require.NoError(t, err)
		assert.Equal(t, "glsa_token", v)

		_, err = ResolveRef(t.Context(), "vault:secret/data/pmm#nope")
		require.ErrorIs(t, err, ErrUnavailable)
		require.EqualError(t, err, "failed to read a credential from Vault: no such secret")

		before, err := Fingerprint(t.Context(), "plain", dsn)
		require.NoError(t, err)
		require.NotEmpty(t, before)
		same, err := Fingerprint(t.Context(), dsn)
		require.NoError(t, err)
		assert.Equal(t, before, same)

		secrets["database/creds/pmm#password"] = "rotated"
		after, err := Fingerprint(t.Context(), dsn)
		require.NoError(t, err)
		assert.NotEqual(t, before, after)
	})
}
