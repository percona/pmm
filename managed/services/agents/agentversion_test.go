package agents

import (
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/hashicorp/go-version"
	"github.com/stretchr/testify/assert"

	"github.com/percona/pmm/managed/models"
)

func TestPMMAgentSupported(t *testing.T) {
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
			errString:    "not supported on pmm-agent",
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
			agentModel := models.Agent{
				AgentID: "Test agent ID",
				Version: pointer.ToString(test.agentVersion),
			}
			err := PMMAgentSupported(&agentModel, prefix, minVersion)
			if test.errString == "" {
				assert.NoError(t, err)
			} else {
				assert.Contains(t, err.Error(), test.errString)
			}
		})
	}

	t.Run("No version info", func(t *testing.T) {
		err := PMMAgentSupported(&models.Agent{AgentID: "Test agent ID"}, prefix, version.Must(version.NewVersion("2.30.0")))
		assert.Contains(t, err.Error(), "has no version info")
	})

	t.Run("Nil agent", func(t *testing.T) {
		err := PMMAgentSupported(nil, prefix, version.Must(version.NewVersion("2.30.0")))
		assert.Contains(t, err.Error(), "nil agent")
	})
}
