// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package agentlocal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/utils/vault"
)

func TestStageSecrets(t *testing.T) {
	t.Parallel()

	t.Run("staged", func(t *testing.T) {
		t.Parallel()

		rec := httptest.NewRecorder()
		StageSecrets(rec, httptest.NewRequestWithContext(t.Context(), http.MethodPost, StageSecretsPath,
			strings.NewReader(`{"data":{"password":"s3cr3t","tls_key":"KEY"}}`)))
		require.Equal(t, http.StatusOK, rec.Code)

		var res StageSecretsResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &res))
		require.Len(t, res.Refs, 2)
		for key, ref := range res.Refs {
			assert.True(t, vault.IsStagedRef(ref), key)
			assert.True(t, strings.HasSuffix(ref, "#"+key), key)
		}
		assert.NotContains(t, rec.Body.String(), "s3cr3t")
	})

	t.Run("nothing to stage", func(t *testing.T) {
		t.Parallel()

		for _, body := range []string{`{}`, `{"data":{}}`, `not JSON`} {
			rec := httptest.NewRecorder()
			StageSecrets(rec, httptest.NewRequestWithContext(t.Context(), http.MethodPost, StageSecretsPath, strings.NewReader(body)))
			assert.Equal(t, http.StatusBadRequest, rec.Code, body)
		}
	})
}
