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
	"strings"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm-managed/models"
)

func TestRDSExporterConfig(t *testing.T) {
	pairs := map[*models.Node]*models.Agent{
		{
			Region:  pointer.ToString("region"),
			Address: "instance",
		}: {
			AWSAccessKey: pointer.ToString("access_key"),
			AWSSecretKey: pointer.ToString("secret_key"),
		},
	}

	actual := rdsExporterConfig(pairs, redactSecrets)
	expected := &agentpb.SetStateRequest_AgentProcess{
		Type:               inventorypb.AgentType_RDS_EXPORTER,
		TemplateLeftDelim:  "{{",
		TemplateRightDelim: "}}",
		Args: []string{
			"--config.file={{ .TextFiles.config }}",
			"--web.listen-address=:{{ .listen_port }}",
		},
		TextFiles: map[string]string{
			`config`: strings.TrimSpace(`
---
instances:
- region: region
  instance: instance
  aws_access_key: access_key
  aws_secret_key: secret_key
			`) + "\n",
		},
		RedactWords: []string{"secret_key"},
	}
	require.Equal(t, expected.Args, actual.Args)
	require.Equal(t, expected.Env, actual.Env)
	require.Equal(t, expected.TextFiles, actual.TextFiles)
	require.Equal(t, expected, actual)
}
