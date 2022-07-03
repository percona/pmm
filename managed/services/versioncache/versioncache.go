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

// Package versioncache provides service software version cache functionality.
package versioncache

import (
	"context"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/agents"
)

var (
	startupDelay              = 10 * time.Second
	serviceCheckInterval      = 24 * time.Hour
	serviceCheckShortInterval = 4 * time.Hour
	minCheckInterval          = 5 * time.Second
)

//go:generate ../../../bin/mockery -name=Versioner -case=snake -inpkg -testonly

// Versioner contains method for retrieving versions of different software.
type Versioner interface {
	GetVersions(pmmAgentID string, softwares []agents.Software) ([]agents.Version, error)
}

// Service is responsible for caching service software versions in the DB.
type Service struct {
	db       *reform.DB
	l        *logrus.Entry
	v        Versioner
	updateCh chan struct{}
}

// New creates new service.
func New(db *reform.DB, v Versioner) *Service {
	return &Service{
		db:       db,
		l:        logrus.WithField("component", "version-cache"),
		v:        v,
		updateCh: make(chan struct{}, 1),
	}
}

type service struct {
	ServiceID     string
	CheckAfter    time.Duration
	WaitNextCheck bool
	PMMAgentID    string
}

// findServiceForUpdate checks if there is any service that needs software versions update in the cache and
// shifts the next check time for this service.
func (s *Service) findServiceForUpdate() (*service, error) {
	results := &service{CheckAfter: minCheckInterval}

	if err := s.db.InTransaction(func(tx *reform.TX) error {
		filter := models.FindServicesSoftwareVersionsFilter{Limit: pointer.ToInt(1)}
		servicesVersions, err := models.FindServicesSoftwareVersions(
			tx.Querier,
			filter,
			models.SoftwareVersionsOrderByNextCheckAt)
		if err != nil {
			return err
		}
		if len(servicesVersions) == 0 {
			// there are no entries in the cache, so perform next check later
			results.CheckAfter = serviceCheckInterval

			return nil
		}
		if servicesVersions[0].NextCheckAt.After(time.Now()) {
			// wait until next service check time
			results.CheckAfter = time.Until(servicesVersions[0].NextCheckAt) + minCheckInterval

			return nil
		}

		results.WaitNextCheck = true
		results.ServiceID = servicesVersions[0].ServiceID

		service, err := models.FindServiceByID(tx.Querier, servicesVersions[0].ServiceID)
		if err != nil {
			return err
		}
		if service.ServiceType != models.MySQLServiceType {
			return nil
		}

		pmmAgents, err := models.FindPMMAgentsForService(tx.Querier, servicesVersions[0].ServiceID)
		if err != nil {
			return err
		}
		if len(pmmAgents) == 0 {
			return errors.Errorf("pmmAgent not found for service")
		}

		results.PMMAgentID = pmmAgents[0].AgentID

		// shift the next check time for this service, so, in case of versions fetch error,
		// it will not loop in trying, but will continue with other services.
		nextCheckAt := time.Now().UTC().Add(serviceCheckShortInterval)
		if _, err := models.UpdateServiceSoftwareVersions(
			tx.Querier, servicesVersions[0].ServiceID,
			models.UpdateServiceSoftwareVersionsParams{NextCheckAt: &nextCheckAt}); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return results, nil
}

func softwareName(s agents.Software) (models.SoftwareName, error) {
	var softwareName models.SoftwareName
	switch software := s.(type) {
	case *agents.Mysqld:
		softwareName = models.MysqldSoftwareName
	case *agents.Xtrabackup:
		softwareName = models.XtrabackupSoftwareName
	case *agents.Xbcloud:
		softwareName = models.XbcloudSoftwareName
	case *agents.Qpress:
		softwareName = models.QpressSoftwareName
	default:
		return "", errors.Errorf("invalid software type %T", software)
	}

	return softwareName, nil
}

// updateVersionsForNextService tries to find a service that needs update and performs update if such service is found.
// Returns desired time interval to wait before next call of the updateVersionsForNextService.
func (s *Service) updateVersionsForNextService() (time.Duration, error) {
	foundService, err := s.findServiceForUpdate()
	if err != nil {
		return minCheckInterval, err
	}

	if !foundService.WaitNextCheck {
		return foundService.CheckAfter, nil
	}

	softwares := []agents.Software{&agents.Mysqld{}, &agents.Xtrabackup{}, &agents.Xbcloud{}, &agents.Qpress{}}
	versions, err := s.v.GetVersions(foundService.PMMAgentID, softwares)
	if err != nil {
		return minCheckInterval, err
	}
	if len(versions) != len(softwares) {
		return minCheckInterval, errors.Errorf("slices length mismatch: versions len %d != softwares len %d",
			len(versions), len(softwares))
	}

	svs := make([]models.SoftwareVersion, 0, len(softwares))
	for i, software := range softwares {
		name, err := softwareName(software)
		if err != nil {
			return minCheckInterval, err
		}

		if versions[i].Error != "" {
			s.l.Warnf("failed to get version of %q software: %s", name, versions[i].Error)
			continue
		}
		if versions[i].Version == "" {
			continue
		}

		svs = append(svs, models.SoftwareVersion{
			Name:    name,
			Version: versions[i].Version,
		})
	}

	nextCheckAt := time.Now().UTC().Add(serviceCheckInterval)
	if _, err := models.UpdateServiceSoftwareVersions(s.db.Querier, foundService.ServiceID,
		models.UpdateServiceSoftwareVersionsParams{
			NextCheckAt:      &nextCheckAt,
			SoftwareVersions: svs,
		},
	); err != nil {
		return minCheckInterval, err
	}

	return minCheckInterval, err
}

// RequestSoftwareVersionsUpdate triggers update service software versions.
func (s *Service) RequestSoftwareVersionsUpdate() {
	select {
	case s.updateCh <- struct{}{}:
	default:
	}
}

// Run runs software version cache service.
func (s *Service) Run(ctx context.Context) {
	time.Sleep(startupDelay) // sleep a while, so the server establishes the connections with agents.

	s.l.Info("Starting...")
	defer s.l.Info("Done.")

	defer close(s.updateCh)

	var checkAfter time.Duration
	for {
		select {
		case <-time.After(checkAfter):
		case <-s.updateCh:
		case <-ctx.Done():
			return
		}

		s.l.Infof("Updating versions...")

		ca, err := s.updateVersionsForNextService()
		if err != nil {
			s.l.Warn(err)
		}

		checkAfter = ca
		s.l.Infof("Done. Next check in %s.", checkAfter)
	}
}
