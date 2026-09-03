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

package alerting

import (
	"fmt"
	"time"

	"github.com/percona/pmm/managed/models"
)

// PMM provisions a fixed set of alert rules from its own built-in templates, so that a server
// monitors itself without anyone creating rules by hand. The rules are grouped into bundles: a
// bundle is a set of rules that share a folder, a gate deciding whether the bundle applies to this
// deployment at all, and a single user-facing toggle.
//
// The identity of every rule below (UID, folder, group) is frozen. Users attach silences and
// notification policies to it, so changing any of it in a later release silently breaks their
// routing. Retiring a rule therefore means moving its UID to retiredRuleUIDs, never deleting it:
// Grafana does not remove a rule that merely disappears from a provisioning file, so a forgotten
// UID becomes an orphan that the user cannot delete through the UI.
const (
	// The HA bundle covers a PMM Server High Availability cluster. Its metrics only exist in HA mode.
	haBundleID = "ha"
	// The components bundle covers what PMM Server itself is built from. It applies to every
	// deployment, standalone included.
	componentsBundleID = "components"

	haFolderTitle         = "PMM High Availability"
	componentsFolderTitle = "PMM Server"

	// Both bundles evaluate in one group.
	provisionedGroupName = "PMM Managed"

	// The evaluation interval must stay a multiple of Grafana's scheduler base interval, otherwise
	// Grafana rejects the whole group and refuses to start.
	provisionedInterval = time.Minute

	// Grafana's default scheduler base interval, which every rule group's evaluation interval has
	// to be a multiple of.
	grafanaSchedulerBaseInterval = 10 * time.Second

	// PMM uses a single Grafana organisation. It has to be set explicitly on each deletion entry:
	// unlike a rule group, a delete entry gets no default, and an entry without it silently
	// deletes from organisation 0 instead.
	provisionedOrgID = 1
)

// provisionedRule names one rule PMM maintains. Everything else about it - title, severity, `for`
// and parameter values - is read from the built-in template, so the template stays the single
// source of truth. TestProvisionedRuleContract asserts those derived values against the frozen
// contract in the specification.
type provisionedRule struct {
	// uid is the rule's permanent identity in Grafana. Frozen.
	uid string
	// templateName is the built-in template the rule is rendered from.
	templateName string
}

// bundleGates are the deployment facts a bundle's gate is decided from. All three are fixed for the
// lifetime of the process: they come from environment variables, so changing one means recreating
// the container. Percona Alerting is deliberately not here - that one is a real setting, changeable
// at runtime, and is checked separately by the renderer.
type bundleGates struct {
	// haEnabled reports whether this deployment is a cluster at all.
	haEnabled bool
	// haAlertsEnabled and componentAlertsEnabled are PMM_ENABLE_HA_ALERTS and
	// PMM_ENABLE_COMPONENT_ALERTS, both defaulting to true.
	haAlertsEnabled        bool
	componentAlertsEnabled bool
}

// provisioningBundle is a set of rules sharing a folder, a gate and a toggle.
type provisioningBundle struct {
	id     string
	folder string
	rules  []provisionedRule
	// enabled reports whether this bundle applies: its own toggle, and for HA the additional
	// condition that the deployment is actually a cluster.
	enabled func(gates bundleGates) bool
}

// builtinBundles is the full catalog, in the order they are rendered.
var builtinBundles = []provisioningBundle{
	{
		id:     haBundleID,
		folder: haFolderTitle,
		rules: []provisionedRule{
			{uid: "pmm-ha-no-leader", templateName: "pmm_ha_no_leader"},
			{uid: "pmm-ha-split-brain", templateName: "pmm_ha_split_brain"},
			{uid: "pmm-ha-quorum-at-risk", templateName: "pmm_ha_quorum_at_risk"},
			{uid: "pmm-ha-leader-flapping", templateName: "pmm_ha_leader_flapping"},
			{uid: "pmm-ha-node-unreachable", templateName: "pmm_ha_node_unreachable"},
		},
		enabled: func(gates bundleGates) bool {
			return gates.haEnabled && gates.haAlertsEnabled
		},
	},
	{
		id:     componentsBundleID,
		folder: componentsFolderTitle,
		rules: []provisionedRule{
			{uid: "pmm-victoriametrics-down", templateName: "pmm_victoriametrics_down"},
			{uid: "pmm-clickhouse-down", templateName: "pmm_clickhouse_down"},
			{uid: "pmm-grafana-down", templateName: "pmm_grafana_down"},
			{uid: "pmm-qan-api2-down", templateName: "pmm_qan_api2_down"},
		},
		enabled: func(gates bundleGates) bool {
			return gates.componentAlertsEnabled
		},
	},
}

// retiredRuleUIDs holds the UIDs of rules PMM used to provision and no longer does. Append only:
// removing a line here leaves the rule behind forever on every server that ever ran the release
// that created it.
var retiredRuleUIDs []string

// catalogUIDs returns every rule UID PMM claims: both bundles, whether or not they apply to this
// deployment, plus the retired ones. A disabled bundle's UIDs still appear in the file as deletions,
// so they are just as much PMM's to account for as the ones being provisioned.
func catalogUIDs() []string {
	var uids []string
	for _, bundle := range builtinBundles {
		for _, rule := range bundle.rules {
			uids = append(uids, rule.uid)
		}
	}
	return append(uids, retiredRuleUIDs...)
}

// validateCatalog checks the invariants Grafana enforces at startup, where a violation means
// Grafana refuses to start rather than reporting a bad rule. Called from the renderer so a mistake
// surfaces as a rendering failure - which degrades to "no bundle" - instead of a dead PMM.
func validateCatalog(templates map[string]models.Template) error {
	if provisionedInterval%grafanaSchedulerBaseInterval != 0 {
		return fmt.Errorf("evaluation interval %s is not a multiple of %s", provisionedInterval, grafanaSchedulerBaseInterval)
	}

	seen := make(map[string]string, len(builtinBundles))
	for _, bundle := range builtinBundles {
		if len(bundle.rules) == 0 {
			return fmt.Errorf("bundle %q has no rules", bundle.id)
		}

		for _, rule := range bundle.rules {
			if other, ok := seen[rule.uid]; ok {
				return fmt.Errorf("rule UID %q is used by both bundle %q and bundle %q", rule.uid, other, bundle.id)
			}
			seen[rule.uid] = bundle.id

			template, ok := templates[rule.templateName]
			if !ok {
				return fmt.Errorf("bundle %q references unknown built-in template %q", bundle.id, rule.templateName)
			}

			// Grafana's file provisioner does not check this, but CreateAlertRule does, and a
			// rule whose `for` is shorter than its evaluation interval never settles.
			if template.For < provisionedInterval {
				return fmt.Errorf("template %q has for=%s, shorter than the evaluation interval %s",
					rule.templateName, template.For, provisionedInterval)
			}
		}
	}

	for _, uid := range retiredRuleUIDs {
		if bundleID, ok := seen[uid]; ok {
			return fmt.Errorf("retired rule UID %q is still provisioned by bundle %q", uid, bundleID)
		}
	}

	return nil
}
