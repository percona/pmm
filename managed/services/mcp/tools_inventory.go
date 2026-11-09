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
	"errors"
	"fmt"
	"net/http"
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
	engineProxySQL   = "proxysql"
	engineHAProxy    = "haproxy"
	engineValkey     = "valkey"
	engineExternal   = "external"

	unknownVersion = "unknown"

	inventoryDefaultLimit = 50
	inventoryMaxLimit     = 200
)

// serviceTypes are PMM's service types, the engines of the service list.
var serviceTypes = []string{engineMySQL, enginePostgreSQL, engineMongoDB, engineProxySQL, engineHAProxy, engineValkey, engineExternal}

const (
	// The query that reads, with the look-back's seconds, the newest sample time of every up series, of any value.
	serviceListQuery = "max by (service_id, service_name, service_type, node_id, node_name, environment, cluster) (tlast_over_time(up[%ds]))"

	// The longest look-back of the service list.
	maxServiceListLookback = 7 * day
)

// versionMetrics maps an engine to the metric and label that carry the server
// version. Confirmed live on PMM 3.8.1: inventory's version field is empty, so
// versions come from the exporters' info metrics.
var versionMetrics = map[string]struct{ metric, label string }{
	engineMySQL:      {"mysql_version_info", "version"},
	enginePostgreSQL: {"pg_static", "short_version"},
	engineMongoDB:    {"mongodb_version_info", "mongodb"},
}

// serviceInfo is one service of the caller's service list.
type serviceInfo struct {
	ServiceID   string
	ServiceName string
	Engine      string
	Version     string
	NodeID      string
	NodeName    string
	Environment string
	Cluster     string
	LastSeen    time.Time
}

type versionInput struct{}

type inventoryInput struct {
	Engine string `json:"engine,omitempty" jsonschema:"Filter by service type: mysql, postgresql, mongodb, proxysql, haproxy, valkey or external"`
	Node   string `json:"node,omitempty" jsonschema:"Filter by node name"`
	Limit  int    `json:"limit,omitempty" jsonschema:"Maximum services to return; default 50, maximum 200"`
	Offset int    `json:"offset,omitempty" jsonschema:"Number of services to skip, for the next page"`
}

// serviceRow is one service in the result of pmm_inventory.
type serviceRow struct {
	ServiceID   string    `json:"service_id" jsonschema:"Service id."`
	ServiceName string    `json:"service_name" jsonschema:"Untrusted. Service name."`
	Engine      string    `json:"engine" jsonschema:"PMM service type: mysql, postgresql, mongodb, proxysql, haproxy, valkey or external."`
	Version     string    `json:"version,omitempty" jsonschema:"Untrusted. Server version, left out when it cannot be read."`
	NodeID      string    `json:"node_id,omitempty" jsonschema:"Id of the node the service runs on."`
	NodeName    string    `json:"node_name,omitempty" jsonschema:"Untrusted. Name of the node the service runs on."`
	Environment string    `json:"environment,omitempty" jsonschema:"Untrusted. Environment label of the service."`
	Cluster     string    `json:"cluster,omitempty" jsonschema:"Untrusted. Cluster label of the service."`
	LastSeen    time.Time `json:"last_seen" jsonschema:"Time of the service's newest up sample, whatever its value."`
	PMMURL      string    `json:"pmm_url,omitempty" jsonschema:"Link to the service's instance summary dashboard in PMM."`
}

// inventoryResult is the result of pmm_inventory.
type inventoryResult struct {
	Services []serviceRow `json:"services" jsonschema:"Monitored services on this page."`
	Page     page         `json:"page" jsonschema:"Paging of the full list."`
}

func (r inventoryResult) render() string {
	if r.Page.Total == 0 {
		return "No services found in your service list (services that sent metrics in the last hour)."
	}
	if len(r.Services) == 0 {
		return r.Page.pastEnd("service list")
	}
	var lines []string
	lines = append(lines, fmt.Sprintf("Monitored services (%d):", r.Page.Total))
	for _, svc := range r.Services {
		version := svc.Version
		if version == "" {
			version = unknownVersion
		}
		row := fmt.Sprintf("- %s | %s %s | id=%s", svc.ServiceName, svc.Engine, version, svc.ServiceID)
		if svc.NodeName != "" {
			row += " | node=" + svc.NodeName
		}
		if svc.Environment != "" {
			row += " | environment=" + svc.Environment
		}
		if svc.Cluster != "" {
			row += " | cluster=" + svc.Cluster
		}
		row += " | last_seen=" + svc.LastSeen.UTC().Format(time.RFC3339)
		if svc.PMMURL != "" {
			row += " | " + link("View in PMM", svc.PMMURL)
		}
		lines = append(lines, row)
	}
	if m := r.Page.marker(); m != "" {
		lines = append(lines, m)
	}
	return strings.Join(lines, "\n")
}

func (s *Service) registerInventoryTools(server *mcp.Server) {
	addTool(s, server, &mcp.Tool{
		Name:  "pmm_version",
		Title: "PMM Server version",
		Description: "Return the PMM Server version. Use it first to confirm the connection and " +
			"that the caller's token is accepted.",
		Annotations: readOnly("PMM Server version"),
	}, s.version)

	addTool(s, server, &mcp.Tool{
		Name:  "pmm_inventory",
		Title: "List monitored services",
		Description: "List the monitored services in your service list: those whose up series you can read and that sent " +
			"metrics in the last hour, with service type, version, node, environment, cluster and last_seen. Use this to " +
			"discover a service_id / service_name and the exact engine version before analysing a query.",
		Annotations: readOnly("List monitored services"),
	}, s.inventory)
}

// versionResult is the result of pmm_version.
type versionResult struct {
	Version        string `json:"version" jsonschema:"PMM Server version."`
	ManagedVersion string `json:"managed_version,omitempty" jsonschema:"pmm-managed full version."`
}

func (r versionResult) render() string {
	text := "PMM Server " + r.Version
	if r.ManagedVersion != "" {
		text += fmt.Sprintf(" (pmm-managed %s)", r.ManagedVersion)
	}
	return text
}

func (s *Service) version(ctx context.Context, req *mcp.CallToolRequest, _ versionInput) (versionResult, error) {
	auth := callerAuthFromHeader(req.Extra.Header)

	v, err := s.api.Version(ctx, auth)
	if err != nil {
		return versionResult{}, err
	}

	out := versionResult{Version: v.Version}
	if v.Managed != nil {
		out.ManagedVersion = v.Managed.FullVersion
	}
	return out, nil
}

func (s *Service) inventory(ctx context.Context, req *mcp.CallToolRequest, in inventoryInput) (inventoryResult, error) {
	if in.Engine != "" && !slices.Contains(serviceTypes, in.Engine) {
		return inventoryResult{}, newToolError(codeInvalidInput, "engine must be one of %s; got '%s'", strings.Join(serviceTypes, ", "), in.Engine)
	}
	err := checkPage(in.Offset, in.Limit, inventoryMaxLimit)
	if err != nil {
		return inventoryResult{}, err
	}
	auth := callerAuthFromHeader(req.Extra.Header)

	services, err := s.listServices(ctx, auth, s.lastHour())
	if err != nil {
		return inventoryResult{}, err
	}

	s.enrichVersions(ctx, auth, services)

	var filtered []serviceInfo
	for _, svc := range services {
		if in.Engine != "" && svc.Engine != in.Engine {
			continue
		}
		if in.Node != "" && svc.NodeName != in.Node {
			continue
		}
		filtered = append(filtered, svc)
	}

	paged, pg, err := paginate(filtered, in.Offset, in.Limit, inventoryDefaultLimit, inventoryMaxLimit)
	if err != nil {
		return inventoryResult{}, err
	}

	base := s.publicBaseURL(ctx, req.Extra.Header)
	rows := make([]serviceRow, 0, len(paged))
	for _, svc := range paged {
		row := serviceRow{
			ServiceID:   svc.ServiceID,
			ServiceName: svc.ServiceName,
			Engine:      svc.Engine,
			NodeID:      svc.NodeID,
			NodeName:    svc.NodeName,
			Environment: svc.Environment,
			Cluster:     svc.Cluster,
			LastSeen:    svc.LastSeen,
			PMMURL:      serviceURL(base, svc),
		}
		if svc.Version != unknownVersion {
			row.Version = svc.Version
		}
		rows = append(rows, row)
	}
	return inventoryResult{Services: rows, Page: pg}, nil
}

// listSpan returns the look-back of the service list for a call window: its last 7 days at most, and whether it cut the window.
func listSpan(w window) (window, bool) {
	floor := w.To.Add(-maxServiceListLookback)
	if floor.After(w.From) {
		return window{From: floor, To: w.To}, true
	}
	return w, false
}

// lastHour is the look-back of a call without a window.
func (s *Service) lastHour() window {
	return window{From: s.now().Add(-hourWindow), To: s.now()}
}

// listServices returns the services whose up series the caller can read and that sent a sample in the look-back of the window.
func (s *Service) listServices(ctx context.Context, auth callerAuth, w window) ([]serviceInfo, error) {
	span, _ := listSpan(w)
	seconds := int64(span.To.Sub(span.From) / time.Second)
	samples, err := s.queryMetrics(ctx, auth, fmt.Sprintf(serviceListQuery, seconds), span.To)
	if err != nil {
		return nil, err
	}

	byID := map[string]serviceInfo{}
	for _, sample := range samples {
		id := sample.Labels["service_id"]
		if id == "" {
			continue
		}
		svc := serviceInfo{
			ServiceID:   id,
			ServiceName: sample.Labels["service_name"],
			Engine:      sample.Labels["service_type"],
			NodeID:      sample.Labels["node_id"],
			NodeName:    sample.Labels["node_name"],
			Environment: sample.Labels["environment"],
			Cluster:     sample.Labels["cluster"],
			LastSeen:    time.Unix(int64(sample.Value), 0).UTC(),
		}
		prev, ok := byID[id]
		if !ok || svc.LastSeen.After(prev.LastSeen) {
			byID[id] = svc
		}
	}

	out := make([]serviceInfo, 0, len(byID))
	for _, svc := range byID {
		out = append(out, svc)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Engine != out[j].Engine {
			return out[i].Engine < out[j].Engine
		}
		if out[i].ServiceName != out[j].ServiceName {
			return out[i].ServiceName < out[j].ServiceName
		}
		return out[i].ServiceID < out[j].ServiceID
	})
	return out, nil
}

// serviceNamed returns, among the services with the name, the one with the newest last_seen.
func serviceNamed(services []serviceInfo, name string) (serviceInfo, bool) {
	var found serviceInfo
	ok := false
	for _, svc := range services {
		if svc.ServiceName != name {
			continue
		}
		if !ok || svc.LastSeen.After(found.LastSeen) {
			found, ok = svc, true
		}
	}
	return found, ok
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
	return s.api.QueryInstant(ctx, auth, uid, promRequest{Query: promql, Time: at, Timeout: metricsTimeout})
}

// metricsDatasourceName is the stable name PMM provisions its VictoriaMetrics
// datasource under (see the Grafana datasources.yml provisioning file). The uid
// is generated, so the name is the only stable handle.
const metricsDatasourceName = "Metrics"

// metricsDatasourceUID looks up the uid of PMM's metrics datasource,
// VictoriaMetrics behind vmproxy, by its provisioned name, so a user's own
// Prometheus-typed datasources are never picked.
//
// It is looked up for every query, with the caller's credentials: a lookup by
// name is one small call, nothing is shared between callers, and nothing goes
// stale when the datasource is re-provisioned or recreated.
func (s *Service) metricsDatasourceUID(ctx context.Context, auth callerAuth) (string, error) {
	ds, err := s.api.GetDatasourceByName(ctx, auth, metricsDatasourceName)
	var se *statusError
	if errors.As(err, &se) && se.status == http.StatusNotFound {
		return "", newToolError(codePMMUnavailable, "no Grafana datasource named '%s' found", metricsDatasourceName)
	}
	if err != nil {
		return "", err
	}
	if ds.Type != "prometheus" || ds.UID == "" {
		return "", newToolError(codePMMUnavailable, "Grafana datasource '%s' is not Prometheus-typed", metricsDatasourceName)
	}
	return ds.UID, nil
}
