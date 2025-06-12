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

	"github.com/percona/saas/pkg/check"
	"github.com/pkg/errors"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"gopkg.in/reform.v1"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
	"github.com/percona/pmm/utils/pdeathsig"
	"github.com/percona/pmm/utils/sqlrows"
	"github.com/percona/pmm/version"
)

const (
	defaultStartDelay = time.Minute

	// Environment variables that affect checks service; only for testing.
	envCheckFile         = "PMM_DEV_ADVISOR_CHECKS_FILE"
	envDisableStartDelay = "PMM_ADVISORS_CHECKS_DISABLE_START_DELAY"
	builtinAdvisorsPath  = "/usr/local/percona/pmm/advisors"
	builtinChecksPath    = "/usr/local/percona/pmm/checks"

	checkExecutionTimeout  = 5 * time.Minute  // limits execution time for every single check
	resultAwaitTimeout     = 20 * time.Second // should be greater than agents.defaultQueryActionTimeout
	scriptExecutionTimeout = 5 * time.Second  // time limit for running pmm-managed-starlark
	resultCheckInterval    = time.Second

	prometheusNamespace = "pmm_managed"
	prometheusSubsystem = "advisor"

	maxSupportedVersion = 2
)

// pmm-agent versions with known changes in Query Actions.
// To match all pre-release versions, add '-0' suffix to the specified version.
var (
	pmmAgent2_6_0   = version.MustParse("2.6.0")
	pmmAgent2_7_0   = version.MustParse("2.7.0")
	pmmAgent2_27_0  = version.MustParse("2.27.0-0")
	pmmAgentInvalid = version.MustParse("3.0.0-invalid")

	b64 = base64.StdEncoding
)

// Service is responsible for interactions with Percona Check service.
type Service struct {
	agentsRegistry agentsRegistry
	db             *reform.DB
	alertsRegistry *registry
	vmClient       v1.API
	clickhouseDB   *sql.DB

	l               *logrus.Entry
	startDelay      time.Duration
	customCheckFile string // For testing

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
		db:             db,
		agentsRegistry: agentsRegistry,
		alertsRegistry: newRegistry(),
		vmClient:       vmClient,
		clickhouseDB:   clickhouseDB,

		l:               l,
		startDelay:      defaultStartDelay,
		customCheckFile: os.Getenv(envCheckFile),

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
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
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

	s.CollectAdvisors(ctx)
	settings, err := models.GetSettings(s.db)
	if err != nil {
		s.l.Errorf("Failed to get settings: %+v.", err)
		return
	}

	s.rareTicker = time.NewTicker(settings.SaaS.AdvisorRunIntervals.RareInterval)
	defer s.rareTicker.Stop()

	s.standardTicker = time.NewTicker(settings.SaaS.AdvisorRunIntervals.StandardInterval)
	defer s.standardTicker.Stop()

	s.frequentTicker = time.NewTicker(settings.SaaS.AdvisorRunIntervals.FrequentInterval)
	defer s.frequentTicker.Stop()

	// delay for the first run to allow all agents to connect
	startCtx, startCancel := context.WithTimeout(ctx, s.startDelay)
	<-startCtx.Done()
	startCancel()
	if ctx.Err() != nil { // check the main context, not startCtx
		return
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		s.runChecksLoop(ctx)
	}()

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

// GetChecksResults returns the failed checks for a given service.
func (s *Service) GetChecksResults(_ context.Context, serviceID string) ([]services.CheckResult, error) {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		return nil, err
	}

	if !settings.IsAdvisorsEnabled() {
		return nil, services.ErrAdvisorsDisabled
	}

	return s.alertsRegistry.getCheckResults(serviceID), nil
}

// runChecksGroup downloads and executes Advisors checks that should run in the interval specified by intervalGroup.
// All checks are executed if intervalGroup is empty.
func (s *Service) runChecksGroup(ctx context.Context, intervalGroup check.Interval) error {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		return errors.WithStack(err)
	}

	if !settings.IsAdvisorsEnabled() {
		return services.ErrAdvisorsDisabled
	}

	s.CollectAdvisors(ctx)
	return s.run(ctx, intervalGroup, nil)
}

// StartChecks downloads and executes advisor checks in asynchronous way.
// If checkNames specified then only matched checks will be executed.
func (s *Service) StartChecks(checkNames []string) error {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		return errors.WithStack(err)
	}

	if !settings.IsAdvisorsEnabled() {
		return services.ErrAdvisorsDisabled
	}

	go func() {
		ctx := context.Background()
		s.CollectAdvisors(ctx)
		if err := s.run(ctx, "", checkNames); err != nil {
			s.l.Errorf("Failed to execute advisor checks: %+v.", err)
		}
	}()

	return nil
}

func (s *Service) run(ctx context.Context, intervalGroup check.Interval, checkNames []string) error {
	if err := intervalGroup.Validate(); err != nil {
		return errors.WithStack(err)
	}

	res, err := s.executeChecks(ctx, intervalGroup, checkNames)
	if err != nil {
		return errors.WithStack(err)
	}

	switch {
	case len(checkNames) != 0:
		// If we run some specific checks, delete previous results for them.
		s.alertsRegistry.deleteByName(checkNames)
	case intervalGroup != "":
		// If we run whole interval group, delete previous results for that group.
		s.alertsRegistry.deleteByInterval(intervalGroup)
	default:
		// If we run all checks, delete all previous results.
		s.alertsRegistry.cleanup()
	}

	s.alertsRegistry.set(res)

	return nil
}

// CleanupAlerts drops all alerts in registry.
func (s *Service) CleanupAlerts() {
	s.alertsRegistry.cleanup()
}

// GetAdvisors returns all available advisors.
func (s *Service) GetAdvisors() ([]check.Advisor, error) {
	cs, err := models.FindCheckSettings(s.db.Querier)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	s.am.Lock()
	defer s.am.Unlock()

	res := make([]check.Advisor, 0, len(s.advisors))
	for _, a := range s.advisors {
		checks := make([]check.Check, 0, len(a.Checks))
		for _, c := range a.Checks {
			if interval, ok := cs[c.Name]; ok {
				c.Interval = check.Interval(interval)
			}
			checks = append(checks, c)
		}
		a.Checks = checks
		res = append(res, a)
	}
	return res, nil
}

// GetChecks retrieves a map of checks from the service.
func (s *Service) GetChecks() (map[string]check.Check, error) {
	cs, err := models.FindCheckSettings(s.db.Querier)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	s.am.Lock()
	defer s.am.Unlock()

	res := make(map[string]check.Check, len(s.checks))
	for _, c := range s.checks {
		if interval, ok := cs[c.Name]; ok {
			c.Interval = check.Interval(interval)
		}

		res[c.Name] = c
	}

	return res, nil
}

// GetDisabledChecks returns disabled checks.
func (s *Service) GetDisabledChecks() ([]string, error) {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		return nil, err
	}

	return settings.SaaS.DisabledAdvisors, nil
}

// DisableChecks disables checks with provided names.
func (s *Service) DisableChecks(checkNames []string) error {
	if len(checkNames) == 0 {
		return nil
	}

	checks, err := s.GetChecks()
	if err != nil {
		return errors.WithStack(err)
	}

	for _, c := range checkNames {
		if _, ok := checks[c]; !ok {
			return errors.Errorf("unknown check %s", c)
		}
	}

	errTx := s.db.InTransaction(func(tx *reform.TX) error {
		params := models.ChangeSettingsParams{DisableAdvisorChecks: checkNames}
		_, err := models.UpdateSettings(tx.Querier, &params)
		return err
	})
	if errTx != nil {
		return errors.Wrap(err, "failed to disable checks")
	}

	return nil
}

// EnableChecks enables checks with provided names.
func (s *Service) EnableChecks(checkNames []string) error {
	if len(checkNames) == 0 {
		return nil
	}

	err := s.db.InTransaction(func(tx *reform.TX) error {
		params := models.ChangeSettingsParams{EnableAdvisorChecks: checkNames}
		_, err := models.UpdateSettings(tx.Querier, &params)
		return err
	})
	if err != nil {
		return errors.Wrap(err, "failed to update disabled checks list")
	}

	return nil
}

// ChangeInterval changes a check's interval to the value received from the UI.
func (s *Service) ChangeInterval(params map[string]check.Interval) error {
	checks, err := s.GetChecks()
	if err != nil {
		return errors.WithStack(err)
	}

	for name, interval := range params {
		c, ok := checks[name]
		if !ok {
			return errors.Errorf("check: %s not found", name)
		}

		// since we re-run checks at regular intervals using a call
		// to s.runChecksGroup which in turn calls s.CollectAdvisors
		// to load/download checks, we must persist any changes
		// to check intervals in the DB so that they can be re-applied
		// once the checks have been re-loaded on restarts.
		errTx := s.db.InTransaction(func(tx *reform.TX) error {
			cs, err := models.FindCheckSettingsByName(tx.Querier, name)
			if err != nil && !errors.Is(err, reform.ErrNoRows) {
				return err
			}

			if cs == nil {
				// record interval change for the first time.
				_, err = models.CreateCheckSettings(tx.Querier, name, models.Interval(interval))
				if err != nil {
					return err
				}
				s.l.Debugf("Saved interval change for check: %s in DB", name)
			} else {
				// update existing interval change.
				_, err = models.ChangeCheckSettings(tx.Querier, name, models.Interval(interval))
				if err != nil {
					return err
				}
				s.l.Debugf("Updated interval change for check: %s in DB", name)
			}

			return nil
		})
		if errTx != nil {
			return errTx
		}

		s.l.Infof("Updated check: %s, interval changed from: %s to: %s", name, c.Interval, interval)
	}

	return nil
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
			return nil, errors.WithStack(ctx.Err())
		}

		res, err := models.FindActionResultByID(s.db.Querier, resultID)
		if err != nil {
			return nil, err
		}

		if !res.Done {
			continue
		}

		if res.Error != "" {
			return nil, errors.Errorf("action %s failed: %s", resultID, res.Error)
		}

		return []byte(res.Output), nil
	}
}

func (s *Service) minPMMAgentVersion(c check.Check) *version.Parsed {
	switch c.Version {
	case 1:
		return s.minPMMAgentVersionForType(c.Type)
	case 2:
		res := pmmAgent2_6_0 // minimum version that can be used with advisors
		for _, query := range c.Queries {
			v := s.minPMMAgentVersionForType(query.Type)
			if v != nil && res.Less(v) {
				res = v
			}
		}

		return res
	default:
		return pmmAgentInvalid
	}
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
		return pmmAgent2_6_0

	case check.MongoDBGetCmdLineOpts:
		return pmmAgent2_7_0

	case check.MongoDBReplSetGetStatus:
		fallthrough
	case check.MongoDBGetDiagnosticData:
		return pmmAgent2_27_0

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

// executeChecks runs checks for all reachable services. If intervalGroup specified only checks from that group will be
// executed. If checkNames specified then only matched checks will be executed.
func (s *Service) executeChecks(ctx context.Context, intervalGroup check.Interval, checkNames []string) ([]services.CheckResult, error) {
	disabledChecks, err := s.GetDisabledChecks()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var res []services.CheckResult
	checks, err := s.GetChecks()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	mySQLChecks, postgreSQLChecks, mongoDBChecks := groupChecksByDB(s.l, checks)

	mySQLChecks = s.filterChecks(mySQLChecks, intervalGroup, disabledChecks, checkNames)
	mySQLCheckResults := s.executeChecksForTargetType(ctx, models.MySQLServiceType, mySQLChecks)
	res = append(res, mySQLCheckResults...)

	postgreSQLChecks = s.filterChecks(postgreSQLChecks, intervalGroup, disabledChecks, checkNames)
	postgreSQLCheckResults := s.executeChecksForTargetType(ctx, models.PostgreSQLServiceType, postgreSQLChecks)
	res = append(res, postgreSQLCheckResults...)

	mongoDBChecks = s.filterChecks(mongoDBChecks, intervalGroup, disabledChecks, checkNames)
	mongoDBCheckResults := s.executeChecksForTargetType(ctx, models.MongoDBServiceType, mongoDBChecks)
	res = append(res, mongoDBCheckResults...)

	return res, nil
}

func (s *Service) executeChecksForTargetType(ctx context.Context, serviceType models.ServiceType, checks map[string]check.Check) []services.CheckResult {
	var res []services.CheckResult
	for _, c := range checks {
		s.l.Infof("Executing check: %s with interval: %s", c.Name, c.Interval)
		pmmAgentVersion := s.minPMMAgentVersion(c)
		targets, err := s.findTargets(serviceType, pmmAgentVersion)
		if err != nil {
			s.l.Warnf("Failed to find proper agents and services for check type: %s and "+
				"min version: %s, reason: %s.", c.Type, pmmAgentVersion, err)
			continue
		}

		for _, target := range targets {
			results, err := s.executeCheck(ctx, target, c)
			if err != nil {
				s.l.Warnf("Failed to execute check %s of type %s on target %s: %+v", c.Name, c.Type, target.AgentID, err)
				s.mChecksExecuted.WithLabelValues(string(target.ServiceType), c.Advisor, c.Name, "error").Inc()
				continue
			}

			res = append(res, results...)

			s.mChecksExecuted.WithLabelValues(string(target.ServiceType), c.Advisor, c.Name, "ok").Inc()
		}
	}

	return res
}

func (s *Service) executeCheck(ctx context.Context, target services.Target, c check.Check) ([]services.CheckResult, error) {
	ctx, cancel := context.WithTimeout(ctx, checkExecutionTimeout)
	defer cancel()

	defer func(t time.Time) {
		s.mChecksExecutionTime.WithLabelValues(string(target.ServiceType), c.Advisor, c.Name).Observe(time.Since(t).Seconds())
	}(time.Now())

	queries := c.Queries
	if c.Version == 1 {
		queries = []check.Query{{Type: c.Type, Query: c.Query}}
	}

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
			return nil, errors.Errorf("unknown check type")
		}
	}

	if err := eg.Wait(); err != nil {
		return nil, errors.Wrap(err, "check query failed")
	}

	res, err := s.processResults(ctx, c, target, resData)
	if err != nil {
		return nil, errors.Wrap(err, "failed to process query result")
	}

	return res, nil
}

func (s *Service) executeMySQLShowQuery(ctx context.Context, query check.Query, target services.Target) ([]byte, error) {
	r, err := models.CreateActionResult(s.db.Querier, target.AgentID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare result")
	}
	defer func() {
		if err = s.db.Delete(r); err != nil {
			s.l.Warnf("Failed to delete action result %s: %s.", r.ID, err)
		}
	}()

	if err = s.agentsRegistry.StartMySQLQueryShowAction(ctx, r.ID, target.AgentID, target.DSN, query.Query, target.Files, target.TDP, target.TLSSkipVerify); err != nil {
		return nil, errors.Wrap(err, "failed to start mySQL show action")
	}
	res, err := s.waitForResult(ctx, r.ID)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return res, nil
}

func (s *Service) executeMySQLSelectQuery(ctx context.Context, query check.Query, target services.Target) (string, error) {
	r, err := models.CreateActionResult(s.db.Querier, target.AgentID)
	if err != nil {
		return "", errors.Wrap(err, "failed to prepare result")
	}
	defer func() {
		if err = s.db.Delete(r); err != nil {
			s.l.Warnf("Failed to delete action result %s: %s.", r.ID, err)
		}
	}()

	if err = s.agentsRegistry.StartMySQLQuerySelectAction(ctx, r.ID, target.AgentID, target.DSN, query.Query, target.Files, target.TDP, target.TLSSkipVerify); err != nil { //nolint:lll
		return "", errors.Wrap(err, "failed to start mySQL select action")
	}
	res, err := s.waitForResult(ctx, r.ID)
	if err != nil {
		return "", errors.WithStack(err)
	}

	return b64.EncodeToString(res), nil
}

func (s *Service) executePostgreSQLShowQuery(ctx context.Context, target services.Target) (string, error) {
	r, err := models.CreateActionResult(s.db.Querier, target.AgentID)
	if err != nil {
		return "", errors.Wrap(err, "failed to prepare result")
	}
	defer func() {
		if err = s.db.Delete(r); err != nil {
			s.l.Warnf("Failed to delete action result %s: %s.", r.ID, err)
		}
	}()
	if err = s.agentsRegistry.StartPostgreSQLQueryShowAction(ctx, r.ID, target.AgentID, target.DSN); err != nil {
		return "", errors.Wrap(err, "failed to start postgreSQL show action")
	}

	res, err := s.waitForResult(ctx, r.ID)
	if err != nil {
		return "", errors.WithStack(err)
	}
	return b64.EncodeToString(res), nil
}

func (s *Service) executePostgreSQLSelectQuery(ctx context.Context, query check.Query, target services.Target) (any, error) {
	var allDBs bool
	var err error
	if value, ok := query.Parameters[check.AllDBs]; ok {
		if allDBs, err = strconv.ParseBool(value); err != nil {
			return nil, errors.Wrap(err, "failed to parse 'all_dbs' query parameter")
		}
	}

	if !allDBs {
		return s.executePostgreSQLSelectQueryForSingleDB(ctx, query, target)
	}

	targets, err := s.splitPGTargetByDB(ctx, target)
	if err != nil {
		return nil, errors.Wrap(err, "failed to split target by db")
	}
	res := make(map[string]string, len(targets))
	for dbName, t := range targets {
		if res[dbName], err = s.executePostgreSQLSelectQueryForSingleDB(ctx, query, t); err != nil {
			return nil, errors.WithStack(err)
		}
	}

	return res, nil
}

func (s *Service) executePostgreSQLSelectQueryForSingleDB(ctx context.Context, query check.Query, target services.Target) (string, error) {
	r, err := models.CreateActionResult(s.db.Querier, target.AgentID)
	if err != nil {
		return "", errors.Wrap(err, "failed to prepare result")
	}
	defer func() {
		if err = s.db.Delete(r); err != nil {
			s.l.Warnf("Failed to delete action result %s: %s.", r.ID, err)
		}
	}()

	if err = s.agentsRegistry.StartPostgreSQLQuerySelectAction(ctx, r.ID, target.AgentID, target.DSN, query.Query); err != nil {
		return "", errors.Wrap(err, "failed to start postgreSQL select action")
	}

	res, err := s.waitForResult(ctx, r.ID)
	if err != nil {
		return "", errors.WithStack(err)
	}

	return b64.EncodeToString(res), nil
}

func (s *Service) executeMongoDBGetParameterQuery(ctx context.Context, target services.Target) (string, error) {
	r, err := models.CreateActionResult(s.db.Querier, target.AgentID)
	if err != nil {
		return "", errors.Wrap(err, "failed to prepare result")
	}
	defer func() {
		if err = s.db.Delete(r); err != nil {
			s.l.Warnf("Failed to delete action result %s: %s.", r.ID, err)
		}
	}()

	if err = s.agentsRegistry.StartMongoDBQueryGetParameterAction(ctx, r.ID, target.AgentID, target.DSN, target.Files, target.TDP); err != nil {
		return "", errors.Wrap(err, "failed to start mongoDB getParameter action")
	}

	res, err := s.waitForResult(ctx, r.ID)
	if err != nil {
		return "", errors.WithStack(err)
	}

	return b64.EncodeToString(res), nil
}

func (s *Service) executeMongoDBBuildInfoQuery(ctx context.Context, target services.Target) (string, error) {
	r, err := models.CreateActionResult(s.db.Querier, target.AgentID)
	if err != nil {
		return "", errors.Wrap(err, "failed to prepare result")
	}
	defer func() {
		if err = s.db.Delete(r); err != nil {
			s.l.Warnf("Failed to delete action result %s: %s.", r.ID, err)
		}
	}()
	if err = s.agentsRegistry.StartMongoDBQueryBuildInfoAction(ctx, r.ID, target.AgentID, target.DSN, target.Files, target.TDP); err != nil {
		return "", errors.Wrap(err, "failed to start mongoDB buildInfo action")
	}

	res, err := s.waitForResult(ctx, r.ID)
	if err != nil {
		return "", errors.WithStack(err)
	}

	return b64.EncodeToString(res), nil
}

func (s *Service) executeMongoDBGetCmdLineOptsQuery(ctx context.Context, target services.Target) (string, error) {
	r, err := models.CreateActionResult(s.db.Querier, target.AgentID)
	if err != nil {
		return "", errors.Wrap(err, "failed to prepare result")
	}
	defer func() {
		if err = s.db.Delete(r); err != nil {
			s.l.Warnf("Failed to delete action result %s: %s.", r.ID, err)
		}
	}()

	if err = s.agentsRegistry.StartMongoDBQueryGetCmdLineOptsAction(ctx, r.ID, target.AgentID, target.DSN, target.Files, target.TDP); err != nil {
		return "", errors.Wrap(err, "failed to start mongoDB getCmdLineOpts action")
	}

	res, err := s.waitForResult(ctx, r.ID)
	if err != nil {
		return "", errors.WithStack(err)
	}

	return b64.EncodeToString(res), nil
}

func (s *Service) executeMongoDBReplSetGetStatusQuery(ctx context.Context, target services.Target) (string, error) {
	r, err := models.CreateActionResult(s.db.Querier, target.AgentID)
	if err != nil {
		return "", errors.Wrap(err, "failed to prepare result")
	}
	defer func() {
		if err = s.db.Delete(r); err != nil {
			s.l.Warnf("Failed to delete action result %s: %s.", r.ID, err)
		}
	}()

	if err = s.agentsRegistry.StartMongoDBQueryReplSetGetStatusAction(ctx, r.ID, target.AgentID, target.DSN, target.Files, target.TDP); err != nil {
		return "", errors.Wrap(err, "failed to start mongoDB replSetGetStatus action")
	}

	res, err := s.waitForResult(ctx, r.ID)
	if err != nil {
		return "", errors.WithStack(err)
	}

	return b64.EncodeToString(res), nil
}

func (s *Service) executeMongoDBGetDiagnosticQuery(ctx context.Context, target services.Target) (string, error) {
	r, err := models.CreateActionResult(s.db.Querier, target.AgentID)
	if err != nil {
		return "", errors.Wrap(err, "failed to prepare result")
	}
	defer func() {
		if err = s.db.Delete(r); err != nil {
			s.l.Warnf("Failed to delete action result %s: %s.", r.ID, err)
		}
	}()

	if err = s.agentsRegistry.StartMongoDBQueryGetDiagnosticDataAction(ctx, r.ID, target.AgentID, target.DSN, target.Files, target.TDP); err != nil {
		return "", errors.Wrap(err, "failed to start mongoDB getDiagnosticData action")
	}

	res, err := s.waitForResult(ctx, r.ID)
	if err != nil {
		return "", errors.WithStack(err)
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
		return "", errors.WithStack(err)
	}

	var lookback time.Time // if not specified use empty time which means "current time"
	if v, ok := query.Parameters[check.Lookback]; ok {
		d, err := time.ParseDuration(v)
		if err != nil {
			return "", errors.Wrap(err, "failed to parse 'lookback' query parameter")
		}

		lookback = time.Now().Add(-d)
	}

	r, warns, err := s.vmClient.Query(ctx, q, lookback)
	if err != nil {
		return "", errors.Wrap(err, "failed to execute instant VM query")
	}

	for _, warn := range warns {
		s.l.Warn(warn)
	}

	res, err := convertVMValue(r)
	if err != nil {
		return "", errors.WithStack(err)
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
		return "", errors.WithStack(err)
	}

	rng := v1.Range{
		End: time.Now(), // use current time as a default for the upper bound of the range
	}

	if v, ok := query.Parameters[check.Lookback]; ok {
		d, err := time.ParseDuration(v)
		if err != nil {
			return "", errors.Wrap(err, "failed to parse 'lookback' query parameter")
		}

		rng.End = time.Now().Add(-d)
	}

	rg, ok := query.Parameters[check.Range]
	if !ok {
		return "", errors.New("'range' query parameter is required for range queries")
	}

	d, err := time.ParseDuration(rg)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse 'range' query parameter")
	}

	rng.Start = rng.End.Add(-d)

	st, ok := query.Parameters[check.Step]
	if !ok {
		return "", errors.New("'step' query parameter is required for range queries")
	}

	rng.Step, err = time.ParseDuration(st)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse 'step' query parameter")
	}

	r, warns, err := s.vmClient.QueryRange(ctx, q, rng)
	if err != nil {
		return "", errors.Wrap(err, "failed to execute range VM query")
	}

	for _, warn := range warns {
		s.l.Warn(warn)
	}

	res, err := convertVMValue(r)
	if err != nil {
		return "", errors.WithStack(err)
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
		return "", errors.WithStack(err)
	}

	query = "SELECT " + query
	rows, err := s.clickhouseDB.QueryContext(ctx, query, nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to execute query")
	}

	columns, dataRows, err := sqlrows.ReadRows(rows)
	if err != nil {
		return "", errors.WithStack(err)
	}

	b, err := agentv1.MarshalActionQuerySQLResult(columns, dataRows)
	if err != nil {
		return "", errors.WithStack(err)
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
		return nil, errors.WithStack(err)
	}

	var data []map[string]any
	if err = json.Unmarshal(b, &data); err != nil {
		return nil, errors.WithStack(err)
	}

	res, err := agentv1.MarshalActionQueryDocsResult(data)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return res, nil
}

func (s *Service) discoverAvailablePGDatabases(ctx context.Context, target services.Target) ([]string, error) {
	query := check.Query{Query: `datname FROM pg_database  
WHERE datallowconn = true AND datistemplate = false AND has_database_privilege(current_user, datname, 'connect')`}

	res, err := s.executePostgreSQLSelectQueryForSingleDB(ctx, query, target)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to select available databases")
	}

	dec, err := b64.DecodeString(res)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode database discovery results")
	}

	data, err := agentv1.UnmarshalActionQueryResult(dec)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal database discovery results")
	}

	r := make([]string, len(data))
	for i, row := range data {
		datname, ok := row["datname"]
		if !ok {
			return nil, errors.New("missing expected 'datname' filed in query response")
		}
		name, ok := datname.(string)
		if !ok {
			return nil, errors.Errorf("unexpected type %T instead of string", datname)
		}

		r[i] = name
	}

	return r, nil
}

func (s *Service) splitPGTargetByDB(ctx context.Context, target services.Target) (map[string]services.Target, error) {
	dbNames, err := s.discoverAvailablePGDatabases(ctx, target)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	dsn, err := url.Parse(target.DSN)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse postrgeSQL DSN")
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
		return "", errors.Wrap(err, "failed to parse query")
	}

	var b strings.Builder
	if err = tm.Execute(&b, data); err != nil {
		return "", errors.Wrap(err, "failed to fill query placeholders")
	}

	return b.String(), nil
}

// StarlarkScriptData represents the data we need to pass to the binary to run starlark scripts.
type StarlarkScriptData struct {
	Version        uint32 `json:"version"`
	Name           string `json:"name"`
	Script         string `json:"script"`
	QueriesResults []any  `json:"queries_results"`
}

func (s *Service) processResults(ctx context.Context, aCheck check.Check, target services.Target, queryResults []any) ([]services.CheckResult, error) {
	l := s.l.WithFields(logrus.Fields{
		"name":       aCheck.Name,
		"service_id": target.ServiceID,
	})

	input := &StarlarkScriptData{
		Version:        aCheck.Version,
		Name:           aCheck.Name,
		Script:         aCheck.Script,
		QueriesResults: queryResults,
	}

	cmdCtx, cancel := context.WithTimeout(ctx, scriptExecutionTimeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "pmm-managed-starlark")
	pdeathsig.Set(cmd, syscall.SIGKILL)

	var stdin, stderr bytes.Buffer
	cmd.Stdin = &stdin
	cmd.Stderr = &stderr

	encoder := json.NewEncoder(&stdin)
	err := encoder.Encode(input)
	if err != nil {
		return nil, errors.Wrap(err, "error encoding data to STDIN")
	}

	procOut, err := cmd.Output()
	if err != nil {
		l.Errorf("Check script failed:\n%s", stderr.String())
		return nil, err
	}

	var results []check.Result
	decoder := json.NewDecoder(bytes.NewReader(procOut))
	err = decoder.Decode(&results)
	if err != nil {
		return nil, errors.Wrap(err, "error processing json output")
	}
	l.Infof("Check script returned %d results.", len(results))
	l.Debugf("Results: %+v.", results)

	checkResults := make([]services.CheckResult, len(results))
	for i, result := range results {
		checkResults[i] = services.CheckResult{
			CheckName:   aCheck.Name,
			AdvisorName: aCheck.Advisor,
			Interval:    aCheck.Interval,
			Target:      target,
			Result:      result,
		}
	}
	return checkResults, nil
}

// findTargets returns slice of available targets for specified service type.
func (s *Service) findTargets(serviceType models.ServiceType, minPMMAgentVersion *version.Parsed) ([]services.Target, error) {
	var targets []services.Target
	monitoredServices, err := models.FindServices(s.db.Querier, models.ServiceFilters{ServiceType: &serviceType})
	if err != nil {
		return nil, err
	}

	for _, service := range monitoredServices {
		// skip pmm own services
		if service.NodeID == models.PMMServerNodeID {
			s.l.Debugf("Skip PMM service, name: %s, type: %s.", service.ServiceName, service.ServiceType)
			continue
		}

		e := s.db.InTransaction(func(tx *reform.TX) error {
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
				AgentID:       pmmAgent.AgentID,
				ServiceID:     service.ServiceID,
				ServiceName:   service.ServiceName,
				ServiceType:   service.ServiceType,
				NodeName:      node.NodeName,
				Labels:        labels,
				DSN:           DSN,
				Files:         agent.Files(),
				TDP:           agent.TemplateDelimiters(service),
				TLSSkipVerify: agent.TLSSkipVerify,
			})
			return nil
		})
		if e != nil {
			s.l.Errorf("Failed to find agents for service %s, reason: %s.", service.ServiceID, e)
		}
	}

	return targets, nil
}

// CollectAdvisors loads advisors from builtin-in advisors directory or user-defined file, and stores versions this pmm-managed version can handle.
func (s *Service) CollectAdvisors(ctx context.Context) {
	var advisors []check.Advisor
	var err error

	defer s.refreshChecksInMemoryMetric()

	if s.customCheckFile != "" {
		s.l.Warnf("Using local test checks file: %s.", s.customCheckFile)
		checks, err := s.loadCustomChecks(s.customCheckFile)
		if err != nil {
			s.l.Errorf("Failed to load local checks file: %s.", err)
			return // keep previously loaded advisors
		}

		advisors = append(advisors, check.Advisor{
			Version:     1,
			Name:        "dev",
			Summary:     "Dev Advisor",
			Description: "Advisor used for developing checks",
			Category:    "development",
			Checks:      checks,
		})
	} else {
		s.l.Warnf("Using builtin test checks file: %s.", builtinAdvisorsPath)
		advisors, err = s.loadBuiltinAdvisors(ctx)
		if err != nil {
			s.l.Errorf("Failed to load built-in advisors: %s.", err)
			return // keep previously downloaded advisors
		}
	}

	s.updateAdvisors(s.filterSupportedChecks(advisors))
}

// loadLocalCheck loads checks from a local, user-defined file.
func (s *Service) loadCustomChecks(file string) ([]check.Check, error) {
	data, err := os.ReadFile(file) //nolint:gosec
	if err != nil {
		return nil, errors.Wrap(err, "failed to read test checks file")
	}

	// be strict about local files
	params := &check.ParseParams{
		DisallowUnknownFields: true,
		DisallowInvalidChecks: true,
	}
	checks, err := check.ParseChecks(bytes.NewReader(data), params)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse test checks file")
	}

	for _, c := range checks {
		if c.Advisor != "dev" {
			return nil, errors.Errorf("Local checks are supposed to be linked to the 'dev' advisor.") //nolint:revive
		}
	}

	return checks, nil
}

// loadBuiltinAdvisors loads builtin advisors.
func (s *Service) loadBuiltinAdvisors(_ context.Context) ([]check.Advisor, error) {
	s.l.Infof("Loading advisors from dir=%s", builtinAdvisorsPath)
	matches, err := filepath.Glob(filepath.Join(builtinAdvisorsPath, "*.yaml"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to find advisor files")
	}

	advisors, err := s.loadAdvisorsFromFiles(matches)
	if err != nil {
		return nil, err
	}

	s.l.Infof("Loading checks from dir=%s", builtinChecksPath)

	matches, err = filepath.Glob(filepath.Join(builtinChecksPath, "*.yaml"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to find check files")
	}

	checks, err := s.loadChecksFromFiles(matches)
	if err != nil {
		return nil, err
	}

	// Link checks to advisors
	for _, c := range checks {
		a, ok := advisors[c.Advisor]
		if !ok {
			return nil, errors.Errorf("check '%s' refers to an unknown advisor '%s'", c.Name, c.Advisor)
		}
		c.Category = a.Category // Inherit category form advisor for backward compatibility
		a.Checks = append(a.Checks, c)
	}

	advisorsSlice := make([]check.Advisor, 0, len(advisors))
	for _, a := range advisors {
		advisorsSlice = append(advisorsSlice, *a)
	}
	return advisorsSlice, nil
}

// loadChecksFromFiles loads Advisor checks from a list of given files.
func (s *Service) loadChecksFromFiles(files []string) ([]check.Check, error) {
	res := make([]check.Check, 0, len(files))
	for _, file := range files {
		s.l.Debugf("Loading check file=%s", file)

		b, err := os.ReadFile(file) //nolint:gosec
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read checks file %s", file)
		}
		checks, err := check.ParseChecks(bytes.NewReader(b), &check.ParseParams{
			DisallowUnknownFields: true,
			DisallowInvalidChecks: true,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse checks from file %s", file)
		}

		if len(checks) != 1 {
			return nil, errors.Errorf("expected exactly one check in %s", file)
		}
		c := checks[0]

		_, fileName := filepath.Split(file)
		if c.Name != strings.TrimSuffix(fileName, ".yaml") {
			return nil, errors.Errorf("check name does not match file name %s", file)
		}

		res = append(res, c)
	}

	return res, nil
}

// loadAdvisorsFromFiles loads Advisors from a list of given files.
func (s *Service) loadAdvisorsFromFiles(files []string) (map[string]*check.Advisor, error) {
	res := make(map[string]*check.Advisor, len(files))
	for _, file := range files {
		s.l.Infof("Loading advisor file=%s", file)

		b, err := os.ReadFile(file) //nolint:gosec
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read advisor file %s", file)
		}
		advisors, err := check.ParseAdvisors(bytes.NewReader(b), &check.ParseParams{
			DisallowUnknownFields: true,
			DisallowInvalidChecks: true,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse advisor from file %s", file)
		}

		if len(advisors) != 1 {
			return nil, errors.Errorf("expected exactly one advisor in %s", file)
		}
		a := advisors[0]

		_, fileName := filepath.Split(file)
		if a.Name != strings.TrimSuffix(fileName, ".yaml") {
			return nil, errors.Errorf("advisor name does not match file name %s", file)
		}

		if _, ok := res[a.Name]; ok {
			return nil, errors.Errorf("advisor name collision detected: %s", a.Name)
		}

		res[a.Name] = &a
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
			if c.Version > maxSupportedVersion {
				s.l.Warnf("Unsupported checks version: %d, max supported version: %d.", c.Version, maxSupportedVersion)
				continue LOOP
			}

			switch c.Version {
			case 1:
				if ok := isQueryTypeSupported(c.Type); !ok {
					s.l.Warnf("Unsupported check type: %s.", c.Type)
					continue LOOP
				}
			case 2:
				for _, query := range c.Queries {
					if ok := isQueryTypeSupported(query.Type); !ok {
						s.l.Warnf("Unsupported query type: %s.", query.Type)
						continue LOOP
					}
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
	s.rareTicker.Reset(rare)
	s.standardTicker.Reset(standard)
	s.frequentTicker.Reset(frequent)
	s.tm.Unlock()

	s.l.Infof("Intervals are changed: rare %s, standard %s, frequent %s", rare, standard, frequent)
}

// Describe implements prom.Collector.
func (s *Service) Describe(ch chan<- *prom.Desc) {
	s.mChecksExecuted.Describe(ch)
	s.mChecksAvailable.Describe(ch)
	s.mChecksExecutionTime.Describe(ch)

	s.alertsRegistry.Describe(ch)
}

// Collect implements prom.Collector.
func (s *Service) Collect(ch chan<- prom.Metric) {
	s.mChecksExecuted.Collect(ch)
	s.mChecksAvailable.Collect(ch)
	s.mChecksExecutionTime.Collect(ch)

	s.alertsRegistry.Collect(ch)
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
		s.mChecksAvailable.WithLabelValues(string(serviceType), c.Advisor, c.Name).Inc()
	}
}

// groupChecksByDB splits provided checks by database and returns three slices: for MySQL, for PostgreSQL and for MongoDB.
func groupChecksByDB(l *logrus.Entry, checks map[string]check.Check) (mySQLChecks, postgreSQLChecks, mongoDBChecks map[string]check.Check) { //nolint:nonamedreturns
	mySQLChecks = make(map[string]check.Check)
	postgreSQLChecks = make(map[string]check.Check)
	mongoDBChecks = make(map[string]check.Check)
	for _, c := range checks {
		switch c.GetFamily() {
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

	return
}

// check interfaces.
var (
	_ prom.Collector = (*Service)(nil)
)
