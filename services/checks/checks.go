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
	"io/ioutil"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	api "github.com/percona-platform/saas/gen/checked"
	"github.com/percona-platform/saas/pkg/check"
	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/utils/tlsconfig"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
)

const (
	defaultHost     = "check.percona.com:443"
	defaultInterval = 24 * time.Hour

	// Environment variables that affect checks service; only for testing.
	envHost      = "PERCONA_TEST_CHECKS_HOST"
	envPublicKey = "PERCONA_TEST_CHECKS_PUBLIC_KEY"
	envInterval  = "PERCONA_TEST_CHECKS_INTERVAL"
	envCheckFile = "PERCONA_TEST_CHECKS_FILE"

	checksTimeout       = time.Hour
	downloadTimeout     = 10 * time.Second
	resultTimeout       = 15 * time.Second
	resultCheckInterval = time.Second
)

var defaultPublicKeys = []string{
	"RWSKCHyoLDYxJ1k0qeayKu3/fsXVS1z8M+0deAClryiHWP99Sr4R/gPP", // PMM 2.6
}

// Service is responsible for interactions with Percona Check service.
type Service struct {
	l          *logrus.Entry
	pmmVersion string
	host       string
	publicKeys []string
	interval   time.Duration

	cm     sync.Mutex
	checks []check.Check

	registry registryService
	db       *reform.DB
}

// New returns Service with given PMM version.
func New(registry registryService, db *reform.DB, pmmVersion string) *Service {
	l := logrus.WithField("component", "check")
	s := &Service{
		l:          l,
		pmmVersion: pmmVersion,
		host:       defaultHost,
		publicKeys: defaultPublicKeys,
		interval:   defaultInterval,
		registry:   registry,
		db:         db,
	}

	if h := os.Getenv(envHost); h != "" {
		l.Warnf("Host changed to %s.", h)
		s.host = h
	}
	if k := os.Getenv(envPublicKey); k != "" {
		s.publicKeys = strings.Split(k, ",")
		l.Warnf("Public keys changed to %q.", k)
	}
	if d, err := time.ParseDuration(os.Getenv(envInterval)); err == nil && d > 0 {
		l.Warnf("Interval changed to %s.", d)
		s.interval = d
	}

	return s
}

// Run runs checks service that grabs checks from Percona Checks service every interval until context is canceled.
func (s *Service) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		// FIXME do that only if STT is enabled in settings https://jira.percona.com/browse/SAAS-30
		if true {
			nCtx, cancel := context.WithTimeout(ctx, checksTimeout)
			s.grabChecks(nCtx)
			s.executeChecks(nCtx)
			cancel()
		}

		select {
		case <-ticker.C:
			// continue with next loop iteration
		case <-ctx.Done():
			return
		}
	}
}

// Checks returns available checks.
func (s *Service) Checks() []check.Check {
	s.cm.Lock()
	defer s.cm.Unlock()

	r := make([]check.Check, 0, len(s.checks))
	return append(r, s.checks...)
}

// waitForResult periodically checks result state and returns it when complete.
func (s *Service) waitForResult(ctx context.Context, resultID string) (*models.ActionResult, error) {
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

		// FIXME we still need to delete old result - they may never be done: https://jira.percona.com/browse/PMM-5840
		if err = s.db.Delete(res); err != nil {
			s.l.Warnf("Failed to delete action result %s: %s.", resultID, err)
		}

		if res.Error != "" {
			return nil, errors.Errorf("Action %s failed: %s.", resultID, res.Error)
		}

		_, err = agentpb.UnmarshalActionQueryResult([]byte(res.Output))
		if err != nil {
			return nil, errors.Errorf("Failed to parse action result : %s.", err)
		}

		return res, nil
	}
}

// executeChecks runs all available checks for all reachable services.
func (s *Service) executeChecks(ctx context.Context) {
	mySQLChecks, postgreSQLChecks, mongoDBChecks := s.groupChecksByDB(s.checks)

	if err := s.executeMySQLChecks(ctx, mySQLChecks); err != nil {
		s.l.Errorf("Failed to execute MySQL checks: %s.", err)
	}

	if err := s.executePostgreSQLChecks(ctx, postgreSQLChecks); err != nil {
		s.l.Errorf("Failed to execute PostgreSQL checks: %s.", err)
	}

	if err := s.executeMongoDBChecks(ctx, mongoDBChecks); err != nil {
		s.l.Errorf("Failed to execute MongoDB checks: %s.", err)
	}
}

// executeMySQLChecks runs MySQL checks for available MySQL services.
func (s *Service) executeMySQLChecks(ctx context.Context, checks []check.Check) error {
	targets, err := s.findTargets(models.MySQLServiceType)
	if err != nil {
		return errors.Wrap(err, "failed to find proper agents and services")
	}

	for _, target := range targets {
		r, err := models.CreateActionResult(s.db.Querier, target.agentID)
		if err != nil {
			s.l.Errorf("Failed to prepare action result for agent %s: %s.", target.agentID, err)
			continue
		}

		for _, c := range checks {
			switch c.Type {
			case check.MySQLShow:
				if err := s.registry.StartMySQLQueryShowAction(ctx, r.ID, target.agentID, target.dsn, c.Query); err != nil {
					s.l.Errorf("Failed to start MySQL show query action for agent %s, reason: %s.", target.agentID, err)
					continue
				}
			case check.MySQLSelect:
				if err := s.registry.StartMySQLQuerySelectAction(ctx, r.ID, target.agentID, target.dsn, c.Query); err != nil {
					s.l.Errorf("Failed to start MySQL select query action for agent %s, reason: %s.", target.agentID, err)
					continue
				}
			default:
				s.l.Errorf("Unknown MySQL check type: %s.", c.Type)
				continue
			}

			nCtx, cancel := context.WithTimeout(ctx, resultTimeout)
			_, err := s.waitForResult(nCtx, r.ID) // TODO returns result
			cancel()
			if err != nil {
				s.l.Errorf("failed to get check result: %s", err)
				continue
			}

			// TODO process result
		}
	}

	return nil
}

// executePostgreSQLChecks runs PostgreSQL checks for available PostgreSQL services.
func (s *Service) executePostgreSQLChecks(ctx context.Context, checks []check.Check) error {
	targets, err := s.findTargets(models.PostgreSQLServiceType)
	if err != nil {
		return errors.Wrap(err, "failed to find proper agents and services")
	}

	for _, target := range targets {
		r, err := models.CreateActionResult(s.db.Querier, target.agentID)
		if err != nil {
			s.l.Errorf("Failed to prepare action result for agent %s: %s.", target.agentID, err)
			continue
		}

		for _, c := range checks {
			switch c.Type {
			case check.PostgreSQLShow:
				if err := s.registry.StartPostgreSQLQueryShowAction(ctx, r.ID, target.agentID, target.dsn); err != nil {
					s.l.Errorf("Failed to start PostgreSQL show query action for agent %s, reason: %s.", target.agentID, err)
					continue
				}
			case check.PostgreSQLSelect:
				if err := s.registry.StartPostgreSQLQuerySelectAction(ctx, r.ID, target.agentID, target.dsn, c.Query); err != nil {
					s.l.Errorf("Failed to start PostgreSQL select query action for agent %s, reason: %s.", target.agentID, err)
					continue
				}
			default:
				s.l.Errorf("Unknown PostgresSQL check type: %s.", c.Type)
				continue
			}

			nCtx, cancel := context.WithTimeout(ctx, resultTimeout)
			_, err := s.waitForResult(nCtx, r.ID) // TODO returns result
			cancel()
			if err != nil {
				s.l.Errorf("failed to get check result: %s", err)
				continue
			}

			// TODO process result
		}
	}

	return nil
}

// executeMongoDBChecks runs MongoDB checks for available MongoDB services.
func (s *Service) executeMongoDBChecks(ctx context.Context, checks []check.Check) error {
	targets, err := s.findTargets(models.MongoDBServiceType)
	if err != nil {
		return errors.Wrap(err, "failed to find proper agents and services")
	}

	for _, target := range targets {
		r, err := models.CreateActionResult(s.db.Querier, target.agentID)
		if err != nil {
			s.l.Errorf("Failed to prepare action result for agent %s: %s.", target.agentID, err)
			continue
		}

		for _, c := range checks {
			switch c.Type {
			case check.MongoDBGetParameter:
				if err := s.registry.StartMongoDBQueryGetParameterAction(context.Background(), r.ID, target.agentID, target.dsn); err != nil {
					s.l.Errorf("Failed to start MongoDB get parameter query action for agent %s, reason: %s.", target.agentID, err)
					continue
				}
			case check.MongoDBBuildInfo:
				if err := s.registry.StartMongoDBQueryBuildInfoAction(context.Background(), r.ID, target.agentID, target.dsn); err != nil {
					s.l.Errorf("Failed to start MongoDB build info query action for agent %s, reason: %s.", target.agentID, err)
					continue
				}

			default:
				s.l.Errorf("Unknown MongoDB check type: %s.", c.Type)
				continue
			}

			nCtx, cancel := context.WithTimeout(ctx, resultTimeout)
			_, err := s.waitForResult(nCtx, r.ID) // TODO returns result
			cancel()
			if err != nil {
				s.l.Errorf("failed to get check result: %s", err)
				continue
			}

			// TODO process result
		}
	}

	return nil
}

// target contains required info about check target
type target struct {
	agentID   string
	serviceID string
	dsn       string
}

// findTargets returns slice of available targets for specified service type.
func (s *Service) findTargets(serviceType models.ServiceType) ([]target, error) {
	var targets []target
	services, err := models.FindServices(s.db.Querier, models.ServiceFilters{ServiceType: &serviceType})
	if err != nil {
		return nil, err
	}

	for _, service := range services {
		e := s.db.InTransaction(func(tx *reform.TX) error {
			a, err := models.FindPMMAgentsForService(s.db.Querier, service.ServiceID)
			if err != nil {
				return err
			}
			if len(a) == 0 {
				return errors.New("no available pmm agents")
			}

			dsn, err := models.FindDSNByServiceIDandPMMAgentID(s.db.Querier, service.ServiceID, a[0].AgentID, "")
			if err != nil {
				return err
			}
			targets = append(targets, target{agentID: a[0].AgentID, serviceID: service.ServiceID, dsn: dsn})
			return nil
		})
		if e != nil {
			s.l.Errorf("Failed to find agents for service %s, reason: %s", service.ServiceID, err)
		}
	}

	return targets, nil
}

// groupChecksByDB splits provided checks by database and returns three slices: for MySQL, for PostgreSQL and for MongoDB.
func (s *Service) groupChecksByDB(checks []check.Check) ([]check.Check, []check.Check, []check.Check) {
	var mySQLChecks, postgreSQLChecks, mongoDBChecks []check.Check

	for _, c := range checks {
		switch c.Type {
		case check.MySQLSelect:
			fallthrough
		case check.MySQLShow:
			mySQLChecks = append(mySQLChecks, c)

		case check.PostgreSQLSelect:
			fallthrough
		case check.PostgreSQLShow:
			postgreSQLChecks = append(postgreSQLChecks, c)

		case check.MongoDBGetParameter:
			fallthrough
		case check.MongoDBBuildInfo:
			mongoDBChecks = append(mongoDBChecks, c)

		default:
			s.l.Warnf("Unknown check type %s, skip it.", c.Type)
		}
	}

	return mySQLChecks, postgreSQLChecks, mongoDBChecks
}

// grabChecks loads checks list.
func (s *Service) grabChecks(ctx context.Context) {
	if f := os.Getenv(envCheckFile); f != "" {
		s.l.Warnf("Using local test checks file: %s.", f)
		if err := s.loadLocalChecks(f); err != nil {
			s.l.Errorf("Failed to load local checks file: %s.", err)
		}
		return
	}

	if err := s.downloadChecks(ctx); err != nil {
		s.l.Errorf("Failed to download checks: %s.", err)
	}
}

// loadLocalCheck loads checks form local file.
func (s *Service) loadLocalChecks(file string) error {
	data, err := ioutil.ReadFile(file) //nolint:gosec
	if err != nil {
		return errors.Wrap(err, "failed to read test checks file")
	}
	checks, err := check.Parse(bytes.NewReader(data))
	if err != nil {
		return errors.Wrap(err, "failed to parse test checks file")
	}

	s.updateChecks(checks)
	return nil
}

// downloadChecks downloads checks form percona service endpoint.
func (s *Service) downloadChecks(ctx context.Context) error {
	s.l.Infof("Downloading checks from %s ...", s.host)

	host, _, err := net.SplitHostPort(s.host)
	if err != nil {
		return errors.Wrap(err, "failed to set checks host")
	}
	tlsConfig := tlsconfig.Get()
	tlsConfig.ServerName = host

	opts := []grpc.DialOption{
		// replacement is marked as experimental
		grpc.WithBackoffMaxDelay(downloadTimeout), //nolint:staticcheck

		grpc.WithBlock(),
		grpc.WithUserAgent("pmm-managed/" + s.pmmVersion),
		grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
	}

	ctx, cancel := context.WithTimeout(ctx, downloadTimeout)
	defer cancel()
	cc, err := grpc.DialContext(ctx, s.host, opts...)
	if err != nil {
		return errors.Wrap(err, "failed to dial")
	}
	defer cc.Close() //nolint:errcheck

	resp, err := api.NewCheckedAPIClient(cc).GetAllChecks(ctx, &api.GetAllChecksRequest{})
	if err != nil {
		return errors.Wrap(err, "failed to request checks service")
	}

	if err = s.verifySignatures(resp); err != nil {
		return err
	}

	checks, err := check.Parse(strings.NewReader(resp.File))
	if err != nil {
		return err
	}

	s.updateChecks(checks)
	return nil
}

// updateChecks update service checks filed value under mutex.
func (s *Service) updateChecks(checks []check.Check) {
	s.cm.Lock()
	defer s.cm.Unlock()

	s.checks = checks
}

// verifySignatures verifies checks signatures and returns error in case of verification problem.
func (s *Service) verifySignatures(resp *api.GetAllChecksResponse) error {
	if len(resp.Signatures) == 0 {
		return errors.New("zero signatures received")
	}

	var err error
	for _, sign := range resp.Signatures {
		for _, key := range s.publicKeys {
			if err = check.Verify([]byte(resp.File), key, sign); err == nil {
				return nil
			}
			s.l.Debugf("Key %q doesn't match signature %q: %s.", key, sign, err)
		}
	}

	return errors.New("no verified signatures")
}
