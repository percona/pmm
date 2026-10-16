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

package base

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"slices"

	"github.com/go-openapi/runtime"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/admin/agentlocal"
	"github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
	"github.com/percona/pmm/utils/vault"
)

// stagedKeys are the JSON fields of the secrets that PMM Server stores in the Vault secret of an Agent with a Vault
// path: the password, and the TLS certificates and keys.
var stagedKeys = []string{ //nolint:gochecknoglobals
	"password", "tls_ca", "tls_cert", "tls_key", "tls_certificate_key", "tls_certificate_key_file_password",
}

// stagingTransport stages, in the local pmm-agent, the secrets of a request that PMM Server is to store in Vault
// through it, and sends PMM Server references to them instead: the secrets reach Vault through pmm-agent alone.
// A request for another pmm-agent, or one that the local pmm-agent cannot stage, is sent as it is.
type stagingTransport struct {
	runtime.ClientTransport
}

// Submit implements runtime.ClientTransport.
func (t stagingTransport) Submit(op *runtime.ClientOperation) (any, error) {
	ctx := context.Background()
	if op.Context != nil { //nolint:staticcheck // the generated clients call Submit, with the context in the operation
		ctx = op.Context //nolint:staticcheck
	}
	err := t.stageSecrets(ctx, op.Params)
	if err != nil {
		logrus.Debugf("Sending the secrets to store in Vault through PMM Server, the local pmm-agent did not stage them: %s.", err)
	}
	return t.ClientTransport.Submit(op)
}

// stageSecrets replaces the secrets in the Body of params, the request parameters, with references to them staged
// in the local pmm-agent, if they are to be stored in the Vault secret of an Agent of that pmm-agent.
func (t stagingTransport) stageSecrets(ctx context.Context, params any) error {
	v := reflect.ValueOf(params)
	if v.Kind() != reflect.Pointer || v.IsNil() || v.Elem().Kind() != reflect.Struct {
		return nil
	}
	body := v.Elem().FieldByName("Body")
	if !body.IsValid() || !body.CanSet() {
		return nil
	}
	var changedAgentID string
	if id := v.Elem().FieldByName("AgentID"); id.IsValid() && id.Kind() == reflect.String {
		changedAgentID = id.String()
	}

	b, err := json.Marshal(body.Interface())
	if err != nil {
		return err
	}
	var request map[string]any
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber() // keeps the numbers as they are
	err = d.Decode(&request)
	if err != nil {
		return nil //nolint:nilerr // not an object, so nothing to stage
	}

	var status *agentlocal.Status
	staged := false
	for _, v := range request {
		agent, ok := v.(map[string]any)
		if !ok {
			continue
		}
		done, err := t.stageAgent(ctx, agent, changedAgentID, &status)
		if err != nil {
			return err
		}
		staged = staged || done
	}
	if !staged {
		return nil
	}

	b, err = json.Marshal(request)
	if err != nil {
		return err
	}
	replaced := reflect.New(body.Type())
	err = json.Unmarshal(b, replaced.Interface())
	if err != nil {
		return err
	}
	body.Set(replaced.Elem())
	logrus.Debugf("Staged the secrets to store in Vault in the local pmm-agent.")
	return nil
}

// stageAgent replaces the secrets of agent, an object of the request, with references to them staged in the local
// pmm-agent, and reports whether it did. The status of the local pmm-agent is got once, when first needed.
func (t stagingTransport) stageAgent(ctx context.Context, agent map[string]any, changedAgentID string, status **agentlocal.Status) (bool, error) {
	secrets := make(map[string]string)
	for _, key := range stagedKeys {
		s, _ := agent[key].(string)
		if s != "" && !vault.IsRef(s) {
			secrets[key] = s
		}
	}
	if len(secrets) == 0 {
		return false, nil
	}

	if *status == nil {
		s, err := agentlocal.GetStatus(agentlocal.DoNotRequestNetworkInfo) //nolint:contextcheck // it takes no context
		if err != nil {
			return false, err
		}
		*status = s
	}
	if !t.storedLocally(ctx, agent, changedAgentID, *status) {
		return false, nil
	}

	refs, err := agentlocal.StageSecrets(ctx, secrets)
	if err != nil {
		return false, err
	}
	for key, ref := range refs {
		agent[key] = ref
	}
	return true, nil
}

// storedLocally reports whether PMM Server is to store the secrets of agent, an object of the request, in Vault
// through the local pmm-agent: the Agent has a Vault path, and that pmm-agent runs it, or is to.
func (t stagingTransport) storedLocally(ctx context.Context, agent map[string]any, changedAgentID string, status *agentlocal.Status) bool {
	if pmmAgentID, _ := agent["pmm_agent_id"].(string); pmmAgentID != "" {
		vaultPath, _ := agent["vault_path"].(string)
		return vaultPath != "" && pmmAgentID == status.AgentID
	}

	// A change of an Agent, whose Vault path the request has only if it changes it.
	if changedAgentID == "" || !slices.ContainsFunc(status.Agents, func(a agentlocal.AgentStatus) bool { return a.AgentID == changedAgentID }) {
		return false
	}
	if vaultPath, _ := agent["vault_path"].(string); vaultPath != "" {
		return true
	}
	vaultPath, err := t.vaultPath(ctx, changedAgentID)
	return err == nil && vaultPath != ""
}

// vaultPath returns the Vault path of the Agent with the given ID.
func (t stagingTransport) vaultPath(ctx context.Context, agentID string) (string, error) {
	params := &agents_service.GetAgentParams{AgentID: agentID, Context: ctx}
	res, err := agents_service.New(t.ClientTransport, nil).GetAgent(params)
	if err != nil {
		return "", err
	}

	b, err := json.Marshal(res.Payload)
	if err != nil {
		return "", err
	}
	var agent map[string]map[string]any
	err = json.Unmarshal(b, &agent)
	if err != nil {
		return "", err
	}
	for _, a := range agent {
		if vaultPath, _ := a["vault_path"].(string); vaultPath != "" {
			return vaultPath, nil
		}
	}
	return "", errors.New("the agent has no Vault path")
}
