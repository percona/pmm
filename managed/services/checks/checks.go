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

// Package checks provides advisor checks functionality.
package checks

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"text/template"
	"time"

	"github.com/google/uuid"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/pi/check"
	"github.com/percona/pmm/managed/pi/common"
	"github.com/percona/pmm/managed/services"
	"github.com/percona/pmm/utils/pdeathsig"
	"github.com/percona/pmm/utils/sqlrows"
	"github.com/percona/pmm/version"
)

const (
	defaultStartDelay = time.Minute

	// Environment variables that affect checks service; only for testing.
	envDisableStartDelay = "PMM_ADVISORS_CHECKS_DISABLE_START_DELAY"
	builtinChecksPath    = "/usr/local/percona/checks"

	checkExecutionTimeout  = 5 * time.Minute  // limits execution time for every single check
	resultAwaitTimeout     = 20 * time.Second // should be greater than agents.defaultQueryActionTimeout
	scriptExecutionTimeout = 5 * time.Second  // time limit for running pmm-managed-starlark
	resultCheckInterval    = time.Second

	prometheusNamespace = "pmm_managed"
	prometheusSubsystem = "advisor"
)

// pmm-agent versions with known changes in Query Actions.
// To match all pre-release versions, add a '-0' suffix to the specified version.
var (
	pmmAgent3_0_0   = version.MustParse("3.0.0-0")
	pmmAgentInvalid = version.MustParse("3.0.0-invalid")

	b64 = base64.StdEncoding
)

// Service is responsible for interactions with Percona Check service.
type Service struct {
	agentsRegistry  agentsRegistry
	db              *reform.DB
	resultsRegistry *registry
	vmClient        v1.API
	clickhouseDB    *sql.DB

	l          *logrus.Entry
	startDelay time.Duration

	// startCheckCh delivers on-demand check runs from StartChecks to
	// runChecksLoop, which owns the service lifecycle context.
	startCheckCh chan checkRunRequest

	am       sync.Mutex
	advisors []check.Advisor
	checks   map[string]check.Check // Checks extracted from advisors and stored by name.

	tm             sync.Mutex
	rareTicker     *time.Ticker
	standardTicker *time.Ticker
	frequentTicker *time.Ticker

	mChecksExecuted      *prom.CounterVec
	mChecksAvailable     *prom.GaugeVec
	mChecksExecutionTime *prom.SummaryVec
}

// queryPlaceholders contain known fields that can be used as placeholders in a check's query.
type queryPlaceholders struct {
	ServiceID   string
	ServiceName string
	NodeName    string
}

// New returns Service with given PMM version.
func New(
	db *reform.DB,
	agentsRegistry agentsRegistry,
	vmClient v1.API,
	clickhouseDB *sql.DB,
) *Service {
	l := logrus.WithField("component", "checks")

	s := &Service{
		db:              db,
		agentsRegistry:  agentsRegistry,
		resultsRegistry: newRegistry(),
		vmClient:        vmClient,
		clickhouseDB:    clickhouseDB,

		l:            l,
		startDelay:   defaultStartDelay,
		startCheckCh: make(chan checkRunRequest, 1),

		mChecksExecuted: prom.NewCounterVec(prom.CounterOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "checks_executed_total",
			Help:      "Number of check scripts executed per service type, advisor and check name",
		}, []string{"service_type", "advisor", "check_name", "status"}),

		mChecksAvailable: prom.NewGaugeVec(prom.GaugeOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "checks_available",
			Help:      "Number of checks loaded in PMM per service type, advisor and check name",
		}, []string{"service_type", "advisor", "check_name"}),

		mChecksExecutionTime: prom.NewSummaryVec(prom.SummaryOpts{
			Namespace:  prometheusNamespace,
			Subsystem:  prometheusSubsystem,
			Name:       "check_execution_time_seconds",
			Help:       "Time taken to execute checks per service type, advisor, and check name",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001}, //nolint:mnd
		}, []string{"service_type", "advisor", "check_name"}),
	}

	if d, _ := strconv.ParseBool(os.Getenv(envDisableStartDelay)); d {
		l.Warn("Start delay disabled.")
		s.startDelay = 0
	}

	return s
}

// Run runs main service loops.
func (s *Service) Run(ctx context.Context) {
	s.l.Info("Starting...")
	defer s.l.Info("Done.")

	err := s.reconcileBuiltinChecks(ctx)
	if err != nil {
		// Keep going with whatever is already stored in the DB.
		s.l.Errorf("Failed to reconcile built-in checks: %+v.", err)
	}
	s.UpdateAdvisorsList(ctx)
	settings, err := models.GetSettings(s.db)
	if err != nil {
		s.l.Errorf("Failed to get settings: %+v.", err)
		return
	}

	s.tm.Lock()
	s.rareTicker = time.NewTicker(settings.SaaS.AdvisorRunIntervals.RareInterval)
	s.standardTicker = time.NewTicker(settings.SaaS.AdvisorRunIntervals.StandardInterval)
	s.frequentTicker = time.NewTicker(settings.SaaS.AdvisorRunIntervals.FrequentInterval)
	s.tm.Unlock()

	defer s.rareTicker.Stop()
	defer s.standardTicker.Stop()
	defer s.frequentTicker.Stop()

	// delay for the first run to allow all agents to connect
	startCtx, startCancel := context.WithTimeout(ctx, s.startDelay)
	<-startCtx.Done()
	startCancel()
	if ctx.Err() != nil { // check the main context, not the startCtx
		return
	}

	var wg sync.WaitGroup

	wg.Go(func() {
		s.runChecksLoop(ctx)
	})

	wg.Wait()
}

// runChecksLoop starts checks execution loop.
func (s *Service) runChecksLoop(ctx context.Context) {
	// First checks run, start all checks from all groups.
	err := s.runChecksGroup(ctx, "") // start all checks
	for {
		if err != nil {
			if errors.Is(err, services.ErrAdvisorsDisabled) {
				s.l.Info("Advisor checks are not enabled, doing nothing.")
			} else {
				s.l.Error(err)
			}
		}

		select {
		case <-ctx.Done():
			return
		case req := <-s.startCheckCh:
			// On-demand run requested via StartChecks.
			s.UpdateAdvisorsList(ctx)
			err = s.run(ctx, "", req.checkNames, req.ri)
		case <-s.rareTicker.C:
			// Start all checks from rare group.
			err = s.runChecksGroup(ctx, check.Rare)
		case <-s.standardTicker.C:
			// Start all checks from standard group.
			err = s.runChecksGroup(ctx, check.Standard)
		case <-s.frequentTicker.C:
			// Start all checks from frequent group.
			err = s.runChecksGroup(ctx, check.Frequent)
		}
	}
}

// GetInsights returns Advisor insights matching the filters,
// together with the total number of matching rows (ignoring pagination).
func (s *Service) GetInsights(ctx context.Context, filters models.InsightFilters, pageIndex, pageSize int) ([]*models.Insight, int, error) {
	results, err := models.FindInsights(ctx, s.db.Querier, filters, pageIndex, pageSize)
	if err != nil {
		return nil, 0, err
	}

	total, err := models.CountInsights(ctx, s.db.Querier, filters)
	if err != nil {
		return nil, 0, err
	}

	return results, total, nil
}

// GetInsightsFilterValues returns the distinct service and node names present in the
// Advisor insights.
func (s *Service) GetInsightsFilterValues(ctx context.Context) ([]string, []string, error) {
	return models.FindInsightFilterValues(ctx, s.db.Querier)
}

// MarkInsightsRead sets the read state on the insights with the given IDs.
func (s *Service) MarkInsightsRead(ctx context.Context, ids []string, isRead bool) error {
	return models.MarkInsightsRead(ctx, s.db.Querier, ids, isRead)
}

// MarkInsightsReadByFilters sets the read state on all insights matching the filters.
func (s *Service) MarkInsightsReadByFilters(ctx context.Context, filters models.InsightFilters, isRead bool) error {
	return models.MarkInsightsReadByFilters(ctx, s.db.Querier, filters, isRead)
}

// runChecksGroup downloads and executes Advisors checks that should run in the interval specified by intervalGroup.
// All checks are executed if intervalGroup is empty.
func (s *Service) runChecksGroup(ctx context.Context, intervalGroup check.Interval) error {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		return err
	}

	if !settings.IsAdvisorsEnabled() {
		return services.ErrAdvisorsDisabled
	}

	s.UpdateAdvisorsList(ctx)
	ri := runInfo{batchID: uuid.NewString(), triggeredBy: models.CheckTriggeredByScheduler}
	return s.run(ctx, intervalGroup, nil, ri)
}

// StartChecks downloads and executes advisor checks in asynchronous way and returns the batch ID.
// If checkNames specified then only matched checks will be executed.
func (s *Service) StartChecks(checkNames []string) (string, error) {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		return "", err
	}

	if !settings.IsAdvisorsEnabled() {
		return "", services.ErrAdvisorsDisabled
	}

	ri := runInfo{batchID: uuid.NewString(), triggeredBy: models.CheckTriggeredByUser}

	// Hand the request off to runChecksLoop, which owns the service lifecycle
	// context. The loop only runs on the leader node, so a non-blocking send
	// drops the request where there is nothing to execute it.
	select {
	case s.startCheckCh <- checkRunRequest{checkNames: checkNames, ri: ri}:
	default:
		s.l.Warn("Advisor checks run is already pending, skipping the request.")
	}

	return ri.batchID, nil
}

// runInfo identifies a single Advisor checks execution batch.
type runInfo struct {
	batchID     string
	triggeredBy models.CheckTriggeredBy
}

// checkRunRequest is an on-demand run handed from StartChecks to runChecksLoop.
type checkRunRequest struct {
	checkNames []string
	ri         runInfo
}

func (s *Service) run(ctx context.Context, intervalGroup check.Interval, checkNames []string, ri runInfo) error {
	err := intervalGroup.Validate()
	if err != nil {
		return err
	}

	res, err := s.executeChecks(ctx, intervalGroup, checkNames, ri)
	if err != nil {
		return err
	}

	switch {
	case len(checkNames) != 0:
		// If we run some specific checks, delete previous results for them.
		s.resultsRegistry.deleteByName(checkNames)
	case intervalGroup != "":
		// If we run whole interval group, delete previous results for that group.
		s.resultsRegistry.deleteByInterval(intervalGroup)
	default:
		// If we run all checks, delete all previous results.
		s.resultsRegistry.cleanup()
	}

	s.resultsRegistry.set(res)

	// Best-effort: email the completed batch to the configured Advisor contact point.
	s.maybeSendAdvisorNotification(ctx, ri.batchID, ri.triggeredBy)

	return nil
}

// CleanupCheckResults drops all check results in the registry.
func (s *Service) CleanupCheckResults() {
	s.resultsRegistry.cleanup()
}

// GetAdvisors returns all available advisors.
func (s *Service) GetAdvisors() ([]check.Advisor, error) {
	s.am.Lock()
	defer s.am.Unlock()

	res := make([]check.Advisor, 0, len(s.advisors))
	res = append(res, s.advisors...)
	return res, nil
}

// GetChecks retrieves a map of checks from the service.
func (s *Service) GetChecks() (map[string]check.Check, error) {
	s.am.Lock()
	defer s.am.Unlock()

	res := make(map[string]check.Check, len(s.checks))
	for _, c := range s.checks {
		res[c.Name] = c
	}

	return res, nil
}

// GetDisabledChecks returns the names of globally-disabled checks.
func (s *Service) GetDisabledChecks(ctx context.Context) ([]string, error) {
	return models.FindDisabledAdvisorCheckNames(ctx, s.db.Querier)
}

// GetDisabledServicesForChecks returns a map of check name to the service IDs
// for which that check is disabled.
func (s *Service) GetDisabledServicesForChecks(ctx context.Context) (map[string][]string, error) {
	return models.FindAdvisorCheckDisabledServices(ctx, s.db.Querier)
}

// DisableChecks disables checks with provided names.
func (s *Service) DisableChecks(ctx context.Context, checkNames []string) error {
	err := s.setChecksDisabled(ctx, checkNames, true)
	if err != nil {
		return fmt.Errorf("failed to disable checks: %w", err)
	}

	return nil
}

// EnableChecks enables checks with provided names.
func (s *Service) EnableChecks(ctx context.Context, checkNames []string) error {
	err := s.setChecksDisabled(ctx, checkNames, false)
	if err != nil {
		return fmt.Errorf("failed to enable checks: %w", err)
	}

	return nil
}

// setChecksDisabled sets the global disabled flag for the named checks.
// Per-service disable settings are left untouched, so they still apply once
// a check is re-enabled globally.
func (s *Service) setChecksDisabled(ctx context.Context, checkNames []string, disabled bool) error {
	if len(checkNames) == 0 {
		return nil
	}

	checks, err := s.GetChecks()
	if err != nil {
		return err
	}

	for _, c := range checkNames {
		if _, ok := checks[c]; !ok {
			return fmt.Errorf("unknown check %s", c)
		}
	}

	return s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		return models.SetAdvisorChecksDisabled(ctx, tx.Querier, checkNames, disabled)
	})
}

// DisableChecksForServices disables a check for the given services, keeping it
// enabled elsewhere. It is not allowed for globally-disabled checks.
func (s *Service) DisableChecksForServices(ctx context.Context, checkName string, serviceIDs []string) error {
	if len(serviceIDs) == 0 {
		return nil
	}

	errTx := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		c, err := models.FindAdvisorCheckByName(tx.WithContext(ctx), checkName)
		if err != nil {
			if errors.Is(err, reform.ErrNoRows) {
				return status.Errorf(codes.NotFound, "advisor check '%s' not found", checkName)
			}
			return err
		}

		if c.Disabled {
			return status.Errorf(codes.FailedPrecondition,
				"advisor check '%s' is disabled globally; enable it globally to manage per-service settings", checkName)
		}

		services, err := models.FindServicesByIDs(tx.WithContext(ctx), serviceIDs)
		if err != nil {
			return err
		}
		for _, id := range serviceIDs {
			if _, ok := services[id]; !ok {
				return status.Errorf(codes.NotFound, "service with ID '%s' not found", id)
			}
		}

		ids, err := c.GetDisabledServiceIDs()
		if err != nil {
			return err
		}

		existing := make(map[string]struct{}, len(ids))
		for _, id := range ids {
			existing[id] = struct{}{}
		}
		for _, id := range serviceIDs {
			if _, ok := existing[id]; !ok {
				existing[id] = struct{}{}
				ids = append(ids, id)
			}
		}

		_, err = models.ChangeAdvisorCheckDisabledServices(ctx, tx.Querier, checkName, ids)
		return err
	})
	if errTx != nil {
		return errTx
	}

	s.l.Infof("Disabled check %s for services: %s.", checkName, strings.Join(serviceIDs, ", "))
	return nil
}

// EnableChecksForServices removes the per-service disable of a check for the
// given services. IDs of already-removed services are accepted so stale
// entries can always be cleaned up.
func (s *Service) EnableChecksForServices(ctx context.Context, checkName string, serviceIDs []string) error {
	if len(serviceIDs) == 0 {
		return nil
	}

	errTx := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		c, err := models.FindAdvisorCheckByName(tx.WithContext(ctx), checkName)
		if err != nil {
			if errors.Is(err, reform.ErrNoRows) {
				return status.Errorf(codes.NotFound, "advisor check '%s' not found", checkName)
			}
			return err
		}

		ids, err := c.GetDisabledServiceIDs()
		if err != nil {
			return err
		}

		remove := make(map[string]struct{}, len(serviceIDs))
		for _, id := range serviceIDs {
			remove[id] = struct{}{}
		}
		kept := make([]string, 0, len(ids))
		for _, id := range ids {
			if _, ok := remove[id]; !ok {
				kept = append(kept, id)
			}
		}

		_, err = models.ChangeAdvisorCheckDisabledServices(ctx, tx.Querier, checkName, kept)
		return err
	})
	if errTx != nil {
		return errTx
	}

	s.l.Infof("Enabled check %s for services: %s.", checkName, strings.Join(serviceIDs, ", "))
	return nil
}

// ChangeInterval changes a check's interval to the value received from the UI.
func (s *Service) ChangeInterval(ctx context.Context, params map[string]check.Interval) error {
	checks, err := s.GetChecks()
	if err != nil {
		return err
	}

	for name := range params {
		_, ok := checks[name]
		if !ok {
			return fmt.Errorf("check: %s not found", name)
		}
	}

	errTx := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		for name, interval := range params {
			_, err := models.ChangeAdvisorCheckInterval(ctx, tx.Querier, name, models.Interval(interval))
			if err != nil {
				return err
			}
			s.l.Infof("Updated check: %s, interval changed from: %s to: %s", name, checks[name].Interval, interval)
		}
		return nil
	})
	if errTx != nil {
		return errTx
	}

	// refresh the in-memory checks so the new effective intervals apply immediately
	s.UpdateAdvisorsList(ctx)
	return nil
}

// CreateAdvisorCheck creates a new user-authored advisor check and reloads the check list.
func (s *Service) CreateAdvisorCheck(ctx context.Context, c check.Check) error {
	c.Version = check.MaxSupportedVersion
	c.UserDefined = true

	err := c.Validate()
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid advisor check: %v", err)
	}

	// the reserved prefix keeps user checks from colliding with current or
	// future Percona-shipped check names
	if !strings.HasPrefix(c.Name, check.UserCheckNamePrefix) {
		return status.Errorf(codes.InvalidArgument,
			"user check name must start with '%s'", check.UserCheckNamePrefix)
	}

	existing, err := s.GetChecks()
	if err != nil {
		return err
	}
	if _, ok := existing[c.Name]; ok {
		return status.Errorf(codes.AlreadyExists, "advisor check '%s' already exists", c.Name)
	}

	m, err := userCheckToModel(c)
	if err != nil {
		return err
	}

	errTx := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		_, err := models.CreateAdvisorCheck(tx.Querier, m)
		return err
	})
	if errTx != nil {
		return fmt.Errorf("failed to create advisor check: %w", errTx)
	}

	s.UpdateAdvisorsList(ctx)
	return nil
}

// UpdateAdvisorCheck updates an existing user-authored advisor check and reloads the check list.
func (s *Service) UpdateAdvisorCheck(ctx context.Context, c check.Check) error {
	c.Version = check.MaxSupportedVersion
	c.UserDefined = true

	err := c.Validate()
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid advisor check: %v", err)
	}

	err = s.ensureUserCheck(c.Name)
	if err != nil {
		return err
	}

	m, err := userCheckToModel(c)
	if err != nil {
		return err
	}

	errTx := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		_, err := models.UpdateAdvisorCheck(tx.Querier, m)
		return err
	})
	if errTx != nil {
		return fmt.Errorf("failed to update advisor check: %w", errTx)
	}

	s.UpdateAdvisorsList(ctx)
	return nil
}

// DeleteAdvisorCheck deletes a user-authored advisor check and reloads the check list.
func (s *Service) DeleteAdvisorCheck(ctx context.Context, name string) error {
	err := s.ensureUserCheck(name)
	if err != nil {
		return err
	}

	errTx := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		return models.RemoveAdvisorCheck(tx.Querier, name)
	})
	if errTx != nil {
		return fmt.Errorf("failed to delete advisor check: %w", errTx)
	}

	s.UpdateAdvisorsList(ctx)
	return nil
}

// TestAdvisorCheck executes an advisor check definition against a single service
// and returns its findings without saving the check or persisting the results.
func (s *Service) TestAdvisorCheck(ctx context.Context, c check.Check, serviceID string) ([]services.CheckResult, string, error) {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		return nil, "", err
	}
	if !settings.IsAdvisorsEnabled() {
		return nil, "", services.ErrAdvisorsDisabled
	}

	c.Version = check.MaxSupportedVersion
	c.UserDefined = true

	err = c.Validate()
	if err != nil {
		return nil, "", status.Errorf(codes.InvalidArgument, "Invalid advisor check: %v", err)
	}

	var serviceType models.ServiceType
	switch c.Family {
	case check.MySQL:
		serviceType = models.MySQLServiceType
	case check.PostgreSQL:
		serviceType = models.PostgreSQLServiceType
	case check.MongoDB:
		serviceType = models.MongoDBServiceType
	default:
		return nil, "", status.Errorf(codes.InvalidArgument, "Unknown check family %s", c.Family)
	}

	targets, err := s.findTargets(ctx, serviceType, s.minPMMAgentVersion(c))
	if err != nil {
		return nil, "", err
	}

	for _, target := range targets {
		if target.ServiceID != serviceID {
			continue
		}
		// collect the script's print() output so check authors can debug their scripts
		var scriptOutput bytes.Buffer
		res, err := s.executeCheck(ctx, target, c, &scriptOutput)
		output := strings.TrimSpace(scriptOutput.String())
		if err != nil {
			// a status error keeps the query/script failure details visible to
			// the caller; plain errors are masked by the gRPC interceptor
			msg := fmt.Sprintf("Failed to execute check '%s' on service '%s': %v", c.Name, target.ServiceName, err)
			if output != "" {
				// keep the error itself on top; the print output goes below it
				msg += "\n\nScript output:\n" + output
			}
			return nil, "", status.Error(codes.FailedPrecondition, msg)
		}
		return res, output, nil
	}

	// no target matched - diagnose why so the error states the actual reason
	service, err := models.FindServiceByID(s.db.WithContext(ctx), serviceID)
	if err != nil {
		return nil, "", err
	}

	switch {
	case service.ServiceName == models.PMMServerPostgreSQLServiceName:
		return nil, "", status.Error(codes.FailedPrecondition,
			"PMM Server's internal PostgreSQL database cannot be targeted by advisor checks")
	case service.ServiceType != serviceType:
		return nil, "", status.Errorf(codes.FailedPrecondition,
			"Service '%s' is a %s service, but this check targets %s services",
			service.ServiceName, service.ServiceType, serviceType)
	default:
		return nil, "", status.Errorf(codes.FailedPrecondition,
			"Service '%s' has no compatible pmm-agent: it may be missing, disconnected or outdated",
			service.ServiceName)
	}
}

// ensureUserCheck verifies that the named check exists and is user-authored.
// It returns a NotFound error for unknown checks and a FailedPrecondition error
// for Percona-shipped checks, whose content is immutable.
func (s *Service) ensureUserCheck(name string) error {
	c, err := models.FindAdvisorCheckByName(s.db.Querier, name)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return status.Errorf(codes.NotFound, "Advisor check '%s' not found", name)
		}
		return err
	}

	if c.Source != models.UserCheckSource {
		return status.Errorf(codes.FailedPrecondition, "Advisor check '%s' is shipped by Percona and cannot be modified", name)
	}
	return nil
}

// userCheckToModel converts a user-authored check.Check into its DB representation.
func userCheckToModel(c check.Check) (*models.AdvisorCheck, error) {
	m, err := checkToModel(c)
	if err != nil {
		return nil, err
	}

	m.Source = models.UserCheckSource
	return m, nil
}

// checkToModel converts a check.Check into its DB representation, leaving the
// source and settings columns at their zero values.
func checkToModel(c check.Check) (*models.AdvisorCheck, error) {
	queries, err := json.Marshal(c.Queries)
	if err != nil {
		return nil, fmt.Errorf("failed to encode queries: %w", err)
	}

	return &models.AdvisorCheck{
		Name:        c.Name,
		Version:     c.Version,
		Summary:     c.Summary,
		Description: c.Description,
		Category:    c.Category,
		Subcategory: c.Subcategory,
		Family:      string(c.Family),
		Interval:    string(c.Interval),
		Queries:     queries,
		Script:      c.Script,
	}, nil
}

// modelToCheck converts a DB advisor check into a check.Check with the
// effective execution interval (user override, if any) applied.
func modelToCheck(m *models.AdvisorCheck) (check.Check, error) {
	var queries []check.Query
	err := json.Unmarshal(m.Queries, &queries)
	if err != nil {
		return check.Check{}, fmt.Errorf("failed to decode queries: %w", err)
	}

	interval := m.Interval
	if m.IntervalOverride != nil {
		interval = *m.IntervalOverride
	}

	return check.Check{
		Version:     m.Version,
		Name:        m.Name,
		Summary:     m.Summary,
		Description: m.Description,
		Category:    m.Category,
		Subcategory: m.Subcategory,
		Family:      check.Family(m.Family),
		Interval:    check.Interval(interval),
		Queries:     queries,
		Script:      m.Script,
		UserDefined: m.Source == models.UserCheckSource,
	}, nil
}

// waitForResult periodically checks result state and returns it when complete.
func (s *Service) waitForResult(ctx context.Context, resultID string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, resultAwaitTimeout)
	defer cancel()

	ticker := time.NewTicker(resultCheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		res, err := models.FindActionResultByID(s.db.Querier, resultID)
		if err != nil {
			return nil, err
		}

		if !res.Done {
			continue
		}

		if res.Error != "" {
			return nil, fmt.Errorf("action %s failed: %s", resultID, res.Error)
		}

		return []byte(res.Output), nil
	}
}

func (s *Service) minPMMAgentVersion(c check.Check) *version.Parsed {
	res := pmmAgent3_0_0 // minimum version that can be used with advisors
	for _, query := range c.Queries {
		v := s.minPMMAgentVersionForType(query.Type)
		if v != nil && res.Less(v) {
			res = v
		}
	}

	return res
}

// minPMMAgentVersion returns the minimal version of pmm-agent that can handle the given check type.
func (s *Service) minPMMAgentVersionForType(t check.Type) *version.Parsed {
	switch t {
	case check.MySQLSelect:
		fallthrough
	case check.MySQLShow:
		fallthrough
	case check.PostgreSQLSelect:
		fallthrough
	case check.PostgreSQLShow:
		fallthrough
	case check.MongoDBBuildInfo:
		fallthrough
	case check.MongoDBGetParameter:
		fallthrough
	case check.MongoDBGetCmdLineOpts:
		fallthrough
	case check.MongoDBReplSetGetStatus:
		fallthrough
	case check.MongoDBGetDiagnosticData:
		return pmmAgent3_0_0

	case check.MetricsRange:
		fallthrough
	case check.MetricsInstant:
		fallthrough
	case check.ClickHouseSelect:
		return nil // These types of queries don't require pmm agent at all, so any version is good.

	default:
		s.l.Warnf("minPMMAgentVersion: unhandled check type %q.", t)
		return pmmAgentInvalid
	}
}

// filterChecks filters checks by several parameters. If group specified then only matched checks will be returned,
// empty group means `any interval`. If enable slice is specified then only matched checks will be returned, empty
// enable slice means `all enabled`. Checks specified in disabled slice are skipped, empty `disabled` slice means
// `nothing disabled`.
func (s *Service) filterChecks(checks map[string]check.Check, group check.Interval, disable, enable []string) map[string]check.Check {
	res := make(map[string]check.Check)
	disableMap := make(map[string]struct{}, len(disable))
	for _, e := range disable {
		disableMap[e] = struct{}{}
	}

	enableMap := make(map[string]struct{}, len(enable))
	for _, e := range enable {
		enableMap[e] = struct{}{}
	}

	for n, c := range checks {
		// If empty group passed, which means `any group`
		// or check has required interval
		// or check has empty interval and required interval is `standard`.
		if group == "" || c.Interval == group || (group == check.Standard && c.Interval == "") {
			// If check enabled explicitly or all checks enabled by passing empty `enable` slice.
			if _, ok := enableMap[c.Name]; ok || len(enableMap) == 0 {
				// Filter disabled checks.
				if _, ok := disableMap[c.Name]; ok {
					s.l.Warnf("Check %s is disabled, skipping it.", c.Name)
					continue
				}

				res[n] = c
			}
		}
	}

	return res
}

// getActiveUserServiceTypes returns a set of service types that have at least one user-added service.
func (s *Service) getActiveUserServiceTypes() (map[models.ServiceType]struct{}, error) {
	serviceTypes, err := models.FindActiveUserServiceTypes(s.db.Querier)
	if err != nil {
		return nil, err
	}

	result := make(map[models.ServiceType]struct{}, len(serviceTypes))
	for _, st := range serviceTypes {
		result[st] = struct{}{}
	}
	return result, nil
}

// executeChecks runs checks for all reachable services. If intervalGroup specified only checks from that group will be
// executed. If checkNames specified then only matched checks will be executed.
func (s *Service) executeChecks(ctx context.Context, intervalGroup check.Interval, checkNames []string, ri runInfo) ([]services.CheckResult, error) {
	disabledChecks, err := s.GetDisabledChecks(ctx)
	if err != nil {
		return nil, err
	}

	disabledServices, err := s.GetDisabledServicesForChecks(ctx)
	if err != nil {
		return nil, err
	}
	disabledTargets := make(map[string]map[string]struct{}, len(disabledServices))
	for name, ids := range disabledServices {
		set := make(map[string]struct{}, len(ids))
		for _, id := range ids {
			set[id] = struct{}{}
		}
		disabledTargets[name] = set
	}

	activeServiceTypes, err := s.getActiveUserServiceTypes()
	if err != nil {
		return nil, err
	}

	var res []services.CheckResult
	checks, err := s.GetChecks()
	if err != nil {
		return nil, err
	}
	mySQLChecks, postgreSQLChecks, mongoDBChecks := groupChecksByDB(s.l, checks)

	// Execute MySQL checks only if MySQL services exist
	if _, hasMySQL := activeServiceTypes[models.MySQLServiceType]; hasMySQL {
		mySQLChecks = s.filterChecks(mySQLChecks, intervalGroup, disabledChecks, checkNames)
		mySQLCheckResults := s.executeChecksForTargetType(ctx, models.MySQLServiceType, mySQLChecks, disabledTargets, ri)
		res = append(res, mySQLCheckResults...)
	} else {
		s.l.Info("Skipping MySQL advisor checks: no MySQL services in inventory")
	}

	// Execute PostgreSQL checks only if PostgreSQL services exist
	if _, hasPostgreSQL := activeServiceTypes[models.PostgreSQLServiceType]; hasPostgreSQL {
		postgreSQLChecks = s.filterChecks(postgreSQLChecks, intervalGroup, disabledChecks, checkNames)
		postgreSQLCheckResults := s.executeChecksForTargetType(ctx, models.PostgreSQLServiceType, postgreSQLChecks, disabledTargets, ri)
		res = append(res, postgreSQLCheckResults...)
	} else {
		s.l.Info("Skipping PostgreSQL advisor checks: no PostgreSQL services in inventory")
	}

	// Execute MongoDB checks only if MongoDB services exist
	if _, hasMongoDB := activeServiceTypes[models.MongoDBServiceType]; hasMongoDB {
		mongoDBChecks = s.filterChecks(mongoDBChecks, intervalGroup, disabledChecks, checkNames)
		mongoDBCheckResults := s.executeChecksForTargetType(ctx, models.MongoDBServiceType, mongoDBChecks, disabledTargets, ri)
		res = append(res, mongoDBCheckResults...)
	} else {
		s.l.Info("Skipping MongoDB advisor checks: no MongoDB services in inventory")
	}

	return res, nil
}

func (s *Service) executeChecksForTargetType(ctx context.Context, serviceType models.ServiceType, checks map[string]check.Check, disabledTargets map[string]map[string]struct{}, ri runInfo) []services.CheckResult { //nolint:lll
	var res []services.CheckResult
	var history []*models.Insight

	for _, c := range checks {
		s.l.Infof("Executing check: %s with interval: %s", c.Name, c.Interval)
		pmmAgentVersion := s.minPMMAgentVersion(c)
		targets, err := s.findTargets(ctx, serviceType, pmmAgentVersion)
		if err != nil {
			s.l.Warnf("Failed to find proper agents and services for check family: %s and "+
				"min version: %s, reason: %s.", c.Family, pmmAgentVersion, err)
			continue
		}

		for _, target := range targets {
			if _, ok := disabledTargets[c.Name][target.ServiceID]; ok {
				s.l.Infof("Check %s is disabled for service %s, skipping it.", c.Name, target.ServiceID)
				continue
			}

			results, err := s.executeCheck(ctx, target, c, nil)
			// stamp each (check, target) outcome with its actual completion time
			checkedAt := models.Now()
			if err != nil {
				s.l.Warnf("Failed to execute check %s of family %s on target %s: %+v", c.Name, c.Family, target.AgentID, err)
				s.mChecksExecuted.WithLabelValues(string(target.ServiceType), c.Subcategory, c.Name, "error").Inc()
				history = append(history, newInsightRecord(c, target, models.CheckResultError, check.Result{Description: err.Error()}, checkedAt, ri))
				continue
			}

			res = append(res, results...)

			s.mChecksExecuted.WithLabelValues(string(target.ServiceType), c.Subcategory, c.Name, "ok").Inc()

			if len(results) == 0 {
				history = append(history, newInsightRecord(c, target, models.CheckResultOK, check.Result{}, checkedAt, ri))
				continue
			}

			for _, finding := range results {
				history = append(history, newInsightRecord(c, target, models.CheckResultFailed, finding.Result, checkedAt, ri))
			}
		}
	}

	err := s.saveInsights(ctx, history)
	if err != nil {
		s.l.Warnf("Failed to save Advisor insights: %+v", err)
	}

	return res
}

// newInsightRecord builds a history record for a single executed (check, target) outcome.
func newInsightRecord(
	c check.Check,
	target services.Target,
	status models.CheckResultStatus,
	result check.Result,
	checkedAt time.Time,
	ri runInfo,
) *models.Insight {
	r := &models.Insight{
		CheckName:      c.Name,
		Category:       c.Category,
		Subcategory:    c.Subcategory,
		Interval:       models.Interval(c.Interval),
		ServiceID:      target.ServiceID,
		ServiceName:    target.ServiceName,
		ServiceType:    target.ServiceType,
		NodeID:         target.NodeID,
		NodeName:       target.NodeName,
		Environment:    target.Environment,
		Cluster:        target.Cluster,
		ReplicationSet: target.ReplicationSet,
		Status:         status,
		Summary:        result.Summary,
		Description:    c.Description,
		Outcome:        result.Description,
		ReadMoreURL:    result.ReadMoreURL,
		Severity:       models.Severity(result.Severity),
		CheckedAt:      checkedAt,
		BatchID:        ri.batchID,
		TriggeredBy:    ri.triggeredBy,
	}
	// OK and error outcomes carry no finding; fall back to the check's own summary
	if r.Summary == "" {
		r.Summary = c.Summary
	}
	switch status {
	case models.CheckResultOK:
		r.Severity = models.Severity(common.Info)
		r.Outcome = "Check passed"
	case models.CheckResultError:
		// the check could not be executed, which is a diagnostic concern, not a database issue
		r.Severity = models.Severity(common.Info)
	case models.CheckResultFailed:
		// keep the severity reported by the finding
	}
	if len(result.Labels) != 0 {
		_ = r.SetLabels(result.Labels)
	}
	return r
}

// saveInsights persists Advisor insights in a single transaction.
func (s *Service) saveInsights(ctx context.Context, history []*models.Insight) error {
	if len(history) == 0 {
		return nil
	}

	return s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		for _, r := range history {
			err := models.CreateInsight(ctx, tx.Querier, r)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// executeCheck runs a single check against a single target. When scriptOutput is
// non-nil, the script's print() output is collected into it.
func (s *Service) executeCheck(ctx context.Context, target services.Target, c check.Check, scriptOutput *bytes.Buffer) ([]services.CheckResult, error) {
	ctx, cancel := context.WithTimeout(ctx, checkExecutionTimeout)
	defer cancel()

	defer func(t time.Time) {
		s.mChecksExecutionTime.WithLabelValues(string(target.ServiceType), c.Subcategory, c.Name).Observe(time.Since(t).Seconds())
	}(time.Now())

	if c.Version < check.MinSupportedVersion || c.Version > check.MaxSupportedVersion {
		return nil, fmt.Errorf("check %s has unsupported version %d", c.Name, c.Version)
	}

	queries := c.Queries

	eg, gCtx := errgroup.WithContext(ctx)
	resData := make([]any, len(queries))

	for i, query := range queries {
		switch query.Type {
		case check.MySQLShow:
			eg.Go(func() error {
				var err error
				resData[i], err = s.executeMySQLShowQuery(gCtx, query, target)
				return err
			})
		case check.MySQLSelect:
			eg.Go(func() error {
				var err error
				resData[i], err = s.executeMySQLSelectQuery(gCtx, query, target)
				return err
			})
		case check.PostgreSQLShow:
			eg.Go(func() error {
				var err error
				resData[i], err = s.executePostgreSQLShowQuery(gCtx, target)
				return err
			})
		case check.PostgreSQLSelect:
			eg.Go(func() error {
				var err error
				resData[i], err = s.executePostgreSQLSelectQuery(gCtx, query, target)
				return err
			})
		case check.MongoDBGetParameter:
			eg.Go(func() error {
				var err error
				resData[i], err = s.executeMongoDBGetParameterQuery(gCtx, target)
				return err
			})
		case check.MongoDBBuildInfo:
			eg.Go(func() error {
				var err error
				resData[i], err = s.executeMongoDBBuildInfoQuery(gCtx, target)
				return err
			})
		case check.MongoDBGetCmdLineOpts:
			eg.Go(func() error {
				var err error
				resData[i], err = s.executeMongoDBGetCmdLineOptsQuery(gCtx, target)
				return err
			})
		case check.MongoDBReplSetGetStatus:
			eg.Go(func() error {
				var err error
				resData[i], err = s.executeMongoDBReplSetGetStatusQuery(gCtx, target)
				return err
			})
		case check.MongoDBGetDiagnosticData:
			eg.Go(func() error {
				var err error
				resData[i], err = s.executeMongoDBGetDiagnosticQuery(gCtx, target)
				return err
			})
		case check.MetricsInstant:
			eg.Go(func() error {
				var err error
				resData[i], err = s.executeMetricsInstantQuery(gCtx, query, target)
				return err
			})
		case check.MetricsRange:
			eg.Go(func() error {
				var err error
				resData[i], err = s.executeMetricsRangeQuery(gCtx, query, target)
				return err
			})
		case check.ClickHouseSelect:
			eg.Go(func() error {
				var err error
				resData[i], err = s.executeClickhouseSelectQuery(gCtx, query, target)
				return err
			})

		default:
			return nil, errors.New("unknown check type")
		}
	}

	err := eg.Wait()
	if err != nil {
		return nil, fmt.Errorf("check query failed: %w", err)
	}

	res, err := s.processResults(ctx, c, target, resData, scriptOutput)
	if err != nil {
		return nil, fmt.Errorf("failed to process query result: %w", err)
	}

	return res, nil
}

func (s *Service) executeMySQLShowQuery(ctx context.Context, query check.Query, target services.Target) ([]byte, error) {
	r, err := models.CreateActionResult(s.db.Querier, target.AgentID)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare result: %w", err)
	}
	defer func() {
		err = s.db.Delete(r)
		if err != nil {
			s.l.Warnf("Failed to delete action result %s: %s.", r.ID, err)
		}
	}()

	err = s.agentsRegistry.StartMySQLQueryShowAction(ctx, r.ID, target.AgentID, target.DSN, query.Query, target.Files, target.TDP, target.TLSSkipVerify)
	if err != nil {
		return nil, fmt.Errorf("failed to start mySQL show action: %w", err)
	}
	res, err := s.waitForResult(ctx, r.ID)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *Service) executeMySQLSelectQuery(ctx context.Context, query check.Query, target services.Target) (string, error) {
	r, err := models.CreateActionResult(s.db.Querier, target.AgentID)
	if err != nil {
		return "", fmt.Errorf("failed to prepare result: %w", err)
	}
	defer func() {
		err = s.db.Delete(r)
		if err != nil {
			s.l.Warnf("Failed to delete action result %s: %s.", r.ID, err)
		}
	}()

	err = s.agentsRegistry.StartMySQLQuerySelectAction(
		ctx, r.ID, target.AgentID,
		target.DSN, query.Query, target.Files, target.TDP, target.TLSSkipVerify,
	)
	if err != nil {
		return "", fmt.Errorf("failed to start mySQL select action: %w", err)
	}
	res, err := s.waitForResult(ctx, r.ID)
	if err != nil {
		return "", err
	}

	return b64.EncodeToString(res), nil
}

func (s *Service) executePostgreSQLShowQuery(ctx context.Context, target services.Target) (string, error) {
	r, err := models.CreateActionResult(s.db.Querier, target.AgentID)
	if err != nil {
		return "", fmt.Errorf("failed to prepare result: %w", err)
	}
	defer func() {
		err = s.db.Delete(r)
		if err != nil {
			s.l.Warnf("Failed to delete action result %s: %s.", r.ID, err)
		}
	}()
	err = s.agentsRegistry.StartPostgreSQLQueryShowAction(ctx, r.ID, target.AgentID, target.DSN)
	if err != nil {
		return "", fmt.Errorf("failed to start postgreSQL show action: %w", err)
	}

	res, err := s.waitForResult(ctx, r.ID)
	if err != nil {
		return "", err
	}
	return b64.EncodeToString(res), nil
}

func (s *Service) executePostgreSQLSelectQuery(ctx context.Context, query check.Query, target services.Target) (any, error) {
	var allDBs bool
	var err error
	if value, ok := query.Parameters[check.AllDBs]; ok {
		allDBs, err = strconv.ParseBool(value)
		if err != nil {
			return nil, fmt.Errorf("failed to parse 'all_dbs' query parameter: %w", err)
		}
	}

	if !allDBs {
		return s.executePostgreSQLSelectQueryForSingleDB(ctx, query, target)
	}

	targets, err := s.splitPGTargetByDB(ctx, target)
	if err != nil {
		return nil, fmt.Errorf("failed to split target by db: %w", err)
	}
	res := make(map[string]string, len(targets))
	for dbName, t := range targets {
		res[dbName], err = s.executePostgreSQLSelectQueryForSingleDB(ctx, query, t)
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

func (s *Service) executePostgreSQLSelectQueryForSingleDB(ctx context.Context, query check.Query, target services.Target) (string, error) {
	r, err := models.CreateActionResult(s.db.Querier, target.AgentID)
	if err != nil {
		return "", fmt.Errorf("failed to prepare result: %w", err)
	}
	defer func() {
		err = s.db.Delete(r)
		if err != nil {
			s.l.Warnf("Failed to delete action result %s: %s.", r.ID, err)
		}
	}()

	err = s.agentsRegistry.StartPostgreSQLQuerySelectAction(ctx, r.ID, target.AgentID, target.DSN, query.Query)
	if err != nil {
		return "", fmt.Errorf("failed to start postgreSQL select action: %w", err)
	}

	res, err := s.waitForResult(ctx, r.ID)
	if err != nil {
		return "", err
	}

	return b64.EncodeToString(res), nil
}

func (s *Service) executeMongoDBGetParameterQuery(ctx context.Context, target services.Target) (string, error) {
	r, err := models.CreateActionResult(s.db.Querier, target.AgentID)
	if err != nil {
		return "", fmt.Errorf("failed to prepare result: %w", err)
	}
	defer func() {
		err = s.db.Delete(r)
		if err != nil {
			s.l.Warnf("Failed to delete action result %s: %s.", r.ID, err)
		}
	}()

	err = s.agentsRegistry.StartMongoDBQueryGetParameterAction(ctx, r.ID, target.AgentID, target.DSN, target.Files, target.TDP)
	if err != nil {
		return "", fmt.Errorf("failed to start mongoDB getParameter action: %w", err)
	}

	res, err := s.waitForResult(ctx, r.ID)
	if err != nil {
		return "", err
	}

	return b64.EncodeToString(res), nil
}

func (s *Service) executeMongoDBBuildInfoQuery(ctx context.Context, target services.Target) (string, error) {
	r, err := models.CreateActionResult(s.db.Querier, target.AgentID)
	if err != nil {
		return "", fmt.Errorf("failed to prepare result: %w", err)
	}
	defer func() {
		err = s.db.Delete(r)
		if err != nil {
			s.l.Warnf("Failed to delete action result %s: %s.", r.ID, err)
		}
	}()
	err = s.agentsRegistry.StartMongoDBQueryBuildInfoAction(ctx, r.ID, target.AgentID, target.DSN, target.Files, target.TDP)
	if err != nil {
		return "", fmt.Errorf("failed to start mongoDB buildInfo action: %w", err)
	}

	res, err := s.waitForResult(ctx, r.ID)
	if err != nil {
		return "", err
	}

	return b64.EncodeToString(res), nil
}

func (s *Service) executeMongoDBGetCmdLineOptsQuery(ctx context.Context, target services.Target) (string, error) {
	r, err := models.CreateActionResult(s.db.Querier, target.AgentID)
	if err != nil {
		return "", fmt.Errorf("failed to prepare result: %w", err)
	}
	defer func() {
		err = s.db.Delete(r)
		if err != nil {
			s.l.Warnf("Failed to delete action result %s: %s.", r.ID, err)
		}
	}()

	err = s.agentsRegistry.StartMongoDBQueryGetCmdLineOptsAction(ctx, r.ID, target.AgentID, target.DSN, target.Files, target.TDP)
	if err != nil {
		return "", fmt.Errorf("failed to start mongoDB getCmdLineOpts action: %w", err)
	}

	res, err := s.waitForResult(ctx, r.ID)
	if err != nil {
		return "", err
	}

	return b64.EncodeToString(res), nil
}

func (s *Service) executeMongoDBReplSetGetStatusQuery(ctx context.Context, target services.Target) (string, error) {
	r, err := models.CreateActionResult(s.db.Querier, target.AgentID)
	if err != nil {
		return "", fmt.Errorf("failed to prepare result: %w", err)
	}
	defer func() {
		err = s.db.Delete(r)
		if err != nil {
			s.l.Warnf("Failed to delete action result %s: %s.", r.ID, err)
		}
	}()

	err = s.agentsRegistry.StartMongoDBQueryReplSetGetStatusAction(ctx, r.ID, target.AgentID, target.DSN, target.Files, target.TDP)
	if err != nil {
		return "", fmt.Errorf("failed to start mongoDB replSetGetStatus action: %w", err)
	}

	res, err := s.waitForResult(ctx, r.ID)
	if err != nil {
		return "", err
	}

	return b64.EncodeToString(res), nil
}

func (s *Service) executeMongoDBGetDiagnosticQuery(ctx context.Context, target services.Target) (string, error) {
	r, err := models.CreateActionResult(s.db.Querier, target.AgentID)
	if err != nil {
		return "", fmt.Errorf("failed to prepare result: %w", err)
	}
	defer func() {
		err = s.db.Delete(r)
		if err != nil {
			s.l.Warnf("Failed to delete action result %s: %s.", r.ID, err)
		}
	}()

	err = s.agentsRegistry.StartMongoDBQueryGetDiagnosticDataAction(ctx, r.ID, target.AgentID, target.DSN, target.Files, target.TDP)
	if err != nil {
		return "", fmt.Errorf("failed to start mongoDB getDiagnosticData action: %w", err)
	}

	res, err := s.waitForResult(ctx, r.ID)
	if err != nil {
		return "", err
	}

	return b64.EncodeToString(res), nil
}

func (s *Service) executeMetricsInstantQuery(ctx context.Context, query check.Query, target services.Target) (string, error) {
	queryData := queryPlaceholders{
		ServiceName: target.ServiceName,
		NodeName:    target.NodeName,
	}

	q, err := fillQueryPlaceholders(query.Query, queryData)
	if err != nil {
		return "", err
	}

	var lookback time.Time // if not specified use empty time which means "current time"
	if v, ok := query.Parameters[check.Lookback]; ok {
		d, err := time.ParseDuration(v)
		if err != nil {
			return "", fmt.Errorf("failed to parse 'lookback' query parameter: %w", err)
		}

		lookback = time.Now().Add(-d)
	}

	r, warns, err := s.vmClient.Query(ctx, q, lookback)
	if err != nil {
		return "", fmt.Errorf("failed to execute instant VM query: %w", err)
	}

	for _, warn := range warns {
		s.l.Warn(warn)
	}

	res, err := convertVMValue(r)
	if err != nil {
		return "", err
	}

	return b64.EncodeToString(res), nil
}

func (s *Service) executeMetricsRangeQuery(ctx context.Context, query check.Query, target services.Target) (string, error) {
	queryData := queryPlaceholders{
		ServiceName: target.ServiceName,
		NodeName:    target.NodeName,
	}

	q, err := fillQueryPlaceholders(query.Query, queryData)
	if err != nil {
		return "", err
	}

	rng := v1.Range{
		End: time.Now(), // use current time as a default for the upper bound of the range
	}

	if v, ok := query.Parameters[check.Lookback]; ok {
		d, err := time.ParseDuration(v)
		if err != nil {
			return "", fmt.Errorf("failed to parse 'lookback' query parameter: %w", err)
		}

		rng.End = time.Now().Add(-d)
	}

	rg, ok := query.Parameters[check.Range]
	if !ok {
		return "", errors.New("'range' query parameter is required for range queries")
	}

	d, err := time.ParseDuration(rg)
	if err != nil {
		return "", fmt.Errorf("failed to parse 'range' query parameter: %w", err)
	}

	rng.Start = rng.End.Add(-d)

	st, ok := query.Parameters[check.Step]
	if !ok {
		return "", errors.New("'step' query parameter is required for range queries")
	}

	rng.Step, err = time.ParseDuration(st)
	if err != nil {
		return "", fmt.Errorf("failed to parse 'step' query parameter: %w", err)
	}

	r, warns, err := s.vmClient.QueryRange(ctx, q, rng)
	if err != nil {
		return "", fmt.Errorf("failed to execute range VM query: %w", err)
	}

	for _, warn := range warns {
		s.l.Warn(warn)
	}

	res, err := convertVMValue(r)
	if err != nil {
		return "", err
	}

	return b64.EncodeToString(res), nil
}

func (s *Service) executeClickhouseSelectQuery(ctx context.Context, checkQuery check.Query, target services.Target) (string, error) {
	queryData := queryPlaceholders{
		ServiceName: target.ServiceName,
		ServiceID:   target.ServiceID,
	}

	query, err := fillQueryPlaceholders(checkQuery.Query, queryData)
	if err != nil {
		return "", err
	}

	query = "SELECT " + query

	rows, err := s.clickhouseDB.QueryContext(ctx, query)
	if err != nil {
		return "", fmt.Errorf("failed to execute query: %w", err)
	}

	columns, dataRows, err := sqlrows.ReadRows(rows)
	if err != nil {
		return "", err
	}

	b, err := agentv1.MarshalActionQuerySQLResult(columns, dataRows)
	if err != nil {
		return "", err
	}

	return b64.EncodeToString(b), nil
}

// convertVMValue converts VM results to format applicable to check input.
func convertVMValue(value model.Value) ([]byte, error) {
	if value.Type() == model.ValScalar {
		// MetricsQL treats scalar type the same as instant vector without labels, since subtle differences between
		// these types usually confuse users. See the corresponding Prometheus docs for details.
		// https://docs.victoriametrics.com/MetricsQL.html#metricsql-features
		return nil, errors.New("unexpected value type")
	}

	// Here we marshal VM value to json and unmarshal it back to form that we need. While it's not so effective
	// from performance standpoint it's easy and clean.
	b, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	var data []map[string]any
	err = json.Unmarshal(b, &data)
	if err != nil {
		return nil, err
	}

	res, err := agentv1.MarshalActionQueryDocsResult(data)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *Service) discoverAvailablePGDatabases(ctx context.Context, target services.Target) ([]string, error) {
	query := check.Query{Query: `datname FROM pg_database  
WHERE datallowconn = true AND datistemplate = false AND has_database_privilege(current_user, datname, 'connect')`}

	res, err := s.executePostgreSQLSelectQueryForSingleDB(ctx, query, target)
	if err != nil {
		return nil, fmt.Errorf("failed to select available databases: %w", err)
	}

	dec, err := b64.DecodeString(res)
	if err != nil {
		return nil, fmt.Errorf("failed to decode database discovery results: %w", err)
	}

	data, err := agentv1.UnmarshalActionQueryResult(dec)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal database discovery results: %w", err)
	}

	r := make([]string, len(data))
	for i, row := range data {
		datname, ok := row["datname"]
		if !ok {
			return nil, errors.New("missing expected 'datname' filed in query response")
		}
		name, ok := datname.(string)
		if !ok {
			return nil, fmt.Errorf("unexpected type %T instead of string", datname)
		}

		r[i] = name
	}

	return r, nil
}

func (s *Service) splitPGTargetByDB(ctx context.Context, target services.Target) (map[string]services.Target, error) {
	dbNames, err := s.discoverAvailablePGDatabases(ctx, target)
	if err != nil {
		return nil, err
	}

	dsn, err := url.Parse(target.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to parse postrgeSQL DSN: %w", err)
	}

	res := make(map[string]services.Target, len(dbNames))
	for _, name := range dbNames {
		nt := target.Copy()
		dsn.Path = name
		nt.DSN = dsn.String()
		res[name] = nt
	}

	return res, nil
}

func fillQueryPlaceholders(query string, data queryPlaceholders) (string, error) {
	tm, err := template.New("query").Parse(query)
	if err != nil {
		return "", fmt.Errorf("failed to parse query: %w", err)
	}

	var b strings.Builder
	err = tm.Execute(&b, data)
	if err != nil {
		return "", fmt.Errorf("failed to fill query placeholders: %w", err)
	}

	return b.String(), nil
}

// StarlarkScriptData represents the data we need to pass to the binary to run starlark scripts.
type StarlarkScriptData struct {
	Version        uint32 `json:"version"`
	Name           string `json:"name"`
	Script         string `json:"script"`
	QueriesResults []any  `json:"queries_results"`
	// CapturePrintOutput makes the script's print() calls emit plain lines on
	// a dedicated pipe (fd 3) so the caller can collect them without mixing
	// them into the stderr error channel (used by check test runs).
	CapturePrintOutput bool `json:"capture_print_output"`
}

// processResults runs the check script in the pmm-managed-starlark sandbox and converts its
// findings. When scriptOutput is non-nil, the script's print() output is collected into it,
// on failures too.
func (s *Service) processResults(ctx context.Context, aCheck check.Check, target services.Target, queryResults []any, scriptOutput *bytes.Buffer) ([]services.CheckResult, error) { //nolint:lll
	l := s.l.WithFields(logrus.Fields{
		"name":       aCheck.Name,
		"service_id": target.ServiceID,
	})

	input := &StarlarkScriptData{
		Version:            aCheck.Version,
		Name:               aCheck.Name,
		Script:             aCheck.Script,
		QueriesResults:     queryResults,
		CapturePrintOutput: scriptOutput != nil,
	}

	cmdCtx, cancel := context.WithTimeout(ctx, scriptExecutionTimeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "pmm-managed-starlark")
	pdeathsig.Set(cmd, syscall.SIGKILL)

	var stdin, stdout, stderr bytes.Buffer
	cmd.Stdin = &stdin
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// print() output arrives on its own pipe (fd 3 in the child), keeping
	// stderr a pure error channel
	var printR, printW *os.File
	if scriptOutput != nil {
		var err error
		printR, printW, err = os.Pipe()
		if err != nil {
			return nil, fmt.Errorf("failed to create print output pipe: %w", err)
		}
		defer printR.Close() //nolint:errcheck
		cmd.ExtraFiles = []*os.File{printW}
	}

	encoder := json.NewEncoder(&stdin)
	err := encoder.Encode(input)
	if err != nil {
		return nil, fmt.Errorf("error encoding data to STDIN: %w", err)
	}

	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start check script: %w", err)
	}

	printDone := make(chan struct{})
	if printW != nil {
		// the child holds its own copy now; closing ours lets the reader see EOF on child exit
		_ = printW.Close()
		go func() {
			defer close(printDone)
			_, _ = io.Copy(scriptOutput, printR)
		}()
	} else {
		close(printDone)
	}

	err = cmd.Wait()
	<-printDone
	if err != nil {
		scriptErr := strings.TrimSpace(stderr.String())
		l.Errorf("Check script failed (%s): %s", err, scriptErr)
		switch {
		case scriptErr != "":
			// the subprocess reported the real cause (script bug, malformed query result, etc.) on stderr
			return nil, errors.New(scriptErr)
		case cmdCtx.Err() != nil:
			return nil, fmt.Errorf("check script execution timed out after %s", scriptExecutionTimeout)
		default:
			return nil, fmt.Errorf("check script execution failed: %w", err)
		}
	}

	procOut := stdout.Bytes()

	var results []check.Result
	decoder := json.NewDecoder(bytes.NewReader(procOut))
	err = decoder.Decode(&results)
	if err != nil {
		return nil, fmt.Errorf("error processing json output: %w", err)
	}
	l.Infof("Check script returned %d results.", len(results))
	l.Debugf("Results: %+v.", results)

	checkResults := make([]services.CheckResult, len(results))
	for i, result := range results {
		err = validateAdvisorSeverity(result.Severity)
		if err != nil {
			return nil, fmt.Errorf("check result %d: %w", i+1, err)
		}
		checkResults[i] = services.CheckResult{
			CheckName:   aCheck.Name,
			Subcategory: aCheck.Subcategory,
			Interval:    aCheck.Interval,
			Target:      target,
			Result:      result,
		}
	}
	return checkResults, nil
}

// validateAdvisorSeverity rejects result severities outside the set advisors use.
// The retired levels (emergency, alert, notice, debug) fail the check run with a
// clear message instead of being coerced silently, so check authors notice and
// migrate their scripts.
func validateAdvisorSeverity(s common.Severity) error {
	switch s {
	case common.Critical, common.Error, common.Warning, common.Info:
		return nil
	default:
		return fmt.Errorf("result severity '%s' is not supported by advisors; use one of: critical, error, warning, info", s)
	}
}

// findTargets returns slice of available targets for specified service type.
func (s *Service) findTargets(ctx context.Context, serviceType models.ServiceType, minPMMAgentVersion *version.Parsed) ([]services.Target, error) {
	var targets []services.Target
	monitoredServices, err := models.FindServices(s.db.WithContext(ctx), models.ServiceFilters{ServiceType: &serviceType})
	if err != nil {
		return nil, err
	}

	for _, service := range monitoredServices {
		// skip PMM Server's internal PostgreSQL database, but allow other services on the PMM Server node
		if service.ServiceName == models.PMMServerPostgreSQLServiceName {
			s.l.Debugf("Skip PMM Server's internal PostgreSQL service, name: %s, type: %s.", service.ServiceName, service.ServiceType)
			continue
		}

		e := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
			pmmAgents, err := models.FindPMMAgentsForService(tx.Querier, service.ServiceID)
			if err != nil {
				return err
			}
			if len(pmmAgents) == 0 {
				return errors.New("no available pmm agents")
			}

			pmmAgents = models.FindPMMAgentsForVersion(s.l, pmmAgents, minPMMAgentVersion)
			if len(pmmAgents) == 0 {
				return errors.New("all available agents are outdated")
			}
			pmmAgent := pmmAgents[0]

			DSN, agent, err := models.FindDSNByServiceIDandPMMAgentID(tx.Querier, service.ServiceID, pmmAgents[0].AgentID, "")
			if err != nil {
				return err
			}

			node, err := models.FindNodeByID(tx.Querier, service.NodeID)
			if err != nil {
				return err
			}

			labels, err := models.MergeLabels(node, service, agent)
			if err != nil {
				return err
			}

			targets = append(targets, services.Target{
				AgentID:        pmmAgent.AgentID,
				ServiceID:      service.ServiceID,
				ServiceName:    service.ServiceName,
				ServiceType:    service.ServiceType,
				NodeID:         node.NodeID,
				NodeName:       node.NodeName,
				Environment:    service.Environment,
				Cluster:        service.Cluster,
				ReplicationSet: service.ReplicationSet,
				Labels:         labels,
				DSN:            DSN,
				Files:          agent.Files(),
				TDP:            agent.TemplateDelimiters(service),
				TLSSkipVerify:  agent.TLSSkipVerify,
			})
			return nil
		})
		if e != nil {
			s.l.Errorf("Failed to find agents for service %s, reason: %s.", service.ServiceID, e)
		}
	}

	return targets, nil
}

// UpdateAdvisorsList loads built-in checks (plus an optional user-defined file),
// groups them into advisors, and stores versions supported by this pmm-managed version.
func (s *Service) UpdateAdvisorsList(ctx context.Context) {
	defer s.refreshChecksInMemoryMetric()

	rows, err := models.FindAdvisorChecks(s.db.WithContext(ctx))
	if err != nil {
		s.l.Errorf("Failed to load advisor checks: %s.", err)
		return // keep previously loaded advisors
	}

	// Skip rows that fail to decode or validate with a warning so a single
	// bad row cannot break the whole load.
	checks := make([]check.Check, 0, len(rows))
	for _, row := range rows {
		c, err := modelToCheck(row)
		if err != nil {
			s.l.Warnf("Failed to decode advisor check '%s': %s.", row.Name, err)
			continue
		}

		err = c.Validate()
		if err != nil {
			s.l.Warnf("Advisor check '%s' is invalid and is ignored: %s.", row.Name, err)
			continue
		}

		checks = append(checks, c)
	}

	s.updateAdvisors(s.filterSupportedChecks(groupChecksIntoAdvisors(checks)))
}

// reconcileBuiltinChecks synchronizes Percona-shipped checks from disk into the
// advisor_checks table: content columns are inserted or refreshed (including
// the placeholder rows created by migration 120 from legacy settings), rows of
// checks removed from the package are pruned, and user-set overrides (interval,
// disabled state, per-service disables) are preserved. It runs once at startup;
// picking up changed check files requires a restart.
func (s *Service) reconcileBuiltinChecks(ctx context.Context) error {
	checks, err := s.loadBuiltinChecks(ctx)
	if err != nil {
		return fmt.Errorf("failed to load built-in checks: %w", err)
	}

	return s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		// Collisions with user-authored rows are impossible: user check names
		// must carry check.UserCheckNamePrefix and Percona checks must not
		// (the latter is enforced by pi-validator).
		names := make([]string, 0, len(checks))
		for _, c := range checks {
			m, err := checkToModel(c)
			if err != nil {
				return err
			}

			err = models.UpsertAdvisorCheckContent(ctx, tx.Querier, m)
			if err != nil {
				return err
			}
			names = append(names, c.Name)
		}

		return models.RemoveAdvisorChecksNotIn(ctx, tx.Querier, names)
	})
}

// loadBuiltinChecks loads builtin checks from the checks directory.
func (s *Service) loadBuiltinChecks(_ context.Context) ([]check.Check, error) {
	s.l.Infof("Loading checks from dir=%s", builtinChecksPath)

	checkFiles, err := filepath.Glob(filepath.Join(builtinChecksPath, "*.yml"))
	if err != nil {
		return nil, fmt.Errorf("failed to find check files: %w", err)
	}

	return s.loadChecksFromFiles(checkFiles)
}

// groupChecksIntoAdvisors groups checks into advisors by their (Category, Subcategory)
// pair, preserving first-seen order.
func groupChecksIntoAdvisors(checks []check.Check) []check.Advisor {
	index := make(map[string]int, len(checks))
	advisors := make([]check.Advisor, 0, len(checks))
	for _, c := range checks {
		key := c.Category + "\x00" + c.Subcategory
		i, ok := index[key]
		if !ok {
			i = len(advisors)
			index[key] = i
			advisors = append(advisors, check.Advisor{Category: c.Category, Subcategory: c.Subcategory})
		}
		advisors[i].Checks = append(advisors[i].Checks, c)
	}

	return advisors
}

// loadChecksFromFiles loads Advisor checks from a list of given files.
func (s *Service) loadChecksFromFiles(files []string) ([]check.Check, error) {
	res := make([]check.Check, 0, len(files))
	for _, file := range files {
		s.l.Debugf("Loading check file=%s", file)

		b, err := os.ReadFile(file) //nolint:gosec
		if err != nil {
			return nil, fmt.Errorf("failed to read checks file %s: %w", file, err)
		}
		checks, err := check.ParseChecks(bytes.NewReader(b), &check.ParseParams{
			DisallowUnknownFields: true,
			DisallowInvalidChecks: true,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to parse checks from file %s: %w", file, err)
		}

		if len(checks) != 1 {
			return nil, fmt.Errorf("expected exactly one check in %s", file)
		}
		c := checks[0]

		_, fileName := filepath.Split(file)
		if c.Name != strings.TrimSuffix(fileName, ".yml") {
			return nil, fmt.Errorf("check name does not match file name %s", file)
		}

		res = append(res, c)
	}

	return res, nil
}

// filterSupportedChecks returns supported advisor checks and prints warning log messages about unsupported.
func (s *Service) filterSupportedChecks(advisors []check.Advisor) []check.Advisor {
	res := make([]check.Advisor, 0, len(advisors))

	for _, advisor := range advisors {
		checks := make([]check.Check, 0, len(advisor.Checks))

	LOOP:
		for _, c := range advisor.Checks {
			if c.Version > check.MaxSupportedVersion {
				s.l.Warnf("Unsupported checks version: %d, max supported version: %d.", c.Version, check.MaxSupportedVersion)
				continue LOOP
			}

			for _, query := range c.Queries {
				if ok := isQueryTypeSupported(query.Type); !ok {
					s.l.Warnf("Unsupported query type: %s.", query.Type)
					continue LOOP
				}
			}

			checks = append(checks, c)
		}
		if len(checks) != 0 {
			advisor.Checks = checks
			res = append(res, advisor)
		}
	}

	return res
}

func isQueryTypeSupported(typ check.Type) bool {
	switch typ {
	case check.MySQLShow:
	case check.MySQLSelect:
	case check.PostgreSQLShow:
	case check.PostgreSQLSelect:
	case check.MongoDBGetParameter:
	case check.MongoDBBuildInfo:
	case check.MongoDBGetCmdLineOpts:
	case check.MongoDBReplSetGetStatus:
	case check.MongoDBGetDiagnosticData:
	case check.MetricsRange:
	case check.MetricsInstant:
	case check.ClickHouseSelect:
	default:
		return false
	}

	return true
}

// updateAdvisors update advisors filed value under mutex.
func (s *Service) updateAdvisors(advisors []check.Advisor) {
	s.am.Lock()
	defer s.am.Unlock()

	s.advisors = advisors

	checks := make(map[string]check.Check)
	for _, a := range s.advisors {
		for _, c := range a.Checks {
			checks[c.Name] = c
		}
	}

	s.checks = checks
}

// UpdateIntervals updates advisor checks restart timer intervals.
func (s *Service) UpdateIntervals(rare, standard, frequent time.Duration) {
	s.tm.Lock()
	defer s.tm.Unlock()
	// Tickers are created by Run; if it has not started on this node (e.g. not
	// the leader), there is nothing to reset - Run reads the new intervals from
	// the persisted settings when it starts.
	if s.rareTicker == nil || s.standardTicker == nil || s.frequentTicker == nil {
		return
	}
	s.rareTicker.Reset(rare)
	s.standardTicker.Reset(standard)
	s.frequentTicker.Reset(frequent)

	s.l.Infof("Intervals are changed: rare %s, standard %s, frequent %s", rare, standard, frequent)
}

// Describe implements prom.Collector.
func (s *Service) Describe(ch chan<- *prom.Desc) {
	s.mChecksExecuted.Describe(ch)
	s.mChecksAvailable.Describe(ch)
	s.mChecksExecutionTime.Describe(ch)

	s.resultsRegistry.Describe(ch)
}

// Collect implements prom.Collector.
func (s *Service) Collect(ch chan<- prom.Metric) {
	s.mChecksExecuted.Collect(ch)
	s.mChecksAvailable.Collect(ch)
	s.mChecksExecutionTime.Collect(ch)

	s.resultsRegistry.Collect(ch)
}

func (s *Service) refreshChecksInMemoryMetric() {
	checks, err := s.GetChecks()
	if err != nil {
		s.l.Warnf("failed to get checks: %+v", err)
		return
	}
	s.mChecksAvailable.Reset()
	mySQLChecks, postgreSQLChecks, mongoDBChecks := groupChecksByDB(s.l, checks)
	s.incChecksInMemoryMetric(models.MySQLServiceType, mySQLChecks)
	s.incChecksInMemoryMetric(models.PostgreSQLServiceType, postgreSQLChecks)
	s.incChecksInMemoryMetric(models.MongoDBServiceType, mongoDBChecks)
}

func (s *Service) incChecksInMemoryMetric(serviceType models.ServiceType, checks map[string]check.Check) {
	for _, c := range checks {
		s.mChecksAvailable.WithLabelValues(string(serviceType), c.Subcategory, c.Name).Inc()
	}
}

// groupChecksByDB splits provided checks by database and returns three slices: for MySQL, for PostgreSQL and for MongoDB.
func groupChecksByDB(l *logrus.Entry, checks map[string]check.Check) (mySQLChecks, postgreSQLChecks, mongoDBChecks map[string]check.Check) { //nolint:nonamedreturns
	mySQLChecks = make(map[string]check.Check)
	postgreSQLChecks = make(map[string]check.Check)
	mongoDBChecks = make(map[string]check.Check)
	for _, c := range checks {
		switch c.Family {
		case check.MySQL:
			mySQLChecks[c.Name] = c
		case check.PostgreSQL:
			postgreSQLChecks[c.Name] = c
		case check.MongoDB:
			mongoDBChecks[c.Name] = c
		default:
			l.Warnf("Unknown check family %s, will be skipped.", c.Family)
		}
	}

	return mySQLChecks, postgreSQLChecks, mongoDBChecks
}

// check interfaces.
var (
	_ prom.Collector = (*Service)(nil)
)
