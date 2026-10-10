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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	omv1 "github.com/percona/pmm/api/om/v1"
	"github.com/percona/pmm/managed/models"
)

// The /v1/om/inventory/* handlers: PMM Extensions' estate, served through PMM.
//
// Two different things are called a "run" one path segment apart, so they are mounted
// apart on purpose. /v1/om/topology/runs is PMM's *own* collection pass -- inventory
// plus VictoriaMetrics, a tenth of a second, never touches a host. /v1/om/inventory/runs
// dispatches a Nomad job per host and takes tens of seconds. Nothing but the path tells
// a caller which one they are about to start, which is why the surface does.
//
// The projection below is deliberately dull. Everything interesting about the estate --
// what counts as failing, when a row is stale, which of three ways an executor is
// unusable -- was decided in the app, and re-deciding any of it here would give a reader
// two answers to the same question.

// defaultInventoryRunLimit is what a caller who passes no limit gets, and
// maxInventoryRunLimit is the most one can ask for. The ceiling exists because the value
// is forwarded to PMM Extensions verbatim: without it a caller could ask the inventory app for an
// unbounded page and wait on it through this proxy. Matches the proto's own
// ListInventoryRunsRequest.limit validation (lte: 100) and PMM Extensions' le=100 -- any of the
// three drifting from the others makes one of them dead code, since the request has to
// clear all of them to reach the database.
const (
	defaultInventoryRunLimit = 20
	maxInventoryRunLimit     = 100
)

// inventoryProbe returns the configured PMM Extensions client, or an error saying it is not.
//
// Reported as FailedPrecondition rather than Unimplemented or NotFound: the endpoints
// exist and work, the deployment has simply not been told where PMM Extensions is, and that is an
// operator's action rather than a missing feature.
//
// Returns extensionsApp, not *probeSource: the inventory handlers below only ever need to call
// against om_inventory, never any of probeSource's own factSource behaviour, and holding
// the narrower handle is what keeps this file from knowing probeSource exists at all.
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

// pmmAgentConnectedByNode returns, for every node with a pmm-agent, whether that
// agent is currently connected -- one query and one registry check per node, not one
// per host in the estate.
//
// Absence from the returned map (rather than a false-valued entry) is the same "not
// connected" answer callers here read either way, since a Go map's missing key
// already zero-values to false; kept as a genuinely sparse map only because building
// it is naturally that shape, not because callers need to distinguish a missing agent
// from a disconnected one.
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

// pmmServerNodeIDs names the node PMM itself runs on.
//
// A set rather than a bool per node, because the question asked of it is "is this
// one the server", and every other node is absent rather than false. PMM's own
// inventory already carries the flag, so this is a read rather than a heuristic on
// the address -- 127.0.0.1 is a property of how the server was registered, not of
// what it is.
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
//
// Not suppression: an entity PMM still knows about returns on the next refresh. It is
// for rows left behind when a node was replaced -- restarting a pmm-agent runs
// `setup --force`, which mints a new node ID, so OM gains a row and keeps the old one.
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
//
// Returns as soon as the refresh is accepted. Conflict is judged per host on the app's
// side, so a scoped refresh is not refused merely because the scheduled sweep happens to
// be running -- only because something else already holds one of the same hosts.
func (s *Service) TriggerInventoryRefresh(ctx context.Context, req *omv1.TriggerInventoryRefreshRequest) (*omv1.TriggerInventoryRefreshResponse, error) {
	probe, err := s.inventoryProbe()
	if err != nil {
		return nil, err
	}

	// Node IDs pass through untranslated: PMM's node ID is also the app's key, which is
	// the whole point of keying the estate on it.
	//
	// Materialised as an empty slice rather than passed through nil. Go marshals a nil
	// slice as JSON `null`, and the app types the field `list[str]` with an empty
	// default -- so `null` is a validation failure, not "no scope". Left as-is, every
	// full-estate refresh through this proxy would have answered 422 while a scoped one
	// worked, which is the shape of bug that gets diagnosed as "the trigger is broken
	// sometimes".
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

// validateMemberConfigs checks one request's per-member settings against the
// hosts it names and against what rs.initiate() will accept, so a set it would
// refuse fails here rather than several steps into a run -- minutes later, with
// mongod already installed everywhere.
//
// Split out of TriggerHostBootstrap, its only caller, to keep that function's
// cognitive complexity within the linter's limit, the same reason
// resolveBootstrapHostOSID sits beside it.
func validateMemberConfigs(nodeIDs []string, memberConfigs map[string]*omv1.BootstrapMemberConfig) error {
	nodeIDSet := make(map[string]bool, len(nodeIDs))
	for _, nodeID := range nodeIDs {
		nodeIDSet[nodeID] = true
	}
	// Accumulated, not returned on the first: a three-member plan with two
	// misconfigured members used to report one of them. Sorted, because ranging a
	// map gives a different order every run and an error message that reshuffles
	// itself is one nobody can diff against the last attempt.
	var violations []string
	for _, nodeID := range slices.Sorted(maps.Keys(memberConfigs)) {
		member := memberConfigs[nodeID]
		if !nodeIDSet[nodeID] {
			violations = append(violations, fmt.Sprintf(
				"member_configs names host %s, which is not in node_ids", nodeID,
			))
			// Nothing else about a host outside the selection is worth checking.
			continue
		}
		// MongoDB's own rs.initiate() rule: a delayed member cannot vote or be
		// eligible for primary -- rejected here rather than left for PMM Extensions to
		// discover only once rs.initiate actually runs, minutes later. Leaving
		// priority or votes unset is the same mistake as setting them wrong,
		// since unset means MongoDB's defaults of 1 and on.
		if member.GetDelaySecs() > 0 && (member.Priority == nil || *member.Priority != 0 || member.Votes == nil || *member.Votes) {
			violations = append(violations, fmt.Sprintf(
				"host %s: a delayed member (delay_secs > 0) must also set priority 0 and votes off", nodeID,
			))
		}
		// The same shape of rule, for hidden members. rs.initiate() refuses a hidden
		// member that can still be elected, because a primary hidden from clients is
		// a set with no reachable primary.
		//
		// Enforced here although PMM-15661 adds it in the install wizard and scopes
		// the backend out: without this the browser is the only thing standing between
		// a direct API call and a run that installs mongod on every host and then
		// fails minutes later inside rs.initiate. Unset priority counts as broken for
		// the same reason it does above -- unset means MongoDB's default of 1.
		if member.GetHidden() && (member.Priority == nil || *member.Priority > 0) {
			violations = append(violations, fmt.Sprintf(
				"host %s: a hidden member must also set priority 0, since MongoDB will not elect a member that clients cannot see", nodeID,
			))
		}
	}

	// A replica set needs a member that can vote and one that can be elected,
	// and rs.initiate() is where a set with neither fails -- minutes into a run,
	// after every host already has mongod installed and started. Counted over
	// node_ids rather than member_configs because a host named nowhere in it
	// keeps MongoDB's defaults and therefore does both.
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

// runIgnoredSettings names every run setting om_bootstrap did not apply, or
// "" when the accepted run matches what was asked for.
//
// Raised in review: these fields are new on the wire, and PMM Extensions' own
// TriggerRunRequest is a plain pydantic model, which ignores fields it does not
// know rather than rejecting them. Against an om_bootstrap older than
// percona/PMM Extensions#1534 the run would be accepted and then come up on PMM Extensions' own
// defaults -- mongod on a different port, and a member meant to be hidden,
// non-voting and delayed joining as an ordinary voting one -- with nothing
// saying so. PMM Extensions echoes every one of these back on the accepted run (its
// RunResponse), so the answer is in hand the moment the run is created; an
// older PMM Extensions simply omits them, which reads here as the zero value and so as a
// mismatch.
//
// Reports the first difference rather than all of them: any single one means
// the app is too old, and the caller's next step is the same either way. Only
// what was actually asked for is compared -- a field PMM left out is PMM Extensions' to
// default, and the value it chose is not a disagreement.
func runIgnoredSettings(planned extensionsTriggerBootstrapRunRequest, accepted *extensionsBootstrapRun) string {
	// Every setting, not the first. A PMM Extensions too old for these fields
	// ignores all of them at once, so naming one made the gap look like a single
	// mis-set value rather than what it is: this PMM talking to an older side-car.
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
	// Named once however many members disagree: they are one wire field, and a reader
	// cannot act on them per host anyway.
	for host, member := range planned.MemberConfigs {
		got, ok := accepted.MemberConfigs[host]
		if !ok || !sameMemberConfig(got, member) {
			ignored = append(ignored, "the per-member replica-set settings")
			break
		}
	}
	return strings.Join(ignored, ", ")
}

// sameMemberConfig compares one host's settings as asked for against as accepted.
//
// An unset priority or votes on the request side is MongoDB's own default, which
// is what PMM Extensions fills in and echoes back, so the two compare equal rather than
// reading as a mismatch on every run that leaves them out.
func sameMemberConfig(accepted, planned extensionsMemberConfig) bool {
	if pointer.GetUint32(planned.Priority) != pointer.GetUint32(accepted.Priority) && planned.Priority != nil {
		return false
	}
	if planned.Votes != nil && pointer.GetBool(planned.Votes) != pointer.GetBool(accepted.Votes) {
		return false
	}
	// Compared only when asked for, like the two above: om_bootstrap echoes None for a
	// member that named no address, and a run that never asked for one must not read
	// as a mismatch. Without this an om_bootstrap too old for the field would accept
	// the request, ignore it, and bring mongod up on the run-level address -- which is
	// 0.0.0.0, the exact default the per-member value exists to avoid.
	if planned.BindIP != nil && pointer.GetString(planned.BindIP) != pointer.GetString(accepted.BindIP) {
		return false
	}
	return planned.Hidden == accepted.Hidden && planned.DelaySecs == accepted.DelaySecs
}

// abandonMisconfiguredRun cancels a run PMM has just decided it cannot use.
//
// Best-effort and logged rather than returned: the caller is already being told
// why its request failed, and a cancel that does not land leaves a run visible
// on the Automations page rather than anything worse. A PMM Extensions too old for the
// settings above is also too old for :cancel, which 404s -- that is the same
// "your PMM Extensions is older than this PMM" answer, so it is not worth reporting twice.
func (s *Service) abandonMisconfiguredRun(ctx context.Context, runID string) {
	_, err := s.bootstrap.cancelRun(ctx, runID)
	if err != nil {
		s.l.Warnf("bootstrap run %s: failed to cancel a run PMM Extensions would not configure: %s", runID, err)
	}
}

// executorUnusable says why a payload cannot be dispatched to this host right now,
// or "" when nothing is known to be wrong.
//
// Reads the same observed.executor sub-document PMM Extensions' own _executor_usable does
// (om_inventory's api_routes.py), so no second call is needed: the host was already
// fetched to read its OS. Having executor_host set is not the same answer: PMM Extensions sets
// that the moment any known executor matches the host, usable or not.
//
// Checked before a run is created because of what the alternative looks like, raised
// on the PMM Extensions side of this work: an unreachable Nomad client surfaces only once
// pre_check -- itself dispatched through Nomad -- fails a few seconds later, by which
// time the run exists and the UI is showing it as in progress.
//
// Only an explicit false rejects. A host whose sub-document is missing entirely is
// left to PMM Extensions, which is the older behaviour and keeps a PMM talking to a side-car that
// does not write this yet able to bootstrap at all; PMM Extensions' own listing filter is
// stricter and reads absence as not eligible, so such a host will not be offered in
// the UI either way.
func executorUnusable(host extensionsHost) string {
	executor, ok := host.Observed["executor"].(map[string]any)
	if !ok {
		return ""
	}

	// Worded without naming the scheduler: this string reaches the install wizard as
	// a gRPC error message, so it is product copy, and PMM-15623 set out to keep
	// "Nomad" out of what a user reads.
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

// supportedBootstrapOSIDs mirrors PMM Extensions' own om_bootstrap.strategy.OperatingSystem
// enum (app/sep/apps/om_bootstrap/strategy.py) -- there is no Go-side equivalent
// type, since PMM otherwise treats os_id as an opaque string sourced from
// om_inventory's probe and forwarded to PMM Extensions verbatim. Checked here purely so an
// unsupported OS fails at trigger time with a clear reason instead of a run
// that starts, dispatches a step, and only then fails on PMM Extensions' own
// _require_package_manager -- PMM Extensions remains the actual source of truth, so a
// third OS lands here only after (never instead of) that enum gaining it.
var supportedBootstrapOSIDs = map[string]bool{"ubuntu": true, "rocky": true}

// supportedBootstrapOSNames lists them for a message, sorted so the sentence a user
// reads does not change between two runs over the same map.
func supportedBootstrapOSNames() string {
	names := make([]string, 0, len(supportedBootstrapOSIDs))
	for id := range supportedBootstrapOSIDs {
		names = append(names, id)
	}
	slices.Sort(names)
	return strings.Join(names, ", ")
}

// refuseNodesNotReadyToInstall rejects the request when any selected node cannot be
// installed onto, and returns the one OS the run will use.
//
// Every problem from every node, in one error. It used to return on the first:
// `resolveBootstrapHostOSID` was called once per node from inside the trigger's loop,
// so three nodes with no OS produced one message about one of them, and the user found
// the second only by fixing the first and running the trigger again. The same was true
// of the executor checks beside it.
//
// The two gates are separate on purpose. A node that cannot be installed onto is that
// node's problem; a selection whose nodes disagree about their OS is a problem with the
// selection, and naming one node for it would blame a machine that is fine.
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
		// Named by the groups rather than by "this one disagrees with that one": with
		// three nodes there is no single odd one out, and the reader has to decide
		// which two to keep.
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

// hostNotReadyReasons returns why an install on this node would fail, as opposed to
// why it must not be attempted at all -- that is hostNotATargetReasons.
//
// The advice matters as much as the reason. "wait for its next inventory probe and try
// again" was wrong: the node's scans had been failing for 160 runs,
// so waiting was never going to help. None of these say wait.
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

// hostDisplayName is what to call this node in a message a person reads.
//
// The node id is a UUID PMM minted; it is not what anyone's inventory, runbook or
// ticket calls the machine, so an error built from it is unactionable. Falls back to
// the id only when PMM has no name, where it is the one identifier that exists.
func hostDisplayName(host extensionsHost, nodeID string) string {
	if host.Name != "" {
		return host.Name
	}
	return nodeID
}

// refuseNodesThatAreNotTargets rejects the whole request when any node must never be
// installed onto, naming every one of them.
//
// Every one, not the first: a user fixing three nodes should not have to run the
// trigger three times to discover there were three (PMM-15664 task 3 applies the same
// rule to the rest of this file's errors). Named rather than identified by the node id
// the caller sent: an error that quotes a bare UUID is unactionable, because the UUID
// is not what anyone's inventory, runbook or ticket calls the machine.
//
// A node PMM has no name for falls back to its id, which is still better than nothing
// and is the only identifier that exists in that case.
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
	// FailedPrecondition, not InvalidArgument: the request names real nodes and is
	// well formed. What is wrong is the state of the estate, which is also why the
	// message says what to do about it rather than only what is wrong.
	return status.Errorf(codes.FailedPrecondition,
		"Operations does not install onto %d of the selected node(s) -- %s. Remove them from the selection.",
		len(blocked), strings.Join(blocked, "; "))
}

// TriggerHostBootstrap plans installing MongoDB on one or three hosts and
// initializing them as one replica set.
//
// PMM-15347 PoC only -- see the RPC's own proto comment and PMM-15347/plan.md for
// scope. Reads every host's own os_id from om_inventory (a general inventory
// fact, PMM-15326: Surface the host's machine-readable OS id) rather than
// asking the caller for it, then hands off to PMM Extensions' om_bootstrap app -- not
// om_inventory, which stays read-only by design -- which does the real
// planning (install_method fixed to "packages", the only strategy implemented
// yet). PMM's own HA-leader-only stepper (stepper.go) drives the returned run
// forward from here; this handler's job ends at planning it.
//
// The om_bootstrap app's own "host" identity is the Nomad *executor* host (the name
// its own dispatch route passes straight through as the Tasks API's
// target -- see dispatch.go's own doc comment on PMM Extensions' side), not PMM's node
// id: the two are different strings for the same machine (a node id is a
// UUID PMM minted; the executor host is whatever name the Nomad client
// registered under, e.g. "pmm-client-node00"), and Nomad only knows the
// latter. GetBootstrapRun passes the executor host straight through --
// arguably more readable for a progress display than a bare UUID -- and only
// registerBootstrapHost (nodeIDForExecutorHost) ever needs the node id back,
// since PMM's own inventory is keyed on that instead.
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

	// Fetched before anything is planned so every node can be judged together: a
	// three-node request with two bad nodes used to report one of them, and the user
	// found the second only by fixing the first and trying again.
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

	// The safety gate, and the reason this is not left to the UI. automation_eligible
	// is advisory: it is computed for a list request, can be minutes stale, and a
	// direct API call never reads it at all. Installing MongoDB onto the PMM Server,
	// or over a replica-set member that already exists, is the most damaging thing
	// Operations can do, so it is refused here as well as offered nowhere.
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
		Hosts:          executorHosts,
		InstallMethod:  "packages",
		OS:             osID,
		MongoDBVersion: req.GetMongodbVersion(),
		ReplicaSetName: req.GetReplicaSetName(),
		DataPath:       req.GetDataPath(),
		LogPath:        req.GetLogPath(),
		Port:           req.GetPort(),
		BindIP:         req.GetBindIp(),
		MemberConfigs:  memberConfigs,
	}
	run, err := s.bootstrap.triggerRun(ctx, planned)
	if err != nil {
		return nil, err
	}

	ignored := runIgnoredSettings(planned, run)
	if ignored != "" {
		s.abandonMisconfiguredRun(ctx, run.ID)
		return nil, status.Errorf(codes.FailedPrecondition,
			"PMM Extensions accepted the run but did not apply %s, so it would come up on its own defaults "+
				"instead of the settings you chose; its om_bootstrap app is older than this PMM. "+
				"The run has been cancelled", ignored)
	}

	if environment, cluster := req.GetEnvironment(), req.GetCluster(); environment != "" || cluster != "" {
		config := &models.OmBootstrapRunConfig{RunID: run.ID, Environment: environment, Cluster: cluster}
		err = models.CreateOmBootstrapRunConfig(s.db.Querier, config)
		if err != nil {
			// The bootstrap itself is already under way on PMM Extensions' side; failing this
			// request now would report an error for a run that is, in fact, running --
			// worse than registering it unlabelled, which is exactly what happens
			// today for every run triggered before this field existed.
			s.l.Warnf("bootstrap run %s: failed to persist environment/cluster: %s", run.ID, err)
		}
	}

	return &omv1.TriggerHostBootstrapResponse{RunId: run.ID}, nil
}

// inventoryHostsByExecutor returns every host the inventory app currently has a
// row for, keyed by its Nomad executor host name -- the identity a bootstrap
// run's progress is keyed on (TriggerHostBootstrap's own doc comment), not the
// node id PMM's own inventory needs. Shared by nodeIDForExecutorHost and
// confirmMonitoringLookup, the two places that need the estate the other way
// round from how ListInventoryHosts reads it.
//
// Fetches the whole estate rather than a filtered query: om_inventory's own
// GET /hosts has no "find by executor_host" filter, and the estate size this
// phase targets (a handful of hosts in one replica set) makes one full fetch
// no real cost -- see ListInventoryHosts's own similar fetch-then-filter
// shape. Whole means every page: GET /hosts answers the paginated envelope
// (PMM-15326: "Bound the estate listings"), so this goes through
// fetchAllPages like the other two readers of that endpoint rather than
// decoding a bare array. A host with no executor at all (never dispatched
// an eligibility probe) is silently dropped rather than keyed on empty
// string.
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

// inventorySyncAppModule is where PMM Extensions mounts the app PMMSyncer
// belongs to -- a different module from probeAppModule, reached through the same
// client and the same `/api/apps/<module>` convention.
const inventorySyncAppModule = "inventory"

// pmmSyncerName is the one syncer triggerInventorySync asks for by name. Syncing
// only PMMSyncer rather than every configured syncer keeps an unrelated source's
// cost off a path that runs when a bootstrap finishes.
const pmmSyncerName = "app.extensions.sync.syncers.pmm.PMMSyncer"

// inventorySyncDebounce is how often triggerInventorySync will actually ask.
//
// The sync is global, not per-run, so one pull serves every run that finished in
// the same window -- and completeSucceededRun reaches this on every 15s tick for
// as long as refreshRetryWindowOpen holds a run, which would otherwise be twenty
// full inventory pulls for one bootstrap.
//
// A run registering *during* an in-flight pull is not served by it: the pull
// snapshots PMM's inventory when it starts. That run waits out the rest of the
// window and is synced by the first tick that gets through, so the debounce
// costs it at most one window -- well inside bootstrapInventoryRefreshWindow,
// and still a different order of magnitude from the app's own schedule.
const inventorySyncDebounce = 45 * time.Second

// inventoryRefreshDebounce is how often triggerScopedInventoryRefresh will
// re-probe the same node.
//
// Shorter than inventorySyncDebounce, because the refresh exists to observe a
// sync that completed since the last attempt: the pull answers 202 and fills the
// app's copy in the background, so the probe fired straight after it is the one
// that finds nothing. Longer than the 15s tick, because nothing the probe reads
// can change faster than the sync feeding it -- without this, one held run costs
// twenty probe runs where ten do the same job.
const inventoryRefreshDebounce = 30 * time.Second

// triggerInventorySync asks PMM Extensions to pull PMM's current services into
// its own inventory now, instead of waiting for PMMSyncer's schedule.
//
// This is the step that was missing between registering a bootstrapped host and
// re-probing it. The registerBootstrapHost call creates the service in *PMM's*
// inventory; the estate's probe resolves a host's services from *PMM Extensions'*
// copy, which only PMMSyncer fills. So triggerScopedInventoryRefresh on its own
// re-probes a host whose service the app has not heard of yet, and finds
// nothing -- by construction, since it fires the moment registration completes.
//
// Observed on a real deployment: two runs finished at 08:40:31 and 08:41:31, each
// fired its scoped refresh at exactly that second, and both found no service.
// The confirm_monitoring step stayed "running" until the app's own 10-minute
// sweep at 08:50:58, so a pair of entirely successful bootstraps read as hung
// for nine minutes.
//
// Best-effort, and nothing for the caller to decide on: a failure leaves
// confirm_monitoring to the app's own schedule, which is exactly where it stood
// before this existed. The caller holds the run on the estate's answer instead
// -- see completeSucceededRun's doc comment.
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
		// Hand the window back. The debounce is armed before the request so two
		// runs completing in one tick cannot both ask, but the pull it was armed
		// for never started -- holding the next 45s of ticks off on its behalf
		// would spend a sixth of the retry window standing down for nothing.
		s.syncRequestedMu.Lock()
		if s.syncRequested.Equal(stamped) {
			s.syncRequested = previous
		}
		s.syncRequestedMu.Unlock()
		s.l.Warnf("failed to trigger an inventory sync after a bootstrap: %s", err)
	}
}

// postInventorySync asks the inventory app for one pull by syncer name, falling
// back to every configured syncer when the app does not recognise the name.
//
// The pmmSyncerName constant is a Python dotted path sitting on the far side of
// a release boundary: PMM Extensions renamed the package holding it (app.sep ->
// app.extensions), and a side-car older than that rename answers 400 -- the
// route rejects an unknown syncer rather than no-opping. Without the fallback
// that 400 costs one Warnf and silently restores the nine-minute hang this whole
// path exists to remove. With it, the mismatch costs the unrelated syncers' work
// on a path that runs when a bootstrap finishes, which is the cheaper of the two.
func (s *Service) postInventorySync(ctx context.Context, probe extensionsApp) error {
	app := probe.client.app(inventorySyncAppModule)
	// Trailing slash: the app answers 307 without it, and a redirect is refused
	// rather than followed so the bearer is never replayed (see refuseRedirect).
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
	// An absent syncer runs every configured one in declaration order -- the
	// route's own documented behaviour, not a guess.
	call.body = map[string]any{}
	return app.call(ctx, call, nil)
}

// triggerScopedInventoryRefresh asks the inventory app to re-probe exactly
// nodeIDs now, rather than leaving confirmMonitoringStep to wait out however
// long the app's own schedule takes to get there on its own -- called by
// completeSucceededRun once a run's hosts are registered.
//
// Best-effort, and debounced per node by inventoryRefreshDebounce: the caller
// reaches this on every tick for as long as the run is held, and probing a node
// again before the sync that feeds the probe can have delivered anything is
// work for no new answer.
//
// A 409 (Aborted here -- see extensionsStatusError) means some other refresh already
// holds one of these hosts, and is expected rather than broken: the app judges
// conflict per host, and the estate sweep it runs on its own schedule holds
// every host it is walking. That makes a refusal likely exactly when a run
// finishes, not rare -- a sweep occupies a sizeable fraction of every schedule
// period -- so it stays un-logged, and starts the debounce window just as
// acceptance does: either way the re-probe this tick wanted is under way. The
// caller holds the run on the estate's answer rather than on this outcome.
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

// refreshDue returns the nodes among nodeIDs that have not been handed to the
// app within inventoryRefreshDebounce of now.
//
// Each node's own stamp decides, rather than its presence in the map: pruning
// below is then a memory bound and nothing more, and losing it costs entries
// that outlive their window instead of nodes that are never refreshed again.
// The difference matters because the failure is silent and permanent -- a node
// held for the life of the server makes every later run on it wait out the whole
// bootstrapInventoryRefreshWindow before confirm_monitoring resolves.
func (s *Service) refreshDue(nodeIDs []string, now time.Time) []string {
	s.refreshRequestedMu.Lock()
	defer s.refreshRequestedMu.Unlock()

	// Bounded by the nodes under bootstrap in one window rather than by every
	// node the server has ever refreshed.
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

// markRefreshed opens the inventoryRefreshDebounce window for nodeIDs. Stamped
// after the call rather than before it, unlike triggerInventorySync's own
// debounce: this one is per node and the caller is the single-leader stepper, so
// there is no second asker to coalesce, and a request that never reached the app
// must not hold the next one off.
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

// nodeIDForExecutorHost resolves a Nomad executor host name back to the PMM
// node id it belongs to -- the reverse of TriggerHostBootstrap's own
// resolution, needed wherever a bootstrap run's progress has to reach PMM's
// own inventory, which is keyed on node id.
func (s *Service) nodeIDForExecutorHost(ctx context.Context, executorHost string) (string, error) {
	hosts, err := s.inventoryHostsByExecutor(ctx)
	if err != nil {
		return "", err
	}
	host, ok := hosts[executorHost]
	if !ok {
		return "", status.Errorf(codes.NotFound,
			"no host in the inventory has executor %q", executorHost)
	}
	return host.NodeID, nil
}

// bootstrapProbe returns the configured om_bootstrap client, or an error saying
// it is not -- the same FailedPrecondition treatment inventoryProbe gives
// om_inventory, for the same reason: this is a deployment nobody has pointed at
// PMM Extensions yet, not a missing feature.
func (s *Service) bootstrapProbe() (*bootstrapClient, error) {
	if s.bootstrap == nil {
		return nil, status.Error(codes.FailedPrecondition,
			"PMM Extensions is not configured; set PMM_EXTENSIONS_URL and PMM_EXTENSIONS_TOKEN to reach the bootstrap app")
	}
	return s.bootstrap, nil
}

// defaultBootstrapRunLimit and maxBootstrapRunLimit mirror defaultInventoryRunLimit and
// maxInventoryRunLimit's own doc comment: the ceiling is forwarded to PMM Extensions verbatim, and
// matches both the proto's ListBootstrapRunsRequest.limit validation and PMM Extensions' own
// om_bootstrap GET /runs le=100.
const (
	defaultBootstrapRunLimit = 20
	maxBootstrapRunLimit     = 100
)

// GetBootstrapRun returns one bootstrap run's current progress.
//
// A thin proxy onto PMM Extensions' om_bootstrap GET /runs/{id}, like every other read in
// this file -- reconciling the run's in-flight dispatches happens on PMM Extensions' own
// side (see om_bootstrap's own doc comment on that route), not here.
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

// CancelBootstrapRun asks PMM Extensions to flag runID for cancellation and best-effort
// stop whatever step is currently dispatching -- see om_bootstrap's own
// :cancel route doc comment. Actually rolling every host back from there is
// PMM's own stepper's job, driven by bootstrap_decision.go's runNeedsRollback
// the next time it observes cancel_requested set, not this handler's.
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

// ListBootstrapRuns returns the bootstrap run history, newest first.
//
// Every status, not just active ones -- this backs an operator-facing history view,
// where a finished run is exactly as worth seeing as a running one.
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

// bootstrapRunConfigLabels returns the environment and cluster runID was
// triggered with, or two empty strings when there is nothing on record -- see
// OmBootstrapRunConfig's own doc comment on why that is the ordinary case, not
// a failure.
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

// confirmMonitoringLookup fetches the inventory app's current hosts for
// confirmMonitoringStep to check, but only when at least one of statuses is
// bootstrapRunSucceeded -- a run still installing, or one that failed or rolled
// back, can only ever report confirm_monitoring as "pending", so there is
// nothing worth an extra PMM Extensions call for. Degrades to nil on failure rather than
// failing the read it backs: a run's own progress is the more important half
// of that response, and a nil map reads every host as still unconfirmed, which
// is the honest answer when the lookup itself is unavailable.
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

// confirmMonitoringStepName names the synthetic, PMM-only step appended to every
// host's finalize_steps. Unlike every other step in this file, om_bootstrap never
// dispatches it -- it is PMM's own read-time confirmation that the service
// registerBootstrapHost created has actually been noticed by the estate's own
// inventory sweep, the same fact HostsPage's "Unregistered mongod" badge reports
// (databaseState in inventory.ts, on the UI side) until it flips. Appended to
// finalize_steps rather than a list of its own so it renders for free wherever a
// host's steps already do.
const confirmMonitoringStepName = "confirm_monitoring"

// confirmMonitoringStep reports whether executorHost's bootstrapped service has
// been noticed yet. "Pending" until the run itself has succeeded (nothing to
// confirm before then), "running" from there until hostsByExecutor shows a
// service for that host, "succeeded" once it does. A nil hostsByExecutor is what
// confirmMonitoringLookup returns when there was nothing to check yet, or its own
// PMM Extensions call failed -- both read as "still running" here, which is honest either
// way: a host genuinely isn't confirmed yet, or PMM cannot currently say.
func confirmMonitoringStep(runStatus, executorHost string, hostsByExecutor map[string]extensionsHost) *omv1.BootstrapStep {
	stepStatus := bootstrapStepPending
	if runStatus == bootstrapRunSucceeded {
		stepStatus = bootstrapStepRunning
		if host, ok := hostsByExecutor[executorHost]; ok && len(host.Services) > 0 {
			stepStatus = bootstrapStepSucceeded
		}
	}
	return &omv1.BootstrapStep{Name: confirmMonitoringStepName, Status: stepStatus}
}

// bootstrapRunToProto projects a extensionsBootstrapRun onto the wire shape
// GetBootstrapRun answers with. Its hostsByExecutor argument comes from
// confirmMonitoringLookup, and may be nil -- see confirmMonitoringStep. Its
// environment and cluster arguments come from bootstrapRunConfigLabels, and are
// empty strings when there is nothing on record for this run.
func bootstrapRunToProto(run *extensionsBootstrapRun, hostsByExecutor map[string]extensionsHost, environment, cluster string) *omv1.GetBootstrapRunResponse {
	hosts := make([]*omv1.BootstrapHost, 0, len(run.Hosts))
	for _, host := range run.Hosts {
		finalizeSteps := bootstrapStepsToProto(host.FinalizeSteps)
		finalizeSteps = append(finalizeSteps,
			confirmMonitoringStep(run.Status, host.Host, hostsByExecutor))
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

// bootstrapStepsToProto projects a slice of extensionsBootstrapStep onto the wire
// shape shared by a host's own steps, its rollback steps, and a run's
// run-level steps.
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

// Bounds on the shape of an update batch, not on its meaning.
//
// The app owns which keys exist and what values are legal, so these do not duplicate its
// rules -- they bound how much structure may cross the hop at all. Both are orders of
// magnitude above any real batch: the settings class this proxies has ten fields and
// nests one level, SCHEDULE and its children.
//
// They are needed because neither hop bounds them. Measured against this tree:
// protojson accepts 200k fields in a 2.3MB body, inside the 4MB default gRPC message
// size, and nests to just under 10k before refusing. The app on the far side is Python,
// where the default recursion limit is 1000, so forwarding either would make PMM the
// thing that broke PMM Extensions. This endpoint is admin-only, which makes it a footgun rather
// than an attack, but a 200k-key batch is an accident worth refusing by name.
const (
	maxConfigFields = 100
	maxConfigDepth  = 10
)

// validateConfigValues refuses a batch that is empty, too wide, or too deeply nested.
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

// exceedsDepth reports whether value nests deeper than limit.
//
// It stops at the limit rather than measuring the true depth, so its own recursion is
// bounded by maxConfigDepth however deep the input goes.
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

// UpdateInventoryConfig applies a batch of configuration changes.
//
// The batch is passed through as the app received it and the app decides what is valid,
// which is what keeps one set of validation rules rather than two. A single bad field
// rejects the whole batch there and nothing is written. Only the batch's shape is
// checked here, by validateConfigValues.
//
// PUT here, PATCH to the app below, deliberately. PMM's API guidelines require the
// standard Update method to be PUT and every other update in this repo is one; PMM Extensions
// reaches the same overrides through a generic settings router that PATCHes
// `/{setting_class}` for every app, so the verb there is not this app's to pick.
// Translating one method is what a proxy is for.
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

	// Read the whole configuration back rather than returning the rows the app echoed.
	//
	// Two reasons, and the second is the one that matters. The guidelines require an
	// Update to answer with the resource, not a diff. And the app answers with one row per
	// key *named in the request*, which is misleading for a nested write: overriding
	// SCHEDULE as a whole object changes what SCHEDULE__every effectively resolves to, and
	// no row says so. A caller that trusted the echo would render a stale child beside a
	// fresh parent.
	//
	// This also retires the rule the UI had to follow -- "re-read after a write, never
	// echo the submitted value" -- by doing it once here, for every caller rather than
	// only the ones that remembered.
	current := []extensionsSetting{}
	read := inventoryCall{method: http.MethodGet, path: "config"}
	err = probe.call(ctx, read, &current)
	if err != nil {
		// The write landed; only the read-back failed. Reporting an error here would tell
		// a caller to retry a change that already applied, so answer with the narrower
		// body the app gave us and log the shortfall.
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

// inventoryHostToProto projects one host row for the wire.
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

// hostNotATargetReasons returns the reasons Operations must never install onto this
// node, from the host document alone.
//
// Split out of automationEligibility because TriggerHostBootstrap enforces exactly
// this set and nothing else (PMM-15664 task 2). Eligibility is advisory -- it is
// computed for a list request and can be minutes stale by the time anyone clicks --
// so the trigger repeats it, and repeating it means sharing the predicate rather
// than writing a second one that can drift.
//
// Deliberately *not* the whole of eligibility. The reachability reasons, the OS and
// the address describe a run that would fail; these three describe a run that would
// succeed and damage something. The trigger keeps its own executor and OS guards
// below for the former.
func hostNotATargetReasons(isPMMServer bool, host extensionsHost) []string {
	if isPMMServer {
		// Alone, not first of several: PMM Server's own image reports os_id "ol", so
		// listing the rest had the row advising that Operations "cannot install onto
		// ol (supported: rocky, ubuntu)" -- which invites someone to reinstall the
		// machine PMM is running on.
		return []string{
			"this is the node PMM Server itself runs on, which Operations never installs onto",
		}
	}
	if len(host.Services) > 0 {
		return []string{"a MongoDB service is already registered on this node"}
	}
	// `else if` in spirit: a node with a registered service usually has the mongod to
	// go with it, and saying both would be one problem reported twice.
	if unregisteredMongodCount(host.Observed) > 0 {
		return []string{"a scan found a mongod running here that PMM has no service for"}
	}
	// Installed but not running: nothing above sees it, and the install's own
	// pre_check missed it too when it sat outside sudo's secure_path, so the run went
	// ahead and left a second mongod beside the first. The scan reads the binary
	// itself, wherever PATH finds it, so its answer is the one to trust here.
	if version := observedString(host.Observed, "installed_version"); version != nil {
		return []string{fmt.Sprintf("a scan found MongoDB %s already installed on this node", *version)}
	}
	return nil
}

// automationEligibility decides whether OM automation (a probe today; provisioning in
// a later phase) can run on a host, and names every unmet condition.
//
// Deliberately one shared definition rather than per-task-type requirements for now:
// probing and the PMM-15347 PoC's bootstrap dispatch both need the same two things
// (a connected agent, a reachable driver-healthy executor), and there is exactly one
// consumer of the distinction so far. See PMM-15347/questions.md Q2 for why a
// requirements-per-task-type mechanism is deliberately not built until a second,
// differently-shaped task type actually needs one.
func automationEligibility(
	executor *omv1.InventoryExecutor,
	pmmAgentConnected bool,
	isPMMServer bool,
	host extensionsHost,
) (bool, []string, bool) {
	// The reasons this node must never be installed onto, which is a different
	// question from whether we can reach it -- and the one TriggerHostBootstrap
	// enforces too, so a direct API call cannot do what the UI refuses.
	reasons := hostNotATargetReasons(isPMMServer, host)
	// Tracked separately from the reasons rather than inferred from them: a consumer
	// matching on the strings would break the first time one is reworded, and the
	// difference decides whether a reader is shown an alarm or a fact.
	byDesign := len(reasons) > 0
	if isPMMServer {
		// Its reason stands alone -- see hostNotATargetReasons. Everything below
		// describes something a user could go and fix, and none of it would make this
		// node a target.
		return false, reasons, true
	}
	if !pmmAgentConnected {
		reasons = append(reasons, "PMM Client is not installed or not connected")
	}
	// Worded in the glossary the UI agreed on (PMM-15659), not in Nomad's terms.
	// These strings are not diagnostics: automationBlockedTitle joins them straight
	// into the tooltip on the Nodes page, so "the Nomad client" and "raw_exec" were
	// user-facing text naming our scheduler, which PMM-15623 set out to remove.
	if executor == nil || !executor.GetReachable() {
		reasons = append(reasons, "this node has no automation agent that answers")
	} else if !executor.GetDriverHealthy() {
		reasons = append(reasons, "this node's automation agent cannot run jobs")
	}
	// The same map the trigger checks against, so eligibility and TriggerHostBootstrap
	// cannot disagree about which OS an install supports. Both are preconditions
	// names, and both failed on the wizard's final button until now.
	switch osID, _ := host.Observed["os_id"].(string); {
	case osID == "":
		reasons = append(reasons, "no scan has reported this node's operating system yet")
	case !supportedBootstrapOSIDs[osID]:
		// By design: nothing is wrong with the node. Operations installs onto two
		// distributions, and this is not one of them -- "Needs attention" would send
		// a reader looking for a fault on a machine that is working perfectly.
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
	// Only meaningful when something is blocking. An eligible node reporting "blocked
	// by design" would be a contradiction a consumer has to reason about.
	return eligible, reasons, !eligible && byDesign
}

// unregisteredMongodCount counts the mongods a scan found that PMM has no service
// for. Reads the same observed key the wire projection does, so the two cannot
// disagree about whether a node carries one.
func unregisteredMongodCount(observed map[string]any) int {
	entries, ok := observed["unregistered_mongods"].([]any)
	if !ok {
		return 0
	}
	return len(entries)
}

// inventoryServiceToProto projects one service row for the wire.
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

// inventoryRunToProto projects one refresh for the wire.
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

// inventoryRunEntitiesToProto projects what a refresh attempted, one row per host.
//
// Host-oriented, because a refresh attempts hosts: a flat service list cannot show a
// machine carrying a PMM client and no database, which is the case OM most exists to
// describe. Each host carries the services on it, and empty is a meaningful answer.
//
// Outcomes only: which host, matched how, answered or not, how long, and the error.
// What the probe found belongs to the estate, which is upserted and stays current --
// carrying it here as well would be a second copy that goes stale the moment the next
// refresh runs. TaskHistoryID is the one exception, and not a contradiction of that
// rule: it carries no observation, only a pointer to where the dispatch's raw output
// -- the observations that were deliberately *not* kept here -- can still be read.
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

// inventorySettingsToProto projects the configuration rows for the wire.
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

// freshnessToProto projects the freshness block.
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

// executorToProto lifts the executor block out of the host document.
//
// Returns nil when the app reported none, which is not the same as three false flags:
// absent means "this sweep did not say", while false means "PMM Extensions looked and the answer
// was no".
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

// unregisteredMongodsToProto lifts the stranger list out of the host document.
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

// observedToStruct carries the whole document through untyped.
//
// The point of the app storing observations as JSON is that collecting a new attribute
// is a payload change rather than a schema change. Enumerating every attribute as a
// proto field would put that coupling back, so the fields above are the ones a table
// sorts by and this is everything: a new attribute shows up in a detail panel the day it
// is collected, and is promoted to a field only when something wants to sort by it.
//
// The collected_at key is dropped: it is metadata about the document rather than an observation,
// and the freshness block already carries the same instant with a name that says so.
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
		// Not fatal: the typed fields above are already extracted, and losing the detail
		// panel is better than failing the whole listing over one unrepresentable value.
		return nil
	}
	return encoded
}

// anyToValue wraps one decoded JSON value, or nil when it cannot be represented.
func anyToValue(value any) *structpb.Value {
	encoded, err := structpb.NewValue(value)
	if err != nil {
		return nil
	}
	return encoded
}

// observedString reads one string attribute out of a document.
//
// Empty and absent are deliberately merged: a missing key and a key holding "" both
// return nil, so the field is omitted from the response either way. For what these read
// -- a config path, an argv, an OS name -- an empty string carries no more information
// than no string at all, and collapsing them keeps the column from showing a blank cell
// that a reader has to interpret.
//
// The pointer is what makes that omission reachable at all: protojson drops an unset
// `optional` scalar entirely, even under EmitUnpopulated.
func observedString(observed map[string]any, key string) *string {
	value, ok := observed[key].(string)
	if !ok || value == "" {
		return nil
	}
	return &value
}

// clampInt32 narrows a count that arrived as a JSON number.
//
// PMM Extensions' counter is decoded into an int, which is 64-bit here, so a plain conversion could
// wrap and report a negative number of consecutive failures. Saturating instead keeps the
// column monotonic: a reader learns "very many", never "minus two billion".
func clampInt32(value int) int32 {
	switch {
	case value > math.MaxInt32:
		return math.MaxInt32
	case value < math.MinInt32:
		return math.MinInt32
	}
	return int32(value)
}

// observedBool reads one boolean attribute out of a document.
func observedBool(observed map[string]any, key string) *bool {
	value, ok := observed[key].(bool)
	if !ok {
		return nil
	}
	return &value
}

// observedDouble reads one numeric attribute out of a document.
func observedDouble(observed map[string]any, key string) *float64 {
	value, ok := observed[key].(float64)
	if !ok {
		return nil
	}
	return &value
}

// observedInt32 reads one integer attribute out of a document.
//
// JSON numbers decode as float64, so this narrows rather than asserts -- and a value that
// will not fit is dropped rather than wrapped. Both callers are a port and a PID, where a
// wrapped result reads as a plausible one: absent says "not collected", 4295 says a port
// something is listening on.
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

// optionalString wraps a nullable string.
func optionalString(value *string) *string {
	if value == nil || *value == "" {
		return nil
	}
	return value
}

// optionalInt32 wraps a nullable integer.
func optionalInt32(value *int32) *int32 {
	if value == nil {
		return nil
	}
	return value
}

// optionalTimestamp wraps a nullable instant.
func optionalTimestamp(value *time.Time) *timestamppb.Timestamp {
	if value == nil {
		return nil
	}
	return timestamppb.New(*value)
}

// parseSepTime parses one of the app's timestamps, tolerating either spelling.
//
// The app serves RFC 3339, but whether the offset is `Z` or `+00:00` depends on which
// column it came from, and a run's started_at has been seen both ways.
func parseSepTime(stamp string) *timestamppb.Timestamp {
	parsed, err := time.Parse(time.RFC3339, stamp)
	if err != nil {
		return nil
	}
	return timestamppb.New(parsed)
}

// The app's vocabularies, mapped onto the wire enums.
//
// Separate from store.go's mappers even where the values coincide, because these translate
// a *different* system's strings: the app owns them, they arrive over HTTP, and a value OM
// has never heard of is a real possibility rather than a corrupted row. Every one of these
// falls through to UNSPECIFIED, which is how "the app said something new" reaches a caller
// as an unknown rather than as a plausible wrong answer.

// extensionsRunStatusToProto maps the app's run status onto the wire enum.
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

// extensionsResolutionToProto maps how the app matched a host to an executor client.
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

// extensionsReloadToProto maps a setting's reload class.
//
// Mirrors ReloadClassification in the app's settings registry one-for-one. Collapsing
// nested_only into "not overridable" was the tempting simplification and it is wrong: a
// nested parent rejects a whole-object write while its children accept one, so a form
// reading the parent would refuse to edit a leaf the API accepts.
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
