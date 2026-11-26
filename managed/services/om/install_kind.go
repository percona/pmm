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
	"encoding/json"
	"time"

	"github.com/percona/pmm/managed/models"
)

// installKind is the operation kind that installs MongoDB on one or three nodes as a managed cluster.
const installKind = "install"

// Install-readiness fact names, as PMM Extensions' install_check scan stores them per install_node and
// the check provider aggregates across the operation's nodes into the one check context the driver reads.
const (
	factRepoReachable         = "repo.reachable"
	factPackageManagerPresent = "install_readiness.package_manager_present"
	factNoMongod              = "install_readiness.no_mongod"
	factFreeSpaceBytes        = "install_readiness.data_dir_free_bytes"
)

// subjectKindInstallNode labels a target node as an install-readiness subject, the key its install_check
// facts are read under.
const subjectKindInstallNode = "install_node"

// installFactMaxAge bounds how old an install-readiness fact may be before it reads as missing.
const installFactMaxAge = 5 * time.Minute

// installMinFreeSpaceBytes is the free space an install needs at the data path before it begins.
const installMinFreeSpaceBytes float64 = 5 << 30

// installDeadline bounds how long an install may stay queued and running before it is timed out.
const installDeadline = 30 * time.Minute

// installParams is an install operation's sealed, non-secret plan, stored in om_operations.params and
// read back by the lock set, the check subjects and the later phases; the admin password is never here.
type installParams struct {
	NodeIDs        []string                   `json:"node_ids"`
	Hosts          map[string]string          `json:"hosts,omitempty"`
	OS             string                     `json:"os,omitempty"`
	ReplicaSetName string                     `json:"replica_set_name"`
	MongoDBVersion string                     `json:"mongodb_version"`
	DataPath       string                     `json:"data_path"`
	LogPath        string                     `json:"log_path"`
	Port           uint32                     `json:"port"`
	BindIP         string                     `json:"bind_ip"`
	MemberConfigs  map[string]json.RawMessage `json:"member_configs,omitempty"`
	Environment    string                     `json:"environment,omitempty"`
	Cluster        string                     `json:"cluster,omitempty"`
	AdminUsername  string                     `json:"admin_username,omitempty"`
	Repo           string                     `json:"repo,omitempty"`
	Packages       []string                   `json:"packages,omitempty"`
	ControlUnit    string                     `json:"control_unit,omitempty"`
}

// installParamsOf reads an install operation's params, returning an empty value when they are unset or
// malformed; the trigger validates them before the row is written.
func installParamsOf(op *models.OmOperation) installParams {
	params := installParams{}
	if len(op.Params) > 0 {
		_ = json.Unmarshal(op.Params, &params)
	}
	return params
}

// InstallKind returns the install kind: it locks its target nodes exclusively and resumes a failed
// run, exempt from attention as it creates a fresh cluster; its steps and hooks come in later phases.
func InstallKind() OperationKind {
	return OperationKind{
		Kind:        installKind,
		ClusterLess: true,
		Resumable:   true,
		Deadline:    installDeadline,
		Attention:   AttentionPolicy{ExemptAll: true},
		LockSet: func(op *models.OmOperation) []LockResource {
			params := installParamsOf(op)
			resources := make([]LockResource, 0, len(params.NodeIDs))
			for _, nodeID := range params.NodeIDs {
				resources = append(resources, LockResource{Kind: "node", ID: nodeID, Mode: models.OmLockExclusive})
			}
			return resources
		},
		Steps: func(op *models.OmOperation) []StepPlan {
			return []StepPlan{{
				Phase: installBootstrapPhase, Step: installBootstrapStep,
				TargetKind: models.OmStepTargetRun, TargetID: op.ID,
			}}
		},
		OnEnterRunning: installOnEnterRunning,
		OnSucceeded:    installOnSucceeded,
		OnTerminal:     installOnTerminal,
		CheckSet:       installReadinessCheckSet(),
		Subjects: func(op *models.OmOperation) []CheckSubject {
			params := installParamsOf(op)
			subjects := make([]CheckSubject, 0, len(params.NodeIDs))
			for _, nodeID := range params.NodeIDs {
				subjects = append(subjects, CheckSubject{Kind: subjectKindInstallNode, Key: nodeID})
			}
			return subjects
		},
	}
}

// installReadinessCheckSet declares install readiness's facts and rules, aggregated across the nodes so
// a boolean passes only when every node does; the OS and version checks stay in the synchronous trigger.
func installReadinessCheckSet() CheckSet {
	return CheckSet{
		Facts: []CheckFact{
			{Name: factRepoReachable, MaxAge: installFactMaxAge},
			{Name: factPackageManagerPresent, MaxAge: installFactMaxAge},
			{Name: factNoMongod, MaxAge: installFactMaxAge},
			{Name: factFreeSpaceBytes, MaxAge: installFactMaxAge},
		},
		Rules: []BlockingRule{
			boolFactRule(factRepoReachable, "the package repository is unreachable from a target node"),
			boolFactRule(factPackageManagerPresent, "a target node has no usable package manager"),
			boolFactRule(factNoMongod, "a target node already runs mongod"),
			freeSpaceRule(factFreeSpaceBytes, installMinFreeSpaceBytes),
		},
	}
}

// boolFactRule blocks when a boolean readiness fact is missing or false, naming the fact as the path.
func boolFactRule(name, reason string) BlockingRule {
	return func(ctx CheckContext) RuleResult {
		ok, _ := ctx.Facts[name].(bool)
		if ok {
			return RuleResult{}
		}
		return RuleResult{Blocked: true, Reason: reason, Path: name}
	}
}

// freeSpaceRule blocks when the least free space across the nodes is below the install minimum, or the
// fact is missing.
func freeSpaceRule(name string, minBytes float64) BlockingRule {
	return func(ctx CheckContext) RuleResult {
		free, ok := ctx.Facts[name].(float64)
		if ok && free >= minBytes {
			return RuleResult{}
		}
		return RuleResult{Blocked: true, Reason: "a target node has insufficient free space", Path: name}
	}
}
