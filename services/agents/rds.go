// pmm-managed
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

	"gopkg.in/yaml.v2"

	"github.com/AlekSi/pointer"
	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"

	"github.com/percona/pmm-managed/models"
)

// rdsInstance represents a single RDS instance information from configuration file.
type rdsInstance struct {
	Region       string `yaml:"region"`
	Instance     string `yaml:"instance"`
	AWSAccessKey string `yaml:"aws_access_key,omitempty"`
	AWSSecretKey string `yaml:"aws_secret_key,omitempty"`
}

// Config contains configuration file information.
type rdsExporterConfigFile struct {
	Instances []rdsInstance `yaml:"instances"`
}

// rdsExporterConfig returns desired configuration of rds_exporter process.
func rdsExporterConfig(pairs map[*models.Node]*models.Agent) *agentpb.SetStateRequest_AgentProcess {
	var config rdsExporterConfigFile
	for node, exporter := range pairs {
		config.Instances = append(config.Instances, rdsInstance{
			Region:       pointer.GetString(node.Region),
			Instance:     node.Address,
			AWSAccessKey: pointer.GetString(exporter.AWSAccessKey),
			AWSSecretKey: pointer.GetString(exporter.AWSSecretKey),
		})
	}

	tdp := templateDelimsPair()

	args := []string{
		"--web.listen-address=:" + tdp.left + " .listen_port " + tdp.right,
		"--config.file=" + tdp.left + " .TextFiles.config " + tdp.right,
	}
	sort.Strings(args)

	b, _ := yaml.Marshal(config)

	return &agentpb.SetStateRequest_AgentProcess{
		Type:               inventorypb.AgentType_RDS_EXPORTER,
		TemplateLeftDelim:  tdp.left,
		TemplateRightDelim: tdp.right,
		Args:               args,
		TextFiles: map[string]string{
			"config": "---\n" + string(b),
		},
	}
}
