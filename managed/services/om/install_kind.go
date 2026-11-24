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

// Install-readiness fact names: the install_check facts collected per install_node, aggregated across
// the operation's nodes into the one check context the driver evaluates.
const (
	factNodesReachable       = "nodes_reachable"
	factOSSupported          = "os_supported"
	factPortFree             = "port_free"
	factNoMongodPresent      = "no_mongod_present"
	factRepoReachable        = "repo_reachable"
	factPackageManagerUsable = "package_manager_usable"
	factFreeSpaceBytes       = "free_space_bytes"
	factHostPackageInstall   = "host.package_install"
)

// subjectKindInstallNode labels a target node as an install-readiness subject, the key its install_check
// facts are read under.
const subjectKindInstallNode = "install_node"

// installFactMaxAge bounds how old an install-readiness fact may be before it reads as missing.
const installFactMaxAge = 5 * time.Minute

// installMinFreeSpaceBytes is the free space an install needs at the data path before it begins.
const installMinFreeSpaceBytes float64 = 5 << 30

// installParams carries an install operation's target nodes and their configuration, stored in
// om_operations.params and read back by the lock set and the steps.
type installParams struct {
	NodeIDs        []string                   `json:"node_ids"`
	MemberConfigs  map[string]json.RawMessage `json:"member_configs,omitempty"`
	MongoDBVersion string                     `json:"mongodb_version"`
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
		Deadline:    30 * time.Minute,
		Attention:   AttentionPolicy{ExemptAll: true},
		LockSet: func(op *models.OmOperation) []LockResource {
			params := installParamsOf(op)
			resources := make([]LockResource, 0, len(params.NodeIDs))
			for _, nodeID := range params.NodeIDs {
				resources = append(resources, LockResource{Kind: "node", ID: nodeID, Mode: models.OmLockExclusive})
			}
			return resources
		},
		Steps:    func(*models.OmOperation) []StepPlan { return nil },
		CheckSet: installReadinessCheckSet(),
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

// installReadinessCheckSet declares install readiness's facts and rules, read over facts aggregated
// across the nodes so a boolean passes only when every node does and free_space_bytes is the least.
func installReadinessCheckSet() CheckSet {
	return CheckSet{
		Facts: []CheckFact{
			{Name: factNodesReachable, MaxAge: installFactMaxAge},
			{Name: factOSSupported, MaxAge: installFactMaxAge},
			{Name: factPortFree, MaxAge: installFactMaxAge},
			{Name: factNoMongodPresent, MaxAge: installFactMaxAge},
			{Name: factRepoReachable, MaxAge: installFactMaxAge},
			{Name: factPackageManagerUsable, MaxAge: installFactMaxAge},
			{Name: factFreeSpaceBytes, MaxAge: installFactMaxAge},
			{Name: factHostPackageInstall, MaxAge: installFactMaxAge},
		},
		Rules: []BlockingRule{
			boolFactRule(factNodesReachable, "a target node is unreachable"),
			boolFactRule(factOSSupported, "a target node runs an unsupported OS"),
			boolFactRule(factPortFree, "the MongoDB port is in use on a target node"),
			boolFactRule(factNoMongodPresent, "a target node already runs mongod"),
			boolFactRule(factRepoReachable, "the package repository is unreachable from a target node"),
			boolFactRule(factPackageManagerUsable, "the package manager is unusable on a target node"),
			boolFactRule(factHostPackageInstall, "a target node cannot install packages, elevation is blocked"),
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
