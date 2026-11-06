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

package agentlocal

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/agent/utils/secrets"
)

// StageSecretsPath is where pmm-admin stages the secrets to store in Vault, see StageSecrets.
const StageSecretsPath = "/local/StageSecrets" //nolint:gosec // a path, not a credential

// maxStageSecretsBody limits the secrets staged at once: a few passwords, and TLS certificates and keys.
const maxStageSecretsBody = 1 << 20

// StageSecretsRequest is the request of StageSecrets.
type StageSecretsRequest struct {
	Data map[string]string `json:"data"`
}

// StageSecretsResponse is the response of StageSecrets.
type StageSecretsResponse struct {
	Refs map[string]string `json:"refs"`
}

// StageSecrets holds the secrets of the request until PMM Server asks pmm-agent to store them in Vault, and answers
// with references to them, which pmm-admin sends PMM Server instead: the secrets reach Vault through pmm-agent alone.
// Staged, a secret is neither stored nor sent anywhere, so this needs no authentication: PMM Server decides where it
// goes, if anywhere, and pmm-agent drops it after a while if it does not.
func StageSecrets(rw http.ResponseWriter, req *http.Request) {
	var r StageSecretsRequest
	err := json.NewDecoder(io.LimitReader(req.Body, maxStageSecretsBody)).Decode(&r)
	if err != nil || len(r.Data) == 0 {
		http.Error(rw, "the request must have the secrets to stage in data", http.StatusBadRequest)
		return
	}

	refs, err := secrets.Stage(r.Data)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusServiceUnavailable)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(rw).Encode(StageSecretsResponse{Refs: refs})
	if err != nil {
		logrus.WithField("component", "local-server").Debugf("Cannot write the staged secrets: %s.", err)
	}
}
