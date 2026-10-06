// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package supervisor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAzureMetricsExporterRegexp(t *testing.T) {
	t.Parallel()

	output := `azure_exporter, version 0.2.0 (branch: , revision: unknown)
  build user:
  build date:       2026-09-28T16:50:22+0000
  go version:       go1.27.1
  platform:         linux/arm64
  tags:             unknown
`
	matches := azureMetricsExporterRegexp.FindStringSubmatch(output)
	require.Len(t, matches, 2)
	assert.Equal(t, "0.2.0", matches[1])
}
