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

// Package pom implements POM -- the PSMDB Open Manager topology and health API.
//
// It reconstructs the MongoDB estate from the sources PMM already owns: the inventory in
// PostgreSQL and the exporter metrics in VictoriaMetrics. PMM stores cluster and
// replication_set as flat string columns on a service and has no topology object; this
// service is what turns those labels back into replica sets and sharded clusters, folds
// in reachability and load, and serves the result as one document.
//
// It is the read half of a split: derivation over data PMM owns lives here, while work
// that has to run on a database host -- collecting argv and installed binary versions,
// restarts, upgrades, configuration changes -- lives in SEP apps driving Nomad clients.
// Those arrive here as another factSource, which is why the merge is by declared
// precedence rather than by which source ran last.
package pom

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"

	pomv1 "github.com/percona/pmm/api/pom/v1"
	"github.com/percona/pmm/managed/models"
)

const (
	// How long a document is served from cache before the next read rebuilds it. Short
	// enough that the page is never meaningfully behind, long enough that a browser
	// refresh does not re-query VictoriaMetrics a dozen times.
	refreshInterval = 30 * time.Second

	// How old the newest observation may be before the document declares itself stale. A
	// stale document is still served -- the UI shows the age and lets the reader judge,
	// which beats replacing data with an error.
	staleAfter = 5 * time.Minute

	// How many runs the in-memory history keeps.
	runHistory = 100

	// What the UI asks for when it passes no limit.
	defaultRunLimit = 25
)

// Run statuses. A run whose sources all answered is a success even if some services were
// never seen: a service that inventory knows and metrics have not is a fact about the
// estate, not a failure of the run.
const (
	runStatusSuccess = "success"
	runStatusPartial = "partial"
	runStatusFailed  = "failed"
)

// Service provides the POM topology API.
type Service struct {
	pomv1.UnimplementedPomServiceServer

	db       *reform.DB
	vmClient victoriaMetricsClient
	l        *logrus.Entry

	// probe is the on-host fact source, or nil when SEP's pom_discovery app is not
	// configured. Held rather than constructed per run so the HTTP client is reused.
	probe *probeSource

	// restored guards the one-time read of the stored document on a cold start.
	restored sync.Once

	// running serialises discovery. One collection at a time is enough, and two
	// concurrent ones would issue the same queries twice and race to publish.
	running sync.Mutex

	mu      sync.Mutex
	latest  *pomv1.GetTopologyResponse
	builtAt time.Time
}

// New returns a new POM service.
func New(db *reform.DB, vmClient victoriaMetricsClient, l *logrus.Entry) *Service {
	return &Service{
		db:       db,
		vmClient: vmClient,
		l:        l,
	}
}

// WithProbeSource attaches SEP's pom_discovery app as a fact source.
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
	probe := &probeSource{
		sepURL: sepURL,
		token:  token,
		client: &http.Client{Timeout: probeRequestTimeout},
		l:      s.l.WithField("source", sourceProbe),
	}
	s.probe = probe
	s.l.Infof("pom_discovery facts at %s", probe.endpoint("facts"))
	return s
}

// GetTopology returns the whole MongoDB estate as one document, rebuilding it when the
// cached one has aged past refreshInterval.
func (s *Service) GetTopology(ctx context.Context, _ *pomv1.GetTopologyRequest) (*pomv1.GetTopologyResponse, error) {
	s.restoreOnce()
	if cached, ok := s.cached(); ok {
		return cached, nil
	}

	response, err := s.discover(ctx)
	if err != nil {
		// Serving a stale document beats serving an error, as long as the envelope says
		// so -- which it does, because stale is computed from observed_at.
		if cached := s.snapshot(); cached != nil {
			s.l.Warnf("serving the cached topology, rebuild failed: %s", err)
			return cached, nil
		}
		return nil, err
	}
	return response, nil
}

// ListDiscoveryRuns returns the recorded runs, newest first.
func (s *Service) ListDiscoveryRuns(_ context.Context, req *pomv1.ListDiscoveryRunsRequest) (*pomv1.ListDiscoveryRunsResponse, error) {
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = defaultRunLimit
	}

	runs, err := s.listRuns(limit)
	if err != nil {
		return nil, err
	}
	return &pomv1.ListDiscoveryRunsResponse{Runs: runs}, nil
}

// GetDiscoveryRun returns one recorded run.
func (s *Service) GetDiscoveryRun(_ context.Context, req *pomv1.GetDiscoveryRunRequest) (*pomv1.GetDiscoveryRunResponse, error) {
	run, err := s.getRun(req.GetRunId())
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "Run %s not found.", req.GetRunId())
		}
		return nil, err
	}
	return &pomv1.GetDiscoveryRunResponse{Run: run}, nil
}

// TriggerDiscovery rebuilds the topology document now and records the run.
//
// Synchronous, unlike SEP's, and it answers with a terminal status: there is no fan-out
// to remote executors to wait on here, so there is nothing to poll for. It refuses
// rather than queues while one is in flight -- two collections would issue the same
// queries twice and race to publish, and the caller wants the answer, not a second run.
func (s *Service) TriggerDiscovery(ctx context.Context, _ *pomv1.TriggerDiscoveryRequest) (*pomv1.TriggerDiscoveryResponse, error) {
	if !s.running.TryLock() {
		// Aborted, not FailedPrecondition: the gateway maps it to 409 Conflict, which is
		// what the UI's Sync button already treats as an expected outcome rather than a
		// failure.
		return nil, status.Error(codes.Aborted, "A discovery run is already in flight.")
	}
	defer s.running.Unlock()

	_, run, err := s.collect(ctx)
	if err != nil {
		return nil, err
	}
	return &pomv1.TriggerDiscoveryResponse{
		RunId:     run.RunId,
		Status:    run.Status,
		StartedAt: run.StartedAt,
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
			_, err := s.discover(ctx)
			if err != nil && ctx.Err() == nil {
				s.l.Warnf("scheduled discovery failed: %s", err)
			}
		}
	}
}

// discover rebuilds unless another caller is already doing it, in which case it waits for
// that one and serves what it published. Two collections of the same estate at the same
// moment differ only in cost.
func (s *Service) discover(ctx context.Context) (*pomv1.GetTopologyResponse, error) {
	if !s.running.TryLock() {
		s.running.Lock()
		s.running.Unlock() //nolint:staticcheck // waiting out the in-flight run, not guarding
		if cached := s.snapshot(); cached != nil {
			return cached, nil
		}
		return nil, status.Error(codes.Unavailable, "Discovery is in flight and no document is available yet.")
	}
	defer s.running.Unlock()

	response, _, err := s.collect(ctx)
	return response, err
}

// collect reads every source, merges, builds the document and records the run. The caller
// must hold s.running.
func (s *Service) collect(ctx context.Context) (*pomv1.GetTopologyResponse, *pomv1.Run, error) {
	startedAt := time.Now()
	runID := uuid.New().String()

	services, nodes, originNode, maxAge, err := s.readInventory()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read inventory: %w", err)
	}

	sources := []factSource{
		inventorySource{nodes: nodes},
		metricsSource{vm: s.vmClient, l: s.l, now: startedAt},
	}
	if s.probe != nil {
		sources = append(sources, *s.probe)
	}
	results := make([]SourceResult, 0, len(sources))
	for _, source := range sources {
		results = append(results, source.collect(ctx, services))
	}

	merged := mergeFacts(results, defaultPrecedence)
	generatedAt := time.Now()
	doc := buildDocument(services, merged, generatedAt, maxAge)

	response := &pomv1.GetTopologyResponse{
		Snapshot: &pomv1.Snapshot{
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

	s.mu.Lock()
	s.latest, s.builtAt = response, generatedAt
	s.mu.Unlock()

	// Recorded after publishing, and never fatal: the document is already correct and
	// already served, so losing the record of a collection is worth less than refusing to
	// answer with it.
	err = s.persist(response, run, originNode)
	if err != nil {
		s.l.Warnf("run %s: failed to record: %s", runID, err)
	}

	s.l.Infof("run %s: %d service(s), %d up, %d cluster(s), %d stale, %s max age, status %s",
		runID, doc.summary.ServicesTotal, doc.summary.ServicesUp, doc.summary.Clusters,
		doc.staleServices, maxAge, run.Status)

	return response, run, nil
}

// restoreOnce loads the newest stored document into the cache the first time it is
// needed, so a restarted pmm-managed answers from the last known estate rather than
// starting from nothing.
//
// It is restored with the age it actually has, not as if it were just collected, so it
// then obeys the same cache rule as anything else: young enough and it is served, too old
// and the read that wanted it rebuilds. Either way there is now something to fall back on
// when a rebuild fails, which is the case that made this worth storing.
func (s *Service) restoreOnce() {
	s.restored.Do(func() {
		response, generatedAt, err := s.restore()
		if err != nil {
			s.l.Warnf("failed to restore the stored topology: %s", err)
			return
		}
		if response == nil {
			return
		}
		s.mu.Lock()
		if s.latest == nil {
			s.latest, s.builtAt = response, generatedAt
		}
		s.mu.Unlock()
		s.l.Infof("restored the topology document generated at %s (%s ago)",
			generatedAt.Format(time.RFC3339), time.Since(generatedAt).Truncate(time.Second))
	})
}

// cached returns the published document while it is younger than refreshInterval.
func (s *Service) cached() (*pomv1.GetTopologyResponse, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.latest != nil && time.Since(s.builtAt) < refreshInterval {
		return s.latest, true
	}
	return nil, false
}

// snapshot returns the published document, however old.
func (s *Service) snapshot() *pomv1.GetTopologyResponse {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.latest
}

// readInventory returns every MongoDB service, its nodes by ID, the PMM Server node name
// to record as the document's vantage point, and how old a volatile observation may be.
func (s *Service) readInventory() ([]*models.Service, map[string]*models.Node, string, time.Duration, error) {
	var (
		services   []*models.Service
		nodesByID  map[string]*models.Node
		originNode string
		maxAge     time.Duration
	)

	serviceType := models.MongoDBServiceType
	errTX := s.db.InTransaction(func(tx *reform.TX) error {
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
// The services_resolved versus probes_ok pair is the diagnostic split: the first says a source
// produced facts for the service at all, the second says it was observed as reachable. A
// run with resolved=9, probes_ok=0 is a healthy join and nine unreachable databases.
func buildRun(
	runID string, startedAt, finishedAt time.Time,
	services []*models.Service, merged map[string]map[string]MergedField,
	doc document, results []SourceResult,
) *pomv1.Run {
	counts := &pomv1.RunCounts{ServicesTotal: int32(len(services))} //nolint:gosec
	for _, service := range services {
		if len(merged[service.ServiceID]) == 0 {
			counts.ServicesOrphaned++
			continue
		}
		counts.ServicesResolved++
	}
	counts.ProbesOk = doc.summary.ServicesUp
	counts.ServicesStale = int32(doc.staleServices) //nolint:gosec

	run := &pomv1.Run{
		RunId:      runID,
		Status:     runStatusSuccess,
		StartedAt:  timestamppb.New(startedAt),
		FinishedAt: timestamppb.New(finishedAt),
		Counts:     counts,
		Sources:    make([]*pomv1.SourceReport, 0, len(results)),
		Errors:     []*pomv1.RunError{},
	}

	failed := 0
	for _, result := range results {
		run.Sources = append(run.Sources, &pomv1.SourceReport{
			Source: result.Source,
			Status: string(result.Status),
			Facts:  int32(len(result.Facts)), //nolint:gosec
			Detail: detailStrings(result.Detail),
		})
		for _, e := range result.Errors {
			run.Errors = append(run.Errors, &pomv1.RunError{
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
		run.Status = runStatusFailed
	case len(run.Errors) > 0 || failed > 0:
		run.Status = runStatusPartial
	}
	return run
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
