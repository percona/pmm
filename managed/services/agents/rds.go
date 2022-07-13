// Copyright (C) 2017 Percona LLC
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
	"sort"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"
	"github.com/prometheus/common/model"
	"gopkg.in/yaml.v3"

	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/managed/models"
)

// rdsInstance represents a single RDS instance information from configuration file.
type rdsInstance struct {
	Region                 string         `yaml:"region"`
	Instance               string         `yaml:"instance"`
	AWSAccessKey           string         `yaml:"aws_access_key,omitempty"`
	AWSSecretKey           string         `yaml:"aws_secret_key,omitempty"`
	DisableBasicMetrics    bool           `yaml:"disable_basic_metrics"`
	DisableEnhancedMetrics bool           `yaml:"disable_enhanced_metrics"`
	Labels                 model.LabelSet `yaml:"labels,omitempty"`
}

// Config contains configuration file information.
type rdsExporterConfigFile struct {
	Instances []rdsInstance `yaml:"instances"`
}

func mergeLabels(node *models.Node, agent *models.Agent) (model.LabelSet, error) {
	labels, err := models.MergeLabels(node, nil, agent)
	if err != nil {
		return nil, err
	}

	res := make(model.LabelSet, len(labels))
	for name, value := range labels {
		res[model.LabelName(name)] = model.LabelValue(value)
	}

	// added to labels anyway
	delete(res, "region")

	if err = res.Validate(); err != nil {
		return nil, errors.Wrap(err, "failed to merge labels")
	}
	return res, nil
}

// rdsExporterConfig returns desired configuration of rds_exporter process.
func rdsExporterConfig(pairs map[*models.Node]*models.Agent, redactMode redactMode) (*agentpb.SetStateRequest_AgentProcess, error) {
	config := rdsExporterConfigFile{
		Instances: make([]rdsInstance, 0, len(pairs)),
	}
	wordsSet := make(map[string]struct{}, len(pairs))
	for node, exporter := range pairs {
		labels, err := mergeLabels(node, exporter)
		if err != nil {
			return nil, err
		}

		config.Instances = append(config.Instances, rdsInstance{
			Region:                 pointer.GetString(node.Region),
			Instance:               node.Address,
			AWSAccessKey:           pointer.GetString(exporter.AWSAccessKey),
			AWSSecretKey:           pointer.GetString(exporter.AWSSecretKey),
			Labels:                 labels,
			DisableBasicMetrics:    exporter.RDSBasicMetricsDisabled,
			DisableEnhancedMetrics: exporter.RDSEnhancedMetricsDisabled,
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
	sort.Strings(args)

	b, err := yaml.Marshal(config)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &agentpb.SetStateRequest_AgentProcess{
		Type:               inventorypb.AgentType_RDS_EXPORTER,
		TemplateLeftDelim:  tdp.Left,
		TemplateRightDelim: tdp.Right,
		Args:               args,
		TextFiles: map[string]string{
			"config": "---\n" + string(b),
		},
		RedactWords: words,
	}, nil
}
