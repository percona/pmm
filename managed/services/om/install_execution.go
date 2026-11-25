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

package om

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/reform.v1"

	omv1 "github.com/percona/pmm/api/om/v1"
	"github.com/percona/pmm/managed/models"
)

// installBootstrapPhase and installBootstrapStep name the one run-level step an install dispatches: it
// creates the om_bootstrap run that installs MongoDB on the nodes and polls it to completion.
const (
	installBootstrapPhase = "run"
	installBootstrapStep  = "bootstrap"
)

// installRunRef is stored as the bootstrap step's result so a re-probe reaches the run it created.
type installRunRef struct {
	RunID string `json:"run_id"`
}

// bootstrapRunner is the subset of the om_bootstrap client the install step uses, so a test can fake it.
type bootstrapRunner interface {
	triggerRun(ctx context.Context, req extensionsTriggerBootstrapRunRequest) (*extensionsBootstrapRun, error)
	getRun(ctx context.Context, runID string) (*extensionsBootstrapRun, error)
}

// installStepExecutor runs an install's bootstrap step: it creates the om_bootstrap run on the first
// probe and reports its progress on the later ones, keeping the run id on the step so a re-probe finds it.
type installStepExecutor struct {
	db        *reform.DB
	bootstrap bootstrapRunner
	op        *models.OmOperation
}

// Run creates the om_bootstrap run on the first probe and maps its status on each later one: a succeeded
// run is ok, a failed or rolled-back run fails, and a still-running one is inconclusive.
func (e *installStepExecutor) Run(ctx context.Context, _ StepPlan, idempotencyKey string) (StepResultKind, []byte, error) {
	ctx = asService(ctx)
	runID, err := e.runIDFromStep(idempotencyKey)
	if err != nil {
		return "", nil, err
	}
	if runID == "" {
		run, err := e.bootstrap.triggerRun(ctx, e.plannedRun())
		if err != nil {
			return "", nil, err
		}
		err = models.RebindOmBootstrapSecretRun(e.db.Querier, e.op.ID, run.ID)
		if err != nil {
			return "", nil, err
		}
		return stepResult(StepResultTimedOut, run.ID)
	}
	run, err := e.bootstrap.getRun(ctx, runID)
	if err != nil {
		return "", nil, err
	}
	switch run.Status {
	case bootstrapRunSucceeded:
		return stepResult(StepResultOk, runID)
	case bootstrapRunFailed, bootstrapRunRolledBack:
		return stepResult(StepResultFailed, runID)
	default:
		return stepResult(StepResultTimedOut, runID)
	}
}

// stepResult pairs a typed result with the run reference stored on the step so a re-probe reaches it.
func stepResult(kind StepResultKind, runID string) (StepResultKind, []byte, error) {
	ref, err := json.Marshal(installRunRef{RunID: runID})
	if err != nil {
		return "", nil, fmt.Errorf("failed to encode the run reference: %w", err)
	}
	return kind, ref, nil
}

// runIDFromStep returns the om_bootstrap run a prior probe recorded on the step, or "" when the first
// probe has not created one yet.
func (e *installStepExecutor) runIDFromStep(idempotencyKey string) (string, error) {
	row, err := e.db.SelectOneFrom(models.OmOperationStepTable, "WHERE idempotency_key = $1", idempotencyKey)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("failed to read the install step: %w", err)
	}
	step := row.(*models.OmOperationStep) //nolint:forcetypeassert
	if len(step.Result) == 0 {
		return "", nil
	}
	ref := installRunRef{}
	err = json.Unmarshal(step.Result, &ref)
	if err != nil {
		return "", fmt.Errorf("failed to read the install run reference: %w", err)
	}
	return ref.RunID, nil
}

// plannedRun assembles the om_bootstrap run request from the operation's sealed plan, keying the member
// configs by executor host as om_bootstrap expects.
func (e *installStepExecutor) plannedRun() extensionsTriggerBootstrapRunRequest {
	params := installParamsOf(e.op)
	hosts := make([]string, 0, len(params.NodeIDs))
	for _, nodeID := range params.NodeIDs {
		hosts = append(hosts, params.Hosts[nodeID])
	}
	memberConfigs := make(map[string]extensionsMemberConfig, len(params.MemberConfigs))
	for nodeID, raw := range params.MemberConfigs {
		member := omv1.BootstrapMemberConfig{}
		err := protojson.Unmarshal(raw, &member)
		if err != nil {
			continue
		}
		memberConfigs[params.Hosts[nodeID]] = extensionsMemberConfig{
			Priority: member.Priority, Votes: member.Votes, Hidden: member.GetHidden(),
			DelaySecs: member.GetDelaySecs(), BindIP: member.BindIp,
		}
	}
	return extensionsTriggerBootstrapRunRequest{
		Hosts:           hosts,
		InstallMethod:   "packages",
		OS:              params.OS,
		MongoDBVersion:  params.MongoDBVersion,
		ReplicaSetName:  params.ReplicaSetName,
		DataPath:        params.DataPath,
		LogPath:         params.LogPath,
		Port:            params.Port,
		BindIP:          params.BindIP,
		MemberConfigs:   memberConfigs,
		CreateAdminUser: params.AdminUsername != "",
	}
}

// installStepExecutorProvider hands the install step executor to an install operation and the pending
// stub to any other kind, which has no steps yet.
type installStepExecutorProvider struct {
	db        *reform.DB
	bootstrap bootstrapRunner
}

// Executor returns the install executor for an install operation, else the pending stub.
//
//nolint:ireturn
func (p installStepExecutorProvider) Executor(op *models.OmOperation, kind OperationKind) (StepExecutor, error) {
	if kind.Kind == installKind {
		return &installStepExecutor{db: p.db, bootstrap: p.bootstrap, op: op}, nil
	}
	return pendingStepExecutor{}, nil
}
