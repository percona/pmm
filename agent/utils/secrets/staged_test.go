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
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/utils/vault"
)

// Not parallel: the tests configure the process-wide Vault client and staged secrets.
func TestStaged(t *testing.T) {
	store := mapGetter{}
	Configure(store)
	t.Cleanup(func() { Configure(nil) })

	t.Run("checked, then stored", func(t *testing.T) {
		refs, err := Stage(map[string]string{"password": "s3cr3t", "tls_key": "KEY"})
		require.NoError(t, err)
		require.Len(t, refs, 2)
		assert.True(t, vault.IsStagedRef(refs["password"]))
		assert.True(t, strings.HasSuffix(refs["password"], "#password"))

		// As PMM Server checks the connection with it, in a DSN.
		ref, err := vault.ParseRef(refs["password"])
		require.NoError(t, err)
		dsn := "root:" + vault.Token(ref.Path, ref.Key, vault.EncodingRaw) + "@tcp(127.0.0.1:3306)/"
		res, values, err := Resolve(t.Context(), dsn)
		require.NoError(t, err)
		assert.Equal(t, "root:s3cr3t@tcp(127.0.0.1:3306)/", res)
		assert.Equal(t, []string{"s3cr3t"}, values)

		// As PMM Server has it stored in Vault.
		require.NoError(t, Write(t.Context(), "secret/data/pmm/mysql", map[string]string{
			"password": refs["password"], "tls_key": refs["tls_key"], "username": "pmm",
		}))
		assert.Equal(t, "s3cr3t", store["secret/data/pmm/mysql#password"])
		assert.Equal(t, "KEY", store["secret/data/pmm/mysql#tls_key"])
		assert.Equal(t, "pmm", store["secret/data/pmm/mysql#username"])

		// Dropped once stored.
		_, err = ResolveRef(t.Context(), refs["password"])
		require.ErrorIs(t, err, ErrStagedNotFound)
	})

	t.Run("expired", func(t *testing.T) {
		now := time.Now()
		staged.m.Lock()
		staged.now = func() time.Time { return now }
		staged.m.Unlock()
		t.Cleanup(func() {
			staged.m.Lock()
			staged.now = time.Now
			staged.m.Unlock()
		})

		refs, err := Stage(map[string]string{"password": "s3cr3t"})
		require.NoError(t, err)
		now = now.Add(stagedTTL)
		err = Write(t.Context(), "secret/data/pmm/other", map[string]string{"password": refs["password"]})
		require.ErrorIs(t, err, ErrStagedNotFound)
		assert.NotContains(t, store, "secret/data/pmm/other#password")
	})

	t.Run("not another pmm-agent's", func(t *testing.T) {
		_, err := ResolveRef(t.Context(), "vault:"+vault.StagedPathPrefix+"0123456789abcdef#password")
		require.ErrorIs(t, err, ErrStagedNotFound)
	})

	t.Run("sealed to this pmm-agent", func(t *testing.T) {
		pub, err := SealingPublicKey()
		require.NoError(t, err)
		publicKey, err := base64.StdEncoding.DecodeString(pub)
		require.NoError(t, err)
		ref, err := vault.Seal(publicKey, "password", "s3cr3t")
		require.NoError(t, err)

		// As PMM Server checks the connection with it, in a DSN.
		r, err := vault.ParseRef(ref)
		require.NoError(t, err)
		res, _, err := Resolve(t.Context(), "root:"+vault.Token(r.Path, r.Key, vault.EncodingRaw)+"@tcp(127.0.0.1:3306)/")
		require.NoError(t, err)
		assert.Equal(t, "root:s3cr3t@tcp(127.0.0.1:3306)/", res)

		// As PMM Server has it stored in Vault.
		require.NoError(t, Write(t.Context(), "secret/data/pmm/sealed", map[string]string{"password": ref}))
		assert.Equal(t, "s3cr3t", store["secret/data/pmm/sealed#password"])

		again, err := SealingPublicKey()
		require.NoError(t, err)
		assert.Equal(t, pub, again, "one key for the process")
	})

	t.Run("bounded", func(t *testing.T) {
		staged.m.Lock()
		clear(staged.secrets)
		staged.m.Unlock()
		t.Cleanup(func() {
			staged.m.Lock()
			clear(staged.secrets)
			staged.m.Unlock()
		})

		for range maxStaged {
			_, err := Stage(map[string]string{"password": "x"})
			require.NoError(t, err)
		}
		_, err := Stage(map[string]string{"password": "x"})
		require.ErrorContains(t, err, "too many secrets are staged")
	})
}
