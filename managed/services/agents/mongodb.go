// Copyright (C) 2024 Percona LLC
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

	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/collectors"
	"github.com/percona/pmm/version"
)

type collectorArgs struct {
	enabled     bool
	enableParam string
}

var (
	// New MongoDB Exporter will be released with PMM agent v2.10.0.
	newMongoExporterPMMVersion = version.MustParse("2.9.99")
	v2_24_99                   = version.MustParse("2.24.99")
	v2_25_99                   = version.MustParse("2.25.99")
	v2_41_1                    = version.MustParse("2.41.1-0")
)

// mongodbExporterConfig returns desired configuration of mongodb_exporter process.
func mongodbExporterConfig(node *models.Node, service *models.Service, exporter *models.Agent, redactMode redactMode,
	pmmAgentVersion *version.Parsed,
) (*agentpb.SetStateRequest_AgentProcess, error) {
	listenAddress := getExporterListenAddress(node, exporter)
	tdp := exporter.TemplateDelimiters(service)

	var args []string
	// Starting with PMM 2.10.0, we are shipping the new mongodb_exporter
	// Starting with PMM 2.25.0, we change the discovering-mode making it to discover all databases.
	// Until now, discovering mode was not working properly and was enabled only if mongodb.collstats-colls=
	// was specified in the command line.
	// Starting with PMM 2.41.1 we added shards collector.
	switch {
	case !pmmAgentVersion.Less(v2_41_1): // >= 2.41.1
		args = v226Args(exporter, tdp, listenAddress)

		if exporter.MongoDBOptions != nil && exporter.MongoDBOptions.EnableAllCollectors {
			args = append(args, "--collector.shards")
		}
	case !pmmAgentVersion.Less(v2_25_99): // >= 2.26
		args = v226Args(exporter, tdp, listenAddress)
	case !pmmAgentVersion.Less(v2_24_99): // >= 2.25
		args = v225Args(exporter, tdp, listenAddress)
	case !pmmAgentVersion.Less(newMongoExporterPMMVersion): // >= 2.10
		args = []string{
			"--mongodb.global-conn-pool",
			"--compatible-mode",
			"--web.listen-address=" + listenAddress + ":" + tdp.Left + " .listen_port " + tdp.Right,
		}
	default:
		args = []string{
			"--collect.collection",
			"--collect.database",
			"--collect.topmetrics",
			"--no-collect.connpoolstats",
			"--no-collect.indexusage",
			"--web.listen-address=" + listenAddress + ":" + tdp.Left + " .listen_port " + tdp.Right,
		}
	}

	args = collectors.FilterOutCollectors("--collect.", args, exporter.DisabledCollectors)

	if pointer.GetString(exporter.MetricsPath) != "" {
		args = append(args, "--web.telemetry-path="+*exporter.MetricsPath)
	}

	args = withLogLevel(args, exporter.LogLevel, pmmAgentVersion, true)

	sort.Strings(args)

	database := ""
	if exporter.MongoDBOptions != nil {
		database = exporter.MongoDBOptions.AuthenticationDatabase
	}
	env := []string{
		fmt.Sprintf("MONGODB_URI=%s", exporter.DSN(service, models.DSNParams{DialTimeout: time.Second, Database: database}, tdp)),
	}

	res := &agentpb.SetStateRequest_AgentProcess{
		Type:               inventorypb.AgentType_MONGODB_EXPORTER,
		TemplateLeftDelim:  tdp.Left,
		TemplateRightDelim: tdp.Right,
		Args:               args,
		Env:                env,
		TextFiles:          exporter.Files(),
	}

	if redactMode != exposeSecrets {
		res.RedactWords = redactWords(exporter)
	}

	if err := ensureAuthParams(exporter, res, pmmAgentVersion, v2_27_99); err != nil {
		return nil, err
	}

	return res, nil
}

func v226Args(exporter *models.Agent, tdp *models.DelimiterPair, listenAddress string) []string {
	collectAll := false
	if exporter.MongoDBOptions != nil {
		collectAll = exporter.MongoDBOptions.EnableAllCollectors
	}

	collstatsLimit := int32(200)
	if exporter.MongoDBOptions != nil && exporter.MongoDBOptions.CollectionsLimit != -1 {
		collstatsLimit = exporter.MongoDBOptions.CollectionsLimit
	}

	collectors := defaultCollectors(collectAll)

	for _, collector := range exporter.DisabledCollectors {
		col, ok := collectors[strings.ToLower(collector)]
		if !ok {
			continue
		}
		col.enabled = false
		collectors[strings.ToLower(collector)] = col
	}

	args := []string{
		"--mongodb.global-conn-pool",
		"--compatible-mode",
		"--web.listen-address=" + listenAddress + ":" + tdp.Left + " .listen_port " + tdp.Right,
		"--discovering-mode",
	}

	if exporter.MongoDBOptions != nil && len(exporter.MongoDBOptions.StatsCollections) != 0 {
		args = append(args, "--mongodb.collstats-colls="+strings.Join(exporter.MongoDBOptions.StatsCollections, ","))
		args = append(args, "--mongodb.indexstats-colls="+strings.Join(exporter.MongoDBOptions.StatsCollections, ","))
	}

	if exporter.MongoDBOptions != nil {
		args = append(args, fmt.Sprintf("--collector.collstats-limit=%d", collstatsLimit))
	}

	for _, collector := range collectors {
		if collector.enabled && collector.enableParam != "" {
			args = append(args, collector.enableParam)
		}
	}

	return args
}

func v225Args(exporter *models.Agent, tdp *models.DelimiterPair, listenAddress string) []string {
	type collectorArgs struct {
		enabled      bool
		enableParam  string
		disableParam string
	}

	collectors := map[string]collectorArgs{
		"diagnosticdata": {
			enabled:      true,
			disableParam: "--no-collector.diagnosticdata",
		},
		"replicasetstatus": {
			enabled:      true,
			disableParam: "--no-collector.replicasetstatus",
		},
		// disabled until we have better information on the resources usage impact
		"dbstats": {
			enabled:     false,
			enableParam: "--collector.dbstats",
		},
		// disabled until we have better information on the resources usage impact
		"topmetrics": {
			enabled:     false,
			enableParam: "--collector.topmetrics",
		},
	}

	for _, collector := range exporter.DisabledCollectors {
		col := collectors[strings.ToLower(collector)]
		col.enabled = false
		collectors[strings.ToLower(collector)] = col
	}

	args := []string{
		"--mongodb.global-conn-pool",
		"--compatible-mode",
		"--web.listen-address=" + listenAddress + ":" + tdp.Left + " .listen_port " + tdp.Right,
		"--discovering-mode",
	}

	if exporter.MongoDBOptions != nil && len(exporter.MongoDBOptions.StatsCollections) != 0 {
		args = append(args, "--mongodb.collstats-colls="+strings.Join(exporter.MongoDBOptions.StatsCollections, ","))
	}

	if exporter.MongoDBOptions != nil && exporter.MongoDBOptions.CollectionsLimit != 0 {
		args = append(args, fmt.Sprintf("--collector.collstats-limit=%d", exporter.MongoDBOptions.CollectionsLimit))
	}

	for _, collector := range collectors {
		if collector.enabled && collector.enableParam != "" {
			args = append(args, collector.enableParam)
		}
		if !collector.enabled && collector.disableParam != "" {
			args = append(args, collector.disableParam)
		}
	}

	return args
}

func defaultCollectors(collectAll bool) map[string]collectorArgs {
	return map[string]collectorArgs{
		"diagnosticdata": {
			enabled:     true,
			enableParam: "--collector.diagnosticdata",
		},
		"replicasetstatus": {
			enabled:     true,
			enableParam: "--collector.replicasetstatus",
		},
		"collstats": {
			enabled:     collectAll,
			enableParam: "--collector.collstats",
		},
		"dbstats": {
			enabled:     collectAll,
			enableParam: "--collector.dbstats",
		},
		"indexstats": {
			enabled:     collectAll,
			enableParam: "--collector.indexstats",
		},
		"topmetrics": {
			enabled:     collectAll,
			enableParam: "--collector.topmetrics",
		},
	}
}

// qanMongoDBProfilerAgentConfig returns desired configuration of qan-mongodb-profiler-agent built-in agent.
func qanMongoDBProfilerAgentConfig(service *models.Service, agent *models.Agent) *agentpb.SetStateRequest_BuiltinAgent {
	tdp := agent.TemplateDelimiters(service)
	return &agentpb.SetStateRequest_BuiltinAgent{
		Type:                 inventorypb.AgentType_QAN_MONGODB_PROFILER_AGENT,
		Dsn:                  agent.DSN(service, models.DSNParams{DialTimeout: time.Second, Database: ""}, nil),
		DisableQueryExamples: agent.QueryExamplesDisabled,
		MaxQueryLength:       agent.MaxQueryLength,
		TextFiles: &agentpb.TextFiles{
			Files:              agent.Files(),
			TemplateLeftDelim:  tdp.Left,
			TemplateRightDelim: tdp.Right,
		},
	}
}
