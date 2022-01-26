// pmm-managed
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
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	api "github.com/percona-platform/saas/gen/check/retrieval"
	"github.com/percona-platform/saas/pkg/check"
	"github.com/percona/pmm/utils/pdeathsig"
	"github.com/percona/pmm/version"
	"github.com/pkg/errors"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/services"
	"github.com/percona/pmm-managed/utils/envvars"
	"github.com/percona/pmm-managed/utils/saasreq"
	"github.com/percona/pmm-managed/utils/signatures"
)

const (
	defaultStartDelay = time.Minute

	// Environment variables that affect checks service; only for testing.
	envCheckFile         = "PERCONA_TEST_CHECKS_FILE"
	envResendInterval    = "PERCONA_TEST_CHECKS_RESEND_INTERVAL"
	envDisableStartDelay = "PERCONA_TEST_CHECKS_DISABLE_START_DELAY"

	checkExecutionTimeout  = 5 * time.Minute  // limits execution time for every single check
	platformRequestTimeout = 2 * time.Minute  // time limit to get checks list from the platform
	resultAwaitTimeout     = 20 * time.Second // should be greater than agents.defaultQueryActionTimeout
	scriptExecutionTimeout = 5 * time.Second  // time limit for running pmm-managed-starlark
	resultCheckInterval    = time.Second

	// sync with API tests
	resolveTimeoutFactor  = 3
	defaultResendInterval = 2 * time.Second

	prometheusNamespace = "pmm_managed"
	prometheusSubsystem = "checks"

	alertsPrefix        = "/stt/"
	maxSupportedVersion = 1
)

// pmm-agent versions with known changes in Query Actions.
var (
	pmmAgent260     = version.MustParse("2.6.0")
	pmmAgent270     = version.MustParse("2.7.0")
	pmmAgentInvalid = version.MustParse("3.0.0-invalid")
)

// Service is responsible for interactions with Percona Check service.
type Service struct {
	agentsRegistry      agentsRegistry
	alertmanagerService alertmanagerService
	db                  *reform.DB
	alertsRegistry      *registry

	l               *logrus.Entry
	host            string
	publicKeys      []string
	startDelay      time.Duration
	resendInterval  time.Duration
	localChecksFile string // For testing

	cm               sync.Mutex
	mySQLChecks      map[string]check.Check
	postgreSQLChecks map[string]check.Check
	mongoDBChecks    map[string]check.Check

	tm             sync.Mutex
	rareTicker     *time.Ticker
	standardTicker *time.Ticker
	frequentTicker *time.Ticker

	mScriptsExecuted *prom.CounterVec
	mAlertsGenerated *prom.CounterVec
}

// New returns Service with given PMM version.
func New(agentsRegistry agentsRegistry, alertmanagerService alertmanagerService, db *reform.DB) (*Service, error) {
	l := logrus.WithField("component", "checks")

	resendInterval := defaultResendInterval
	if d, err := time.ParseDuration(os.Getenv(envResendInterval)); err == nil && d > 0 {
		l.Warnf("Interval changed to %s.", d)
		resendInterval = d
	}

	host, err := envvars.GetSAASHost()
	if err != nil {
		return nil, err
	}

	s := &Service{
		agentsRegistry:      agentsRegistry,
		alertmanagerService: alertmanagerService,
		db:                  db,
		alertsRegistry:      newRegistry(resolveTimeoutFactor * resendInterval),

		l:               l,
		host:            host,
		startDelay:      defaultStartDelay,
		resendInterval:  defaultResendInterval,
		localChecksFile: os.Getenv(envCheckFile),

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

	if k := envvars.GetPublicKeys(); k != nil {
		l.Warnf("Public keys changed to %q.", k)
		s.publicKeys = k
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

	return s, nil
}

// Run runs main service loops.
func (s *Service) Run(ctx context.Context) {
	s.l.Info("Starting...")
	defer s.l.Info("Done.")

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
func (s *Service) GetSecurityCheckResults() ([]services.STTCheckResult, error) {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		return nil, err
	}

	if !settings.SaaS.STTEnabled {
		return nil, services.ErrSTTDisabled
	}

	return s.alertsRegistry.getCheckResults(), nil
}

// runChecksGroup downloads and executes STT checks in synchronous way.
// If intervalGroup is empty.
func (s *Service) runChecksGroup(ctx context.Context, intervalGroup check.Interval) error {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		return errors.WithStack(err)
	}

	if !settings.SaaS.STTEnabled {
		return services.ErrSTTDisabled
	}

	return s.run(ctx, intervalGroup, nil)
}

// StartChecks downloads and executes STT checks in asynchronous way.
// If checkNames specified then only matched checks will be executed.
func (s *Service) StartChecks(checkNames []string) error {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		return errors.WithStack(err)
	}

	if !settings.SaaS.STTEnabled {
		return services.ErrSTTDisabled
	}

	go func() {
		if err := s.run(context.Background(), "", checkNames); err != nil {
			s.l.Errorf("Failed to execute STT checks: %+v.", err)
		}
	}()

	return nil
}

func (s *Service) run(ctx context.Context, intervalGroup check.Interval, checkNames []string) error {
	if err := intervalGroup.Validate(); err != nil {
		return errors.WithStack(err)
	}

	s.collectChecks(ctx)

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

// getMySQLChecks returns available MySQL checks.
func (s *Service) getMySQLChecks() map[string]check.Check {
	s.cm.Lock()
	defer s.cm.Unlock()

	r := make(map[string]check.Check)
	for k, v := range s.mySQLChecks {
		r[k] = v
	}
	return r
}

// getPostgreSQLChecks returns available PostgreSQL checks.
func (s *Service) getPostgreSQLChecks() map[string]check.Check {
	s.cm.Lock()
	defer s.cm.Unlock()

	r := make(map[string]check.Check)
	for k, v := range s.postgreSQLChecks {
		r[k] = v
	}
	return r
}

// getMongoDBChecks returns available MongoDB checks.
func (s *Service) getMongoDBChecks() map[string]check.Check {
	s.cm.Lock()
	defer s.cm.Unlock()

	r := make(map[string]check.Check)
	for k, v := range s.mongoDBChecks {
		r[k] = v
	}
	return r
}

// GetAllChecks returns all available checks.
func (s *Service) GetAllChecks() map[string]check.Check {
	checks := make(map[string]check.Check)
	for k, v := range s.getMySQLChecks() {
		checks[k] = v
	}

	for k, v := range s.getPostgreSQLChecks() {
		checks[k] = v
	}

	for k, v := range s.getMongoDBChecks() {
		checks[k] = v
	}
	return checks
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

	m := s.GetAllChecks()

	for _, c := range checkNames {
		if _, ok := m[c]; !ok {
			return errors.Errorf("unknown check %s", c)
		}
	}

	err := s.db.InTransaction(func(tx *reform.TX) error {
		params := models.ChangeSettingsParams{DisableSTTChecks: checkNames}
		_, err := models.UpdateSettings(tx.Querier, &params)
		return err
	})
	if err != nil {
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
	checkMap := s.GetAllChecks()
	if len(checkMap) == 0 {
		return errors.New("no checks loaded")
	}

	for name, interval := range params {
		c, ok := checkMap[name]
		if !ok {
			return errors.Errorf("check: %s not found", name)
		}
		c.Interval = interval

		// since we re-run checks at regular intervals using a call
		// to s.runChecksGroup which in turn calls s.collectChecks
		// to load/download checks, we must persist any changes
		// to check intervals in the DB so that they can be re-applied
		// once the checks have been re-loaded on restarts.
		e := s.db.InTransaction(func(tx *reform.TX) error {
			cs, err := models.FindCheckSettingsByName(tx.Querier, c.Name)
			// record interval change for the first time.
			if err == reform.ErrNoRows {
				cs, err := models.CreateCheckSettings(tx.Querier, c.Name, models.Interval(c.Interval))
				if err != nil {
					return err
				}
				s.l.Debugf("Saved interval change for check: %s in DB", cs.Name)
				s.updateCheck(c)
				return nil
			}

			// update existing interval change.
			if cs != nil {
				cs, err := models.ChangeCheckSettings(tx.Querier, c.Name, models.Interval(c.Interval))
				if err != nil {
					return err
				}
				s.l.Debugf("Updated interval change for check: %s in DB", cs.Name)
				s.updateCheck(c)
				return nil
			}
			return err
		})
		if e != nil {
			return e
		}
	}

	return nil
}

// updateCheck updates a check with an updated interval in the appropriate check group.
func (s *Service) updateCheck(newCheck check.Check) {
	switch newCheck.Type {
	case check.MySQLSelect:
		fallthrough
	case check.MySQLShow:
		s.cm.Lock()
		defer s.cm.Unlock()
		oldCheck := s.mySQLChecks[newCheck.Name]
		s.mySQLChecks[newCheck.Name] = newCheck
		s.l.Infof("Updated check: %s, interval changed from: %s to: %s", oldCheck.Name, oldCheck.Interval, newCheck.Interval)

	case check.PostgreSQLSelect:
		fallthrough
	case check.PostgreSQLShow:
		s.cm.Lock()
		defer s.cm.Unlock()
		oldCheck := s.postgreSQLChecks[newCheck.Name]
		s.postgreSQLChecks[newCheck.Name] = newCheck
		s.l.Infof("Updated check: %s, interval changed from: %s to: %s", oldCheck.Name, oldCheck.Interval, newCheck.Interval)

	case check.MongoDBGetParameter:
		fallthrough
	case check.MongoDBBuildInfo:
		fallthrough
	case check.MongoDBGetCmdLineOpts:
		s.cm.Lock()
		defer s.cm.Unlock()
		oldCheck := s.mongoDBChecks[newCheck.Name]
		s.mongoDBChecks[newCheck.Name] = newCheck
		s.l.Infof("Updated check: %s, interval changed from: %s to: %s", oldCheck.Name, oldCheck.Interval, newCheck.Interval)

	default:
		s.l.Warnf("Unknown check type %s, skip it.", newCheck.Type)
	}
}

// waitForResult periodically checks result state and returns it when complete.
func (s *Service) waitForResult(ctx context.Context, resultID string) ([]byte, error) {
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

		if err = s.db.Delete(res); err != nil {
			s.l.Warnf("Failed to delete action result %s: %s.", resultID, err)
		}

		if res.Error != "" {
			return nil, errors.Errorf("action %s failed: %s", resultID, res.Error)
		}

		return []byte(res.Output), nil
	}
}

// minPMMAgentVersion returns the minimal version of pmm-agent that can handle the given check type.
func (s *Service) minPMMAgentVersion(t check.Type) *version.Parsed {
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
		return pmmAgent260

	case check.MongoDBGetCmdLineOpts:
		return pmmAgent270

	default:
		s.l.Warnf("minPMMAgentVersion: unhandled check type %q.", t)
		return pmmAgentInvalid
	}
}

// filterChecks filters checks by several parameters. If group specified then only matched checks will be returned,
// empty group means `any interval`. If enable slice is specified then only matched checks will be returned, empty
// enable slice means `all enabled`. Checks specified in disabled slice are skipped, empty `disabled` slice means
// `nothing disabled`.
func (s *Service) filterChecks(checks map[string]check.Check, group check.Interval, disable, enable []string) []check.Check {
	var res []check.Check
	disableMap := make(map[string]struct{}, len(disable))
	for _, e := range disable {
		disableMap[e] = struct{}{}
	}

	enableMap := make(map[string]struct{}, len(enable))
	for _, e := range enable {
		enableMap[e] = struct{}{}
	}

	for _, c := range checks {
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

				res = append(res, c)
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

	var checkResults []services.STTCheckResult

	mySQLChecks := s.filterChecks(s.getMySQLChecks(), intervalGroup, disabledChecks, checkNames)
	mySQLCheckResults := s.executeMySQLChecks(ctx, mySQLChecks)
	checkResults = append(checkResults, mySQLCheckResults...)

	postgreSQLChecks := s.filterChecks(s.getPostgreSQLChecks(), intervalGroup, disabledChecks, checkNames)
	postgreSQLCheckResults := s.executePostgreSQLChecks(ctx, postgreSQLChecks)
	checkResults = append(checkResults, postgreSQLCheckResults...)

	mongoDBChecks := s.filterChecks(s.getMongoDBChecks(), intervalGroup, disabledChecks, checkNames)
	mongoDBCheckResults := s.executeMongoDBChecks(ctx, mongoDBChecks)
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

// executeMySQLChecks runs specified checks for available MySQL service.
func (s *Service) executeMySQLChecks(ctx context.Context, checks []check.Check) []services.STTCheckResult {
	var res []services.STTCheckResult
	for _, c := range checks {
		s.l.Infof("Executing check: %s with interval: %s", c.Name, c.Interval)
		pmmAgentVersion := s.minPMMAgentVersion(c.Type)
		targets, err := s.findTargets(models.MySQLServiceType, pmmAgentVersion)
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

			s.mScriptsExecuted.WithLabelValues(string(models.MySQLServiceType)).Inc()
			s.mAlertsGenerated.WithLabelValues(string(models.MySQLServiceType), string(c.Type)).Add(float64(len(results)))
		}
	}

	return res
}

// executePostgreSQLChecks runs specified PostgreSQL checks for available PostgreSQL services.
func (s *Service) executePostgreSQLChecks(ctx context.Context, checks []check.Check) []services.STTCheckResult {
	var res []services.STTCheckResult
	for _, c := range checks {
		s.l.Infof("Executing check: %s with interval: %s", c.Name, c.Interval)
		pmmAgentVersion := s.minPMMAgentVersion(c.Type)
		targets, err := s.findTargets(models.PostgreSQLServiceType, pmmAgentVersion)
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

			s.mScriptsExecuted.WithLabelValues(string(models.PostgreSQLServiceType)).Inc()
			s.mAlertsGenerated.WithLabelValues(string(models.PostgreSQLServiceType), string(c.Type)).Add(float64(len(results)))
		}
	}

	return res
}

// executeMongoDBChecks runs specified MongoDB checks for available MongoDB services.
func (s *Service) executeMongoDBChecks(ctx context.Context, checks []check.Check) []services.STTCheckResult {
	var res []services.STTCheckResult
	for _, c := range checks {
		s.l.Infof("Executing check: %s with interval: %s", c.Name, c.Interval)
		pmmAgentVersion := s.minPMMAgentVersion(c.Type)
		targets, err := s.findTargets(models.MongoDBServiceType, pmmAgentVersion)
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

			s.mScriptsExecuted.WithLabelValues(string(models.MongoDBServiceType)).Inc()
			s.mAlertsGenerated.WithLabelValues(string(models.MongoDBServiceType), string(c.Type)).Add(float64(len(results)))
		}
	}

	return res
}

func (s *Service) executeCheck(ctx context.Context, target services.Target, c check.Check) ([]services.STTCheckResult, error) {
	nCtx, cancel := context.WithTimeout(ctx, checkExecutionTimeout)
	defer cancel()

	r, err := models.CreateActionResult(s.db.Querier, target.AgentID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare result")
	}

	switch c.Type {
	case check.MySQLShow:
		err = s.agentsRegistry.StartMySQLQueryShowAction(nCtx, r.ID, target.AgentID, target.DSN, c.Query, target.Files, target.TDP, target.TLSSkipVerify)
	case check.MySQLSelect:
		err = s.agentsRegistry.StartMySQLQuerySelectAction(nCtx, r.ID, target.AgentID, target.DSN, c.Query, target.Files, target.TDP, target.TLSSkipVerify)
	case check.PostgreSQLShow:
		err = s.agentsRegistry.StartPostgreSQLQueryShowAction(ctx, r.ID, target.AgentID, target.DSN)
	case check.PostgreSQLSelect:
		err = s.agentsRegistry.StartPostgreSQLQuerySelectAction(ctx, r.ID, target.AgentID, target.DSN, c.Query)
	case check.MongoDBGetParameter:
		err = s.agentsRegistry.StartMongoDBQueryGetParameterAction(ctx, r.ID, target.AgentID, target.DSN, target.Files, target.TDP)
	case check.MongoDBBuildInfo:
		err = s.agentsRegistry.StartMongoDBQueryBuildInfoAction(ctx, r.ID, target.AgentID, target.DSN, target.Files, target.TDP)
	case check.MongoDBGetCmdLineOpts:
		err = s.agentsRegistry.StartMongoDBQueryGetCmdLineOptsAction(ctx, r.ID, target.AgentID, target.DSN, target.Files, target.TDP)
	default:
		return nil, errors.Errorf("unknown check type")
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to start query")
	}

	res, err := s.processResults(nCtx, c, target, r.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to process action result")
	}

	return res, nil
}

// StarlarkScriptData represents the data we need to pass to the binary to run starlark scripts.
type StarlarkScriptData struct {
	Version     uint32 `json:"version"`
	Name        string `json:"name"`
	Script      string `json:"script"`
	QueryResult []byte `json:"query_result"`
}

func (s *Service) processResults(ctx context.Context, sttCheck check.Check, target services.Target, resID string) ([]services.STTCheckResult, error) {
	nCtx, cancel := context.WithTimeout(ctx, resultAwaitTimeout)
	r, err := s.waitForResult(nCtx, resID)
	cancel()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get action result")
	}

	l := s.l.WithFields(logrus.Fields{
		"name":       sttCheck.Name,
		"id":         resID,
		"service_id": target.ServiceID,
	})

	input := &StarlarkScriptData{
		Version:     sttCheck.Version,
		Name:        sttCheck.Name,
		Script:      sttCheck.Script,
		QueryResult: r,
	}

	cmdCtx, cancel := context.WithTimeout(ctx, scriptExecutionTimeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "pmm-managed-starlark")
	pdeathsig.Set(cmd, syscall.SIGKILL)

	var stdin, stderr bytes.Buffer
	cmd.Stdin = &stdin
	cmd.Stderr = &stderr

	encoder := json.NewEncoder(&stdin)
	err = encoder.Encode(input)
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

	checkResults := make([]services.STTCheckResult, len(results))
	for i, result := range results {
		checkResults[i] = services.STTCheckResult{
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
func (s *Service) groupChecksByDB(checks []check.Check) (mySQLChecks, postgreSQLChecks, mongoDBChecks map[string]check.Check) {
	mySQLChecks = make(map[string]check.Check)
	postgreSQLChecks = make(map[string]check.Check)
	mongoDBChecks = make(map[string]check.Check)
	for _, c := range checks {
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
			mongoDBChecks[c.Name] = c

		default:
			s.l.Warnf("Unknown check type %s, skip it.", c.Type)
		}
	}

	return
}

// filterChecksByInterval filters checks according to their interval buckets.
func filterChecksByInterval(checks []check.Check, interval check.Interval) []check.Check {
	if interval == "" { // all checks
		return checks
	}

	var res []check.Check
	for _, c := range checks {
		// Empty check interval equals standard interval.
		if c.Interval == interval || (interval == check.Standard && c.Interval == "") {
			res = append(res, c)
		}
	}

	return res
}

// collectChecks loads checks from file or SaaS, and stores versions this pmm-managed can handle.
func (s *Service) collectChecks(ctx context.Context) {
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

	checks = s.filterSupportedChecks(checks)

	checkSettingsMap, err := models.FindCheckSettings(s.db.Querier)
	if err != nil && err != reform.ErrNoRows {
		s.l.Errorf("Failed to retrieve checks settings: %s.", err)
		return
	}

	for i, c := range checks {
		if interval, ok := checkSettingsMap[c.Name]; ok {
			c.Interval = check.Interval(interval)
			checks[i] = c
		}
	}

	mySQLChecks, postgreSQLChecks, mongoDBChecks := s.groupChecksByDB(checks)

	s.updateChecks(mySQLChecks, postgreSQLChecks, mongoDBChecks)
}

// loadLocalCheck loads checks form local file.
func (s *Service) loadLocalChecks(file string) ([]check.Check, error) {
	data, err := ioutil.ReadFile(file) //nolint:gosec
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
	s.l.Infof("Downloading checks from %s ...", s.host)

	nCtx, cancel := context.WithTimeout(ctx, platformRequestTimeout)
	defer cancel()

	var accessToken string
	if ssoDetails, err := models.GetPerconaSSODetails(nCtx, s.db.Querier); err == nil {
		accessToken = ssoDetails.AccessToken.AccessToken
	}

	endpoint := fmt.Sprintf("https://%s/v1/check/GetAllChecks", s.host)
	bodyBytes, err := saasreq.MakeRequest(nCtx, http.MethodPost, endpoint, accessToken, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to dial")
	}

	var resp *api.GetAllChecksResponse
	if err := json.Unmarshal(bodyBytes, &resp); err != nil {
		return nil, err
	}

	if err = signatures.Verify(s.l, resp.File, resp.Signatures, s.publicKeys); err != nil {
		return nil, err
	}

	// be liberal about files from SaaS for smooth transition to future versions
	params := &check.ParseParams{
		DisallowUnknownFields: false,
		DisallowInvalidChecks: false,
	}
	checks, err := check.Parse(strings.NewReader(resp.File), params)
	if err != nil {
		return nil, err
	}

	return checks, nil
}

// filterSupportedChecks returns supported checks and prints warning log messages about unsupported.
func (s *Service) filterSupportedChecks(checks []check.Check) []check.Check {
	res := make([]check.Check, 0, len(checks))
	for _, c := range checks {
		if c.Version > maxSupportedVersion {
			s.l.Warnf("Unsupported checks version: %d, max supported version: %d.", c.Version, maxSupportedVersion)
			continue
		}

		switch c.Type {
		case check.MySQLShow:
		case check.MySQLSelect:
		case check.PostgreSQLShow:
		case check.PostgreSQLSelect:
		case check.MongoDBGetParameter:
		case check.MongoDBBuildInfo:
		case check.MongoDBGetCmdLineOpts:
		default:
			s.l.Warnf("Unsupported check type: %s.", c.Type)
			continue
		}

		res = append(res, c)
	}

	return res
}

// updateChecks update service checks filed value under mutex.
func (s *Service) updateChecks(mySQLChecks, postgreSQLChecks, mongoDBChecks map[string]check.Check) {
	s.cm.Lock()
	defer s.cm.Unlock()

	s.mySQLChecks = mySQLChecks
	s.postgreSQLChecks = postgreSQLChecks
	s.mongoDBChecks = mongoDBChecks
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

// check interfaces
var (
	_ prom.Collector = (*Service)(nil)
)
