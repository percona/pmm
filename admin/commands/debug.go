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

package commands

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/admin/agentlocal"
	"github.com/percona/pmm/admin/pkg/flags"
	"github.com/percona/pmm/api/inventory/v1/json/client"
	agentsService "github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
	services "github.com/percona/pmm/api/inventory/v1/json/client/services_service"
	"github.com/percona/pmm/api/inventory/v1/types"
)

var debugResultT = ParseTemplate(`
Agent ID: {{ .AgentID }}
Agent Type: {{ .AgentType }}
Status: {{ .Status }}
{{ if .ListenPort }}Listen Port: {{ .ListenPort }}{{ end }}
{{ if .ScrapeHealth }}
VMAgent Scrape Health: {{ .ScrapeHealth }}{{ if .ScrapeError }}
VMAgent Scrape Error: {{ .ScrapeError }}{{ end }}{{ end }}

{{ range .Resolutions }}
=== Resolution: {{ .Resolution }} ===
{{ if .ExporterURL }}Exporter URL: {{ .ExporterURL }}{{ end }}
{{ if .CollectorOptions }}Collector Options: {{ .CollectorOptions }}{{ end }}
Collection Time: {{ .CollectionTime }}
{{ if .Error }}Error: {{ .Error }}{{ else }}{{ if .OutputFile }}Metrics saved to: {{ .OutputFile }}
Metrics count: {{ .MetricsCount }} metrics{{ end }}{{ if not .OutputFile }}No metrics collected{{ end }}{{ end }}

{{ end }}{{ if .LogsFile }}Logs saved to: {{ .LogsFile }} ({{ .LogsLines }} lines){{ end }}
{{ if .VmagentLogsFile }}VMAgent logs saved to: {{ .VmagentLogsFile }} ({{ .VmagentLogsLines }} lines){{ end }}
`)

// debugResolutionResult holds the result for a single resolution
type debugResolutionResult struct {
	Resolution       string        `json:"resolution"`
	ExporterURL      string        `json:"exporter_url,omitempty"`
	CollectorOptions string        `json:"collector_options,omitempty"`
	CollectionTime   time.Duration `json:"collection_time"`
	OutputFile       string        `json:"output_file,omitempty"`
	MetricsCount     int           `json:"metrics_count,omitempty"`
	Error            string        `json:"error,omitempty"`
}

type debugResult struct {
	AgentID          string                  `json:"agent_id"`
	AgentType        string                  `json:"agent_type"`
	Status           string                  `json:"status"`
	ListenPort       int64                   `json:"listen_port,omitempty"`
	Resolutions      []debugResolutionResult `json:"resolutions"`
	LogsFile         string                  `json:"logs_file,omitempty"`
	LogsLines        int                     `json:"logs_lines,omitempty"`
	VmagentLogsFile  string                  `json:"vmagent_logs_file,omitempty"`
	VmagentLogsLines int                     `json:"vmagent_logs_lines,omitempty"`
	ScrapeHealth     string                  `json:"scrape_health,omitempty"`
	ScrapeError      string                  `json:"scrape_error,omitempty"`
	Error            string                  `json:"error,omitempty"`
}

func (res *debugResult) Result() {}

func (res *debugResult) String() string {
	return RenderTemplate(debugResultT, res)
}

var (
	// DebugServiceTypesKeys lists all possible service types for debug command
	DebugServiceTypesKeys = []string{"mysql", "mongodb", "postgresql", "valkey", "proxysql", "haproxy", "external", "node"}
)

// DebugCommand is used by Kong for CLI flags and commands.
type DebugCommand struct {
	ServiceType   string `arg:"" enum:"${debugServiceTypesEnum}" help:"Service type, one of: ${enum}"`
	ServiceName   string `arg:"" optional:"" help:"Service name (optional, will auto-detect if only one service of this type exists)"`
	AgentID       string `help:"Specific Agent ID to debug (optional)"`
	Resolution    string `help:"Resolution to test (lr=low, mr=medium, hr=high). If not specified, collects all available resolutions"`
	Timeout       int    `help:"Timeout for metrics collection in seconds" default:"30"`
	OutputDir     string `help:"Output directory for metrics files (default: current directory)"`
	AgentPassword string `help:"Password for exporter authentication (default: agent ID)"`
	LogLines      int    `help:"Number of log lines to include (default: 100)" default:"100"`
}

// RunCmdWithContext runs debug command.
func (cmd *DebugCommand) RunCmdWithContext(ctx context.Context, globals *flags.GlobalFlags) (Result, error) {
	// Validate resolution if specified
	if cmd.Resolution != "" && !cmd.isValidResolution(cmd.Resolution) {
		return nil, errors.Errorf("invalid resolution %q, must be one of: %s, %s, %s",
			cmd.Resolution, resolutionLR, resolutionMR, resolutionHR)
	}

	// Find agent(s) to debug
	agents, err := cmd.findAgentsToDebug(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find agents to debug")
	}

	if len(agents) == 0 {
		return nil, errors.Errorf("no agents found for service type '%s'", cmd.ServiceType)
	}

	// Handle multiple agents - prompt user to select one
	var agentInfo *agentInfo
	if len(agents) > 1 && cmd.AgentID == "" {
		selectedAgent, err := cmd.promptAgentSelection(agents)
		if err != nil {
			return nil, err
		}
		agentInfo = selectedAgent
	} else {
		// Use the first (or only) agent
		agentInfo = agents[0]
	}

	result := &debugResult{
		AgentID:    agentInfo.AgentID,
		AgentType:  agentInfo.AgentType,
		Status:     agentInfo.Status,
		ListenPort: agentInfo.ListenPort,
	}

	timestamp := time.Now().Format("20060102_150405")

	// Only collect metrics for exporter agents, not QAN agents
	if agentInfo.Category == AgentCategoryExporter {
		if agentInfo.ListenPort <= 0 {
			result.Error = "Exporter agent does not have a listen port configured"
			return result, nil
		}

		// Get vmagent data (scrape health, collectors) in a single API call
		vmagentInfo, collectorsMap, err := cmd.getVMAgentData(ctx, agentInfo.AgentID)
		if err != nil {
			logrus.Warnf("Failed to get vmagent data: %v", err)
			result.Error = fmt.Sprintf("vmagent not available: %v", err)
			return result, nil
		}

		// Set scrape health info
		result.ScrapeHealth = vmagentInfo.ScrapeHealth
		if vmagentInfo.LastError != "" {
			result.ScrapeError = vmagentInfo.LastError
		}

		// Fetch vmagent logs
		vmagentLogsFile := cmd.getOutputPath("pmm_debug_%s_%s_vmagent_logs.txt",
			strings.ToLower(agentInfo.AgentType), timestamp)
		vmagentLogsLines, err := cmd.fetchAgentLogs(ctx, vmagentInfo.AgentID, vmagentLogsFile, cmd.LogLines, globals)
		if err != nil {
			logrus.Warnf("Failed to fetch vmagent logs: %v", err)
		} else {
			result.VmagentLogsFile = vmagentLogsFile
			result.VmagentLogsLines = vmagentLogsLines
		}

		// Determine which resolutions to collect
		var resolutions []string
		if cmd.Resolution != "" {
			// If user specified a specific resolution, use only that one (if it exists in vmagent)
			if _, exists := collectorsMap[cmd.Resolution]; exists {
				resolutions = []string{cmd.Resolution}
			} else {
				result.Error = fmt.Sprintf("resolution %s not found in vmagent for agent %s", cmd.Resolution, agentInfo.AgentID)
				return result, nil
			}
		} else {
			// Use all available resolutions from vmagent
			for resolution := range collectorsMap {
				resolutions = append(resolutions, resolution)
			}
			logrus.Debugf("Available resolutions from vmagent for agent %s: %v", agentInfo.AgentID, resolutions)
		}

		// Collect metrics for each resolution
		for _, resolution := range resolutions {
			resResult := cmd.collectResolutionMetrics(ctx, agentInfo, resolution, timestamp, collectorsMap, globals)
			result.Resolutions = append(result.Resolutions, resResult)
		}
	} else {
		// QAN agents don't provide metrics, only logs
		logrus.Infof("Agent %s is a QAN agent, skipping metrics collection (logs only)", agentInfo.AgentID)
	}

	// Fetch agent logs (for both exporter and QAN agents)
	logsFile := cmd.getOutputPath("pmm_debug_%s_%s_logs.txt",
		strings.ToLower(agentInfo.AgentType), timestamp)

	logsLines, err := cmd.fetchAgentLogs(ctx, agentInfo.AgentID, logsFile, cmd.LogLines, globals)
	if err != nil {
		logrus.Warnf("Failed to fetch agent logs: %v", err)
	} else {
		result.LogsFile = logsFile
		result.LogsLines = logsLines
	}

	return result, nil
}

// isValidResolution checks if the resolution is valid
func (cmd *DebugCommand) isValidResolution(resolution string) bool {
	return resolution == resolutionLR || resolution == resolutionMR || resolution == resolutionHR
}

// getOutputPath generates a file path in the specified output directory
func (cmd *DebugCommand) getOutputPath(pattern string, args ...interface{}) string {
	filename := fmt.Sprintf(pattern, args...)
	if cmd.OutputDir != "" {
		return filepath.Join(cmd.OutputDir, filename)
	}
	return filename
}

// promptAgentSelection prompts the user to select an agent from a list
func (cmd *DebugCommand) promptAgentSelection(agents []*agentInfo) (*agentInfo, error) {
	fmt.Printf("\nFound %d agents for service type '%s':\n\n", len(agents), cmd.ServiceType) //nolint:forbidigo

	for i, agent := range agents {
		fmt.Printf("  %d. Agent ID: %s\n", i+1, agent.AgentID)                 //nolint:forbidigo
		fmt.Printf("     Type: %s, Status: %s", agent.AgentType, agent.Status) //nolint:forbidigo
		if agent.ListenPort > 0 {
			fmt.Printf(", Port: %d", agent.ListenPort) //nolint:forbidigo
		}
		fmt.Println() //nolint:forbidigo
	}

	fmt.Printf("\nSelect an agent to debug [1-%d]: ", len(agents)) //nolint:forbidigo

	var selection int
	_, err := fmt.Scanln(&selection)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read selection")
	}

	if selection < 1 || selection > len(agents) {
		return nil, errors.Errorf("invalid selection %d, must be between 1 and %d", selection, len(agents))
	}

	return agents[selection-1], nil
}

// collectResolutionMetrics collects metrics for a specific resolution
func (cmd *DebugCommand) collectResolutionMetrics(ctx context.Context, agentInfo *agentInfo, resolution, timestamp string, collectorsMap map[string][]string, globals *flags.GlobalFlags) debugResolutionResult {
	result := debugResolutionResult{
		Resolution: resolution,
	}

	// Get collectors for the resolution (for display purposes)
	if collectors, ok := collectorsMap[resolution]; ok && len(collectors) > 0 {
		result.CollectorOptions = strings.Join(collectors, ", ")
		logrus.Debugf("Using collectors from vmagent for agent %s resolution %s: %v", agentInfo.AgentID, resolution, collectors)
	} else if agentInfo.AgentType == "NODE_EXPORTER" {
		// Node exporters should have collectors
		result.Error = fmt.Sprintf("No collectors found for resolution %s", resolution)
		return result
	}

	// Get collectors slice for this resolution
	var collectors []string
	if collectorsList, ok := collectorsMap[resolution]; ok {
		collectors = collectorsList
	}

	// Get exporter URL
	exporterURL, err := cmd.buildExporterURL(ctx, agentInfo, resolution, collectors)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to build exporter URL: %v", err)
		return result
	}
	result.ExporterURL = exporterURL

	// Generate output filename
	outputFile := cmd.getOutputPath("pmm_debug_%s_%s_%s.txt",
		strings.ToLower(agentInfo.AgentType), resolution, timestamp)
	result.OutputFile = outputFile

	// Collect metrics with timing
	start := time.Now()
	metricsCount, err := cmd.collectMetricsToFile(ctx, exporterURL, outputFile, time.Duration(cmd.Timeout)*time.Second)
	result.CollectionTime = time.Since(start)

	if err != nil {
		result.Error = err.Error()
	} else {
		result.MetricsCount = metricsCount
	}

	return result
}

// AgentCategory defines the category of an agent
type AgentCategory string

const (
	// AgentCategoryExporter represents exporter agents that provide metrics
	AgentCategoryExporter AgentCategory = "exporter"
	// AgentCategoryQAN represents QAN agents that only provide query analytics data
	AgentCategoryQAN AgentCategory = "qan"
)

// agentInfo holds basic agent information
type agentInfo struct {
	AgentID       string
	AgentType     string
	Status        string
	ListenPort    int64
	ServiceID     string
	AgentPassword string
	Category      AgentCategory // Category of the agent (exporter or qan)
}

const (
	defaultUsername    = "pmm"
	defaultHTTPTimeout = 10 * time.Second
	defaultMetricsPath = "/metrics"
	defaultLogsTimeout = 30 * time.Second
	metricsBufferSize  = 4096
	resolutionHR       = "hr"
	resolutionMR       = "mr"
	resolutionLR       = "lr"
)

// vmagentTargetsResponse represents the response from vmagent /api/v1/targets endpoint
type vmagentTargetsResponse struct {
	Status string `json:"status"`
	Data   struct {
		ActiveTargets []vmagentTarget `json:"activeTargets"`
	} `json:"data"`
}

// vmagentTarget represents a single target in the vmagent response
type vmagentTarget struct {
	DiscoveredLabels map[string]string `json:"discoveredLabels"`
	Labels           map[string]string `json:"labels"`
	ScrapeURL        string            `json:"scrapeUrl"`
	Health           string            `json:"health"`
	LastError        string            `json:"lastError"`
	LastScrape       string            `json:"lastScrape"`
	ScrapeInterval   string            `json:"scrapeInterval"`
	ScrapeTimeout    string            `json:"scrapeTimeout"`
}

// vmagentInfo holds vmagent information including scrape health
type vmagentInfo struct {
	AgentID      string
	Port         int64
	ScrapeHealth string // "up" or "down"
	LastError    string
}

// findAgentsToDebug finds agents to debug based on service type and other criteria
func (cmd *DebugCommand) findAgentsToDebug(ctx context.Context) ([]*agentInfo, error) {
	// If specific agent ID is provided, use it directly
	if cmd.AgentID != "" {
		agent, err := cmd.getAgentByID(ctx, cmd.AgentID)
		if err != nil {
			return nil, err
		}
		return []*agentInfo{agent}, nil
	}

	// Find agents based on service type
	return cmd.findAgentsByServiceType(ctx)
}

// getAgentByID retrieves agent information by agent ID from inventory API
func (cmd *DebugCommand) getAgentByID(ctx context.Context, agentID string) (*agentInfo, error) {
	params := &agentsService.GetAgentParams{
		AgentID: agentID,
		Context: ctx,
	}

	resp, err := client.Default.AgentsService.GetAgent(params)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get agent from inventory")
	}

	// Extract agent information from the response
	return cmd.extractAgentInfo(resp.Payload), nil
}

// findAgentsByServiceType finds agents based on service type
func (cmd *DebugCommand) findAgentsByServiceType(ctx context.Context) ([]*agentInfo, error) {
	// Get local node information
	status, err := agentlocal.GetStatus(agentlocal.DoNotRequestNetworkInfo)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get local agent status")
	}

	var agents []*agentInfo

	// Handle node exporter specially
	if cmd.ServiceType == "node" {
		return cmd.findNodeExporters(ctx, status.NodeID)
	}

	// Find services of the specified type
	serviceType := GetServiceTypeConstant(cmd.ServiceType)
	if serviceType == "" {
		return nil, errors.Errorf("unsupported service type: %s", cmd.ServiceType)
	}

	servicesParams := &services.ListServicesParams{
		NodeID:      &status.NodeID,
		ServiceType: &serviceType,
		Context:     ctx,
	}

	servicesResp, err := client.Default.ServicesService.ListServices(servicesParams)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list services")
	}

	// Extract service IDs based on service type and optional service name filter
	serviceIDs := cmd.extractServiceIDs(servicesResp.Payload)

	// Find agents for each service
	for _, serviceID := range serviceIDs {
		agentsParams := &agentsService.ListAgentsParams{
			ServiceID: &serviceID,
			Context:   ctx,
		}

		agentsResp, err := client.Default.AgentsService.ListAgents(agentsParams)
		if err != nil {
			logrus.Warnf("Failed to list agents for service %s: %v", serviceID, err)
			continue
		}

		// Extract all agents (exporters and QAN) for this service
		serviceAgents := cmd.extractServiceAgents(agentsResp.Payload, serviceID)
		agents = append(agents, serviceAgents...)
	}

	return agents, nil
}

// findNodeExporters finds node exporter agents
func (cmd *DebugCommand) findNodeExporters(ctx context.Context, nodeID string) ([]*agentInfo, error) {
	agentsParams := &agentsService.ListAgentsParams{
		NodeID:  &nodeID,
		Context: ctx,
	}

	agentsResp, err := client.Default.AgentsService.ListAgents(agentsParams)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list node agents")
	}

	var nodeAgents []*agentInfo
	for _, agent := range agentsResp.Payload.NodeExporter {
		if agent.Disabled {
			continue
		}
		nodeAgents = append(nodeAgents, createAgentInfo(
			agent.AgentID, "NODE_EXPORTER", GetAgentStatus(agent.Status),
			int64(agent.ListenPort), "", AgentCategoryExporter))
	}

	return nodeAgents, nil
}

// extractAgentInfo extracts agent information from API response
func (cmd *DebugCommand) extractAgentInfo(agentResp interface{}) *agentInfo {
	// This would need to be implemented based on the actual API response structure
	// For now, return a placeholder
	return &agentInfo{
		AgentID:   "unknown",
		AgentType: "UNKNOWN",
		Status:    "UNKNOWN",
	}
}

// extractServiceIDs extracts service IDs from services response, optionally filtering by service name
func (cmd *DebugCommand) extractServiceIDs(payload *services.ListServicesOKBody) []string {
	var serviceIDs []string

	// Helper function to check if service name matches (if specified)
	matchesName := func(serviceName string) bool {
		return cmd.ServiceName == "" || cmd.ServiceName == serviceName
	}

	switch cmd.ServiceType {
	case "mysql":
		for _, svc := range payload.Mysql {
			if matchesName(svc.ServiceName) {
				serviceIDs = append(serviceIDs, svc.ServiceID)
			}
		}
	case "mongodb":
		for _, svc := range payload.Mongodb {
			if matchesName(svc.ServiceName) {
				serviceIDs = append(serviceIDs, svc.ServiceID)
			}
		}
	case "postgresql":
		for _, svc := range payload.Postgresql {
			if matchesName(svc.ServiceName) {
				serviceIDs = append(serviceIDs, svc.ServiceID)
			}
		}
	case "valkey":
		for _, svc := range payload.Valkey {
			if matchesName(svc.ServiceName) {
				serviceIDs = append(serviceIDs, svc.ServiceID)
			}
		}
	case "proxysql":
		for _, svc := range payload.Proxysql {
			if matchesName(svc.ServiceName) {
				serviceIDs = append(serviceIDs, svc.ServiceID)
			}
		}
	case "haproxy":
		for _, svc := range payload.Haproxy {
			if matchesName(svc.ServiceName) {
				serviceIDs = append(serviceIDs, svc.ServiceID)
			}
		}
	case "external":
		for _, svc := range payload.External {
			if matchesName(svc.ServiceName) {
				serviceIDs = append(serviceIDs, svc.ServiceID)
			}
		}
	}

	return serviceIDs
}

// createAgentInfo creates an agentInfo struct from common agent fields
func createAgentInfo(agentID, agentType, status string, listenPort int64, serviceID string, category AgentCategory) *agentInfo {
	return &agentInfo{
		AgentID:       agentID,
		AgentType:     agentType,
		Status:        status,
		ListenPort:    int64(listenPort),
		ServiceID:     serviceID,
		AgentPassword: agentID, // Default password is agent ID
		Category:      category,
	}
}

// extractServiceAgents extracts exporter and QAN agents for a service from agents response
func (cmd *DebugCommand) extractServiceAgents(payload *agentsService.ListAgentsOKBody, serviceID string) []*agentInfo {
	var serviceAgents []*agentInfo

	// Extract different types of exporter and QAN agents based on service type
	switch cmd.ServiceType {
	case "mysql":
		// MySQL Exporter
		for _, agent := range payload.MysqldExporter {
			if agent.Disabled || agent.ServiceID != serviceID {
				continue
			}
			serviceAgents = append(serviceAgents, createAgentInfo(
				agent.AgentID, "MYSQLD_EXPORTER", GetAgentStatus(agent.Status),
				int64(agent.ListenPort), serviceID, AgentCategoryExporter))
		}
		// QAN MySQL PerfSchema
		for _, agent := range payload.QANMysqlPerfschemaAgent {
			if agent.Disabled || agent.ServiceID != serviceID {
				continue
			}
			serviceAgents = append(serviceAgents, createAgentInfo(
				agent.AgentID, "QAN_MYSQL_PERFSCHEMA_AGENT", GetAgentStatus(agent.Status),
				0, serviceID, AgentCategoryQAN)) // QAN agents don't have listen ports
		}
		// QAN MySQL SlowLog
		for _, agent := range payload.QANMysqlSlowlogAgent {
			if agent.Disabled || agent.ServiceID != serviceID {
				continue
			}
			serviceAgents = append(serviceAgents, createAgentInfo(
				agent.AgentID, "QAN_MYSQL_SLOWLOG_AGENT", GetAgentStatus(agent.Status),
				0, serviceID, AgentCategoryQAN))
		}
	case "mongodb":
		// MongoDB Exporter
		for _, agent := range payload.MongodbExporter {
			if agent.Disabled || agent.ServiceID != serviceID {
				continue
			}
			serviceAgents = append(serviceAgents, createAgentInfo(
				agent.AgentID, "MONGODB_EXPORTER", GetAgentStatus(agent.Status),
				int64(agent.ListenPort), serviceID, AgentCategoryExporter))
		}
		// QAN MongoDB Profiler
		for _, agent := range payload.QANMongodbProfilerAgent {
			if agent.Disabled || agent.ServiceID != serviceID {
				continue
			}
			serviceAgents = append(serviceAgents, createAgentInfo(
				agent.AgentID, "QAN_MONGODB_PROFILER_AGENT", GetAgentStatus(agent.Status),
				0, serviceID, AgentCategoryQAN))
		}
	case "postgresql":
		// PostgreSQL Exporter
		for _, agent := range payload.PostgresExporter {
			if agent.Disabled || agent.ServiceID != serviceID {
				continue
			}
			serviceAgents = append(serviceAgents, createAgentInfo(
				agent.AgentID, "POSTGRES_EXPORTER", GetAgentStatus(agent.Status),
				int64(agent.ListenPort), serviceID, AgentCategoryExporter))
		}
		// QAN PostgreSQL PgStatements
		for _, agent := range payload.QANPostgresqlPgstatementsAgent {
			if agent.Disabled || agent.ServiceID != serviceID {
				continue
			}
			serviceAgents = append(serviceAgents, createAgentInfo(
				agent.AgentID, "QAN_POSTGRESQL_PGSTATEMENTS_AGENT", GetAgentStatus(agent.Status),
				0, serviceID, AgentCategoryQAN))
		}
		// QAN PostgreSQL PgStatMonitor
		for _, agent := range payload.QANPostgresqlPgstatmonitorAgent {
			if agent.Disabled || agent.ServiceID != serviceID {
				continue
			}
			serviceAgents = append(serviceAgents, createAgentInfo(
				agent.AgentID, "QAN_POSTGRESQL_PGSTATMONITOR_AGENT", GetAgentStatus(agent.Status),
				0, serviceID, AgentCategoryQAN))
		}
	case "valkey":
		for _, agent := range payload.ValkeyExporter {
			if agent.Disabled || agent.ServiceID != serviceID {
				continue
			}
			serviceAgents = append(serviceAgents, createAgentInfo(
				agent.AgentID, "VALKEY_EXPORTER", GetAgentStatus(agent.Status),
				int64(agent.ListenPort), serviceID, AgentCategoryExporter))
		}
	case "proxysql":
		for _, agent := range payload.ProxysqlExporter {
			if agent.Disabled || agent.ServiceID != serviceID {
				continue
			}
			serviceAgents = append(serviceAgents, createAgentInfo(
				agent.AgentID, "PROXYSQL_EXPORTER", GetAgentStatus(agent.Status),
				int64(agent.ListenPort), serviceID, AgentCategoryExporter))
		}
	case "external":
		for _, agent := range payload.ExternalExporter {
			if agent.Disabled || agent.ServiceID != serviceID {
				continue
			}
			serviceAgents = append(serviceAgents, createAgentInfo(
				agent.AgentID, "EXTERNAL_EXPORTER", "RUNNING", // External exporters don't have status in the same way
				int64(agent.ListenPort), serviceID, AgentCategoryExporter))
		}
	}

	return serviceAgents
}

// getVMAgentData fetches vmagent information and collector parameters for the given exporter agent
// Returns vmagent info and a map where key is resolution and value is a slice of collector names
func (cmd *DebugCommand) getVMAgentData(ctx context.Context, exporterAgentID string) (*vmagentInfo, map[string][]string, error) {
	// First, find the vmagent
	localStatus, err := agentlocal.GetRawStatus(ctx, agentlocal.DoNotRequestNetworkInfo)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get local agent status")
	}

	var vmagentPort int64
	var vmagentID string
	if localStatus.AgentsInfo != nil {
		for _, agent := range localStatus.AgentsInfo {
			if agent.AgentType != nil && *agent.AgentType == types.AgentTypeVMAgent {
				vmagentPort = agent.ListenPort
				vmagentID = agent.AgentID
				break
			}
		}
	}

	if vmagentPort == 0 {
		return nil, nil, errors.New("vmagent not found or not running")
	}

	// Fetch targets from vmagent API
	vmagentURL := fmt.Sprintf("http://%s/api/v1/targets",
		net.JoinHostPort(agentlocal.Localhost, strconv.FormatInt(vmagentPort, 10)))

	req, err := http.NewRequestWithContext(ctx, "GET", vmagentURL, nil)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create HTTP request for vmagent targets")
	}

	client := &http.Client{Timeout: defaultHTTPTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to fetch vmagent targets")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, errors.Errorf("vmagent returned status %d: %s", resp.StatusCode, resp.Status)
	}

	var targetsResp vmagentTargetsResponse
	if err := json.NewDecoder(resp.Body).Decode(&targetsResp); err != nil {
		return nil, nil, errors.Wrap(err, "failed to decode vmagent targets response")
	}

	// Initialize vmagent info
	info := &vmagentInfo{
		AgentID:      vmagentID,
		Port:         vmagentPort,
		ScrapeHealth: "unknown",
	}

	// Initialize collectors map
	collectorsMap := make(map[string][]string)

	// Check health and extract collectors for all targets of this exporter agent
	var hasHealthyTarget bool
	var lastError string

	for _, target := range targetsResp.Data.ActiveTargets {
		// Check if this target is for our agent
		if target.Labels["agent_id"] == exporterAgentID {
			// Check scrape health
			if target.Health == "up" {
				hasHealthyTarget = true
			} else if target.Health == "down" && target.LastError != "" {
				lastError = target.LastError
			}

			// Extract resolution from job name
			jobName, ok := target.Labels["job"]
			if !ok {
				continue
			}

			var resolution string
			for _, res := range []string{resolutionHR, resolutionMR, resolutionLR} {
				if strings.HasSuffix(jobName, "_"+res) {
					resolution = res
					break
				}
			}

			if resolution == "" {
				continue
			}

			// Parse the scrape URL to extract collectors from query parameters
			parsedURL, err := url.Parse(target.ScrapeURL)
			if err != nil {
				logrus.Debugf("Failed to parse scrape URL %s: %v", target.ScrapeURL, err)
				continue
			}

			// Extract collect[] parameters
			if collectParams, ok := parsedURL.Query()["collect[]"]; ok && len(collectParams) > 0 {
				collectorsMap[resolution] = collectParams
			}
		}
	}

	// Set scrape health status
	if hasHealthyTarget {
		info.ScrapeHealth = "up"
	} else if lastError != "" {
		info.ScrapeHealth = "down"
		info.LastError = lastError
	}

	if len(collectorsMap) == 0 {
		return info, nil, errors.Errorf("no targets found for agent %s in vmagent", exporterAgentID)
	}

	return info, collectorsMap, nil
}

// buildExporterURL constructs the exporter URL for metrics collection with authentication
func (cmd *DebugCommand) buildExporterURL(ctx context.Context, agent *agentInfo, resolution string, collectors []string) (string, error) {
	password := cmd.AgentPassword
	if password == "" {
		password = agent.AgentPassword // Default to agent's password (typically agent ID)
	}

	// Build URL with credentials
	u := &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("127.0.0.1:%d", agent.ListenPort),
		Path:   defaultMetricsPath,
	}

	// Set authentication (username is always "pmm")
	u.User = url.UserPassword(defaultUsername, password)

	// Add collectors as query parameters for all agent types
	if len(collectors) > 0 {
		// Build collect[] query parameters directly from collectors slice
		params := url.Values{}
		for _, collector := range collectors {
			params.Add("collect[]", collector)
		}
		u.RawQuery = params.Encode()
		logrus.Debugf("Built URL with collectors for agent %s: %s", agent.AgentID, u.String())
	}

	return u.String(), nil
}

// collectMetricsToFile fetches metrics from the exporter endpoint and saves to file
func (cmd *DebugCommand) collectMetricsToFile(ctx context.Context, exporterURL string, outputFile string, timeout time.Duration) (int, error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: timeout,
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", exporterURL, nil)
	if err != nil {
		return 0, errors.Wrap(err, "failed to create HTTP request")
	}

	// Set appropriate headers
	req.Header.Set("Accept", "text/plain")
	req.Header.Set("User-Agent", "pmm-admin-debug")

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		return 0, errors.Wrap(err, "failed to fetch metrics")
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return 0, errors.Errorf("exporter returned status %d: %s", resp.StatusCode, resp.Status)
	}

	// Create output file
	file, err := os.Create(outputFile)
	if err != nil {
		return 0, errors.Wrap(err, "failed to create output file")
	}
	defer file.Close()

	// Write response body to file and count metrics
	metricsCount := 0
	scanner := io.TeeReader(resp.Body, file)

	// Count metrics while writing to file
	buf := make([]byte, metricsBufferSize)
	for {
		n, err := scanner.Read(buf)
		if n > 0 {
			// Count lines that look like metrics (not comments or empty lines)
			lines := strings.Split(string(buf[:n]), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" && !strings.HasPrefix(line, "#") {
					metricsCount++
				}
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, errors.Wrap(err, "failed to read response body")
		}
	}

	return metricsCount, nil
}

// fetchAgentLogs fetches agent logs from pmm-agent and saves to file
func (cmd *DebugCommand) fetchAgentLogs(ctx context.Context, agentID string, outputFile string, maxLines int, globals *flags.GlobalFlags) (int, error) {
	// Build logs.zip URL with agent_id filter
	pmmAgentURL := fmt.Sprintf("http://%s:%d/logs.zip?agent_id=%s",
		agentlocal.Localhost, globals.PMMAgentListenPort, url.QueryEscape(agentID))

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: defaultLogsTimeout,
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", pmmAgentURL, nil)
	if err != nil {
		return 0, errors.Wrap(err, "failed to create HTTP request for logs")
	}

	// Fetch logs zip
	resp, err := client.Do(req)
	if err != nil {
		return 0, errors.Wrap(err, "failed to fetch logs from pmm-agent")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, errors.Errorf("pmm-agent returned status %d: %s", resp.StatusCode, resp.Status)
	}

	// Read zip content
	zipData, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, errors.Wrap(err, "failed to read logs zip")
	}

	// Parse zip
	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return 0, errors.Wrap(err, "failed to parse logs zip")
	}

	// Find the log file for this agent (should only be one since we filtered by agent_id)
	var agentLogs string
	for _, file := range zipReader.File {
		if strings.HasSuffix(file.Name, ".log") {
			rc, err := file.Open()
			if err != nil {
				return 0, errors.Wrapf(err, "failed to open %s in zip", file.Name)
			}

			logData, err := io.ReadAll(rc)
			rc.Close()

			if err != nil {
				return 0, errors.Wrapf(err, "failed to read %s", file.Name)
			}

			agentLogs = string(logData)
			break
		}
	}

	if agentLogs == "" {
		return 0, errors.Errorf("no logs found for agent %s", agentID)
	}

	// Extract last N lines
	lines := strings.Split(agentLogs, "\n")
	startIdx := 0
	if len(lines) > maxLines {
		startIdx = len(lines) - maxLines
	}
	relevantLogs := strings.Join(lines[startIdx:], "\n")

	// Write to file
	err = os.WriteFile(outputFile, []byte(relevantLogs), 0600)
	if err != nil {
		return 0, errors.Wrap(err, "failed to write logs to file")
	}

	return len(lines) - startIdx, nil
}
