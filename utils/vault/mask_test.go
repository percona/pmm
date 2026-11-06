// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package vault

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaskConfigSecrets(t *testing.T) {
	t.Parallel()

	in := `scrape_configs:
  - job_name: mysqld_exporter
    basic_auth:
      username: pmm
      password: metrics-pw
    bearer_token: tok
server:
    password: vault:secret/data/pmm/agents#token
vault:
    token: ab
    secret-id: "s3cr3t"
    role-id: role
password: top-level
`
	assert.Equal(t, `scrape_configs:
  - job_name: mysqld_exporter
    basic_auth:
      username: pmm
      password: '***'
    bearer_token: '***'
server:
    password: vault:secret/data/pmm/agents#token
vault:
    token: '***'
    secret-id: '***'
    role-id: role
password: '***'
`, string(MaskConfigSecrets([]byte(in))))
}
