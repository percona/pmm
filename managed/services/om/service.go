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

// Package om implements the OpenManager topology and health API: it groups PMM's MongoDB services
// by cluster label and folds in reachability and load from VictoriaMetrics.
package om

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"

	omv1 "github.com/percona/pmm/api/om/v1"
	"github.com/percona/pmm/managed/models"
)

const (
	refreshInterval = 30 * time.Second
	staleAfter      = 5 * time.Minute
	runHistory      = 100
	defaultRunLimit = 25
	maxRunLimit     = runHistory
)

// Run statuses as the run row stores them; runStatusToProto maps them to the wire enum.
const (
	runStatusSuccess = "success"
	runStatusPartial = "partial"
	runStatusFailed  = "failed"
)

// Service provides the OM topology API.
type Service struct {
	omv1.UnimplementedOmServiceServer

	db       *reform.DB
	vmClient victoriaMetricsClient
	l        *logrus.Entry

	// ha is nil in most tests, which means this node is the leader.
	ha haChecker

	probe *probeSource

	// tick overrides refreshInterval; only tests set it.
	tick time.Duration

	bootstrap    *bootstrapClient
	agents       agentConnectionChecker
	stateUpdater agentStateUpdater

	// syncRequested debounces triggerInventorySync globally: one pull covers every run finishing in the same window.
	syncRequested   time.Time
	syncRequestedMu sync.Mutex

	// refreshRequested debounces triggerScopedInventoryRefresh per node; refreshDue prunes it.
	refreshRequested   map[string]time.Time
	refreshRequestedMu sync.Mutex

	restored sync.Once

	// running serialises collection.
	running sync.Mutex

	mu     sync.Mutex
	latest *omv1.GetTopologyResponse

	runsSuccess atomic.Int64
	runsPartial atomic.Int64
	runsFailed  atomic.Int64
}

// New returns a new OM service.
func New(db *reform.DB, vmClient victoriaMetricsClient, ha haChecker, l *logrus.Entry) *Service {
	return &Service{
		db:       db,
		vmClient: vmClient,
		ha:       ha,
		l:        l,
	}
}

// WithProbeSource attaches PMM Extensions' om_inventory app as a fact source; an empty URL leaves it off.
func (s *Service) WithProbeSource(extensionsURL, token string) *Service {
	if extensionsURL == "" {
		s.l.Info("PMM Extensions is not configured; on-host facts will be absent")
		return s
	}
	client := newExtensionsClient(extensionsURL, token)
	probe := &probeSource{
		app: client.app(probeAppModule),
		l:   s.l.WithField("source", sourceProbe),
	}
	s.probe = probe
	s.l.Infof("om_inventory estate at %s", probe.app.endpoint(""))
	return s
}

// Enabled returns true if OpenManager is enabled.
func (s *Service) Enabled() bool {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		s.l.WithError(err).Error("can't get settings")
		return false
	}
	return settings.IsOMEnabled()
}

// CheckAvailable returns why PMM Extensions' om_inventory app cannot be used, or nil when it can.
func (s *Service) CheckAvailable(ctx context.Context) error {
	probe, err := s.inventoryProbe()
	if err != nil {
		return err
	}
	var page extensionsPage[probeService]
	err = probe.call(ctx, inventoryCall{method: http.MethodGet, path: probeServicesPath, query: url.Values{"limit": {"1"}}}, &page)
	if status.Code(err) == codes.NotFound {
		return status.Errorf(codes.FailedPrecondition, "PMM Extensions does not serve the om_inventory app: %s", status.Convert(err).Message())
	}
	return err
}

// SyncInventoryEnabled sets om_inventory's ENABLED flag and, on enabling, triggers an immediate sweep so the
// estate does not wait out a full SCHEDULE interval. Failures are logged, not returned.
func (s *Service) SyncInventoryEnabled(ctx context.Context, enabled bool) {
	_ = s.syncInventoryEnabled(ctx, enabled)
}

// syncInventoryEnabled returns only the ENABLED write's error: a failed sweep trigger is logged and left to the
// app's own SCHEDULE.
func (s *Service) syncInventoryEnabled(ctx context.Context, enabled bool) error {
	if s.probe == nil || s.probe.app.client == nil {
		return nil
	}
	err := s.probe.app.patchConfig(ctx, map[string]any{"ENABLED": enabled})
	if err != nil {
		s.l.WithError(err).WithField("enabled", enabled).
			Warn("failed to sync OpenManager's on/off state to PMM Extensions' om_inventory app")
		return err
	}
	if !enabled {
		return nil
	}
	err = s.probe.app.triggerRun(ctx)
	if err != nil {
		s.l.WithError(err).Warn("failed to trigger an immediate PMM Extensions inventory sweep after enabling OpenManager")
	}
	return nil
}

// WithBootstrapSource attaches PMM Extensions' om_bootstrap app for RunBootstrapStepper and TriggerHostBootstrap.
func (s *Service) WithBootstrapSource(extensionsURL, token string) *Service {
	if extensionsURL == "" {
		s.l.Info("PMM Extensions is not configured; MongoDB bootstrap will be unavailable")
		return s
	}
	client := newExtensionsClient(extensionsURL, token)
	s.bootstrap = &bootstrapClient{app: client.app(bootstrapAppModule)}
	return s
}

// WithAgentRegistry attaches the pmm-agent connectivity checker ListInventoryHosts uses for automation eligibility.
// Without it, every host reads as not connected.
func (s *Service) WithAgentRegistry(r agentConnectionChecker) *Service {
	s.agents = r
	return s
}

// WithStateUpdater attaches the pmm-agent state pusher registerBootstrapHost uses to start a new mongodb_exporter.
// Without it, a bootstrapped host's exporter row exists but pmm-agent is never told to start it.
func (s *Service) WithStateUpdater(u agentStateUpdater) *Service {
	s.stateUpdater = u
	return s
}

// GetTopology returns the whole MongoDB estate as one document, from memory or the stored snapshot.
// It never collects: collection is leader-only.
func (s *Service) GetTopology(ctx context.Context, _ *omv1.GetTopologyRequest) (*omv1.GetTopologyResponse, error) { //nolint:unparam
	s.restoreOnce(ctx)
	if cached := s.snapshot(); cached != nil {
		return withFreshStale(cached), nil
	}
	return emptyTopologyResponse(), nil
}

// ListTopologyRuns returns the recorded runs, newest first.
func (s *Service) ListTopologyRuns(ctx context.Context, req *omv1.ListTopologyRunsRequest) (*omv1.ListTopologyRunsResponse, error) {
	limit := int(req.GetLimit())
	switch {
	case limit <= 0:
		limit = defaultRunLimit
	case limit > maxRunLimit:
		limit = maxRunLimit
	}

	runs, err := s.listRuns(ctx, limit)
	if err != nil {
		return nil, err
	}
	return &omv1.ListTopologyRunsResponse{Runs: runs}, nil
}

// GetTopologyRun returns one recorded run.
func (s *Service) GetTopologyRun(ctx context.Context, req *omv1.GetTopologyRunRequest) (*omv1.GetTopologyRunResponse, error) {
	run, err := s.getRun(ctx, req.GetRunId())
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "Run %s not found.", req.GetRunId())
		}
		return nil, err
	}
	return &omv1.GetTopologyRunResponse{Run: run}, nil
}

// TriggerTopologyCollection rebuilds the topology document now and records the run.
//
//nolint:unparam
func (s *Service) TriggerTopologyCollection(ctx context.Context, _ *omv1.TriggerTopologyCollectionRequest) (*omv1.TriggerTopologyCollectionResponse, error) {
	if s.ha != nil && !s.ha.IsLeader() {
		if leader := s.ha.LeaderID(); leader != "" {
			return nil, status.Errorf(codes.FailedPrecondition,
				"This node is not the HA leader; %s is. Only the leader collects.", leader)
		}
		return nil, status.Error(codes.FailedPrecondition,
			"This node is not the HA leader. Only the leader collects.")
	}

	if !s.running.TryLock() {
		// Aborted maps to 409 Conflict, which the UI's Sync button treats as an expected outcome.
		return nil, status.Error(codes.Aborted, "A collection run is already in flight.")
	}
	defer s.running.Unlock()

	_, run, err := s.collect(ctx)
	if err != nil {
		return nil, err
	}
	return &omv1.TriggerTopologyCollectionResponse{
		RunId:     run.RunId,
		Status:    run.Status,
		StartTime: run.StartTime,
	}, nil
}

// Run refreshes the document on a timer until ctx is cancelled. It also syncs the OpenManager switch to om_inventory,
// retrying until it lands, since a server that starts enabled sees no ChangeSettings transition.
func (s *Service) Run(ctx context.Context) {
	reconciled := s.syncInventoryEnabled(ctx, s.Enabled()) == nil

	interval := s.tick
	if interval == 0 {
		interval = refreshInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Before the Enabled() gate: syncing an off switch is how PMM Extensions is told to stop sweeping.
			if !reconciled {
				reconciled = s.syncInventoryEnabled(ctx, s.Enabled()) == nil
			}
			if !s.Enabled() {
				continue
			}
			_, err := s.discover(ctx)
			if err != nil && ctx.Err() == nil {
				s.l.Warnf("scheduled collection failed: %s", err)
			}
		}
	}
}

// discover collects, or if a collection is already in flight, waits for it and serves what it published.
func (s *Service) discover(ctx context.Context) (*omv1.GetTopologyResponse, error) {
	if !s.running.TryLock() {
		s.running.Lock()
		s.running.Unlock() //nolint:staticcheck // waiting out the in-flight run, not guarding
		if cached := s.snapshot(); cached != nil {
			return cached, nil
		}
		return nil, status.Error(codes.Unavailable, "A collection is in flight and no document is available yet.")
	}
	defer s.running.Unlock()

	response, _, err := s.collect(ctx)
	return response, err
}

// collect builds the document from every source and records the run. The caller must hold s.running.
func (s *Service) collect(ctx context.Context) (*omv1.GetTopologyResponse, *omv1.TopologyRun, error) {
	startedAt := time.Now()
	runID := uuid.New().String()

	services, nodes, originNode, maxAge, err := s.readInventory(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read inventory: %w", err)
	}

	sources := s.sources(nodes, startedAt)
	results := make([]SourceResult, 0, len(sources))
	for _, source := range sources {
		results = append(results, source.collect(ctx, services))
	}

	merged := mergeFacts(results, defaultPrecedence)
	generatedAt := time.Now()
	doc := applyHealth(buildDocument(services, merged, generatedAt, maxAge))

	response := &omv1.GetTopologyResponse{
		Snapshot: &omv1.Snapshot{
			GeneratedAt:   timestamppb.New(generatedAt),
			ObservedAt:    observedAt(doc),
			Stale:         isStale(doc, generatedAt),
			SchemaVersion: schemaVersion,
			RunId:         runID,
		},
		OriginNode:    optional(originNode),
		SourceQueries: sourceQueries,
		Summary:       doc.summary,
		Environments:  doc.environments,
	}
	run := buildRun(runID, startedAt, generatedAt, services, merged, doc, results)
	s.recordRunOutcome(run.Status)

	s.mu.Lock()
	s.latest = response
	s.mu.Unlock()

	err = s.persist(ctx, response, run, originNode)
	if err != nil {
		s.l.Warnf("run %s: failed to record: %s", runID, err)
	}

	s.l.Infof("run %s: %d service(s), %d up, %d cluster(s), %d stale, %s max age, status %s",
		runID, doc.summary.TotalServices, doc.summary.UpServices, doc.summary.Clusters,
		doc.staleServices, maxAge, run.Status)

	return response, run, nil
}

func (s *Service) sources(nodes map[string]*models.Node, startedAt time.Time) []factSource {
	sources := []factSource{
		inventorySource{nodes: nodes},
		metricsSource{vm: s.vmClient, l: s.l, now: startedAt},
	}
	if s.probe != nil {
		sources = append(sources, *s.probe)
	}
	return sources
}

func (s *Service) restoreOnce(ctx context.Context) {
	s.restored.Do(func() {
		response, generatedAt, err := s.restore(ctx)
		if err != nil {
			s.l.Warnf("failed to restore the stored topology: %s", err)
			return
		}
		if response == nil {
			return
		}
		s.mu.Lock()
		if s.latest == nil {
			s.latest = response
		}
		s.mu.Unlock()
		s.l.Infof("restored the topology document generated at %s (%s ago)",
			generatedAt.Format(time.RFC3339), time.Since(generatedAt).Truncate(time.Second))
	})
}

func (s *Service) snapshot() *omv1.GetTopologyResponse {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.latest
}

// withFreshStale returns a shallow copy of response with Snapshot.Stale recomputed against the current time.
// It must not mutate response, which concurrent readers share.
func withFreshStale(response *omv1.GetTopologyResponse) *omv1.GetTopologyResponse {
	if response == nil || response.Snapshot == nil {
		return response
	}
	// Built field by field: copying a proto message by value copies its internal mutex (go vet copylocks).
	snapshot := &omv1.Snapshot{
		GeneratedAt:   response.Snapshot.GeneratedAt,
		ObservedAt:    response.Snapshot.ObservedAt,
		Stale:         snapshotStale(response.Snapshot.ObservedAt),
		SchemaVersion: response.Snapshot.SchemaVersion,
		RunId:         response.Snapshot.RunId,
	}
	return &omv1.GetTopologyResponse{
		Snapshot:      snapshot,
		OriginNode:    response.OriginNode,
		SourceQueries: response.SourceQueries,
		Summary:       response.Summary,
		Environments:  response.Environments,
	}
}

// emptyTopologyResponse is served before any document exists. Stale is true so the UI shows "no data yet"
// rather than what looks like an empty estate.
func emptyTopologyResponse() *omv1.GetTopologyResponse {
	return &omv1.GetTopologyResponse{
		Snapshot: &omv1.Snapshot{
			Stale:         true,
			SchemaVersion: schemaVersion,
		},
		SourceQueries: sourceQueries,
		Summary:       &omv1.Summary{ProcessRoleCounts: map[string]int32{}},
		Environments:  []*omv1.Environment{},
	}
}

// readInventory returns every MongoDB service, all nodes by ID, the PMM Server node name, and how old a volatile
// observation may be.
func (s *Service) readInventory(ctx context.Context) ([]*models.Service, map[string]*models.Node, string, time.Duration, error) {
	var (
		services   []*models.Service
		nodesByID  map[string]*models.Node
		originNode string
		maxAge     time.Duration
	)

	serviceType := models.MongoDBServiceType
	errTX := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		var err error
		services, err = models.FindServices(tx.Querier, models.ServiceFilters{ServiceType: &serviceType})
		if err != nil {
			return err
		}

		settings, err := models.GetSettings(tx)
		if err != nil {
			return err
		}
		maxAge = max(settings.MetricsResolutions.HR*volatileMaxAgeFactor, minVolatileMaxAge)

		nodes, err := models.FindNodes(tx.Querier, models.NodeFilters{})
		if err != nil {
			return err
		}
		nodesByID = make(map[string]*models.Node, len(nodes))
		for _, node := range nodes {
			nodesByID[node.NodeID] = node
			if node.IsPMMServerNode {
				originNode = node.NodeName
			}
		}
		return nil
	})
	if errTX != nil {
		return nil, nil, "", 0, errTX
	}
	return services, nodesByID, originNode, maxAge, nil
}

func buildRun(
	runID string, startedAt, finishedAt time.Time,
	services []*models.Service, merged map[string]map[string]MergedField,
	doc document, results []SourceResult,
) *omv1.TopologyRun {
	counts := &omv1.TopologyRunCounts{TotalServices: int32(len(services))} //nolint:gosec
	for _, service := range services {
		if len(merged[service.ServiceID]) == 0 {
			counts.OrphanedServices++
			continue
		}
		counts.ResolvedServices++
	}
	counts.SuccessfulProbes = doc.summary.UpServices
	counts.StaleServices = int32(doc.staleServices) //nolint:gosec

	run := &omv1.TopologyRun{
		RunId:     runID,
		Status:    omv1.RunStatus_RUN_STATUS_SUCCESS,
		StartTime: timestamppb.New(startedAt),
		EndTime:   timestamppb.New(finishedAt),
		Counts:    counts,
		Sources:   make([]*omv1.SourceReport, 0, len(results)),
		Errors:    []*omv1.TopologyRunError{},
	}

	failed := 0
	for _, result := range results {
		run.Sources = append(run.Sources, &omv1.SourceReport{
			Source: result.Source,
			Status: sourceStatusToProto(result.Status),
			Facts:  int32(len(result.Facts)), //nolint:gosec
			Detail: detailStrings(result.Detail),
		})
		for _, e := range result.Errors {
			run.Errors = append(run.Errors, &omv1.TopologyRunError{
				Scope:       e.Scope,
				ServiceName: optional(e.ServiceName),
				Code:        e.Code,
				Message:     e.Message,
			})
		}
		if result.Status == SourceFailed {
			failed++
		}
	}

	switch {
	case failed > 0 && failed == len(results):
		run.Status = omv1.RunStatus_RUN_STATUS_FAILED
	case len(run.Errors) > 0 || failed > 0:
		run.Status = omv1.RunStatus_RUN_STATUS_PARTIAL
	}
	return run
}

func (s *Service) recordRunOutcome(status omv1.RunStatus) {
	switch status {
	case omv1.RunStatus_RUN_STATUS_SUCCESS:
		s.runsSuccess.Add(1)
	case omv1.RunStatus_RUN_STATUS_PARTIAL:
		s.runsPartial.Add(1)
	case omv1.RunStatus_RUN_STATUS_FAILED:
		s.runsFailed.Add(1)
	default:
		// buildRun only ever sets one of the three above.
	}
}

func detailStrings(detail map[string]any) map[string]string {
	out := make(map[string]string, len(detail))
	for key, value := range detail {
		out[key] = fmt.Sprint(value)
	}
	return out
}

func observedAt(doc document) *timestamppb.Timestamp {
	if doc.observedAt.IsZero() {
		return nil
	}
	return timestamppb.New(doc.observedAt)
}

func isStale(doc document, generatedAt time.Time) bool {
	if doc.observedAt.IsZero() {
		return false
	}
	return generatedAt.Sub(doc.observedAt) > staleAfter
}
