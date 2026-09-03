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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/prometheus/common/model"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/pi/common"
	"github.com/percona/pmm/managed/services"
)

// Grafana reads alert rules from a provisioning file at startup and on an explicit reload. The
// schema below is Grafana's, not PMM's: it looks similar to the ruler API that CreateRule posts to,
// but the field names differ (noDataState here, no_data_state there), so it needs its own structs.
//
// Grafana interpolates environment variables into most string fields of a provisioning file, but
// reads query models and annotations raw. Anything rendered into a label value must therefore not
// contain a dollar sign, or Grafana expands it into an empty string and keeps the broken result
// without reporting an error.
type provisioningFile struct {
	APIVersion  int                      `json:"apiVersion"`
	Groups      []provisioningGroup      `json:"groups"`
	DeleteRules []provisioningRuleDelete `json:"deleteRules"`
}

type provisioningGroup struct {
	OrgID    int                `json:"orgId"`
	Name     string             `json:"name"`
	Folder   string             `json:"folder"`
	Interval string             `json:"interval"`
	Rules    []provisioningRule `json:"rules"`
}

type provisioningRule struct {
	UID          string            `json:"uid"`
	Title        string            `json:"title"`
	Condition    string            `json:"condition"`
	For          string            `json:"for"`
	NoDataState  string            `json:"noDataState"`
	ExecErrState string            `json:"execErrState"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	Data         []services.Data   `json:"data"`
}

type provisioningRuleDelete struct {
	OrgID int    `json:"orgId"`
	UID   string `json:"uid"`
}

const (
	// No Data has to map to Normal, so that a rule stays silent when its series are absent: the
	// HA metrics on a standalone server, or a component this deployment does not scrape. The file
	// provisioner defaults this to NoData, not to Grafana's UI default of OK, so it must be set.
	provisionedNoDataState = "OK"

	// The execution error state deliberately differs from the Alerting that CreateRule uses. These
	// rules are owned by PMM and a user cannot fix them, so an execution error must not page an
	// entire fleet; it is reported through pmm_alerting_provisioning_* metrics instead.
	provisionedExecErrState = "OK"

	// A label marks each rule as maintained by PMM, so users can match on it in notification
	// policies and silences. Grafana's own file provenance is metadata on the rule rather than a
	// label the alert carries, so it cannot serve that purpose.
	provisionedLabel = "pmm_provisioned"
)

// renderProvisioningFile builds the Grafana alerting provisioning file for this server. It is a
// pure function of the built-in templates, the resolved Metrics datasource UID and the settings,
// so that the caller can compare its output against what was last applied and do nothing when
// nothing changed.
//
// A bundle that does not apply contributes no group and instead has every one of its UIDs listed
// for deletion, because Grafana leaves a rule in place when it merely disappears from the file.
func renderProvisioningFile(
	templates map[string]models.Template,
	datasourceUID string,
	settings *models.Settings,
	gates bundleGates,
	notOurs map[string]string,
) ([]byte, error) {
	if datasourceUID == "" {
		return nil, errors.New("metrics datasource UID is empty")
	}

	err := validateCatalog(templates)
	if err != nil {
		return nil, fmt.Errorf("invalid provisioning catalog: %w", err)
	}

	file := provisioningFile{
		APIVersion:  1,
		Groups:      []provisioningGroup{},
		DeleteRules: []provisioningRuleDelete{},
	}

	// Percona Alerting owns these templates, so turning it off has to take the rules with it.
	// Nothing else in PMM disables Grafana's unified alerting, so rules left behind would keep
	// evaluating while the user's Alerting page says the feature is off.
	alertingEnabled := settings.IsAlertingEnabled()

	for _, bundle := range builtinBundles {
		if !alertingEnabled || !bundle.enabled(gates) {
			for _, rule := range bundle.rules {
				if _, taken := notOurs[rule.uid]; taken {
					// Deleting a rule PMM does not own would destroy someone else's work.
					continue
				}
				file.DeleteRules = append(file.DeleteRules, provisioningRuleDelete{OrgID: provisionedOrgID, UID: rule.uid})
			}
			continue
		}

		group := provisioningGroup{
			OrgID:    provisionedOrgID,
			Name:     provisionedGroupName,
			Folder:   bundle.folder,
			Interval: model.Duration(provisionedInterval).String(),
			Rules:    make([]provisioningRule, 0, len(bundle.rules)),
		}

		for _, rule := range bundle.rules {
			if _, taken := notOurs[rule.uid]; taken {
				// Claiming a UID someone else owns either kills Grafana or silently overwrites
				// their rule, depending on how theirs was made. Leave it alone; the rule is simply
				// absent until they release it, which the caller logs and counts.
				continue
			}

			rendered, err := renderProvisionedRule(rule, templates[rule.templateName], datasourceUID)
			if err != nil {
				return nil, fmt.Errorf("failed to render rule %q: %w", rule.uid, err)
			}
			group.Rules = append(group.Rules, rendered)
		}

		// A group with no rules left is not written at all: Grafana rejects an empty group.
		if len(group.Rules) == 0 {
			continue
		}

		file.Groups = append(file.Groups, group)
	}

	for _, uid := range retiredRuleUIDs {
		if _, taken := notOurs[uid]; taken {
			continue
		}
		file.DeleteRules = append(file.DeleteRules, provisioningRuleDelete{OrgID: provisionedOrgID, UID: uid})
	}

	return json.MarshalIndent(file, "", "  ")
}

// renderProvisionedRule builds one rule from its template, reproducing what CreateRule would
// produce from the same template, with one deliberate difference: the node_name and service_name
// label templates are not added. They name a monitored database service, which none of these rules
// has, and their values contain dollar signs that Grafana's provisioning interpolation would eat.
func renderProvisionedRule(rule provisionedRule, template models.Template, datasourceUID string) (provisioningRule, error) {
	alertTemplate, err := parseAlertTemplate(template.Yaml)
	if err != nil {
		return provisioningRule{}, fmt.Errorf("failed to parse template: %w", err)
	}

	params, err := defaultParamsValues(template.Params)
	if err != nil {
		return provisioningRule{}, err
	}
	paramsMap := params.AsStringMap()

	data, condition, err := buildGrafanaRuleData(alertTemplate, datasourceUID, paramsMap, nil)
	if err != nil {
		return provisioningRule{}, fmt.Errorf("failed to build rule data: %w", err)
	}

	templateAnnotations, err := template.GetAnnotations()
	if err != nil {
		return provisioningRule{}, fmt.Errorf("failed to get template annotations: %w", err)
	}
	annotations := make(map[string]string, len(templateAnnotations))
	err = transformMaps(templateAnnotations, annotations, paramsMap)
	if err != nil {
		return provisioningRule{}, fmt.Errorf("failed to fill template annotations: %w", err)
	}

	templateLabels, err := template.GetLabels()
	if err != nil {
		return provisioningRule{}, fmt.Errorf("failed to get template labels: %w", err)
	}
	labels := make(map[string]string, len(templateLabels))
	err = transformMaps(templateLabels, labels, paramsMap)
	if err != nil {
		return provisioningRule{}, fmt.Errorf("failed to fill template labels: %w", err)
	}

	labels["percona_alerting"] = "1"
	labels["severity"] = common.Severity(template.Severity).String()
	labels["template_name"] = template.Name
	labels[provisionedLabel] = "1"

	for name, value := range labels {
		if strings.ContainsRune(value, '$') {
			return provisioningRule{}, fmt.Errorf("label %q contains a dollar sign, which Grafana provisioning would expand: %q", name, value)
		}
	}

	return provisioningRule{
		UID:          rule.uid,
		Title:        template.Summary,
		Condition:    condition,
		For:          model.Duration(template.For).String(),
		NoDataState:  provisionedNoDataState,
		ExecErrState: provisionedExecErrState,
		Labels:       labels,
		Annotations:  annotations,
		Data:         data,
	}, nil
}

// defaultParamsValues takes each template parameter at its shipped default. Provisioned rules are
// not configurable by design: a user who wants a different threshold creates their own rule from
// the same template.
func defaultParamsValues(definitions models.AlertExprParamsDefinitions) (AlertExprParamsValues, error) {
	values := make(AlertExprParamsValues, 0, len(definitions))
	for _, definition := range definitions {
		switch definition.Type {
		case models.Float:
			if definition.FloatParam == nil || definition.FloatParam.Default == nil {
				return nil, fmt.Errorf("parameter %q has no default value", definition.Name)
			}
			values = append(values, AlertExprParamValue{
				Name:       definition.Name,
				Type:       models.Float,
				FloatValue: *definition.FloatParam.Default,
			})
		case models.Bool, models.String:
			return nil, fmt.Errorf("parameter %q has unsupported type %q for a provisioned rule", definition.Name, definition.Type)
		default:
			return nil, fmt.Errorf("parameter %q has unknown type %q", definition.Name, definition.Type)
		}
	}

	return values, nil
}

// validateProvisioningFile parses rendered content back and asserts the invariants Grafana asserts
// while starting up. Grafana treats a bad provisioning file as a fatal startup error, which would
// take down the whole PMM user interface and API rather than only alerting, so nothing is written
// to disk until it has passed this.
func validateProvisioningFile(content []byte) error {
	var file provisioningFile
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&file)
	if err != nil {
		return fmt.Errorf("failed to parse rendered file: %w", err)
	}

	if file.APIVersion != 1 {
		return fmt.Errorf("unexpected apiVersion %d", file.APIVersion)
	}

	provisioned := make(map[string]struct{})
	for _, group := range file.Groups {
		err = validateProvisioningGroup(group, provisioned)
		if err != nil {
			return err
		}
	}

	return validateProvisioningDeletions(file.DeleteRules, provisioned)
}

func validateProvisioningGroup(group provisioningGroup, provisioned map[string]struct{}) error {
	switch {
	case group.Name == "":
		return errors.New("rule group has no name")
	case group.Folder == "":
		return fmt.Errorf("rule group %q has no folder", group.Name)
	case group.Interval == "":
		return fmt.Errorf("rule group %q has no interval", group.Name)
	case group.OrgID != provisionedOrgID:
		return fmt.Errorf("rule group %q has orgId %d", group.Name, group.OrgID)
	}

	for _, rule := range group.Rules {
		err := validateProvisioningRule(rule)
		if err != nil {
			return err
		}
		provisioned[rule.UID] = struct{}{}
	}

	return nil
}

func validateProvisioningRule(rule provisioningRule) error {
	switch {
	case rule.UID == "":
		return fmt.Errorf("rule %q has no UID", rule.Title)
	case rule.Title == "":
		return fmt.Errorf("rule %q has no title", rule.UID)
	case rule.Condition == "":
		return fmt.Errorf("rule %q has no condition", rule.UID)
	case rule.For == "":
		return fmt.Errorf("rule %q has no for duration", rule.UID)
	case len(rule.Data) == 0:
		return fmt.Errorf("rule %q has no data", rule.UID)
	}

	// Grafana parses these with the Prometheus duration parser, which accepts units Go's
	// time.ParseDuration does not, so validate the way Grafana will.
	forDuration, err := model.ParseDuration(rule.For)
	if err != nil {
		return fmt.Errorf("rule %q has an unparseable for duration %q: %w", rule.UID, rule.For, err)
	}
	if time.Duration(forDuration) < provisionedInterval {
		return fmt.Errorf("rule %q has for=%s, shorter than the evaluation interval", rule.UID, rule.For)
	}

	var conditionFound bool
	for _, query := range rule.Data {
		if query.RefID == "" {
			return fmt.Errorf("rule %q has a query with no refId", rule.UID)
		}
		if query.DatasourceUID == "" {
			return fmt.Errorf("rule %q query %q has no datasource UID", rule.UID, query.RefID)
		}
		if query.RefID == rule.Condition {
			conditionFound = true
		}
	}
	if !conditionFound {
		return fmt.Errorf("rule %q names condition %q, which is not one of its queries", rule.UID, rule.Condition)
	}

	return nil
}

func validateProvisioningDeletions(deletions []provisioningRuleDelete, provisioned map[string]struct{}) error {
	for _, deletion := range deletions {
		if deletion.UID == "" {
			return errors.New("deletion entry has no UID")
		}
		// A deletion entry gets no default organisation, unlike a rule group, so one without an
		// explicit orgId deletes from organisation 0 and silently does nothing.
		if deletion.OrgID != provisionedOrgID {
			return fmt.Errorf("deletion of %q has orgId %d", deletion.UID, deletion.OrgID)
		}
		if _, ok := provisioned[deletion.UID]; ok {
			return fmt.Errorf("rule %q is both provisioned and deleted by the same file", deletion.UID)
		}
	}

	return nil
}
