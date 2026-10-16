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
