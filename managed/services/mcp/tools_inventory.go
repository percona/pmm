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

package mcp

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	engineMySQL      = "mysql"
	enginePostgreSQL = "postgresql"
	engineMongoDB    = "mongodb"

	unknownVersion = "unknown"
)

// versionMetrics maps an engine to the metric and label that carry the server
// version. Confirmed live on PMM 3.8.1: inventory's version field is empty, so
// versions come from the exporters' info metrics.
var versionMetrics = map[string]struct{ metric, label string }{
	engineMySQL:      {"mysql_version_info", "version"},
	enginePostgreSQL: {"pg_static", "short_version"},
	engineMongoDB:    {"mongodb_version_info", "mongodb"},
}

// serviceInfo is one monitored database service.
type serviceInfo struct {
	ServiceID   string
	ServiceName string
	Engine      string
	Version     string
	NodeName    string
	Address     string
	Port        int64
}

type versionInput struct{}

type inventoryInput struct {
	Engine string `json:"engine,omitempty" jsonschema:"Filter by engine: mysql, postgresql or mongodb"`
	Node   string `json:"node,omitempty" jsonschema:"Filter by node name"`
}

func (s *Service) registerInventoryTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:  "pmm_version",
		Title: "PMM Server version",
		Description: "Return the PMM Server version. Use it first to confirm the connection and " +
			"that the caller's token is accepted.",
		Annotations: readOnly("PMM Server version"),
	}, s.version)

	mcp.AddTool(server, &mcp.Tool{
		Name:  "pmm_inventory",
		Title: "List monitored services",
		Description: "List database services monitored by PMM, with engine and version. Use this to " +
			"discover a service_id / service_name and the exact engine version before analysing a query.",
		Annotations: readOnly("List monitored services"),
	}, s.inventory)
}

func (s *Service) version(ctx context.Context, req *mcp.CallToolRequest, _ versionInput) (*mcp.CallToolResult, any, error) {
	auth := callerAuthFromHeader(req.Extra.Header)

	v, err := s.api.Version(ctx, auth)
	if err != nil {
		return nil, nil, s.fail("pmm_version", err)
	}

	text := "PMM Server " + v.Version
	if v.Managed != nil && v.Managed.FullVersion != "" {
		text += fmt.Sprintf(" (pmm-managed %s)", v.Managed.FullVersion)
	}
	return textResult(text), nil, nil
}

func (s *Service) inventory(ctx context.Context, req *mcp.CallToolRequest, in inventoryInput) (*mcp.CallToolResult, any, error) {
	if in.Engine != "" && !slices.Contains([]string{engineMySQL, enginePostgreSQL, engineMongoDB}, in.Engine) {
		return nil, nil, newToolError(codeInvalidInput, "engine must be one of mysql, postgresql, mongodb; got '%s'", in.Engine)
	}
	auth := callerAuthFromHeader(req.Extra.Header)

	services, err := s.listServices(ctx, auth)
	if err != nil {
		return nil, nil, s.fail("pmm_inventory", err)
	}

	s.enrichVersions(ctx, auth, services)

	var out []serviceInfo
	for _, svc := range services {
		if in.Engine != "" && svc.Engine != in.Engine {
			continue
		}
		if in.Node != "" && svc.NodeName != in.Node {
			continue
		}
		out = append(out, svc)
	}
	if len(out) == 0 {
		return textResult("No monitored services found."), nil, nil
	}

	lines := make([]string, 0, len(out)+1)
	lines = append(lines, fmt.Sprintf("Monitored services (%d):", len(out)))
	for _, svc := range out {
		row := fmt.Sprintf("- %s | %s %s | id=%s", svc.ServiceName, svc.Engine, svc.Version, svc.ServiceID)
		if svc.NodeName != "" {
			row += " | node=" + svc.NodeName
		}
		if svc.Address != "" {
			row += " | address=" + svc.Address
			if svc.Port != 0 {
				row += fmt.Sprintf(":%d", svc.Port)
			}
		}
		lines = append(lines, row)
	}
	return textResult(strings.Join(lines, "\n")), nil, nil
}

// listServices fetches the inventory and resolves node names. Node lookup
// failures are tolerated: the listing is still useful without them.
func (s *Service) listServices(ctx context.Context, auth callerAuth) ([]serviceInfo, error) {
	resp, err := s.api.ListServices(ctx, auth)
	if err != nil {
		return nil, err
	}

	nodeNames := map[string]string{}
	nodes, err := s.api.ListNodes(ctx, auth)
	if err != nil {
		s.l.WithField("tool", "pmm_inventory").Debugf("Cannot list nodes, node names will be empty: %s.", err)
	} else {
		for _, n := range nodes.Generic {
			nodeNames[n.NodeID] = n.NodeName
		}
		for _, n := range nodes.Container {
			nodeNames[n.NodeID] = n.NodeName
		}
		for _, n := range nodes.Remote {
			nodeNames[n.NodeID] = n.NodeName
		}
		for _, n := range nodes.RemoteRDS {
			nodeNames[n.NodeID] = n.NodeName
		}
		for _, n := range nodes.RemoteAzureDatabase {
			nodeNames[n.NodeID] = n.NodeName
		}
	}

	var out []serviceInfo
	for _, m := range resp.Mysql {
		out = append(out, serviceInfo{
			ServiceID: m.ServiceID, ServiceName: m.ServiceName, Engine: engineMySQL,
			Version: m.Version, NodeName: nodeNames[m.NodeID], Address: m.Address, Port: m.Port,
		})
	}
	for _, p := range resp.Postgresql {
		out = append(out, serviceInfo{
			ServiceID: p.ServiceID, ServiceName: p.ServiceName, Engine: enginePostgreSQL,
			Version: p.Version, NodeName: nodeNames[p.NodeID], Address: p.Address, Port: p.Port,
		})
	}
	for _, m := range resp.Mongodb {
		out = append(out, serviceInfo{
			ServiceID: m.ServiceID, ServiceName: m.ServiceName, Engine: engineMongoDB,
			Version: m.Version, NodeName: nodeNames[m.NodeID], Address: m.Address, Port: m.Port,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Engine != out[j].Engine {
			return out[i].Engine < out[j].Engine
		}
		return out[i].ServiceName < out[j].ServiceName
	})
	return out, nil
}

// enrichVersions fills empty versions from the exporters' version metrics,
// one instant query per engine. Any failure degrades to "unknown".
func (s *Service) enrichVersions(ctx context.Context, auth callerAuth, services []serviceInfo) {
	l := s.l.WithField("tool", "pmm_inventory")
	needed := map[string]bool{}
	for _, svc := range services {
		if svc.Version == "" {
			needed[svc.Engine] = true
		}
	}

	versions := map[string]map[string]string{}
	for engine := range needed {
		vm, ok := versionMetrics[engine]
		if !ok {
			continue
		}
		samples, err := s.queryMetrics(ctx, auth, vm.metric, time.Time{})
		if err != nil {
			l.Debugf("Cannot read %s, versions will be unknown: %s.", vm.metric, err)
			continue
		}
		byService := map[string]string{}
		for _, sample := range samples {
			if name := sample.Labels["service_name"]; name != "" {
				byService[name] = sample.Labels[vm.label]
			}
		}
		versions[engine] = byService
	}

	for i := range services {
		if services[i].Version != "" {
			continue
		}
		if v := versions[services[i].Engine][services[i].ServiceName]; v != "" {
			services[i].Version = v
		} else {
			services[i].Version = unknownVersion
		}
	}
}

// queryMetrics runs an instant PromQL query against PMM's metrics datasource.
func (s *Service) queryMetrics(ctx context.Context, auth callerAuth, promql string, at time.Time) ([]metricSample, error) {
	uid, err := s.metricsDatasourceUID(ctx, auth)
	if err != nil {
		return nil, err
	}
	return s.api.QueryInstant(ctx, auth, uid, promql, at)
}

// metricsDatasourceUID discovers (once) the uid of the Prometheus-typed
// datasource, which is PMM's VictoriaMetrics behind vmproxy.
func (s *Service) metricsDatasourceUID(ctx context.Context, auth callerAuth) (string, error) {
	s.dsMu.Lock()
	defer s.dsMu.Unlock()
	if s.dsUID != "" {
		return s.dsUID, nil
	}

	datasources, err := s.api.ListDatasources(ctx, auth)
	if err != nil {
		return "", err
	}
	for _, ds := range datasources {
		if ds.Type == "prometheus" && ds.UID != "" {
			s.dsUID = ds.UID
			return ds.UID, nil
		}
	}
	return "", newToolError(codePMMUnavailable, "no Prometheus-typed datasource found in Grafana")
}
