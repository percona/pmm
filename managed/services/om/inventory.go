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
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	omv1 "github.com/percona/pmm/api/om/v1"
)

// The /v1/om/inventory/* handlers: SEP's estate, served through PMM.
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

// defaultInventoryRunLimit is what a caller who passes no limit gets.
const defaultInventoryRunLimit = 20

// inventoryProbe returns the configured SEP client, or an error saying it is not.
//
// Reported as FailedPrecondition rather than Unimplemented or NotFound: the endpoints
// exist and work, the deployment has simply not been told where SEP is, and that is an
// operator's action rather than a missing feature.
func (s *Service) inventoryProbe() (*probeSource, error) {
	if s.probe == nil {
		return nil, status.Error(codes.FailedPrecondition,
			"SEP is not configured; set PMM_SEP_URL and PMM_SEP_TOKEN to reach the inventory app")
	}
	return s.probe, nil
}

// ListInventoryHosts returns every host the inventory app has a row for.
func (s *Service) ListInventoryHosts(ctx context.Context, req *omv1.ListInventoryHostsRequest) (*omv1.ListInventoryHostsResponse, error) {
	probe, err := s.inventoryProbe()
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	if req.GetHasService() != nil {
		query.Set("has_service", strconv.FormatBool(req.GetHasService().GetValue()))
	}
	if req.GetFailing() != nil {
		query.Set("failing", strconv.FormatBool(req.GetFailing().GetValue()))
	}
	if req.GetExecutor() != nil {
		query.Set("executor", strconv.FormatBool(req.GetExecutor().GetValue()))
	}

	hosts := []sepHost{}
	call := inventoryCall{method: http.MethodGet, path: "hosts", query: query}
	if err := probe.call(ctx, call, &hosts); err != nil { //nolint:noinlineerr
		return nil, err
	}

	response := &omv1.ListInventoryHostsResponse{Hosts: make([]*omv1.InventoryHost, 0, len(hosts))}
	for _, host := range hosts {
		response.Hosts = append(response.Hosts, inventoryHostToProto(host))
	}
	return response, nil
}

// GetInventoryHost returns one host.
func (s *Service) GetInventoryHost(ctx context.Context, req *omv1.GetInventoryHostRequest) (*omv1.GetInventoryHostResponse, error) {
	probe, err := s.inventoryProbe()
	if err != nil {
		return nil, err
	}

	host := sepHost{}
	call := inventoryCall{method: http.MethodGet, path: inventoryPath("hosts", req.GetNodeId())}
	if err := probe.call(ctx, call, &host); err != nil { //nolint:noinlineerr
		return nil, err
	}
	return &omv1.GetInventoryHostResponse{Host: inventoryHostToProto(host)}, nil
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
	if err := probe.call(ctx, call, nil); err != nil { //nolint:noinlineerr
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
	if nodeID := req.GetNodeId().GetValue(); nodeID != "" {
		query.Set("node_id", nodeID)
	}
	if req.GetFailing() != nil {
		query.Set("failing", strconv.FormatBool(req.GetFailing().GetValue()))
	}

	services := []sepService{}
	call := inventoryCall{method: http.MethodGet, path: "services", query: query}
	if err := probe.call(ctx, call, &services); err != nil { //nolint:noinlineerr
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

	service := sepService{}
	call := inventoryCall{method: http.MethodGet, path: inventoryPath("services", req.GetServiceId())}
	if err := probe.call(ctx, call, &service); err != nil { //nolint:noinlineerr
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
	if err := probe.call(ctx, call, nil); err != nil { //nolint:noinlineerr
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

	limit := req.GetLimit()
	if limit <= 0 {
		limit = defaultInventoryRunLimit
	}
	query := url.Values{"limit": []string{strconv.FormatInt(int64(limit), 10)}}

	runs := []sepRun{}
	call := inventoryCall{method: http.MethodGet, path: "runs", query: query}
	if err := probe.call(ctx, call, &runs); err != nil { //nolint:noinlineerr
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

	run := sepRun{}
	call := inventoryCall{method: http.MethodGet, path: inventoryPath("runs", req.GetRunId())}
	if err := probe.call(ctx, call, &run); err != nil { //nolint:noinlineerr
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
	if err := probe.call(ctx, call, &accepted); err != nil { //nolint:noinlineerr
		return nil, err
	}

	response := &omv1.TriggerInventoryRefreshResponse{
		RunId:  accepted.RunID,
		Status: sepRunStatusToProto(accepted.Status),
		Scope:  accepted.Scope,
	}
	if accepted.StartedAt != nil {
		if parsed := parseSepTime(*accepted.StartedAt); parsed != nil {
			response.StartTime = parsed
		}
	}
	return response, nil
}

// GetInventoryConfig returns the inventory app's configuration.
func (s *Service) GetInventoryConfig(ctx context.Context, _ *omv1.GetInventoryConfigRequest) (*omv1.GetInventoryConfigResponse, error) {
	probe, err := s.inventoryProbe()
	if err != nil {
		return nil, err
	}

	settings := []sepSetting{}
	call := inventoryCall{method: http.MethodGet, path: "config"}
	if err := probe.call(ctx, call, &settings); err != nil { //nolint:noinlineerr
		return nil, err
	}
	return &omv1.GetInventoryConfigResponse{Settings: inventorySettingsToProto(settings)}, nil
}

// UpdateInventoryConfig applies a batch of configuration changes.
//
// The batch is passed through as the app received it and the app decides what is valid,
// which is what keeps one set of validation rules rather than two. A single bad field
// rejects the whole batch there and nothing is written.
//
// PUT here, PATCH to the app below, deliberately. PMM's API guidelines require the
// standard Update method to be PUT and every other update in this repo is one; SEP
// reaches the same overrides through a generic settings router that PATCHes
// `/{setting_class}` for every app, so the verb there is not this app's to pick.
// Translating one method is what a proxy is for.
func (s *Service) UpdateInventoryConfig(ctx context.Context, req *omv1.UpdateInventoryConfigRequest) (*omv1.UpdateInventoryConfigResponse, error) {
	probe, err := s.inventoryProbe()
	if err != nil {
		return nil, err
	}
	if len(req.GetValues().GetFields()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "values must name at least one field to change")
	}

	applied := []sepSetting{}
	call := inventoryCall{method: http.MethodPatch, path: "config", body: req.GetValues().AsMap()}
	if err := probe.call(ctx, call, &applied); err != nil { //nolint:noinlineerr
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
	current := []sepSetting{}
	read := inventoryCall{method: http.MethodGet, path: "config"}
	if err := probe.call(ctx, read, &current); err != nil {
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
	if err := probe.call(ctx, call, nil); err != nil { //nolint:noinlineerr
		return nil, err
	}
	return &omv1.DeleteInventoryConfigOverrideResponse{}, nil
}

// inventoryHostToProto projects one host row for the wire.
func inventoryHostToProto(host sepHost) *omv1.InventoryHost {
	out := &omv1.InventoryHost{
		NodeId:              host.NodeID,
		Name:                host.Name,
		Address:             optionalString(host.Address),
		ExecutorHost:        optionalString(host.ExecutorHost),
		Os:                  observedString(host.Observed, "os"),
		Kernel:              observedString(host.Observed, "kernel"),
		Executor:            executorToProto(host.Observed),
		UnregisteredMongods: unregisteredMongodsToProto(host.Observed),
		Observed:            observedToStruct(host.Observed),
		Freshness:           freshnessToProto(host.sepFreshness),
		Services:            make([]*omv1.InventoryService, 0, len(host.Services)),
	}
	for _, service := range host.Services {
		out.Services = append(out.Services, inventoryServiceToProto(service))
	}
	return out
}

// inventoryServiceToProto projects one service row for the wire.
func inventoryServiceToProto(service sepService) *omv1.InventoryService {
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
		Freshness:        freshnessToProto(service.sepFreshness),
	}
}

// inventoryRunToProto projects one refresh for the wire.
func inventoryRunToProto(run sepRun) *omv1.InventoryRun {
	return &omv1.InventoryRun{
		RunId:     run.RunID,
		Status:    sepRunStatusToProto(run.Status),
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
		},
		Scope: run.Scope,
		Error: optionalString(run.Error),
	}
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
// refresh runs.
func inventoryRunEntitiesToProto(nodes []sepRunNode) []*omv1.InventoryRunEntity {
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
			Resolution:      sepResolutionToProto(node.Resolution),
			Answered:        node.Answered,
			DurationSeconds: optionalDouble(node.Duration),
			Error:           optionalString(node.Error),
			Services:        services,
		})
	}
	return entities
}

// inventorySettingsToProto projects the configuration rows for the wire.
func inventorySettingsToProto(settings []sepSetting) []*omv1.InventorySetting {
	out := make([]*omv1.InventorySetting, 0, len(settings))
	for _, setting := range settings {
		out = append(out, &omv1.InventorySetting{
			Key:          setting.Key,
			Value:        anyToValue(setting.Value),
			DefaultValue: anyToValue(setting.DefaultValue),
			Type:         setting.Type,
			Reload:       sepReloadToProto(setting.Reload),
			HasOverride:  setting.HasOverride,
			IsAdvanced:   setting.IsAdvanced,
			Description:  optionalString(setting.Description),
		})
	}
	return out
}

// freshnessToProto projects the freshness block.
func freshnessToProto(f sepFreshness) *omv1.InventoryFreshness {
	return &omv1.InventoryFreshness{
		FirstSeenAt:         optionalTimestamp(f.FirstSeenAt),
		LastAttemptAt:       optionalTimestamp(f.LastAttemptAt),
		LastSuccessAt:       optionalTimestamp(f.LastSuccessAt),
		FailingSince:        optionalTimestamp(f.FailingSince),
		ConsecutiveFailures: int32(f.ConsecutiveFailures), //nolint:gosec
		LastError:           optionalString(f.LastError),
	}
}

// executorToProto lifts the executor block out of the host document.
//
// Returns nil when the app reported none, which is not the same as three false flags:
// absent means "this sweep did not say", while false means "SEP looked and the answer
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
		out.Detail = wrapperspb.String(value)
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
// Every reader here returns a wrapper rather than a bare value because protojson drops
// an unset `optional` scalar entirely, even under EmitUnpopulated. A silently absent
// nullable field is a column that reads as "not collected" when the truth may be
// "collected, and it was empty".
func observedString(observed map[string]any, key string) *wrapperspb.StringValue {
	value, ok := observed[key].(string)
	if !ok || value == "" {
		return nil
	}
	return wrapperspb.String(value)
}

// observedBool reads one boolean attribute out of a document.
func observedBool(observed map[string]any, key string) *wrapperspb.BoolValue {
	value, ok := observed[key].(bool)
	if !ok {
		return nil
	}
	return wrapperspb.Bool(value)
}

// observedDouble reads one numeric attribute out of a document.
func observedDouble(observed map[string]any, key string) *wrapperspb.DoubleValue {
	value, ok := observed[key].(float64)
	if !ok {
		return nil
	}
	return wrapperspb.Double(value)
}

// observedInt32 reads one integer attribute out of a document.
//
// JSON numbers decode as float64, so this narrows rather than asserts -- and a value that
// will not fit is dropped rather than wrapped. Both callers are a port and a PID, where a
// wrapped result reads as a plausible one: absent says "not collected", 4295 says a port
// something is listening on.
func observedInt32(observed map[string]any, key string) *wrapperspb.Int32Value {
	value, ok := observed[key].(float64)
	if !ok {
		return nil
	}
	if math.IsNaN(value) || value < math.MinInt32 || value > math.MaxInt32 {
		return nil
	}
	return wrapperspb.Int32(int32(value))
}

// optionalString wraps a nullable string.
func optionalString(value *string) *wrapperspb.StringValue {
	if value == nil || *value == "" {
		return nil
	}
	return wrapperspb.String(*value)
}

// optionalInt32 wraps a nullable integer.
func optionalInt32(value *int32) *wrapperspb.Int32Value {
	if value == nil {
		return nil
	}
	return wrapperspb.Int32(*value)
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

// sepRunStatusToProto maps the app's run status onto the wire enum.
func sepRunStatusToProto(status string) omv1.RunStatus {
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

// sepResolutionToProto maps how the app matched a host to an executor client.
func sepResolutionToProto(resolution string) omv1.ExecutorResolution {
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

// sepReloadToProto maps a setting's reload class.
//
// Mirrors ReloadClassification in the app's settings registry one-for-one. Collapsing
// nested_only into "not overridable" was the tempting simplification and it is wrong: a
// nested parent rejects a whole-object write while its children accept one, so a form
// reading the parent would refuse to edit a leaf the API accepts.
func sepReloadToProto(reload string) omv1.SettingReload {
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
