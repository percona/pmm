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

package agents

import (
	"fmt"
	"maps"
	"sort"

	"github.com/AlekSi/pointer"
	"github.com/prometheus/common/model"
	"gopkg.in/yaml.v3"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/version"
)

// rdsInstance represents a single RDS instance information from configuration file.
type rdsInstance struct {
	Region                 string         `yaml:"region"`
	Instance               string         `yaml:"instance"`
	AWSAccessKey           string         `yaml:"aws_access_key,omitempty"`
	AWSSecretKey           string         `yaml:"aws_secret_key,omitempty"`
	AWSRoleArn             string         `yaml:"aws_role_arn,omitempty"`
	DisableBasicMetrics    bool           `yaml:"disable_basic_metrics"`
	DisableEnhancedMetrics bool           `yaml:"disable_enhanced_metrics"`
	Labels                 model.LabelSet `yaml:"labels,omitempty"`
}

// Config contains configuration file information.
type rdsExporterConfigFile struct {
	Instances []rdsInstance `yaml:"instances"`
}

// serviceStandardLabels are the Service labels that rds_exporter series carry, so that access roles
// filtering on them also match the metrics of the RDS Node (PMM-15868).
var serviceStandardLabels = []string{"environment", "cluster", "replication_set"}

// sharedServiceLabels returns the serviceStandardLabels on which all given Services have the same non-empty value.
// A Node without Services gets none, and a label on which its Services disagree is left out.
func sharedServiceLabels(services []*models.Service) (map[string]string, error) {
	var res map[string]string
	for i, service := range services {
		labels, err := service.UnifiedLabels()
		if err != nil {
			return nil, err
		}

		if i == 0 {
			res = make(map[string]string, len(serviceStandardLabels))
			for _, name := range serviceStandardLabels {
				if value, ok := labels[name]; ok {
					res[name] = value
				}
			}
			continue
		}

		for name, value := range res {
			if labels[name] != value {
				delete(res, name)
			}
		}
	}

	return res, nil
}

// mergeLabels returns the labels of the rds_exporter series of the given Node, merged in the order
// models.MergeLabels uses for the series of a Service: Node, Service, Agent.
func mergeLabels(node *models.Node, services []*models.Service, agent *models.Agent) (model.LabelSet, error) {
	labels, err := models.MergeLabels(node, nil, nil)
	if err != nil {
		return nil, err
	}

	shared, err := sharedServiceLabels(services)
	if err != nil {
		return nil, err
	}
	maps.Copy(labels, shared)

	agentLabels, err := agent.UnifiedLabels()
	if err != nil {
		return nil, err
	}
	maps.Copy(labels, agentLabels)

	res := make(model.LabelSet, len(labels))
	for name, value := range labels {
		res[model.LabelName(name)] = model.LabelValue(value)
	}

	// added to labels anyway
	delete(res, "region")

	err = res.Validate()
	if err != nil {
		return nil, fmt.Errorf("failed to merge labels: %w", err)
	}
	return res, nil
}

// checkRDSExporterSupported returns an error when exporter must not be sent to pmmAgent.
// A role-based rds_exporter needs pmm-agent 3.4.0: an older one accepts the config, reports
// RUNNING and scrapes nothing. Both models.CreateAgent and models.ChangeAgent refuse such a row,
// but they only see the version reported when the row is saved, so a pmm-agent downgraded since
// then is caught here, every time its state is sent.
func checkRDSExporterSupported(pmmAgent, exporter *models.Agent) error {
	if exporter.AWSOptions.AWSRoleARN == "" {
		return nil
	}
	return models.IsAgentSupported(pmmAgent, "AWS IAM role assumption", models.PMMAgentMinVersionForAWSRoleARN)
}

// rdsExporterConfig returns desired configuration of rds_exporter process.
// The nodeServices map holds the Services of each RDS Node by Node ID.
func rdsExporterConfig(
	pairs map[*models.Node]*models.Agent,
	nodeServices map[string][]*models.Service,
	redactMode redactMode,
	pmmAgentVersion *version.Parsed,
) (*agentv1.SetStateRequest_AgentProcess, error) {
	config := rdsExporterConfigFile{
		Instances: make([]rdsInstance, 0, len(pairs)),
	}
	wordsSet := make(map[string]struct{}, len(pairs))
	for node, exporter := range pairs {
		labels, err := mergeLabels(node, nodeServices[node.NodeID], exporter)
		if err != nil {
			return nil, err
		}

		config.Instances = append(config.Instances, rdsInstance{
			Region:                 pointer.GetString(node.Region),
			Instance:               node.InstanceID,
			AWSAccessKey:           exporter.AWSOptions.AWSAccessKey,
			AWSSecretKey:           exporter.AWSOptions.AWSSecretKey,
			AWSRoleArn:             exporter.AWSOptions.AWSRoleARN,
			Labels:                 labels,
			DisableBasicMetrics:    exporter.AWSOptions.RDSBasicMetricsDisabled,
			DisableEnhancedMetrics: exporter.AWSOptions.RDSEnhancedMetricsDisabled,
		})

		if redactMode != exposeSecrets {
			for _, word := range redactWords(exporter) {
				wordsSet[word] = struct{}{}
			}
		}
	}

	// sort by region and id
	sort.Slice(config.Instances, func(i, j int) bool {
		if config.Instances[i].Region != config.Instances[j].Region {
			return config.Instances[i].Region < config.Instances[j].Region
		}
		return config.Instances[i].Instance < config.Instances[j].Instance
	})

	words := make([]string, 0, len(wordsSet))
	for w := range wordsSet {
		words = append(words, w)
	}
	sort.Strings(words)

	tdp := models.TemplateDelimsPair()

	args := []string{
		"--web.listen-address=:" + tdp.Left + " .listen_port " + tdp.Right,
		"--config.file=" + tdp.Left + " .TextFiles.config " + tdp.Right,
	}

	args = withLogLevel(args, toOneLogLevel(pairs), pmmAgentVersion, true)

	sort.Strings(args)

	b, err := yaml.Marshal(config)
	if err != nil {
		return nil, err
	}

	return &agentv1.SetStateRequest_AgentProcess{
		Type:               inventoryv1.AgentType_AGENT_TYPE_RDS_EXPORTER,
		TemplateLeftDelim:  tdp.Left,
		TemplateRightDelim: tdp.Right,
		Args:               args,
		TextFiles: map[string]string{
			"config": "---\n" + string(b),
		},
		RedactWords: words,
	}, nil
}

// toOneLogLevel returns first if all equals.
func toOneLogLevel(pairs map[*models.Node]*models.Agent) *string {
	var (
		logLevel      *string
		logLevelFirst bool
	)

	for _, exporter := range pairs {
		if !logLevelFirst {
			logLevel = exporter.LogLevel

			logLevelFirst = true

			continue
		}

		if pointer.GetString(logLevel) != pointer.GetString(exporter.LogLevel) {
			return nil
		}
	}

	return logLevel
}
