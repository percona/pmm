// Copyright (C) 2017 Percona LLC
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

// Package checks provides security checks functionality.
package checks

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"text/template"
	"time"

	"github.com/percona-platform/saas/pkg/check"
	"github.com/percona-platform/saas/pkg/common"
	"github.com/pkg/errors"
	metrics "github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
	"github.com/percona/pmm/managed/utils/envvars"
	"github.com/percona/pmm/managed/utils/platform"
	"github.com/percona/pmm/managed/utils/signatures"
	"github.com/percona/pmm/utils/pdeathsig"
	"github.com/percona/pmm/version"
)

const (
	defaultStartDelay = time.Minute

	// Environment variables that affect checks service; only for testing.
	envCheckFile         = "PERCONA_TEST_CHECKS_FILE"
	envResendInterval    = "PERCONA_TEST_CHECKS_RESEND_INTERVAL"
	envDisableStartDelay = "PERCONA_TEST_CHECKS_DISABLE_START_DELAY"

	checkExecutionTimeout  = 5 * time.Minute  // limits execution time for every single check
	platformRequestTimeout = 2 * time.Minute  // time limit to get checks list from the portal
	resultAwaitTimeout     = 20 * time.Second // should be greater than agents.defaultQueryActionTimeout
	scriptExecutionTimeout = 5 * time.Second  // time limit for running pmm-managed-starlark
	resultCheckInterval    = time.Second

	// Sync with API tests.
	resolveTimeoutFactor  = 3
	defaultResendInterval = 2 * time.Second

	prometheusNamespace = "pmm_managed"
	prometheusSubsystem = "checks"

	alertsPrefix        = "/stt/"
	maxSupportedVersion = 2
)

// pmm-agent versions with known changes in Query Actions.
// To match all pre-release versions add '-0' suffix to specified version.
var (
	pmmAgent2_6_0   = version.MustParse("2.6.0")
	pmmAgent2_7_0   = version.MustParse("2.7.0")
	pmmAgent2_27_0  = version.MustParse("2.27.0-0")
	pmmAgentInvalid = version.MustParse("3.0.0-invalid")
)

// Service is responsible for interactions with Percona Check service.
type Service struct {
	platformClient      *platform.Client
	agentsRegistry      agentsRegistry
	alertmanagerService alertmanagerService
	db                  *reform.DB
	alertsRegistry      *registry
	vmClient            v1.API

	l                  *logrus.Entry
	startDelay         time.Duration
	resendInterval     time.Duration
	platformPublicKeys []string
	localChecksFile    string // For testing

	cm     sync.Mutex
	checks map[string]check.Check

	tm             sync.Mutex
	rareTicker     *time.Ticker
	standardTicker *time.Ticker
	frequentTicker *time.Ticker

	mScriptsExecuted *prom.CounterVec
	mAlertsGenerated *prom.CounterVec
}

// New returns Service with given PMM version.
func New(db *reform.DB, platformClient *platform.Client, agentsRegistry agentsRegistry, alertmanagerService alertmanagerService, VMAddress string) (*Service, error) {
	l := logrus.WithField("component", "checks")

	resendInterval := defaultResendInterval
	if d, err := time.ParseDuration(os.Getenv(envResendInterval)); err == nil && d > 0 {
		l.Warnf("Interval changed to %s.", d)
		resendInterval = d
	}

	vmClient, err := metrics.NewClient(metrics.Config{Address: VMAddress})
	if err != nil {
		return nil, err
	}

	var platformPublicKeys []string
	if k := envvars.GetPlatformPublicKeys(); k != nil {
		l.Warnf("Percona Platform public keys changed to %q.", k)
		platformPublicKeys = k
	}

	s := &Service{
		db:                  db,
		agentsRegistry:      agentsRegistry,
		alertmanagerService: alertmanagerService,
		alertsRegistry:      newRegistry(resolveTimeoutFactor * resendInterval),
		vmClient:            v1.NewAPI(vmClient),

		l:                  l,
		platformClient:     platformClient,
		startDelay:         defaultStartDelay,
		resendInterval:     resendInterval,
		platformPublicKeys: platformPublicKeys,
		localChecksFile:    os.Getenv(envCheckFile),

		mScriptsExecuted: prom.NewCounterVec(prom.CounterOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "scripts_executed_total",
			Help:      "Counter of check scripts executed per service type",
		}, []string{"service_type"}),

		mAlertsGenerated: prom.NewCounterVec(prom.CounterOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "alerts_generated_total",
			Help:      "Counter of alerts generated per service type per check type",
		}, []string{"service_type", "check_type"}),
	}

	if d, _ := strconv.ParseBool(os.Getenv(envDisableStartDelay)); d {
		l.Warn("Start delay disabled.")
		s.startDelay = 0
	}

	s.mScriptsExecuted.WithLabelValues(string(models.MySQLServiceType))
	s.mScriptsExecuted.WithLabelValues(string(models.PostgreSQLServiceType))
	s.mScriptsExecuted.WithLabelValues(string(models.MongoDBServiceType))

	s.mAlertsGenerated.WithLabelValues(string(models.MySQLServiceType), string(check.MySQLShow))
	s.mAlertsGenerated.WithLabelValues(string(models.MySQLServiceType), string(check.MySQLSelect))
	s.mAlertsGenerated.WithLabelValues(string(models.PostgreSQLServiceType), string(check.PostgreSQLShow))
	s.mAlertsGenerated.WithLabelValues(string(models.PostgreSQLServiceType), string(check.PostgreSQLSelect))
	s.mAlertsGenerated.WithLabelValues(string(models.MongoDBServiceType), string(check.MongoDBBuildInfo))
	s.mAlertsGenerated.WithLabelValues(string(models.MongoDBServiceType), string(check.MongoDBGetCmdLineOpts))
	s.mAlertsGenerated.WithLabelValues(string(models.MongoDBServiceType), string(check.MongoDBGetParameter))
	s.mAlertsGenerated.WithLabelValues(string(models.MongoDBServiceType), string(check.MongoDBReplSetGetStatus))
	s.mAlertsGenerated.WithLabelValues(string(models.MongoDBServiceType), string(check.MongoDBGetDiagnosticData))

	return s, nil
}

// Run runs main service loops.
func (s *Service) Run(ctx context.Context) {
	s.l.Info("Starting...")
	defer s.l.Info("Done.")

	s.CollectChecks(ctx)
	settings, err := models.GetSettings(s.db)
	if err != nil {
		s.l.Errorf("Failed to get settings: %+v.", err)
		return
	}

	s.rareTicker = time.NewTicker(settings.SaaS.STTCheckIntervals.RareInterval)
	defer s.rareTicker.Stop()

	s.standardTicker = time.NewTicker(settings.SaaS.STTCheckIntervals.StandardInterval)
	defer s.standardTicker.Stop()

	s.frequentTicker = time.NewTicker(settings.SaaS.STTCheckIntervals.FrequentInterval)
	defer s.frequentTicker.Stop()

	// delay for the first run to allow all agents to connect
	startCtx, startCancel := context.WithTimeout(ctx, s.startDelay)
	<-startCtx.Done()
	startCancel()
	if ctx.Err() != nil { // check main context, not startCtx
		return
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		s.resendAlerts(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		s.runChecksLoop(ctx)
	}()

	wg.Wait()
}

// resendAlerts resends collected alerts until ctx is canceled.
func (s *Service) resendAlerts(ctx context.Context) {
	t := time.NewTicker(s.resendInterval)
	defer t.Stop()

	for {
		s.alertmanagerService.SendAlerts(ctx, s.alertsRegistry.collect())

		select {
		case <-ctx.Done():
			return
		case <-t.C:
			// nothing, continue for loop
		}
	}
}

// runChecksLoop starts checks execution loop.
func (s *Service) runChecksLoop(ctx context.Context) {
	// First checks run, start all checks from all groups.
	err := s.runChecksGroup(ctx, "") // start all checks
	for {
		switch err {
		case nil:
			// nothing, continue
		case services.ErrSTTDisabled:
			s.l.Info("STT is not enabled, doing nothing.")
		default:
			s.l.Error(err)
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

// GetSecurityCheckResults returns the results of the STT checks that were run. It returns services.ErrSTTDisabled if STT is disabled.
func (s *Service) GetSecurityCheckResults() ([]services.CheckResult, error) {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		return nil, err
	}

	if settings.SaaS.STTDisabled {
		return nil, services.ErrSTTDisabled
	}

	return s.alertsRegistry.getCheckResults(), nil
}

// GetChecksResults returns the failed checks for a given service from AlertManager.
func (s *Service) GetChecksResults(ctx context.Context, serviceID string) ([]services.CheckResult, error) {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		return nil, err
	}

	if settings.SaaS.STTDisabled {
		return nil, services.ErrSTTDisabled
	}

	filters := &services.FilterParams{
		IsCheck:   true,
		ServiceID: serviceID,
	}
	res, err := s.alertmanagerService.GetAlerts(ctx, filters)
	if err != nil {
		return nil, err
	}

	checkResults := make([]services.CheckResult, 0, len(res))
	for _, alert := range res {
		checkResults = append(checkResults, services.CheckResult{
			CheckName: alert.Labels[model.AlertNameLabel],
			Silenced:  len(alert.Status.SilencedBy) != 0,
			AlertID:   alert.Labels["alert_id"],
			Interval:  check.Interval(alert.Labels["interval_group"]),
			Target: services.Target{
				AgentID:     alert.Labels["agent_id"],
				ServiceID:   alert.Labels["service_id"],
				ServiceName: alert.Labels["service_name"],
				NodeName:    alert.Labels["node_name"],
				Labels:      alert.Labels,
			},
			Result: check.Result{
				Summary:     alert.Annotations["summary"],
				Description: alert.Annotations["description"],
				ReadMoreURL: alert.Annotations["read_more_url"],
				Severity:    common.ParseSeverity(alert.Labels["severity"]),
				Labels:      alert.Labels,
			},
		})
	}
	return checkResults, nil
}

// ToggleCheckAlert toggles the silence state of the check with the provided alertID.
func (s *Service) ToggleCheckAlert(ctx context.Context, alertID string, silence bool) error {
	filters := &services.FilterParams{
		IsCheck: true,
		AlertID: alertID,
	}
	res, err := s.alertmanagerService.GetAlerts(ctx, filters)
	if err != nil {
		return errors.Wrapf(err, "failed to get alerts with id: %s", alertID)
	}

	if silence {
		err = s.alertmanagerService.SilenceAlerts(ctx, res)
	} else {
		err = s.alertmanagerService.UnsilenceAlerts(ctx, res)
	}

	return err
}

// runChecksGroup downloads and executes Advisors checks that should run in the interval specified by intervalGroup.
// All checks are executed if intervalGroup is empty.
func (s *Service) runChecksGroup(ctx context.Context, intervalGroup check.Interval) error {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		return errors.WithStack(err)
	}

	if settings.SaaS.STTDisabled {
		return services.ErrSTTDisabled
	}

	s.CollectChecks(ctx)
	return s.run(ctx, intervalGroup, nil)
}

// StartChecks downloads and executes STT checks in asynchronous way.
// If checkNames specified then only matched checks will be executed.
func (s *Service) StartChecks(checkNames []string) error {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		return errors.WithStack(err)
	}

	if settings.SaaS.STTDisabled {
		return services.ErrSTTDisabled
	}

	go func() {
		ctx := context.Background()
		s.CollectChecks(ctx)
		if err := s.run(ctx, "", checkNames); err != nil {
			s.l.Errorf("Failed to execute STT checks: %+v.", err)
		}
	}()

	return nil
}

func (s *Service) run(ctx context.Context, intervalGroup check.Interval, checkNames []string) error {
	if err := intervalGroup.Validate(); err != nil {
		return errors.WithStack(err)
	}

	if err := s.executeChecks(ctx, intervalGroup, checkNames); err != nil {
		return errors.WithStack(err)
	}

	s.alertmanagerService.SendAlerts(ctx, s.alertsRegistry.collect())

	return nil
}

// CleanupAlerts drops all alerts in registry.
func (s *Service) CleanupAlerts() {
	s.alertsRegistry.cleanup()
}

// GetChecks returns all available checks.
func (s *Service) GetChecks() (map[string]check.Check, error) {
	cs, err := models.FindCheckSettings(s.db.Querier)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	s.cm.Lock()
	defer s.cm.Unlock()

	r := make(map[string]check.Check, len(s.checks))
	for n, c := range s.checks {
		if interval, ok := cs[n]; ok {
			c.Interval = check.Interval(interval)
		}
		r[n] = c
	}
	return r, nil
}

// GetDisabledChecks returns disabled checks.
func (s *Service) GetDisabledChecks() ([]string, error) {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		return nil, err
	}

	return settings.SaaS.DisabledSTTChecks, nil
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
		params := models.ChangeSettingsParams{DisableSTTChecks: checkNames}
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
		params := models.ChangeSettingsParams{EnableSTTChecks: checkNames}
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
		// to s.runChecksGroup which in turn calls s.CollectChecks
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
func (s *Service) executeChecks(ctx context.Context, intervalGroup check.Interval, checkNames []string) error {
	disabledChecks, err := s.GetDisabledChecks()
	if err != nil {
		return errors.WithStack(err)
	}

	var checkResults []services.CheckResult
	checks, err := s.GetChecks()
	if err != nil {
		return errors.WithStack(err)
	}
	mySQLChecks, postgreSQLChecks, mongoDBChecks := s.groupChecksByDB(checks)

	mySQLChecks = s.filterChecks(mySQLChecks, intervalGroup, disabledChecks, checkNames)
	mySQLCheckResults := s.executeChecksForTargetType(ctx, models.MySQLServiceType, mySQLChecks)
	checkResults = append(checkResults, mySQLCheckResults...)

	postgreSQLChecks = s.filterChecks(postgreSQLChecks, intervalGroup, disabledChecks, checkNames)
	postgreSQLCheckResults := s.executeChecksForTargetType(ctx, models.PostgreSQLServiceType, postgreSQLChecks)
	checkResults = append(checkResults, postgreSQLCheckResults...)

	mongoDBChecks = s.filterChecks(mongoDBChecks, intervalGroup, disabledChecks, checkNames)
	mongoDBCheckResults := s.executeChecksForTargetType(ctx, models.MongoDBServiceType, mongoDBChecks)
	checkResults = append(checkResults, mongoDBCheckResults...)

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

	s.alertsRegistry.set(checkResults)

	return nil
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
				continue
			}
			res = append(res, results...)

			s.mScriptsExecuted.WithLabelValues(string(serviceType)).Inc()
			s.mAlertsGenerated.WithLabelValues(string(serviceType), string(c.Type)).Add(float64(len(results)))
		}
	}

	return res
}

func (s *Service) executeCheck(ctx context.Context, target services.Target, c check.Check) ([]services.CheckResult, error) {
	ctx, cancel := context.WithTimeout(ctx, checkExecutionTimeout)
	defer cancel()

	queries := c.Queries
	if c.Version == 1 {
		queries = []check.Query{{Type: c.Type, Query: c.Query}}
	}

	eg, gCtx := errgroup.WithContext(ctx)
	resData := make([][]byte, len(queries))

	for i, query := range queries {
		i, query := i, query
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
				resData[i], err = s.executePostrgreSQLSelectQuery(gCtx, query, target)
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

		default:
			return nil, errors.Errorf("unknown check type")
		}
	}

	if err := eg.Wait(); err != nil {
		return nil, errors.Wrap(err, "check query failed")
	}

	res, err := s.processResults(ctx, c, target, resData)
	if err != nil {
		return nil, errors.Wrap(err, "failed to process action result")
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

func (s *Service) executeMySQLSelectQuery(ctx context.Context, query check.Query, target services.Target) ([]byte, error) {
	r, err := models.CreateActionResult(s.db.Querier, target.AgentID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare result")
	}
	defer func() {
		if err = s.db.Delete(r); err != nil {
			s.l.Warnf("Failed to delete action result %s: %s.", r.ID, err)
		}
	}()

	if err = s.agentsRegistry.StartMySQLQuerySelectAction(ctx, r.ID, target.AgentID, target.DSN, query.Query, target.Files, target.TDP, target.TLSSkipVerify); err != nil { //nolint:lll
		return nil, errors.Wrap(err, "failed to start mySQL select action")
	}
	res, err := s.waitForResult(ctx, r.ID)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return res, nil
}

func (s *Service) executePostgreSQLShowQuery(ctx context.Context, target services.Target) ([]byte, error) {
	r, err := models.CreateActionResult(s.db.Querier, target.AgentID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare result")
	}
	defer func() {
		if err = s.db.Delete(r); err != nil {
			s.l.Warnf("Failed to delete action result %s: %s.", r.ID, err)
		}
	}()
	if err = s.agentsRegistry.StartPostgreSQLQueryShowAction(ctx, r.ID, target.AgentID, target.DSN); err != nil {
		return nil, errors.Wrap(err, "failed to start postgreSQL show action")
	}

	res, err := s.waitForResult(ctx, r.ID)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return res, nil
}

func (s *Service) executePostrgreSQLSelectQuery(ctx context.Context, query check.Query, target services.Target) ([]byte, error) {
	r, err := models.CreateActionResult(s.db.Querier, target.AgentID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare result")
	}
	defer func() {
		if err = s.db.Delete(r); err != nil {
			s.l.Warnf("Failed to delete action result %s: %s.", r.ID, err)
		}
	}()

	if err = s.agentsRegistry.StartPostgreSQLQuerySelectAction(ctx, r.ID, target.AgentID, target.DSN, query.Query); err != nil {
		return nil, errors.Wrap(err, "failed to start postgreSQL select action")
	}

	res, err := s.waitForResult(ctx, r.ID)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return res, nil
}

func (s *Service) executeMongoDBGetParameterQuery(ctx context.Context, target services.Target) ([]byte, error) {
	r, err := models.CreateActionResult(s.db.Querier, target.AgentID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare result")
	}
	defer func() {
		if err = s.db.Delete(r); err != nil {
			s.l.Warnf("Failed to delete action result %s: %s.", r.ID, err)
		}
	}()

	if err = s.agentsRegistry.StartMongoDBQueryGetParameterAction(ctx, r.ID, target.AgentID, target.DSN, target.Files, target.TDP); err != nil {
		return nil, errors.Wrap(err, "failed to start mongoDB getParameter action")
	}

	res, err := s.waitForResult(ctx, r.ID)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return res, nil
}

func (s *Service) executeMongoDBBuildInfoQuery(ctx context.Context, target services.Target) ([]byte, error) {
	r, err := models.CreateActionResult(s.db.Querier, target.AgentID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare result")
	}
	defer func() {
		if err = s.db.Delete(r); err != nil {
			s.l.Warnf("Failed to delete action result %s: %s.", r.ID, err)
		}
	}()
	if err = s.agentsRegistry.StartMongoDBQueryBuildInfoAction(ctx, r.ID, target.AgentID, target.DSN, target.Files, target.TDP); err != nil {
		return nil, errors.Wrap(err, "failed to start mongoDB buildInfo action")
	}

	res, err := s.waitForResult(ctx, r.ID)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return res, nil
}

func (s *Service) executeMongoDBGetCmdLineOptsQuery(ctx context.Context, target services.Target) ([]byte, error) {
	r, err := models.CreateActionResult(s.db.Querier, target.AgentID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare result")
	}
	defer func() {
		if err = s.db.Delete(r); err != nil {
			s.l.Warnf("Failed to delete action result %s: %s.", r.ID, err)
		}
	}()

	if err = s.agentsRegistry.StartMongoDBQueryGetCmdLineOptsAction(ctx, r.ID, target.AgentID, target.DSN, target.Files, target.TDP); err != nil {
		return nil, errors.Wrap(err, "failed to start mongoDB getCmdLineOpts action")
	}

	res, err := s.waitForResult(ctx, r.ID)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return res, nil
}

func (s *Service) executeMongoDBReplSetGetStatusQuery(ctx context.Context, target services.Target) ([]byte, error) {
	r, err := models.CreateActionResult(s.db.Querier, target.AgentID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare result")
	}
	defer func() {
		if err = s.db.Delete(r); err != nil {
			s.l.Warnf("Failed to delete action result %s: %s.", r.ID, err)
		}
	}()

	if err = s.agentsRegistry.StartMongoDBQueryReplSetGetStatusAction(ctx, r.ID, target.AgentID, target.DSN, target.Files, target.TDP); err != nil {
		return nil, errors.Wrap(err, "failed to start mongoDB replSetGetStatus action")
	}

	res, err := s.waitForResult(ctx, r.ID)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return res, nil
}

func (s *Service) executeMongoDBGetDiagnosticQuery(ctx context.Context, target services.Target) ([]byte, error) {
	r, err := models.CreateActionResult(s.db.Querier, target.AgentID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare result")
	}
	defer func() {
		if err = s.db.Delete(r); err != nil {
			s.l.Warnf("Failed to delete action result %s: %s.", r.ID, err)
		}
	}()

	if err = s.agentsRegistry.StartMongoDBQueryGetDiagnosticDataAction(ctx, r.ID, target.AgentID, target.DSN, target.Files, target.TDP); err != nil {
		return nil, errors.Wrap(err, "failed to start mongoDB getDiagnosticData action")
	}

	res, err := s.waitForResult(ctx, r.ID)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return res, nil
}

func (s *Service) executeMetricsInstantQuery(ctx context.Context, query check.Query, target services.Target) ([]byte, error) {
	q, err := fillQueryPlaceholders(query.Query, target)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var lookback time.Time // if not specified use empty time which means "current time"
	if v, ok := query.Parameters[check.Lookback]; ok {
		d, err := time.ParseDuration(v)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse 'lookback' query parameter")
		}

		lookback = time.Now().Add(-d)
	}

	r, warns, err := s.vmClient.Query(ctx, q, lookback)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute instant VM query")
	}

	for _, warn := range warns {
		s.l.Warn(warn)
	}

	res, err := convertVMValue(r)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return res, nil
}

func (s *Service) executeMetricsRangeQuery(ctx context.Context, query check.Query, target services.Target) ([]byte, error) {
	q, err := fillQueryPlaceholders(query.Query, target)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	rng := v1.Range{
		End: time.Now(), // use current time as a default for the upper bound of the range
	}

	if v, ok := query.Parameters[check.Lookback]; ok {
		d, err := time.ParseDuration(v)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse 'lookback' query parameter")
		}

		rng.End = time.Now().Add(-d)
	}

	rg, ok := query.Parameters[check.Range]
	if !ok {
		return nil, errors.New("'range' query parameter is required for range queries")
	}

	d, err := time.ParseDuration(rg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse 'range' query parameter")
	}

	rng.Start = rng.End.Add(-d)

	st, ok := query.Parameters[check.Step]
	if !ok {
		return nil, errors.New("'step' query parameter is required for range queries")
	}

	rng.Step, err = time.ParseDuration(st)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse 'step' query parameter")
	}

	r, warns, err := s.vmClient.QueryRange(ctx, q, rng)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute range VM query")
	}

	for _, warn := range warns {
		s.l.Warn(warn)
	}

	res, err := convertVMValue(r)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return res, nil
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

	var data []map[string]interface{}
	if err = json.Unmarshal(b, &data); err != nil {
		return nil, errors.WithStack(err)
	}

	res, err := agentpb.MarshalActionQueryDocsResult(data)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return res, nil
}

func fillQueryPlaceholders(query string, target services.Target) (string, error) {
	tm, err := template.New("query").Parse(query)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse query")
	}

	data := struct {
		ServiceName string
		NodeName    string
	}{
		ServiceName: target.ServiceName,
		NodeName:    target.NodeName,
	}

	var b strings.Builder
	if err = tm.Execute(&b, data); err != nil {
		return "", errors.Wrap(err, "failed to fill query placeholders")
	}

	return b.String(), nil
}

// StarlarkScriptData represents the data we need to pass to the binary to run starlark scripts.
type StarlarkScriptData struct {
	Version        uint32   `json:"version"`
	Name           string   `json:"name"`
	Script         string   `json:"script"`
	QueriesResults [][]byte `json:"queries_results"`
}

func (s *Service) processResults(ctx context.Context, sttCheck check.Check, target services.Target, queryResults [][]byte) ([]services.CheckResult, error) {
	l := s.l.WithFields(logrus.Fields{
		"name":       sttCheck.Name,
		"service_id": target.ServiceID,
	})

	input := &StarlarkScriptData{
		Version:        sttCheck.Version,
		Name:           sttCheck.Name,
		Script:         sttCheck.Script,
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
			CheckName: sttCheck.Name,
			Interval:  sttCheck.Interval,
			Target:    target,
			Result:    result,
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

// groupChecksByDB splits provided checks by database and returns three slices: for MySQL, for PostgreSQL and for MongoDB.
func (s *Service) groupChecksByDB(checks map[string]check.Check) (mySQLChecks, postgreSQLChecks, mongoDBChecks map[string]check.Check) {
	mySQLChecks = make(map[string]check.Check)
	postgreSQLChecks = make(map[string]check.Check)
	mongoDBChecks = make(map[string]check.Check)
	for _, c := range checks {
		switch c.Version {
		case 1:
			switch c.Type {
			case check.MySQLSelect:
				fallthrough
			case check.MySQLShow:
				mySQLChecks[c.Name] = c

			case check.PostgreSQLSelect:
				fallthrough
			case check.PostgreSQLShow:
				postgreSQLChecks[c.Name] = c

			case check.MongoDBGetParameter:
				fallthrough
			case check.MongoDBBuildInfo:
				fallthrough
			case check.MongoDBGetCmdLineOpts:
				fallthrough
			case check.MongoDBReplSetGetStatus:
				fallthrough
			case check.MongoDBGetDiagnosticData:
				mongoDBChecks[c.Name] = c

			default:
				s.l.Warnf("Unknown check type %s, skip it.", c.Type)
			}
		case 2:
			switch c.Family {
			case check.MySQL:
				mySQLChecks[c.Name] = c
			case check.PostgreSQL:
				postgreSQLChecks[c.Name] = c
			case check.MongoDB:
				mongoDBChecks[c.Name] = c
			default:
				s.l.Warnf("Unknown check family %s, skip it.", c.Family)
			}
		}
	}

	return
}

// CollectChecks loads checks from file or SaaS, and stores versions this pmm-managed can handle.
func (s *Service) CollectChecks(ctx context.Context) {
	var checks []check.Check
	var err error
	if s.localChecksFile != "" {
		s.l.Warnf("Using local test checks file: %s.", s.localChecksFile)
		checks, err = s.loadLocalChecks(s.localChecksFile)
		if err != nil {
			s.l.Errorf("Failed to load local checks file: %s.", err)
			return // keep previously loaded checks
		}
	} else {
		checks, err = s.downloadChecks(ctx)
		if err != nil {
			s.l.Errorf("Failed to download checks: %s.", err)
			return // keep previously downloaded checks
		}
	}

	s.updateChecks(s.filterSupportedChecks(checks))
}

// loadLocalCheck loads checks form local file.
func (s *Service) loadLocalChecks(file string) ([]check.Check, error) {
	data, err := os.ReadFile(file) //nolint:gosec
	if err != nil {
		return nil, errors.Wrap(err, "failed to read test checks file")
	}

	// be strict about local files
	params := &check.ParseParams{
		DisallowUnknownFields: true,
		DisallowInvalidChecks: true,
	}
	checks, err := check.Parse(bytes.NewReader(data), params)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse test checks file")
	}

	return checks, nil
}

// downloadChecks downloads checks form percona service endpoint.
func (s *Service) downloadChecks(ctx context.Context) ([]check.Check, error) {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		return nil, err
	}

	if settings.Telemetry.Disabled {
		s.l.Debug("Checks downloading skipped due to disabled telemetry.")
		return nil, nil
	}

	nCtx, cancel := context.WithTimeout(ctx, platformRequestTimeout)
	defer cancel()

	resp, err := s.platformClient.GetChecks(nCtx)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if err = signatures.Verify(s.l, resp.File, resp.Signatures, s.platformPublicKeys); err != nil {
		return nil, err
	}

	// be liberal about files from SaaS for smooth transition to future versions
	params := &check.ParseParams{
		DisallowUnknownFields: false,
		DisallowInvalidChecks: false,
	}

	checks, err := check.Parse(strings.NewReader(resp.File), params)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return checks, nil
}

// filterSupportedChecks returns supported checks and prints warning log messages about unsupported.
func (s *Service) filterSupportedChecks(checks []check.Check) []check.Check {
	res := make([]check.Check, 0, len(checks))

checksLoop:
	for _, c := range checks {
		if c.Version > maxSupportedVersion {
			s.l.Warnf("Unsupported checks version: %d, max supported version: %d.", c.Version, maxSupportedVersion)
			continue
		}

		switch c.Version {
		case 1:
			if ok := isQueryTypeSupported(c.Type); !ok {
				s.l.Warnf("Unsupported check type: %s.", c.Type)
				continue
			}
		case 2:
			for _, query := range c.Queries {
				if ok := isQueryTypeSupported(query.Type); !ok {
					s.l.Warnf("Unsupported query type: %s.", query.Type)
					continue checksLoop
				}
			}
		}

		res = append(res, c)
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
	default:
		return false
	}

	return true
}

// updateChecks update service checks filed value under mutex.
func (s *Service) updateChecks(checks []check.Check) {
	s.cm.Lock()
	defer s.cm.Unlock()

	s.checks = make(map[string]check.Check)
	for _, c := range checks {
		s.checks[c.Name] = c
	}
}

// UpdateIntervals updates STT restart timers intervals.
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
	s.mScriptsExecuted.Describe(ch)
	s.mAlertsGenerated.Describe(ch)
}

// Collect implements prom.Collector.
func (s *Service) Collect(ch chan<- prom.Metric) {
	s.mScriptsExecuted.Collect(ch)
	s.mAlertsGenerated.Collect(ch)
}

// check interfaces.
var (
	_ prom.Collector = (*Service)(nil)
)
