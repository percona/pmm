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

// Package om implements OM -- the OpenManager topology and health API.
//
// It groups the MongoDB estate from the sources PMM already owns: the inventory in
// PostgreSQL and the exporter metrics in VictoriaMetrics. PMM stores cluster and
// replication_set as flat string columns on a service and has no topology object; this
// service groups services that share a label, folds in reachability and load, and serves
// the result as one document. That is a grouped service inventory, not a reconstructed
// topology graph: a sharded cluster here is a flat list of mongos, configsvr and shardsvr
// rows that happen to share a label, not a shard map with a discovered member list. See
// ClusterType in api/om/v1/om.proto for the one thing inferred about a group's shape.
//
// It is the read half of a split: derivation over data PMM owns lives here, while work
// that has to run on a database host -- collecting argv and installed binary versions,
// restarts, upgrades, configuration changes -- lives in SEP apps driving Nomad clients.
// Those arrive here as another factSource, which is why the merge is by declared
// precedence rather than by which source ran last.
package om

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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
	// How often the leader's own ticker rebuilds the document. Collection is
	// leader-driven, not request-driven -- see Run and GetTopology -- so this paces the
	// ticker rather than gating a read.
	refreshInterval = 30 * time.Second

	// How old the newest observation may be before the document declares itself stale. A
	// stale document is still served -- the UI shows the age and lets the reader judge,
	// which beats replacing data with an error.
	staleAfter = 5 * time.Minute

	// How many runs the in-memory history keeps.
	runHistory = 100

	// What the UI asks for when it passes no limit, and the most a caller may ask for.
	// The ceiling is runHistory: asking for more than the history keeps cannot return
	// more rows, so a larger number is only a larger query.
	defaultRunLimit = 25
	maxRunLimit     = runHistory
)

// Run statuses, as the run row stores them. A run whose sources all answered is a success
// even if some services were never seen: a service that inventory knows and metrics have
// not is a fact about the estate, not a failure of the run.
//
// Kept as strings because they are persisted. The wire carries omv1.RunStatus instead, and
// runStatusToProto in store.go is the only crossing -- a stored row stays legible in psql
// and survives a renumbering of the enum.
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

	// ha reports leadership, so TriggerTopologyCollection can refuse on a follower. Nil
	// in most tests, which is read as "every node is the leader" -- the single-node
	// default.
	ha haChecker

	// probe is the on-host fact source, or nil when SEP's om_inventory app is not
	// configured. Held rather than constructed per run so the HTTP client is reused.
	probe *probeSource

	// agents reports pmm-agent connectivity for ListInventoryHosts' eligibility
	// computation, or nil when not wired up (every host then reads as
	// pmm_agent_connected: false -- see agentConnectionChecker's doc comment).
	agents agentConnectionChecker

	// restored guards the one-time read of the stored document on a cold start.
	restored sync.Once

	// running serialises collection. One collection at a time is enough, and two
	// concurrent ones would issue the same queries twice and race to publish.
	running sync.Mutex

	mu     sync.Mutex
	latest *omv1.GetTopologyResponse

	// Completed-run tally by status, read by MetricsCollector on every Prometheus scrape
	// and written once per run by recordRunOutcome. Atomic because a scrape can land while
	// a collection is in flight.
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

// WithProbeSource attaches SEP's om_inventory app as a fact source.
//
// Takes where SEP is, not where the app is: the app path is this package's business,
// and the sources that follow -- upgrade, restart -- will hang off the same SEP.
//
// Optional by design: an empty URL leaves the source off, and the document is built
// from PMM's own inventory and metrics alone. That is the difference between "no probe
// has run here" and "the probe failed", and the run receipt reports which.
func (s *Service) WithProbeSource(sepURL, token string) *Service {
	if sepURL == "" {
		s.l.Info("SEP is not configured; on-host facts will be absent")
		return s
	}
	client := &sepClient{
		baseURL: sepURL,
		token:   token,
		http:    &http.Client{Timeout: probeRequestTimeout},
	}
	probe := &probeSource{
		app: client.app(probeAppModule),
		l:   s.l.WithField("source", sourceProbe),
	}
	s.probe = probe
	s.l.Infof("om_inventory estate at %s", probe.app.endpoint(""))
	return s
}

// WithAgentRegistry attaches the pmm-agent connectivity checker ListInventoryHosts
// uses to compute automation eligibility.
//
// Optional, matching WithProbeSource's shape: unset in most tests, and in any build
// that has not wired one up, every host then reads as not connected. See
// agentConnectionChecker's doc comment for why that is the fail-closed default rather
// than treating an unknown state as eligible.
func (s *Service) WithAgentRegistry(r agentConnectionChecker) *Service {
	s.agents = r
	return s
}

// Enabled returns true if OpenManager is enabled, so every /v1/om/* RPC and the
// scheduled collection in Run refuse while it is off, via the same generic
// gRPC-service-enabled interceptor BackupService and the other preview features use.
func (s *Service) Enabled() bool {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		s.l.WithError(err).Error("can't get settings")
		return false
	}
	return settings.IsOMEnabled()
}

// IsAvailable reports whether SEP's om_inventory app is configured and reachable.
//
// Used to gate turning OpenManager on: an admin flipping the switch with no inventory
// app to talk to would enable a UI backed by a source that can never answer, with no
// way to tell "off" from "broken" apart from reading logs. It does not drive anything
// on SEP's side -- this is the same read every scheduled collection already performs
// via probeSource.collect, just run once up front rather than waited out.
func (s *Service) IsAvailable(ctx context.Context) bool {
	if s.probe == nil || s.probe.app.client == nil {
		return false
	}
	_, err := s.probe.fetch(ctx)
	return err == nil
}

// SyncInventoryEnabled tells SEP's om_inventory app whether OpenManager is on, and
// on enabling, kicks an immediate sweep instead of leaving the estate to wait out
// SCHEDULE's own interval.
//
// PATCHes ENABLED rather than SCHEDULE: the app keeps its own configured cadence
// (an operator's SCHEDULE override) independent of whether OpenManager is turned
// on, so toggling this switch off and back on does not reset a customized interval
// back to the app's default. See OmInventorySettings in SEP for the other half.
//
// The immediate sweep exists because a freshly (re-)enabled periodic task in SEP's
// beat store is not due until one full SCHEDULE interval has elapsed -- there is no
// "run once now, then repeat" concept in an interval schedule, so a 60-minute
// cadence would otherwise leave the estate empty for up to an hour after being
// turned on. Mirrors triggerOMCollectionIfJustEnabled, PMM's own equivalent kick for
// its topology page.
//
// Both calls are best-effort: a stale write, or a sweep that does not fire, means
// SEP is briefly out of step with PMM's switch, not a broken settings change, so
// failure is logged rather than returned to the caller -- matching
// triggerOMCollectionIfJustEnabled, the other side effect ChangeSettings fires on
// this same transition.
func (s *Service) SyncInventoryEnabled(ctx context.Context, enabled bool) {
	if s.probe == nil || s.probe.app.client == nil {
		return
	}
	err := s.probe.app.patchConfig(ctx, map[string]any{"ENABLED": enabled})
	if err != nil {
		s.l.WithError(err).WithField("enabled", enabled).
			Warn("failed to sync OpenManager's on/off state to SEP's om_inventory app")
		return
	}
	if !enabled {
		return
	}
	err = s.probe.app.triggerRun(ctx)
	if err != nil {
		s.l.WithError(err).Warn("failed to trigger an immediate SEP inventory sweep after enabling OpenManager")
	}
}

// GetTopology returns the whole MongoDB estate as one document.
//
// A pure read path: memory, then the stored snapshot, never a collection. Collection is
// leader-only (Run, TriggerTopologyCollection), and a request-triggered rebuild here
// would make every follower behind the load balancer a writer and a pruner racing the
// leader. Staleness is recomputed against the clock on every call rather than trusted
// from whenever the document was built or restored, so a follower that never collects
// still reports its document's true age.
func (s *Service) GetTopology(ctx context.Context, _ *omv1.GetTopologyRequest) (*omv1.GetTopologyResponse, error) { //nolint:unparam
	s.restoreOnce(ctx)
	if cached := s.snapshot(); cached != nil {
		return withFreshStale(cached), nil
	}
	// Genuinely cold: no in-memory document and nothing stored yet, e.g. a fresh estate
	// whose leader has not completed its first pass. An empty document with stale=true
	// is the honest answer -- Unavailable would read as an error for a normal startup
	// state, and blocking for the leader's first run would reintroduce the
	// request-triggered wait this path exists to remove.
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
// Synchronous, unlike SEP's, and it answers with a terminal status: there is no fan-out
// to remote executors to wait on here, so there is nothing to poll for. It refuses
// rather than queues while one is in flight -- two collections would issue the same
// queries twice and race to publish, and the caller wants the answer, not a second run.
//
// Refuses on a non-leader for the same reason: a follower that collected would persist a
// run and prune the shared history right alongside the leader's own pass.
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
		// Aborted, not FailedPrecondition: the gateway maps it to 409 Conflict, which is
		// what the UI's Sync button already treats as an expected outcome rather than a
		// failure.
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

// Run refreshes the document on a timer until ctx is cancelled.
//
// Collection is driven rather than left to whoever happens to read: the run history is
// only worth having if it exists when nobody is looking, and a document assembled purely
// on demand can say nothing about the interval since the last one.
func (s *Service) Run(ctx context.Context) {
	ticker := time.NewTicker(refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
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

// discover rebuilds unless another caller is already doing it, in which case it waits for
// that one and serves what it published. Two collections of the same estate at the same
// moment differ only in cost.
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

// collect reads every source, merges, builds the document and records the run. The caller
// must hold s.running.
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

	// Recorded after publishing, and never fatal: the document is already correct and
	// already served, so losing the record of a collection is worth less than refusing to
	// answer with it.
	err = s.persist(ctx, response, run, originNode)
	if err != nil {
		s.l.Warnf("run %s: failed to record: %s", runID, err)
	}

	s.l.Infof("run %s: %d service(s), %d up, %d cluster(s), %d stale, %s max age, status %s",
		runID, doc.summary.TotalServices, doc.summary.UpServices, doc.summary.Clusters,
		doc.staleServices, maxAge, run.Status)

	return response, run, nil
}

// sources returns the fact sources one collection reads, in the order they run.
//
// Extracted so a test can see the set and its order without driving a whole collection:
// which sources are wired in, and whether probe is included, is exactly what review
// found untested. Both nodes and startedAt are per-run state the sources close over
// rather than package globals, so two overlapping calls (the ticker and a manual
// trigger racing, however briefly) never share one.
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

// restoreOnce loads the newest stored document into memory the first time it is needed,
// so a restarted pmm-managed answers from the last known estate rather than nothing.
//
// Only "is there a document at all" is decided here. How fresh it is remains a question
// the read path answers itself, every time it is asked -- see withFreshStale -- rather
// than baking the answer in as of the moment this ran.
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

// snapshot returns the published document, however old.
func (s *Service) snapshot() *omv1.GetTopologyResponse {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.latest
}

// withFreshStale returns response with Snapshot.Stale recomputed against the current
// time, leaving the rest of the message untouched.
//
// A document served from memory can sit there for a long time on a node that never
// collects -- every follower, once GetTopology stops triggering a collection of its own
// -- so staleness has to be a property of when it is read, not frozen as of whenever it
// was built or restored. Response is shared across concurrent readers, so this returns a
// shallow copy rather than mutating it in place.
func withFreshStale(response *omv1.GetTopologyResponse) *omv1.GetTopologyResponse {
	if response == nil || response.Snapshot == nil {
		return response
	}
	// Built field by field rather than copied (`clone := *response`): a proto message
	// carries a sync.Mutex in its generated MessageState, and copying that by value is
	// exactly the bug go vet's copylocks check exists to catch.
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

// emptyTopologyResponse is what GetTopology answers on a genuinely cold estate: nothing
// in memory, nothing stored yet. That is the normal state for the first moments of a
// fresh install, not an error -- observed_at stays unset and stale is true so the UI can
// render "collecting, no data yet" rather than a blank document that looks like an empty
// estate.
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

// readInventory returns every MongoDB service, its nodes by ID, the PMM Server node name
// to record as the document's vantage point, and how old a volatile observation may be.
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

		// The freshness rule follows whatever PMM is actually scraping at, rather than
		// hardcoding an interval that a settings change would silently invalidate.
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

// buildRun records what one pass saw.
//
// The resolved_services versus successful_probes pair is the diagnostic split: the first says a
// source produced facts for the service at all, the second says it was observed as reachable.
// A run with resolved=9, successful=0 is a healthy join and nine unreachable databases.
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

// recordRunOutcome tallies one completed run by status, for MetricsCollector's
// pmm_managed_om_runs_total counter.
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

// detailStrings renders a source's counters for the wire, which carries them as strings
// so a source can report whatever it has without the contract naming every counter.
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
