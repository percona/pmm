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

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForceNewAgentTokenSetupFlag(t *testing.T) {
	for _, tt := range []struct {
		name     string
		args     []string
		expected bool
	}{
		{name: "disabled by default"},
		{name: "enabled by flag", args: []string{"--force-new-agent-token"}, expected: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{}
			app, _ := Application(cfg)
			args := append([]string{"setup"}, tt.args...)
			args = append(args, "127.0.0.1", "generic", "test-node")

			_, err := app.Parse(args)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, cfg.Setup.ForceNewAgentToken)
		})
	}
}
