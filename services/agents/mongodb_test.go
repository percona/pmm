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
	"testing"

	"github.com/AlekSi/pointer"
	api "github.com/percona/pmm/api/agent"
	"github.com/stretchr/testify/assert"

	"github.com/percona/pmm-managed/models"
)

func TestMongodbExporterConfig(t *testing.T) {
	mongo := &models.Service{
		Address: pointer.ToString("1.2.3.4"),
		Port:    pointer.ToUint16(27017),
	}
	exporter := &models.Agent{
		Username: pointer.ToString("username"),
		Password: pointer.ToString("s3cur3 p@$$w0r4."),
	}
	actual := mongodbExporterConfig(mongo, exporter)
	expected := &api.SetStateRequest_AgentProcess{
		Type:               api.Type_MONGODB_EXPORTER,
		TemplateLeftDelim:  "{{",
		TemplateRightDelim: "}}",
		Args: []string{
			"--collect.collection",
			"--collect.database",
			"--collect.indexusage",
			"--collect.topmetrics",
			"--web.listen-address=:{{ .listen_port }}",
		},
		Env: []string{
			"MONGODB_URI=mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:27017",
		},
	}
	assert.Equal(t, expected.Args, actual.Args)
	assert.Equal(t, expected.Env, actual.Env)
	assert.Equal(t, expected, actual)
}
