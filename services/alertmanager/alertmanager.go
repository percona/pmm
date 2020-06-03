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

// Package alertmanager contains business logic of working with Alertmanager.
package alertmanager

import (
	"context"
	"io/ioutil"
	"os"
	"strings"
	"syscall"
	"time"

	httptransport "github.com/go-openapi/runtime/client"
	"github.com/percona/pmm/api/alertmanager/amclient"
	"github.com/percona/pmm/api/alertmanager/amclient/alert"
	"github.com/percona/pmm/api/alertmanager/amclient/general"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"
)

const (
	resendInterval      = 30 * time.Second
	alertmanagerDataDir = "/srv/alertmanager/data"
	prometheusDir       = "/srv/prometheus"
	path                = "/srv/alertmanager/alertmanager.base.yml"
)

// Service is responsible for interactions with Alertmanager.
type Service struct {
	db *reform.DB
	r  *Registry
	l  *logrus.Entry
}

// New creates new service.
func New(db *reform.DB, alertsRegistry *Registry) *Service {
	return &Service{
		db: db,
		r:  alertsRegistry,
		l:  logrus.WithField("component", "alertmanager"),
	}
}

// Run runs Alertmanager configuration update loop until ctx is canceled.
func (svc *Service) Run(ctx context.Context) {
	svc.l.Info("Starting...")
	defer svc.l.Info("Done.")

	svc.createDataDir()
	svc.generateBaseConfig()

	t := time.NewTicker(resendInterval)
	defer t.Stop()

	for {
		svc.sendAlerts(ctx)

		select {
		case <-ctx.Done():
			return
		case <-t.C:
			// nothing, continue for loop
		}
	}
}

// createDataDir creates Alertmanager directories if not exists in the persistent volume.
func (svc *Service) createDataDir() {
	// try to create Alertmanager data dir if not exists.
	if err := os.MkdirAll(alertmanagerDataDir, 0775); err != nil {
		svc.l.Errorf("Cannot create datadir for Alertmanager %v.", err)
		return
	}

	alertmanagerDataDirStat, err := os.Stat(alertmanagerDataDir)
	if err != nil {
		svc.l.Errorf("Cannot get stat of %q: %v.", alertmanagerDataDir, err)
		return
	}

	// Check and fix permissions.
	if alertmanagerDataDirStat.Mode()&os.ModePerm != os.FileMode(0775) {
		if err := os.Chmod(alertmanagerDataDir, 0775); err != nil {
			svc.l.Errorf("Cannot chmod datadir for Alertmanager %v.", err)
		}
	}

	alertmanagerDataDirSysStat := alertmanagerDataDirStat.Sys().(*syscall.Stat_t)
	aUID, aGID := int(alertmanagerDataDirSysStat.Uid), int(alertmanagerDataDirSysStat.Gid)

	prometheusDirStat, err := os.Stat(prometheusDir)
	if err != nil {
		svc.l.Errorf("Cannot get stat of %q: %v.", prometheusDir, err)
		return
	}

	prometheusDirSysStat := prometheusDirStat.Sys().(*syscall.Stat_t)
	pUID, pGID := int(prometheusDirSysStat.Uid), int(prometheusDirSysStat.Gid)
	// Chown user and group as Prometheus has if they are not same.
	if aUID != pUID || aGID != pGID {
		if err := os.Chown(alertmanagerDataDir, pUID, pGID); err != nil {
			svc.l.Errorf("Cannot chown datadir for Alertmanager %v.", err)
		}
	}
}

// generateBaseConfig generates /srv/alertmanager/alertmanager.base.yml if it is not present.
//
// TODO That's a temporary measure until we start generating /etc/alertmanager.yml
// using /srv/alertmanager/alertmanager.base.yml as a base. See supervisord config.
func (svc *Service) generateBaseConfig() {
	_, err := os.Stat(path)
	svc.l.Debugf("%s status: %v", path, err)

	if os.IsNotExist(err) {
		defaultBase := strings.TrimSpace(`
---
# You can edit this file; changes will be preserved.

route:
  receiver: empty
  routes: []

receivers:
  - name: empty
`) + "\n"
		err = ioutil.WriteFile(path, []byte(defaultBase), 0644) //nolint:gosec
		svc.l.Infof("%s created: %v.", path, err)
	}
}

// sendAlerts sends alerts collected in the Registry.
func (svc *Service) sendAlerts(ctx context.Context) {
	alerts := svc.r.collect()
	if len(alerts) == 0 {
		return
	}

	svc.l.Infof("Sending %d alerts...", len(alerts))
	_, err := amclient.Default.Alert.PostAlerts(&alert.PostAlertsParams{
		Alerts:  alerts,
		Context: ctx,
	})
	if err != nil {
		svc.l.Error(err)
	}
}

// IsReady verifies that Alertmanager works.
func (svc *Service) IsReady(ctx context.Context) error {
	_, err := amclient.Default.General.GetStatus(&general.GetStatusParams{
		Context: ctx,
	})
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// configure default client; we use it mainly because we can't remove it from generated code
//nolint:gochecknoinits
func init() {
	amclient.Default.SetTransport(httptransport.New("127.0.0.1:9093", "/alertmanager/api/v2", []string{"http"}))
}
