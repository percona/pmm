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

package models

import (
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/hashicorp/go-version"
	"github.com/stretchr/testify/assert"
)

func TestPMMAgentSupported(t *testing.T) {
	t.Parallel()
	prefix := "testing prefix"
	minVersion := version.Must(version.NewVersion("2.30.5"))

	tests := []struct {
		name         string
		agentVersion string
		errString    string
	}{
		{
			name:         "Empty version string",
			agentVersion: "",
			errString:    "failed to parse PMM agent version",
		},
		{
			name:         "Wrong version string",
			agentVersion: "Some version",
			errString:    "failed to parse PMM agent version",
		},
		{
			name:         "Less than min version",
			agentVersion: "2.30.4",
			errString:    "not supported by pmm-agent",
		},
		{
			name:         "Equals min version",
			agentVersion: "2.30.5",
			errString:    "",
		},
		{
			name:         "Greater than min version",
			agentVersion: "2.30.6",
			errString:    "",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			agentModel := Agent{
				AgentID: "Test agent ID",
				Version: pointer.ToString(test.agentVersion),
			}
			err := isAgentSupported(&agentModel, prefix, minVersion)
			if test.errString == "" {
				assert.NoError(t, err)
			} else {
				assert.Contains(t, err.Error(), test.errString)
			}
		})
	}

	t.Run("No version info", func(t *testing.T) {
		err := isAgentSupported(&Agent{AgentID: "Test agent ID"}, prefix, version.Must(version.NewVersion("2.30.0")))
		assert.Contains(t, err.Error(), "has no version info")
	})

	t.Run("Nil agent", func(t *testing.T) {
		err := isAgentSupported(nil, prefix, version.Must(version.NewVersion("2.30.0")))
		assert.Contains(t, err.Error(), "nil agent")
	})
}
