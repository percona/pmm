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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// stageSecretsPath is where the local pmm-agent stages secrets, see StageSecrets.
const stageSecretsPath = "/local/StageSecrets" //nolint:gosec // a path, not a credential

// maxStageSecretsResponse limits the response of StageSecrets, which holds a reference to each secret.
const maxStageSecretsResponse = 1 << 20

// ErrStagingNotSupported is returned by StageSecrets by a pmm-agent that does not stage secrets.
var ErrStagingNotSupported = errors.New("pmm-agent does not stage secrets")

// StageSecrets has the local pmm-agent hold data until PMM Server asks it to store it in Vault, and returns
// references to its keys to send PMM Server instead, so that the secrets reach Vault through pmm-agent alone.
func StageSecrets(ctx context.Context, data map[string]string) (map[string]string, error) {
	body, err := json.Marshal(map[string]any{"data": data})
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second) //nolint:mnd
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+address+stageSecretsPath, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck

	b, err := io.ReadAll(io.LimitReader(resp.Body, maxStageSecretsResponse))
	if err != nil {
		return nil, err
	}
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound, http.StatusMethodNotAllowed:
		return nil, ErrStagingNotSupported
	default:
		return nil, fmt.Errorf("pmm-agent did not stage the secrets: %d %s", resp.StatusCode, bytes.TrimSpace(b))
	}

	var res struct {
		Refs map[string]string `json:"refs"`
	}
	err = json.Unmarshal(b, &res)
	if err != nil {
		return nil, err
	}
	if len(res.Refs) != len(data) {
		return nil, errors.New("pmm-agent did not stage every secret")
	}
	return res.Refs, nil
}
