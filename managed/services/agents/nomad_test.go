package agents

import (
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

func TestGenerateNomadClientConfig(t *testing.T) {
	t.Run("Basic", func(t *testing.T) {
		node := &models.Node{
			NodeName: "node-name",
			NodeID:   "node-id",
			Address:  "node-address",
		}
		agent := &models.Agent{
			LogLevel: pointer.To("debug"),
		}
		tdp := models.TemplateDelimsPair()
		config, err := generateNomadClientConfig(node, agent, tdp)
		require.NoError(t, err)
		assert.Contains(t, config, "NodeName = \"node-name\"")
	})
}
