package agents

import (
	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/version"
)

// valkeyExporterConfig returns desired configuration of valkey_exporter process.
// todo: to be implemented in PMM-13837
func valkeyExporterConfig(node *models.Node, service *models.Service, exporter *models.Agent, mode redactMode, pmmAgentVersion *version.Parsed) *agentv1.SetStateRequest_AgentProcess {
	tdp := exporter.TemplateDelimiters(service)
	var args []string

	args = withLogLevel(args, exporter.LogLevel, pmmAgentVersion, true)

	return &agentv1.SetStateRequest_AgentProcess{
		Type:               inventoryv1.AgentType_AGENT_TYPE_VALKEY_EXPORTER,
		TemplateLeftDelim:  tdp.Left,
		TemplateRightDelim: tdp.Right,
		Args:               args,
	}
}
