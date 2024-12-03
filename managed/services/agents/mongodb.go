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

package agents

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/AlekSi/pointer"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/collectors"
	"github.com/percona/pmm/version"
)

var (
	// New MongoDB Exporter will be released with PMM agent v2.10.0.
	newMongoExporterPMMVersion = version.MustParse("2.9.99")
	v2_25_0                    = version.MustParse("2.25.0-0")
	v2_26_0                    = version.MustParse("2.26.0-0")
	v2_41_1                    = version.MustParse("2.41.1-0")
	v2_42_0                    = version.MustParse("2.42.0-0")
	v2_43_0                    = version.MustParse("2.43.0-0")
)

// mongodbExporterConfig returns desired configuration of mongodb_exporter process.
func mongodbExporterConfig(node *models.Node, service *models.Service, exporter *models.Agent, redactMode redactMode,
	pmmAgentVersion *version.Parsed,
) (*agentv1.SetStateRequest_AgentProcess, error) {
	listenAddress := getExporterListenAddress(node, exporter)
	tdp := exporter.TemplateDelimiters(service)

	args := getArgs(exporter, tdp, listenAddress, pmmAgentVersion)

	if pointer.GetString(exporter.ExporterOptions.MetricsPath) != "" {
		args = append(args, "--web.telemetry-path="+*exporter.ExporterOptions.MetricsPath)
	}

	args = withLogLevel(args, exporter.LogLevel, pmmAgentVersion, true)

	sort.Strings(args)

	database := exporter.MongoDBOptions.AuthenticationDatabase
	env := []string{
		fmt.Sprintf("MONGODB_URI=%s", exporter.DSN(service, models.DSNParams{DialTimeout: time.Second, Database: database}, tdp, pmmAgentVersion)),
	}

	res := &agentv1.SetStateRequest_AgentProcess{
		Type:               inventoryv1.AgentType_AGENT_TYPE_MONGODB_EXPORTER,
		TemplateLeftDelim:  tdp.Left,
		TemplateRightDelim: tdp.Right,
		Args:               args,
		Env:                env,
		TextFiles:          exporter.Files(),
	}

	if redactMode != exposeSecrets {
		res.RedactWords = redactWords(exporter)
	}

	if err := ensureAuthParams(exporter, res, pmmAgentVersion, v2_28_00, false); err != nil {
		return nil, err
	}

	return res, nil
}

// getArgs returns the appropriate arguments based on the PMM agent version.
func getArgs(exporter *models.Agent, tdp *models.DelimiterPair, listenAddress string, pmmAgentVersion *version.Parsed) []string {
	var args []string

	switch {
	case !pmmAgentVersion.Less(v2_25_0): // >= 2.26.0
		args = buildBaseArgs(listenAddress, tdp)
		args = append(args, "--discovering-mode")

		defaultEnabledCollectors := []string{"diagnosticdata", "replicasetstatus"}
		collectAll := exporter.MongoDBOptions.EnableAllCollectors

		if !pmmAgentVersion.Less(v2_26_0) {
			defaultEnabledCollectors = []string{}
			args = append(args, "--collector.diagnosticdata", "--collector.replicasetstatus")
			if collectAll {
				args = append(args, "--collector.collstats", "--collector.dbstats", "--collector.indexstats", "--collector.topmetrics")
			}
		}
		if !pmmAgentVersion.Less(v2_41_1) && collectAll { // >= 2.41.1
			args = append(args, "--collector.shards")
		}
		if !pmmAgentVersion.Less(v2_42_0) && collectAll { // >= 2.42.0
			args = append(args, "--collector.currentopmetrics")
		}
		if !pmmAgentVersion.Less(v2_43_0) { // >= 2.43.0, enable by default
			args = append(args, "--collector.fcv")
		}
		if !pmmAgentVersion.Less(v2_43_0) { // >= 2.43.0, enable pbm collector by default
			args = append(args, "--collector.pbm")
		}

		args = collectors.FilterOutCollectors("--collector.", args, exporter.ExporterOptions.DisabledCollectors)
		args = append(args, collectors.DisableDefaultEnabledCollectors("--no-collector.", defaultEnabledCollectors, exporter.ExporterOptions.DisabledCollectors)...)

		if len(exporter.MongoDBOptions.StatsCollections) != 0 {
			args = append(args, "--mongodb.collstats-colls="+strings.Join(exporter.MongoDBOptions.StatsCollections, ","))
			if !pmmAgentVersion.Less(v2_26_0) {
				args = append(args, "--mongodb.indexstats-colls="+strings.Join(exporter.MongoDBOptions.StatsCollections, ","))
			}
		}

		collstatsLimit := int32(200)
		if exporter.MongoDBOptions.CollectionsLimit != -1 {
			collstatsLimit = exporter.MongoDBOptions.CollectionsLimit
		}
		args = append(args, fmt.Sprintf("--collector.collstats-limit=%d", collstatsLimit))

	case !pmmAgentVersion.Less(newMongoExporterPMMVersion): // >= 2.10.0
		args = buildBaseArgs(listenAddress, tdp)

	default: // < 2.10.0
		args = []string{
			"--collect.collection",
			"--collect.database",
			"--collect.topmetrics",
			"--no-collect.connpoolstats",
			"--no-collect.indexusage",
			"--web.listen-address=" + listenAddress + ":" + tdp.Left + " .listen_port " + tdp.Right, //nolint:goconst
		}

		args = collectors.FilterOutCollectors("--collect.", args, exporter.ExporterOptions.DisabledCollectors)
	}

	return args
}

func buildBaseArgs(listenAddress string, tdp *models.DelimiterPair) []string {
	return []string{
		"--mongodb.global-conn-pool",
		"--compatible-mode",
		"--web.listen-address=" + listenAddress + ":" + tdp.Left + " .listen_port " + tdp.Right,
	}
}

// qanMongoDBProfilerAgentConfig returns desired configuration of qan-mongodb-profiler-agent built-in agent.
func qanMongoDBProfilerAgentConfig(service *models.Service, agent *models.Agent, pmmAgentVersion *version.Parsed) *agentv1.SetStateRequest_BuiltinAgent {
	tdp := agent.TemplateDelimiters(service)

	return &agentv1.SetStateRequest_BuiltinAgent{
		Type:                 inventoryv1.AgentType_AGENT_TYPE_QAN_MONGODB_PROFILER_AGENT,
		Dsn:                  agent.DSN(service, models.DSNParams{DialTimeout: time.Second, Database: ""}, nil, pmmAgentVersion),
		DisableQueryExamples: agent.QANOptions.QueryExamplesDisabled,
		MaxQueryLength:       agent.QANOptions.MaxQueryLength,
		TextFiles: &agentv1.TextFiles{
			Files:              agent.Files(),
			TemplateLeftDelim:  tdp.Left,
			TemplateRightDelim: tdp.Right,
		},
	}
}
