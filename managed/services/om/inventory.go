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

package om

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"maps"
	"math"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/xdg-go/stringprep"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	omv1 "github.com/percona/pmm/api/om/v1"
	"github.com/percona/pmm/managed/models"
)

// maxInventoryRunLimit matches ListInventoryRunsRequest.limit's lte: 100 and the app's le=100,
// since the limit is forwarded to PMM Extensions verbatim.
const (
	defaultInventoryRunLimit = 20
	maxInventoryRunLimit     = 100
)

func (s *Service) inventoryProbe() (extensionsApp, error) {
	if s.probe == nil {
		return extensionsApp{}, status.Error(codes.FailedPrecondition,
			"PMM Extensions is not configured; set PMM_EXTENSIONS_URL and PMM_EXTENSIONS_TOKEN to reach the inventory app")
	}
	return s.probe.app, nil
}

// ListInventoryHosts returns every host the inventory app has a row for.
func (s *Service) ListInventoryHosts(ctx context.Context, req *omv1.ListInventoryHostsRequest) (*omv1.ListInventoryHostsResponse, error) {
	probe, err := s.inventoryProbe()
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	if req.HasService != nil {
		query.Set("has_service", strconv.FormatBool(req.GetHasService()))
	}
	if req.Failing != nil {
		query.Set("failing", strconv.FormatBool(req.GetFailing()))
	}
	if req.Executor != nil {
		query.Set("executor", strconv.FormatBool(req.GetExecutor()))
	}

	hosts, err := fetchAllPages(func(offset, limit int) (extensionsPage[extensionsHost], error) {
		pageQuery := url.Values{}
		maps.Copy(pageQuery, query)
		pageQuery.Set("offset", strconv.Itoa(offset))
		pageQuery.Set("limit", strconv.Itoa(limit))
		page := extensionsPage[extensionsHost]{}
		call := inventoryCall{method: http.MethodGet, path: "hosts", query: pageQuery}
		err := probe.call(ctx, call, &page)
		return page, err
	})
	if err != nil {
		return nil, err
	}

	connectedNodes, err := s.pmmAgentConnectedByNode()
	if err != nil {
		return nil, err
	}

	serverNodes, err := s.pmmServerNodeIDs()
	if err != nil {
		return nil, err
	}

	response := &omv1.ListInventoryHostsResponse{Hosts: make([]*omv1.InventoryHost, 0, len(hosts))}
	for _, host := range hosts {
		proto := inventoryHostToProto(host, connectedNodes[host.NodeID], serverNodes[host.NodeID])
		if req.AutomationEligible != nil && proto.AutomationEligible != req.GetAutomationEligible() {
			continue
		}
		response.Hosts = append(response.Hosts, proto)
	}
	return response, nil
}

func (s *Service) pmmAgentConnectedByNode() (map[string]bool, error) {
	if s.agents == nil {
		return nil, nil //nolint:nilnil // absent registry: every lookup below reads as not connected
	}
	agentType := models.PMMAgentType
	pmmAgents, err := models.FindAgents(s.db.Querier, models.AgentFilters{AgentType: &agentType})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list pmm-agents: %s", err)
	}
	connected := make(map[string]bool, len(pmmAgents))
	for _, agent := range pmmAgents {
		if agent.RunsOnNodeID == nil {
			continue
		}
		connected[*agent.RunsOnNodeID] = s.agents.IsConnected(agent.AgentID)
	}
	return connected, nil
}

func (s *Service) pmmServerNodeIDs() (map[string]bool, error) {
	if s.db == nil {
		return nil, nil //nolint:nilnil // absent store: every lookup below reads as "not the server"
	}
	nodes, err := models.FindNodes(s.db.Querier, models.NodeFilters{})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list nodes: %s", err)
	}
	servers := make(map[string]bool, 1)
	for _, node := range nodes {
		if node.IsPMMServerNode {
			servers[node.NodeID] = true
		}
	}
	return servers, nil
}

// GetInventoryHost returns one host.
func (s *Service) GetInventoryHost(ctx context.Context, req *omv1.GetInventoryHostRequest) (*omv1.GetInventoryHostResponse, error) {
	probe, err := s.inventoryProbe()
	if err != nil {
		return nil, err
	}

	host := extensionsHost{}
	call := inventoryCall{method: http.MethodGet, path: inventoryPath("hosts", req.GetNodeId())}
	err = probe.call(ctx, call, &host)
	if err != nil {
		return nil, err
	}

	connectedNodes, err := s.pmmAgentConnectedByNode()
	if err != nil {
		return nil, err
	}

	serverNodes, err := s.pmmServerNodeIDs()
	if err != nil {
		return nil, err
	}
	return &omv1.GetInventoryHostResponse{
		Host: inventoryHostToProto(host, connectedNodes[host.NodeID], serverNodes[host.NodeID]),
	}, nil
}

// DeleteInventoryHost forgets a host and the services on it.
// It is not suppression: a host PMM still knows about returns on the next refresh.
func (s *Service) DeleteInventoryHost(ctx context.Context, req *omv1.DeleteInventoryHostRequest) (*omv1.DeleteInventoryHostResponse, error) {
	probe, err := s.inventoryProbe()
	if err != nil {
		return nil, err
	}

	call := inventoryCall{method: http.MethodDelete, path: inventoryPath("hosts", req.GetNodeId())}
	err = probe.call(ctx, call, nil)
	if err != nil {
		return nil, err
	}
	return &omv1.DeleteInventoryHostResponse{}, nil
}

// ListInventoryServices returns every service the inventory app has a row for.
func (s *Service) ListInventoryServices(ctx context.Context, req *omv1.ListInventoryServicesRequest) (*omv1.ListInventoryServicesResponse, error) {
	probe, err := s.inventoryProbe()
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	if nodeID := req.GetNodeId(); nodeID != "" {
		query.Set("node_id", nodeID)
	}
	if req.Failing != nil {
		query.Set("failing", strconv.FormatBool(req.GetFailing()))
	}

	services, err := fetchAllPages(func(offset, limit int) (extensionsPage[extensionsService], error) {
		pageQuery := url.Values{}
		maps.Copy(pageQuery, query)
		pageQuery.Set("offset", strconv.Itoa(offset))
		pageQuery.Set("limit", strconv.Itoa(limit))
		page := extensionsPage[extensionsService]{}
		call := inventoryCall{method: http.MethodGet, path: "services", query: pageQuery}
		err := probe.call(ctx, call, &page)
		return page, err
	})
	if err != nil {
		return nil, err
	}

	response := &omv1.ListInventoryServicesResponse{Services: make([]*omv1.InventoryService, 0, len(services))}
	for _, service := range services {
		response.Services = append(response.Services, inventoryServiceToProto(service))
	}
	return response, nil
}

// GetInventoryService returns one service.
func (s *Service) GetInventoryService(ctx context.Context, req *omv1.GetInventoryServiceRequest) (*omv1.GetInventoryServiceResponse, error) {
	probe, err := s.inventoryProbe()
	if err != nil {
		return nil, err
	}

	service := extensionsService{}
	call := inventoryCall{method: http.MethodGet, path: inventoryPath("services", req.GetServiceId())}
	err = probe.call(ctx, call, &service)
	if err != nil {
		return nil, err
	}
	return &omv1.GetInventoryServiceResponse{Service: inventoryServiceToProto(service)}, nil
}

// DeleteInventoryService forgets one service.
func (s *Service) DeleteInventoryService(ctx context.Context, req *omv1.DeleteInventoryServiceRequest) (*omv1.DeleteInventoryServiceResponse, error) {
	probe, err := s.inventoryProbe()
	if err != nil {
		return nil, err
	}

	call := inventoryCall{method: http.MethodDelete, path: inventoryPath("services", req.GetServiceId())}
	err = probe.call(ctx, call, nil)
	if err != nil {
		return nil, err
	}
	return &omv1.DeleteInventoryServiceResponse{}, nil
}

// ListInventoryRuns returns the inventory app's refresh history.
func (s *Service) ListInventoryRuns(ctx context.Context, req *omv1.ListInventoryRunsRequest) (*omv1.ListInventoryRunsResponse, error) {
	probe, err := s.inventoryProbe()
	if err != nil {
		return nil, err
	}

	since, until := req.GetSince(), req.GetUntil()
	if since != nil && until != nil && until.AsTime().Before(since.AsTime()) {
		return nil, status.Error(codes.InvalidArgument, "until must not be before since")
	}

	limit := req.GetLimit()
	switch {
	case limit <= 0:
		limit = defaultInventoryRunLimit
	case limit > maxInventoryRunLimit:
		limit = maxInventoryRunLimit
	}
	query := url.Values{"limit": []string{strconv.FormatInt(int64(limit), 10)}}
	if since != nil {
		query.Set("since", since.AsTime().UTC().Format(time.RFC3339Nano))
	}
	if until != nil {
		query.Set("until", until.AsTime().UTC().Format(time.RFC3339Nano))
	}

	runs := []extensionsRun{}
	call := inventoryCall{method: http.MethodGet, path: "runs", query: query}
	err = probe.call(ctx, call, &runs)
	if err != nil {
		return nil, err
	}

	response := &omv1.ListInventoryRunsResponse{Runs: make([]*omv1.InventoryRun, 0, len(runs))}
	for _, run := range runs {
		response.Runs = append(response.Runs, inventoryRunToProto(run))
	}
	return response, nil
}

// GetInventoryRun returns one refresh.
func (s *Service) GetInventoryRun(ctx context.Context, req *omv1.GetInventoryRunRequest) (*omv1.GetInventoryRunResponse, error) {
	probe, err := s.inventoryProbe()
	if err != nil {
		return nil, err
	}

	run := extensionsRun{}
	call := inventoryCall{method: http.MethodGet, path: inventoryPath("runs", req.GetRunId())}
	err = probe.call(ctx, call, &run)
	if err != nil {
		return nil, err
	}
	return &omv1.GetInventoryRunResponse{
		Run:      inventoryRunToProto(run),
		Entities: inventoryRunEntitiesToProto(run.Nodes),
	}, nil
}

// TriggerInventoryRefresh probes the estate, or the named hosts within it.
// It returns once the refresh is accepted; the app judges conflicts per host.
func (s *Service) TriggerInventoryRefresh(ctx context.Context, req *omv1.TriggerInventoryRefreshRequest) (*omv1.TriggerInventoryRefreshResponse, error) {
	probe, err := s.inventoryProbe()
	if err != nil {
		return nil, err
	}

	// A nil slice marshals as JSON null, which the app rejects; an empty list means the whole estate.
	nodeIDs := req.GetNodeIds()
	if nodeIDs == nil {
		nodeIDs = []string{}
	}
	body := map[string]any{"node_ids": nodeIDs}

	accepted := struct {
		RunID     string   `json:"run_id"`
		Status    string   `json:"status"`
		StartedAt *string  `json:"started_at"`
		Scope     []string `json:"scope"`
	}{}
	call := inventoryCall{method: http.MethodPost, path: "runs", body: body}
	err = probe.call(ctx, call, &accepted)
	if err != nil {
		return nil, err
	}

	response := &omv1.TriggerInventoryRefreshResponse{
		RunId:  accepted.RunID,
		Status: extensionsRunStatusToProto(accepted.Status),
		Scope:  accepted.Scope,
	}
	if accepted.StartedAt != nil {
		if parsed := parseSepTime(*accepted.StartedAt); parsed != nil {
			response.StartTime = parsed
		}
	}
	return response, nil
}

// validateMemberConfigs checks per-member settings against node_ids and the rules rs.initiate() enforces,
// so a set it would refuse fails here rather than minutes into a run.
func validateMemberConfigs(nodeIDs []string, memberConfigs map[string]*omv1.BootstrapMemberConfig) error {
	nodeIDSet := make(map[string]bool, len(nodeIDs))
	for _, nodeID := range nodeIDs {
		nodeIDSet[nodeID] = true
	}
	var violations []string
	for _, nodeID := range slices.Sorted(maps.Keys(memberConfigs)) {
		member := memberConfigs[nodeID]
		if !nodeIDSet[nodeID] {
			violations = append(violations, fmt.Sprintf(
				"member_configs names host %s, which is not in node_ids", nodeID,
			))
			continue
		}
		// rs.initiate() requires a delayed member to have priority 0 and no vote. Unset counts as wrong,
		// since unset means MongoDB's defaults of priority 1 and a vote.
		if member.GetDelaySecs() > 0 && (member.Priority == nil || *member.Priority != 0 || member.Votes == nil || *member.Votes) {
			violations = append(violations, fmt.Sprintf(
				"host %s: a delayed member (delay_secs > 0) must also set priority 0 and votes off", nodeID,
			))
		}
		// rs.initiate() refuses a hidden member that can be elected; unset priority means the default of 1.
		if member.GetHidden() && (member.Priority == nil || *member.Priority > 0) {
			violations = append(violations, fmt.Sprintf(
				"host %s: a hidden member must also set priority 0, since MongoDB will not elect a member that clients cannot see", nodeID,
			))
		}
	}

	// Counted over node_ids: a host absent from member_configs keeps MongoDB's defaults, so it both
	// votes and can be elected.
	voters, electable := 0, 0
	for _, nodeID := range nodeIDs {
		member, named := memberConfigs[nodeID]
		if !named || member.Votes == nil || *member.Votes {
			voters++
		}
		if !named || member.Priority == nil || *member.Priority > 0 {
			electable++
		}
	}
	if voters == 0 {
		violations = append(violations,
			"member_configs leaves no host with a vote; a replica set needs at least one voting member")
	}
	if electable == 0 {
		violations = append(violations,
			"member_configs leaves every host with priority 0; a replica set needs at least one member that can become primary")
	}

	if len(violations) > 0 {
		return status.Error(codes.InvalidArgument, strings.Join(violations, "; "))
	}
	return nil
}

// runIgnoredSettings names the requested settings the accepted run does not echo back, or "".
// An om_bootstrap older than these fields accepts the request but silently ignores them.
func runIgnoredSettings(planned extensionsTriggerBootstrapRunRequest, accepted *extensionsBootstrapRun) string {
	var ignored []string
	if planned.DataPath != "" && accepted.DataPath != planned.DataPath {
		ignored = append(ignored, "the data path")
	}
	if planned.LogPath != "" && accepted.LogPath != planned.LogPath {
		ignored = append(ignored, "the log path")
	}
	if planned.Port != 0 && accepted.Port != planned.Port {
		ignored = append(ignored, "the port")
	}
	if planned.BindIP != "" && accepted.BindIP != planned.BindIP {
		ignored = append(ignored, "the bind address")
	}
	for host, member := range planned.MemberConfigs {
		got, ok := accepted.MemberConfigs[host]
		if !ok || !sameMemberConfig(got, member) {
			ignored = append(ignored, "the per-member replica-set settings")
			break
		}
	}
	if _, ok := runStep(*accepted, createAdminUserStep); planned.CreateAdminUser && !ok {
		ignored = append(ignored, "the administrator user")
	}
	return strings.Join(ignored, ", ")
}

// sameMemberConfig compares requested and accepted settings for one host. Optional fields left unset
// are skipped, since the app echoes back its own default for them.
func sameMemberConfig(accepted, planned extensionsMemberConfig) bool {
	if pointer.GetUint32(planned.Priority) != pointer.GetUint32(accepted.Priority) && planned.Priority != nil {
		return false
	}
	if planned.Votes != nil && pointer.GetBool(planned.Votes) != pointer.GetBool(accepted.Votes) {
		return false
	}
	if planned.BindIP != nil && pointer.GetString(planned.BindIP) != pointer.GetString(accepted.BindIP) {
		return false
	}
	return planned.Hidden == accepted.Hidden && planned.DelaySecs == accepted.DelaySecs
}

// abandonRun cancels a run PMM cannot carry through, best-effort.
func (s *Service) abandonRun(ctx context.Context, runID, reason string) {
	s.l.Warnf("bootstrap run %s: cancelling it: %s", runID, reason)
	_, err := s.bootstrap.cancelRun(ctx, runID)
	if err != nil {
		s.l.Warnf("bootstrap run %s: failed to cancel it: %s", runID, err)
	}
}

// newAdminSecret returns a secret for the named administrator (nil if none) and any password PMM generated.
func newAdminSecret(req *omv1.TriggerHostBootstrapRequest) (*models.OmBootstrapSecret, string, error) {
	username, password := req.GetAdminUsername(), req.GetAdminPassword()
	if username == "" {
		if password != "" {
			return nil, "", status.Error(codes.InvalidArgument, "admin_password needs admin_username")
		}
		return nil, "", nil
	}
	if username == bootstrapMongoDBUsername {
		return nil, "", status.Errorf(codes.InvalidArgument,
			"admin_username %s is reserved for PMM's monitoring user", bootstrapMongoDBUsername)
	}
	generated := ""
	if password == "" {
		generated = rand.Text()
		password = generated
	}
	// mongod runs a SCRAM-SHA-256 password through SASLprep, so one it rejects would fail every createUser attempt.
	_, err := stringprep.SASLprep.Prepare(password)
	if err != nil {
		return nil, "", status.Errorf(codes.InvalidArgument,
			"admin_password has characters MongoDB does not accept in a password: %s", err)
	}

	secret, err := newBootstrapSecret()
	if err != nil {
		return nil, "", status.Errorf(codes.Internal, "failed to generate the run's credentials: %s", err)
	}
	secret.AdminUsername, secret.AdminPassword = username, password
	return secret, generated, nil
}

// executorUnusable says why nothing can be dispatched to this host, or "" if nothing is known to be wrong.
// Only an explicit false rejects; a host without an executor sub-document is left to PMM Extensions.
func executorUnusable(host extensionsHost) string {
	executor, ok := host.Observed["executor"].(map[string]any)
	if !ok {
		return ""
	}

	// These strings reach the install wizard as user-facing text, so they do not name the scheduler.
	reachable, ok := executor["reachable"].(bool)
	if ok && !reachable {
		return "its automation agent is not reachable"
	}
	driverHealthy, ok := executor["driver_healthy"].(bool)
	if ok && !driverHealthy {
		return "its automation agent's job driver is not healthy"
	}
	return ""
}

// supportedBootstrapOSIDs mirrors om_bootstrap's OperatingSystem enum, which remains the source of truth,
// so an unsupported OS fails at trigger time rather than mid-run.
var supportedBootstrapOSIDs = map[string]bool{"ubuntu": true, "rocky": true}

func supportedBootstrapOSNames() string {
	names := make([]string, 0, len(supportedBootstrapOSIDs))
	for id := range supportedBootstrapOSIDs {
		names = append(names, id)
	}
	slices.Sort(names)
	return strings.Join(names, ", ")
}

// refuseNodesNotReadyToInstall rejects the request, naming every node that cannot be installed onto,
// and otherwise returns the one OS the selected nodes share.
func refuseNodesNotReadyToInstall(nodeIDs []string, hosts []extensionsHost) (string, error) {
	blocked := make([]string, 0, len(hosts))
	osIDs := make(map[string][]string)
	for i, host := range hosts {
		name := hostDisplayName(host, nodeIDs[i])
		problems := hostNotReadyReasons(host)
		if len(problems) > 0 {
			blocked = append(blocked, fmt.Sprintf("%s: %s", name, strings.Join(problems, "; ")))
			continue
		}
		osID, _ := host.Observed["os_id"].(string)
		osIDs[osID] = append(osIDs[osID], name)
	}
	if len(blocked) > 0 {
		return "", status.Errorf(codes.FailedPrecondition,
			"%d of the selected node(s) cannot be installed onto -- %s. Each has to be fixed on the node itself; "+
				"its newest scan on the Nodes page says what failed.",
			len(blocked), strings.Join(blocked, "; "))
	}
	if len(osIDs) > 1 {
		groups := make([]string, 0, len(osIDs))
		for osID, names := range osIDs {
			groups = append(groups, fmt.Sprintf("%s on %s", osID, strings.Join(names, ", ")))
		}
		slices.Sort(groups)
		return "", status.Errorf(codes.InvalidArgument,
			"the selected nodes do not all run the same operating system (%s); a mixed-OS replica set is out of "+
				"phase-1 scope, so select nodes that run one of them",
			strings.Join(groups, "; "))
	}
	for osID := range osIDs {
		return osID, nil
	}
	return "", status.Error(codes.InvalidArgument, "node_ids is empty")
}

// hostNotReadyReasons returns why an install on this node would fail, as opposed to why it must not be
// attempted at all, which is hostNotATargetReasons.
func hostNotReadyReasons(host extensionsHost) []string {
	var reasons []string
	if host.ExecutorHost == nil || *host.ExecutorHost == "" {
		reasons = append(reasons, "no automation agent is registered for it, so nothing can be dispatched to it")
	} else if unusable := executorUnusable(host); unusable != "" {
		reasons = append(reasons, unusable)
	}
	switch osID, _ := host.Observed["os_id"].(string); {
	case osID == "":
		reasons = append(reasons,
			"no scan has reported its operating system, so its scans are not landing")
	case !supportedBootstrapOSIDs[osID]:
		reasons = append(reasons,
			fmt.Sprintf("it runs %s, which Operations does not support installing onto (supported: %s)",
				osID, supportedBootstrapOSNames()))
	}
	return reasons
}

func hostDisplayName(host extensionsHost, nodeID string) string {
	if host.Name != "" {
		return host.Name
	}
	return nodeID
}

// refuseNodesThatAreNotTargets rejects the request, naming every node that must never be installed onto.
func (s *Service) refuseNodesThatAreNotTargets(nodeIDs []string, hosts []extensionsHost) error {
	serverNodes, err := s.pmmServerNodeIDs()
	if err != nil {
		return err
	}

	blocked := make([]string, 0, len(hosts))
	for i, host := range hosts {
		reasons := hostNotATargetReasons(serverNodes[nodeIDs[i]], host)
		if len(reasons) == 0 {
			continue
		}
		blocked = append(blocked, fmt.Sprintf("%s: %s",
			hostDisplayName(host, nodeIDs[i]), strings.Join(reasons, "; ")))
	}
	if len(blocked) == 0 {
		return nil
	}
	return status.Errorf(codes.FailedPrecondition,
		"Operations does not install onto %d of the selected node(s) -- %s. Remove them from the selection.",
		len(blocked), strings.Join(blocked, "; "))
}

// TriggerHostBootstrap plans installing MongoDB on one or three hosts and initializing them as one replica set.
// The om_bootstrap app keys hosts by executor host name rather than node ID, since that is the name Nomad knows.
func (s *Service) TriggerHostBootstrap(ctx context.Context, req *omv1.TriggerHostBootstrapRequest) (*omv1.TriggerHostBootstrapResponse, error) {
	probe, err := s.inventoryProbe()
	if err != nil {
		return nil, err
	}
	if s.bootstrap == nil {
		return nil, status.Error(codes.FailedPrecondition,
			"PMM Extensions is not configured; set PMM_EXTENSIONS_URL and PMM_EXTENSIONS_TOKEN to reach the bootstrap app")
	}
	nodeIDs := req.GetNodeIds()
	if len(nodeIDs) != 1 && len(nodeIDs) != 3 {
		return nil, status.Errorf(codes.InvalidArgument,
			"node_ids must have exactly one or three entries, got %d", len(nodeIDs))
	}

	err = validateMemberConfigs(nodeIDs, req.GetMemberConfigs())
	if err != nil {
		return nil, err
	}

	secret, generatedPassword, err := newAdminSecret(req)
	if err != nil {
		return nil, err
	}

	hosts := make([]extensionsHost, 0, len(nodeIDs))
	for _, nodeID := range nodeIDs {
		host := extensionsHost{}
		call := inventoryCall{method: http.MethodGet, path: inventoryPath("hosts", nodeID)}
		err = probe.call(ctx, call, &host)
		if err != nil {
			return nil, err
		}
		hosts = append(hosts, host)
	}

	// Enforced here, not only in the UI: automation_eligible is advisory, can be minutes stale,
	// and a direct API call never reads it.
	err = s.refuseNodesThatAreNotTargets(nodeIDs, hosts)
	if err != nil {
		return nil, err
	}

	osID, err := refuseNodesNotReadyToInstall(nodeIDs, hosts)
	if err != nil {
		return nil, err
	}

	// Nothing below can fail: every node has been judged, so this only assembles.
	executorHosts := make([]string, 0, len(nodeIDs))
	memberConfigs := make(map[string]extensionsMemberConfig, len(req.GetMemberConfigs()))
	for i, nodeID := range nodeIDs {
		executorHost := *hosts[i].ExecutorHost
		executorHosts = append(executorHosts, executorHost)
		if member, ok := req.GetMemberConfigs()[nodeID]; ok {
			memberConfigs[executorHost] = extensionsMemberConfig{
				Priority:  member.Priority,
				Votes:     member.Votes,
				Hidden:    member.GetHidden(),
				DelaySecs: member.GetDelaySecs(),
				BindIP:    member.BindIp,
			}
		}
	}

	planned := extensionsTriggerBootstrapRunRequest{
		Hosts:           executorHosts,
		InstallMethod:   "packages",
		OS:              osID,
		MongoDBVersion:  req.GetMongodbVersion(),
		ReplicaSetName:  req.GetReplicaSetName(),
		DataPath:        req.GetDataPath(),
		LogPath:         req.GetLogPath(),
		Port:            req.GetPort(),
		BindIP:          req.GetBindIp(),
		MemberConfigs:   memberConfigs,
		CreateAdminUser: secret != nil,
	}
	run, err := s.bootstrap.triggerRun(ctx, planned)
	if err != nil {
		return nil, err
	}

	ignored := runIgnoredSettings(planned, run)
	if ignored != "" {
		s.abandonRun(ctx, run.ID, "PMM Extensions did not apply "+ignored)
		return nil, status.Errorf(codes.FailedPrecondition,
			"PMM Extensions accepted the run but did not apply %s, so it would come up on its own defaults "+
				"instead of the settings you chose; its om_bootstrap app is older than this PMM. "+
				"The run has been cancelled", ignored)
	}

	if secret != nil {
		secret.RunID = run.ID
		err = models.CreateOmBootstrapSecret(s.db.Querier, secret)
		if err != nil {
			s.abandonRun(ctx, run.ID, fmt.Sprintf("failed to store its credentials: %s", err))
			return nil, status.Errorf(codes.Internal,
				"failed to store the run's credentials, so the run has been cancelled: %s", err)
		}
	}

	if environment, cluster := req.GetEnvironment(), req.GetCluster(); environment != "" || cluster != "" {
		config := &models.OmBootstrapRunConfig{RunID: run.ID, Environment: environment, Cluster: cluster}
		err = models.CreateOmBootstrapRunConfig(s.db.Querier, config)
		if err != nil {
			// The run is already under way, so failing the request would misreport it; leave it unlabelled instead.
			s.l.Warnf("bootstrap run %s: failed to persist environment/cluster: %s", run.ID, err)
		}
	}

	return &omv1.TriggerHostBootstrapResponse{RunId: run.ID, GeneratedAdminPassword: generatedPassword}, nil
}

// inventoryHostsByExecutor returns every inventory host that has an executor, keyed by executor host name.
func (s *Service) inventoryHostsByExecutor(ctx context.Context) (map[string]extensionsHost, error) {
	probe, err := s.inventoryProbe()
	if err != nil {
		return nil, err
	}
	hosts, err := fetchAllPages(func(offset, limit int) (extensionsPage[extensionsHost], error) {
		query := url.Values{}
		query.Set("offset", strconv.Itoa(offset))
		query.Set("limit", strconv.Itoa(limit))
		page := extensionsPage[extensionsHost]{}
		call := inventoryCall{method: http.MethodGet, path: "hosts", query: query}
		err := probe.call(ctx, call, &page)
		return page, err
	})
	if err != nil {
		return nil, err
	}
	byExecutor := make(map[string]extensionsHost, len(hosts))
	for _, host := range hosts {
		if host.ExecutorHost != nil {
			byExecutor[*host.ExecutorHost] = host
		}
	}
	return byExecutor, nil
}

// inventorySyncAppModule is where PMM Extensions mounts the app PMMSyncer belongs to.
const inventorySyncAppModule = "inventory"

// pmmSyncerName limits the sync to PMMSyncer, keeping other syncers' cost off the bootstrap completion path.
const pmmSyncerName = "app.extensions.sync.syncers.pmm.PMMSyncer"

// inventorySyncDebounce coalesces sync requests: the sync is global, and each 15s tick holding a finished
// run would otherwise trigger a full inventory pull.
const inventorySyncDebounce = 45 * time.Second

// inventoryRefreshDebounce is shorter than inventorySyncDebounce so a re-probe can observe a completed sync,
// and longer than the 15s tick because what the probe reads cannot change faster than the sync feeds it.
const inventoryRefreshDebounce = 30 * time.Second

// triggerInventorySync asks PMM Extensions to pull PMM's services now, best-effort, so a re-probe of a
// just-registered host finds its service without waiting for PMMSyncer's schedule.
func (s *Service) triggerInventorySync(ctx context.Context) {
	probe, err := s.inventoryProbe()
	if err != nil {
		return
	}

	stamped := time.Now()
	s.syncRequestedMu.Lock()
	if stamped.Sub(s.syncRequested) < inventorySyncDebounce {
		s.syncRequestedMu.Unlock()
		return
	}
	previous := s.syncRequested
	s.syncRequested = stamped
	s.syncRequestedMu.Unlock()

	err = s.postInventorySync(ctx, probe)
	if err != nil {
		// The pull never started, so hand the debounce window back rather than standing down for nothing.
		s.syncRequestedMu.Lock()
		if s.syncRequested.Equal(stamped) {
			s.syncRequested = previous
		}
		s.syncRequestedMu.Unlock()
		s.l.Warnf("failed to trigger an inventory sync after a bootstrap: %s", err)
	}
}

// postInventorySync asks the inventory app to run PMMSyncer, falling back to every configured syncer
// when an app older than the syncer's package rename rejects the name.
func (s *Service) postInventorySync(ctx context.Context, probe extensionsApp) error {
	app := probe.client.app(inventorySyncAppModule)
	// Trailing slash: the app answers 307 without it, and redirects are refused (see refuseRedirect).
	call := inventoryCall{
		method: http.MethodPost,
		path:   "sync/",
		body:   map[string]any{"syncer": pmmSyncerName},
	}
	err := app.call(ctx, call, nil)
	if status.Code(err) != codes.InvalidArgument {
		return err
	}

	s.l.Warnf("the inventory app does not know syncer %s, so asking it to run every configured syncer instead: %s",
		pmmSyncerName, err)
	// Without a syncer name the route runs every configured syncer.
	call.body = map[string]any{}
	return app.call(ctx, call, nil)
}

// triggerScopedInventoryRefresh asks the app to re-probe nodeIDs now; best-effort and debounced per node.
// A 409 (Aborted) means another refresh holds one of the hosts, which is expected while a sweep runs.
func (s *Service) triggerScopedInventoryRefresh(ctx context.Context, nodeIDs []string) {
	probe, err := s.inventoryProbe()
	if err != nil {
		return
	}
	due := s.refreshDue(nodeIDs, time.Now())
	if len(due) == 0 {
		return
	}

	call := inventoryCall{method: http.MethodPost, path: "runs", body: map[string]any{"node_ids": due}}
	err = probe.call(ctx, call, nil)
	if err != nil && status.Code(err) != codes.Aborted {
		s.l.Warnf("failed to trigger a scoped inventory refresh for %v: %s", due, err)
		return
	}
	s.markRefreshed(due, time.Now())
}

func (s *Service) refreshDue(nodeIDs []string, now time.Time) []string {
	s.refreshRequestedMu.Lock()
	defer s.refreshRequestedMu.Unlock()

	// Pruning only bounds memory; each node's own stamp decides whether it is due.
	for nodeID, at := range s.refreshRequested {
		if now.Sub(at) >= inventoryRefreshDebounce {
			delete(s.refreshRequested, nodeID)
		}
	}
	due := make([]string, 0, len(nodeIDs))
	for _, nodeID := range nodeIDs {
		at, held := s.refreshRequested[nodeID]
		if !held || now.Sub(at) >= inventoryRefreshDebounce {
			due = append(due, nodeID)
		}
	}
	return due
}

// markRefreshed opens the debounce window for nodeIDs. Callers stamp after the request, so one that never
// reached the app does not hold off the next.
func (s *Service) markRefreshed(nodeIDs []string, now time.Time) {
	s.refreshRequestedMu.Lock()
	defer s.refreshRequestedMu.Unlock()

	if s.refreshRequested == nil {
		s.refreshRequested = make(map[string]time.Time, len(nodeIDs))
	}
	for _, nodeID := range nodeIDs {
		s.refreshRequested[nodeID] = now
	}
}

func (s *Service) nodeIDForExecutorHost(ctx context.Context, executorHost string) (string, error) {
	hosts, err := s.inventoryHostsByExecutor(ctx)
	if err != nil {
		return "", err
	}
	host, ok := hosts[executorHost]
	if !ok {
		return "", status.Errorf(codes.NotFound,
			"no host in the inventory has executor %s", executorHost)
	}
	return host.NodeID, nil
}

func (s *Service) bootstrapProbe() (*bootstrapClient, error) {
	if s.bootstrap == nil {
		return nil, status.Error(codes.FailedPrecondition,
			"PMM Extensions is not configured; set PMM_EXTENSIONS_URL and PMM_EXTENSIONS_TOKEN to reach the bootstrap app")
	}
	return s.bootstrap, nil
}

// maxBootstrapRunLimit matches ListBootstrapRunsRequest.limit's validation and om_bootstrap's le=100.
const (
	defaultBootstrapRunLimit = 20
	maxBootstrapRunLimit     = 100
)

// GetBootstrapRun returns one bootstrap run's current progress.
func (s *Service) GetBootstrapRun(ctx context.Context, req *omv1.GetBootstrapRunRequest) (*omv1.GetBootstrapRunResponse, error) {
	bootstrap, err := s.bootstrapProbe()
	if err != nil {
		return nil, err
	}

	run, err := bootstrap.getRun(ctx, req.GetRunId())
	if err != nil {
		return nil, err
	}

	environment, cluster := s.bootstrapRunConfigLabels(run.ID)
	return bootstrapRunToProto(run, s.confirmMonitoringLookup(ctx, run.Status), environment, cluster), nil
}

// CancelBootstrapRun asks PMM Extensions to flag a run for cancellation and, best-effort, stop its
// current step. Rolling the hosts back is left to the stepper.
func (s *Service) CancelBootstrapRun(ctx context.Context, req *omv1.CancelBootstrapRunRequest) (*omv1.CancelBootstrapRunResponse, error) {
	bootstrap, err := s.bootstrapProbe()
	if err != nil {
		return nil, err
	}

	run, err := bootstrap.cancelRun(ctx, req.GetRunId())
	if err != nil {
		return nil, err
	}

	environment, cluster := s.bootstrapRunConfigLabels(run.ID)
	return &omv1.CancelBootstrapRunResponse{
		Run: bootstrapRunToProto(run, s.confirmMonitoringLookup(ctx, run.Status), environment, cluster),
	}, nil
}

// ListBootstrapRuns returns the bootstrap run history in every status, newest first.
func (s *Service) ListBootstrapRuns(ctx context.Context, req *omv1.ListBootstrapRunsRequest) (*omv1.ListBootstrapRunsResponse, error) {
	bootstrap, err := s.bootstrapProbe()
	if err != nil {
		return nil, err
	}

	limit := int(req.GetLimit())
	switch {
	case limit <= 0:
		limit = defaultBootstrapRunLimit
	case limit > maxBootstrapRunLimit:
		limit = maxBootstrapRunLimit
	}

	runs, err := bootstrap.listRuns(ctx, "", limit)
	if err != nil {
		return nil, err
	}

	statuses := make([]string, len(runs))
	for i, run := range runs {
		statuses[i] = run.Status
	}
	hostsByExecutor := s.confirmMonitoringLookup(ctx, statuses...)

	proto := make([]*omv1.GetBootstrapRunResponse, 0, len(runs))
	for i := range runs {
		environment, cluster := s.bootstrapRunConfigLabels(runs[i].ID)
		proto = append(proto, bootstrapRunToProto(&runs[i], hostsByExecutor, environment, cluster))
	}
	return &omv1.ListBootstrapRunsResponse{Runs: proto}, nil
}

func (s *Service) bootstrapRunConfigLabels(runID string) (string, string) {
	if s.db == nil {
		return "", ""
	}
	config, err := models.FindOmBootstrapRunConfigByRunID(s.db.Querier, runID)
	if err != nil {
		if !errors.Is(err, models.ErrNotFound) {
			s.l.Warnf("bootstrap run %s: failed to load its environment/cluster: %s", runID, err)
		}
		return "", ""
	}
	return config.Environment, config.Cluster
}

// confirmMonitoringLookup fetches the inventory hosts only when a run has succeeded, as nothing else can be
// confirmed. It returns nil on failure, which reads every host as still unconfirmed.
func (s *Service) confirmMonitoringLookup(ctx context.Context, statuses ...string) map[string]extensionsHost {
	needed := slices.Contains(statuses, bootstrapRunSucceeded)
	if !needed {
		return nil
	}
	hosts, err := s.inventoryHostsByExecutor(ctx)
	if err != nil {
		s.l.Warnf("failed to confirm bootstrap monitoring against the inventory app: %s", err)
		return nil
	}
	return hosts
}

// confirmMonitoringStepName is a PMM-only step, never dispatched by om_bootstrap, confirming the inventory
// has noticed the registered service. It is appended to finalize_steps so it renders wherever steps do.
const confirmMonitoringStepName = "confirm_monitoring"

func confirmMonitoringStep(runStatus, executorHost string, port uint32, hostsByExecutor map[string]extensionsHost) *omv1.BootstrapStep {
	stepStatus := bootstrapStepPending
	if runStatus == bootstrapRunSucceeded {
		stepStatus = bootstrapStepRunning
		if hasBootstrapService(hostsByExecutor[executorHost], executorHost, port) {
			stepStatus = bootstrapStepSucceeded
		}
	}
	return &omv1.BootstrapStep{Name: confirmMonitoringStepName, Status: stepStatus}
}

func bootstrapRunToProto(run *extensionsBootstrapRun, hostsByExecutor map[string]extensionsHost, environment, cluster string) *omv1.GetBootstrapRunResponse {
	hosts := make([]*omv1.BootstrapHost, 0, len(run.Hosts))
	for _, host := range run.Hosts {
		finalizeSteps := bootstrapStepsToProto(host.FinalizeSteps)
		finalizeSteps = append(finalizeSteps,
			confirmMonitoringStep(run.Status, host.Host, run.Port, hostsByExecutor))
		hosts = append(hosts, &omv1.BootstrapHost{
			Host:          host.Host,
			Steps:         bootstrapStepsToProto(host.Steps),
			RollbackSteps: bootstrapStepsToProto(host.RollbackSteps),
			FinalizeSteps: finalizeSteps,
		})
	}
	return &omv1.GetBootstrapRunResponse{
		RunId:           run.ID,
		Status:          run.Status,
		Hosts:           hosts,
		RunSteps:        bootstrapStepsToProto(run.RunSteps),
		Error:           run.Error,
		ReplicaSetName:  run.ReplicaSetName,
		MongodbVersion:  run.MongoDBVersion,
		StartedAt:       timestamppb.New(run.StartedAt),
		FinishedAt:      optionalTimestamp(run.FinishedAt),
		Environment:     optional(environment),
		Cluster:         optional(cluster),
		CancelRequested: run.CancelRequested,
	}
}

func bootstrapStepsToProto(steps []extensionsBootstrapStep) []*omv1.BootstrapStep {
	proto := make([]*omv1.BootstrapStep, 0, len(steps))
	for _, step := range steps {
		proto = append(proto, &omv1.BootstrapStep{
			Name:         step.Name,
			Status:       step.Status,
			Detail:       step.Detail,
			AttemptCount: int32(step.AttemptCount), //nolint:gosec // an attempt count never approaches int32's range
		})
	}
	return proto
}

// GetInventoryConfig returns the inventory app's configuration.
func (s *Service) GetInventoryConfig(ctx context.Context, _ *omv1.GetInventoryConfigRequest) (*omv1.GetInventoryConfigResponse, error) {
	probe, err := s.inventoryProbe()
	if err != nil {
		return nil, err
	}

	settings := []extensionsSetting{}
	call := inventoryCall{method: http.MethodGet, path: "config"}
	err = probe.call(ctx, call, &settings)
	if err != nil {
		return nil, err
	}
	return &omv1.GetInventoryConfigResponse{Settings: inventorySettingsToProto(settings)}, nil
}

// maxConfigFields and maxConfigDepth bound an update batch's shape, which neither gRPC nor protojson
// bounds usefully; deep nesting would exceed the Python app's recursion limit.
const (
	maxConfigFields = 100
	maxConfigDepth  = 10
)

func validateConfigValues(values *structpb.Struct) error {
	fields := values.GetFields()
	switch {
	case len(fields) == 0:
		return status.Error(codes.InvalidArgument, "values must name at least one field to change")
	case len(fields) > maxConfigFields:
		return status.Errorf(codes.InvalidArgument,
			"values names %d fields, at most %d may change in one call", len(fields), maxConfigFields)
	}
	for key, value := range fields {
		if exceedsDepth(value, maxConfigDepth-1) {
			return status.Errorf(codes.InvalidArgument,
				"values.%s nests deeper than %d levels", key, maxConfigDepth)
		}
	}
	return nil
}

// exceedsDepth stops at limit rather than measuring the true depth, so its recursion stays bounded.
func exceedsDepth(value *structpb.Value, limit int) bool {
	if limit <= 0 {
		return value.GetStructValue() != nil || value.GetListValue() != nil
	}
	for _, field := range value.GetStructValue().GetFields() {
		if exceedsDepth(field, limit-1) {
			return true
		}
	}
	for _, element := range value.GetListValue().GetValues() {
		if exceedsDepth(element, limit-1) {
			return true
		}
	}
	return false
}

// UpdateInventoryConfig applies a batch of configuration changes, all or nothing, as validated by the app.
// It is PUT per PMM's API guidelines; the app's generic settings router uses PATCH.
func (s *Service) UpdateInventoryConfig(ctx context.Context, req *omv1.UpdateInventoryConfigRequest) (*omv1.UpdateInventoryConfigResponse, error) {
	probe, err := s.inventoryProbe()
	if err != nil {
		return nil, err
	}
	err = validateConfigValues(req.GetValues())
	if err != nil {
		return nil, err
	}

	applied := []extensionsSetting{}
	call := inventoryCall{method: http.MethodPatch, path: "config", body: req.GetValues().AsMap()}
	err = probe.call(ctx, call, &applied)
	if err != nil {
		return nil, err
	}

	// Read the configuration back rather than return the echo: the app echoes only the keys named in the
	// request, which misses children whose effective value a nested write changed.
	current := []extensionsSetting{}
	read := inventoryCall{method: http.MethodGet, path: "config"}
	err = probe.call(ctx, read, &current)
	if err != nil {
		// The write landed, so an error would invite a retry; fall back to the rows the app echoed.
		s.l.Warnf("inventory config updated, but reading it back failed: %s", err)
		return &omv1.UpdateInventoryConfigResponse{Settings: inventorySettingsToProto(applied)}, nil
	}
	return &omv1.UpdateInventoryConfigResponse{Settings: inventorySettingsToProto(current)}, nil
}

// DeleteInventoryConfigOverride reverts one field to its deployed value.
func (s *Service) DeleteInventoryConfigOverride(
	ctx context.Context, req *omv1.DeleteInventoryConfigOverrideRequest,
) (*omv1.DeleteInventoryConfigOverrideResponse, error) {
	probe, err := s.inventoryProbe()
	if err != nil {
		return nil, err
	}

	call := inventoryCall{method: http.MethodDelete, path: inventoryPath("config", req.GetKey())}
	err = probe.call(ctx, call, nil)
	if err != nil {
		return nil, err
	}
	return &omv1.DeleteInventoryConfigOverrideResponse{}, nil
}

func inventoryHostToProto(host extensionsHost, pmmAgentConnected bool, isPMMServer bool) *omv1.InventoryHost {
	executor := executorToProto(host.Observed)
	eligible, reasons, byDesign := automationEligibility(executor, pmmAgentConnected, isPMMServer, host)
	out := &omv1.InventoryHost{
		NodeId:                    host.NodeID,
		Name:                      host.Name,
		Address:                   optionalString(host.Address),
		ExecutorHost:              optionalString(host.ExecutorHost),
		Os:                        observedString(host.Observed, "os"),
		Kernel:                    observedString(host.Observed, "kernel"),
		Executor:                  executor,
		UnregisteredMongods:       unregisteredMongodsToProto(host.Observed),
		Observed:                  observedToStruct(host.Observed),
		Freshness:                 freshnessToProto(host.extensionsFreshness),
		Services:                  make([]*omv1.InventoryService, 0, len(host.Services)),
		PmmAgentConnected:         pmmAgentConnected,
		AutomationEligible:        eligible,
		AutomationBlockedReasons:  reasons,
		AutomationBlockedByDesign: byDesign,
		IsPmmServerNode:           isPMMServer,
	}
	for _, service := range host.Services {
		out.Services = append(out.Services, inventoryServiceToProto(service))
	}
	return out
}

// hostNotATargetReasons returns why Operations must never install onto this node. TriggerHostBootstrap
// enforces these too, since eligibility is advisory and can be stale.
func hostNotATargetReasons(isPMMServer bool, host extensionsHost) []string {
	if isPMMServer {
		return []string{
			"this is the node PMM Server itself runs on, which Operations never installs onto",
		}
	}
	if len(host.Services) > 0 {
		return []string{"a MongoDB service is already registered on this node"}
	}
	if unregisteredMongodCount(host.Observed) > 0 {
		return []string{"a scan found a mongod running here that PMM has no service for"}
	}
	// Installed but not running: the install's pre_check misses a binary outside sudo's secure_path,
	// while the scan finds it wherever PATH does.
	if version := observedString(host.Observed, "installed_version"); version != nil {
		return []string{fmt.Sprintf("a scan found MongoDB %s already installed on this node", *version)}
	}
	return nil
}

// automationEligibility decides whether OM automation can run on a host and names every unmet condition.
// The last result reports whether the host is blocked by design rather than by a fault.
func automationEligibility(
	executor *omv1.InventoryExecutor,
	pmmAgentConnected bool,
	isPMMServer bool,
	host extensionsHost,
) (bool, []string, bool) {
	reasons := hostNotATargetReasons(isPMMServer, host)
	byDesign := len(reasons) > 0
	if isPMMServer {
		// The PMM Server reason stands alone: its image reports os_id "ol", and an unsupported-OS reason
		// would invite reinstalling the machine PMM runs on.
		return false, reasons, true
	}
	if !pmmAgentConnected {
		reasons = append(reasons, "PMM Client is not installed or not connected")
	}
	// These reasons are shown on the Nodes page, so they do not name the scheduler.
	if executor == nil || !executor.GetReachable() {
		reasons = append(reasons, "this node has no automation agent that answers")
	} else if !executor.GetDriverHealthy() {
		reasons = append(reasons, "this node's automation agent cannot run jobs")
	}
	switch osID, _ := host.Observed["os_id"].(string); {
	case osID == "":
		reasons = append(reasons, "no scan has reported this node's operating system yet")
	case !supportedBootstrapOSIDs[osID]:
		// Nothing is wrong with the node; Operations just does not install onto its distribution.
		byDesign = true
		reasons = append(reasons, fmt.Sprintf(
			"this node runs %s, which Operations cannot install onto (supported: %s)",
			osID, supportedBootstrapOSNames(),
		))
	}
	if host.Address == nil || *host.Address == "" {
		reasons = append(reasons, "PMM has no address for this node")
	}
	eligible := len(reasons) == 0
	return eligible, reasons, !eligible && byDesign
}

func unregisteredMongodCount(observed map[string]any) int {
	entries, ok := observed["unregistered_mongods"].([]any)
	if !ok {
		return 0
	}
	return len(entries)
}

func inventoryServiceToProto(service extensionsService) *omv1.InventoryService {
	return &omv1.InventoryService{
		ServiceId:        service.ServiceID,
		NodeId:           service.NodeID,
		Name:             service.Name,
		Port:             optionalInt32(service.Port),
		Role:             optionalString(service.Role),
		InstalledVersion: observedString(service.Observed, "installed_version"),
		RunningVersion:   observedString(service.Observed, "version"),
		ConfigPath:       observedString(service.Observed, "config_path"),
		Argv:             observedString(service.Observed, "argv"),
		ProbeStatus:      observedString(service.Observed, "probe_status"),
		ServerRunning:    observedBool(service.Observed, "server_running"),
		UptimeSeconds:    observedDouble(service.Observed, "uptime_seconds"),
		ReplicationSet:   observedString(service.Observed, "replication_set"),
		Observed:         observedToStruct(service.Observed),
		Freshness:        freshnessToProto(service.extensionsFreshness),
	}
}

func inventoryRunToProto(run extensionsRun) *omv1.InventoryRun {
	return &omv1.InventoryRun{
		RunId:     run.RunID,
		Status:    extensionsRunStatusToProto(run.Status),
		StartTime: optionalTimestamp(run.StartedAt),
		EndTime:   optionalTimestamp(run.FinishedAt),
		Counts: &omv1.InventoryRunCounts{
			TotalServices:    run.Counts.ServicesTotal,
			ResolvedServices: run.Counts.ServicesResolved,
			OrphanedServices: run.Counts.ServicesOrphaned,
			AnsweredServices: run.Counts.ServicesAnswered,
			TotalHosts:       run.Counts.HostsTotal,
			ProbeableHosts:   run.Counts.HostsProbeable,
			AnsweredHosts:    run.Counts.HostsAnswered,
			FinishedHosts:    run.Counts.HostsFinished,
		},
		Scope:        run.Scope,
		Error:        optionalString(run.Error),
		FailingNodes: failingNodesToProto(run.FailingNodes),
	}
}

// failingNodesToProto keeps PMM Extensions' order, which is by name.
func failingNodesToProto(nodes []extensionsRunFailingNode) []*omv1.InventoryRunFailingNode {
	out := make([]*omv1.InventoryRunFailingNode, 0, len(nodes))
	for _, node := range nodes {
		out = append(out, &omv1.InventoryRunFailingNode{NodeId: node.NodeID, Name: pointer.GetString(node.Name)})
	}
	return out
}

func inventoryRunEntitiesToProto(nodes []extensionsRunNode) []*omv1.InventoryRunEntity {
	entities := make([]*omv1.InventoryRunEntity, 0, len(nodes))
	for _, node := range nodes {
		services := make([]*omv1.InventoryRunEntityService, 0, len(node.Services))
		for _, service := range node.Services {
			services = append(services, &omv1.InventoryRunEntityService{
				ServiceId:   optionalString(service.ServiceID),
				ServiceName: optionalString(service.ServiceName),
				Answered:    service.Answered,
				Error:       optionalString(service.Error),
			})
		}
		entities = append(entities, &omv1.InventoryRunEntity{
			NodeId:          node.NodeID,
			HostName:        optionalString(node.HostName),
			ExecutorHost:    optionalString(node.ExecutorHost),
			Resolution:      extensionsResolutionToProto(node.Resolution),
			Answered:        node.Answered,
			DurationSeconds: optionalDouble(node.Duration),
			TaskHistoryId:   node.TaskHistoryID,
			Error:           optionalString(node.Error),
			Services:        services,
		})
	}
	return entities
}

func inventorySettingsToProto(settings []extensionsSetting) []*omv1.InventorySetting {
	out := make([]*omv1.InventorySetting, 0, len(settings))
	for _, setting := range settings {
		out = append(out, &omv1.InventorySetting{
			Key:          setting.Key,
			Value:        anyToValue(setting.Value),
			DefaultValue: anyToValue(setting.DefaultValue),
			Type:         setting.Type,
			Reload:       extensionsReloadToProto(setting.Reload),
			HasOverride:  setting.HasOverride,
			IsAdvanced:   setting.IsAdvanced,
			Description:  optionalString(setting.Description),
		})
	}
	return out
}

func freshnessToProto(f extensionsFreshness) *omv1.InventoryFreshness {
	return &omv1.InventoryFreshness{
		FirstSeenAt:         optionalTimestamp(f.FirstSeenAt),
		LastAttemptAt:       optionalTimestamp(f.LastAttemptAt),
		LastSuccessAt:       optionalTimestamp(f.LastSuccessAt),
		FailingSince:        optionalTimestamp(f.FailingSince),
		ConsecutiveFailures: clampInt32(f.ConsecutiveFailures),
		LastError:           optionalString(f.LastError),
		LastErrorCode:       optionalString(f.LastErrorCode),
		LastRunId:           optionalString(f.LastRunID),
	}
}

// executorToProto returns nil when the app reported no executor block, which differs from all flags false.
func executorToProto(observed map[string]any) *omv1.InventoryExecutor {
	nested, ok := observed["executor"].(map[string]any)
	if !ok {
		return nil
	}
	out := &omv1.InventoryExecutor{}
	if value, ok := nested["registered"].(bool); ok {
		out.Registered = value
	}
	if value, ok := nested["reachable"].(bool); ok {
		out.Reachable = value
	}
	if value, ok := nested["driver_healthy"].(bool); ok {
		out.DriverHealthy = value
	}
	if value, ok := nested["detail"].(string); ok && value != "" {
		out.Detail = &value
	}
	return out
}

func unregisteredMongodsToProto(observed map[string]any) []*omv1.UnregisteredMongod {
	entries, ok := observed["unregistered_mongods"].([]any)
	if !ok {
		return nil
	}
	out := make([]*omv1.UnregisteredMongod, 0, len(entries))
	for _, entry := range entries {
		fields, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, &omv1.UnregisteredMongod{
			Port:       observedInt32(fields, "port"),
			ConfigPath: observedString(fields, "config_path"),
			Argv:       observedString(fields, "argv"),
			Program:    observedString(fields, "program"),
			Pid:        observedInt32(fields, "pid"),
		})
	}
	return out
}

// observedToStruct passes the observed document through untyped, so a newly collected attribute needs no
// proto change. The collected_at field is dropped: the freshness block carries the same instant.
func observedToStruct(observed map[string]any) *structpb.Struct {
	if len(observed) == 0 {
		return nil
	}
	filtered := make(map[string]any, len(observed))
	for key, value := range observed {
		if key == observedCollectedAt {
			continue
		}
		filtered[key] = value
	}
	encoded, err := structpb.NewStruct(filtered)
	if err != nil {
		// Losing the detail panel beats failing the whole listing over one unrepresentable value.
		return nil
	}
	return encoded
}

func anyToValue(value any) *structpb.Value {
	encoded, err := structpb.NewValue(value)
	if err != nil {
		return nil
	}
	return encoded
}

func observedString(observed map[string]any, key string) *string {
	value, ok := observed[key].(string)
	if !ok || value == "" {
		return nil
	}
	return &value
}

// clampInt32 saturates rather than wraps, so a huge count never reads as negative.
func clampInt32(value int) int32 {
	switch {
	case value > math.MaxInt32:
		return math.MaxInt32
	case value < math.MinInt32:
		return math.MinInt32
	}
	return int32(value)
}

func observedBool(observed map[string]any, key string) *bool {
	value, ok := observed[key].(bool)
	if !ok {
		return nil
	}
	return &value
}

func observedDouble(observed map[string]any, key string) *float64 {
	value, ok := observed[key].(float64)
	if !ok {
		return nil
	}
	return &value
}

// observedInt32 drops a value that does not fit int32 rather than wrap it into a plausible port or PID.
func observedInt32(observed map[string]any, key string) *int32 {
	value, ok := observed[key].(float64)
	if !ok {
		return nil
	}
	if math.IsNaN(value) || value < math.MinInt32 || value > math.MaxInt32 {
		return nil
	}
	narrowed := int32(value)
	return &narrowed
}

func optionalString(value *string) *string {
	if value == nil || *value == "" {
		return nil
	}
	return value
}

func optionalInt32(value *int32) *int32 {
	if value == nil {
		return nil
	}
	return value
}

func optionalTimestamp(value *time.Time) *timestamppb.Timestamp {
	if value == nil {
		return nil
	}
	return timestamppb.New(*value)
}

func parseSepTime(stamp string) *timestamppb.Timestamp {
	parsed, err := time.Parse(time.RFC3339, stamp)
	if err != nil {
		return nil
	}
	return timestamppb.New(parsed)
}

func extensionsRunStatusToProto(status string) omv1.RunStatus {
	switch status {
	case "running":
		return omv1.RunStatus_RUN_STATUS_RUNNING
	case "success":
		return omv1.RunStatus_RUN_STATUS_SUCCESS
	case "partial":
		return omv1.RunStatus_RUN_STATUS_PARTIAL
	case "failed":
		return omv1.RunStatus_RUN_STATUS_FAILED
	case "skipped":
		return omv1.RunStatus_RUN_STATUS_SKIPPED
	default:
		return omv1.RunStatus_RUN_STATUS_UNSPECIFIED
	}
}

func extensionsResolutionToProto(resolution string) omv1.ExecutorResolution {
	switch resolution {
	case "name":
		return omv1.ExecutorResolution_EXECUTOR_RESOLUTION_NAME
	case "address":
		return omv1.ExecutorResolution_EXECUTOR_RESOLUTION_ADDRESS
	case "orphaned":
		return omv1.ExecutorResolution_EXECUTOR_RESOLUTION_ORPHANED
	default:
		return omv1.ExecutorResolution_EXECUTOR_RESOLUTION_UNSPECIFIED
	}
}

func extensionsReloadToProto(reload string) omv1.SettingReload {
	switch reload {
	case "hot":
		return omv1.SettingReload_SETTING_RELOAD_HOT
	case "nested_only":
		return omv1.SettingReload_SETTING_RELOAD_NESTED_ONLY
	case "not_overridable":
		return omv1.SettingReload_SETTING_RELOAD_NOT_OVERRIDABLE
	default:
		return omv1.SettingReload_SETTING_RELOAD_UNSPECIFIED
	}
}
