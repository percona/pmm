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
	"time"

	"github.com/AlekSi/pointer"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/percona/pmm/api/alertmanager/amclient"
	"github.com/percona/pmm/api/alertmanager/amclient/alert"
	"github.com/percona/pmm/api/alertmanager/amclient/general"
	"github.com/percona/pmm/api/alertmanager/amclient/silence"
	"github.com/percona/pmm/api/alertmanager/ammodels"
	"github.com/percona/promconfig/alertmanager"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/yaml.v3"

	"github.com/percona/pmm-managed/utils/dir"
)

const (
	alertmanagerDir     = "/srv/alertmanager"
	alertmanagerDataDir = "/srv/alertmanager/data"
	dirPerm             = os.FileMode(0o775)

	alertmanagerConfigPath     = "/etc/alertmanager.yml"
	alertmanagerBaseConfigPath = "/srv/alertmanager/alertmanager.base.yml"
)

// Service is responsible for interactions with Alertmanager.
type Service struct {
	db *reform.DB
	l  *logrus.Entry
}

// New creates new service.
func New(db *reform.DB) *Service {
	return &Service{
		db: db,
		l:  logrus.WithField("component", "alertmanager"),
	}
}

// Run runs Alertmanager configuration update loop until ctx is canceled.
func (svc *Service) Run(ctx context.Context) {
	svc.l.Info("Starting...")
	defer svc.l.Info("Done.")

	err := dir.CreateDataDir(alertmanagerDir, "pmm", "pmm", dirPerm)
	if err != nil {
		svc.l.Error(err)
	}
	err = dir.CreateDataDir(alertmanagerDataDir, "pmm", "pmm", dirPerm)
	if err != nil {
		svc.l.Error(err)
	}

	svc.generateBaseConfig()
	svc.updateConfiguration(ctx)

	// we don't have "configuration update loop" yet, so do nothing
	// TODO implement loop similar to victoriametrics.Service.Run

	<-ctx.Done()
}

// generateBaseConfig generates /srv/alertmanager/alertmanager.base.yml if it is not present.
func (svc *Service) generateBaseConfig() {
	_, err := os.Stat(alertmanagerBaseConfigPath)
	svc.l.Debugf("%s status: %v", alertmanagerBaseConfigPath, err)

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
		err = ioutil.WriteFile(alertmanagerBaseConfigPath, []byte(defaultBase), 0o644) //nolint:gosec
		svc.l.Infof("%s created: %v.", alertmanagerBaseConfigPath, err)
	}
}

// updateConfiguration updates Alertmanager configuration.
func (svc *Service) updateConfiguration(ctx context.Context) {
	// TODO split into marshalConfig and configAndReload like in victoriametrics.Service

	// if /etc/alertmanager.yml already exists, read its contents.
	var content []byte
	_, err := os.Stat(alertmanagerConfigPath)
	if err == nil {
		svc.l.Infof("%s exists, checking content", alertmanagerConfigPath)
		content, err = ioutil.ReadFile(alertmanagerConfigPath)
		if err != nil {
			svc.l.Errorf("Failed to load alertmanager config %s: %s", alertmanagerConfigPath, err)
		}
	}

	// copy the base config if `/etc/alertmanager.yml` is not present or
	// is already present but does not have any config.
	if os.IsNotExist(err) || string(content) == "---\n" {
		var cfg alertmanager.Config
		buf, err := ioutil.ReadFile(alertmanagerBaseConfigPath)
		if err != nil {
			svc.l.Errorf("Failed to load alertmanager base config %s: %s", alertmanagerBaseConfigPath, err)
			return
		}
		if err := yaml.Unmarshal(buf, &cfg); err != nil {
			svc.l.Errorf("Failed to parse alertmanager base config %s: %s.", alertmanagerBaseConfigPath, err)
			return
		}

		// TODO add custom information to this config.
		b, err := yaml.Marshal(cfg)
		if err != nil {
			svc.l.Errorf("Failed to marshal alertmanager config %s: %s.", alertmanagerConfigPath, err)
			return
		}
		b = append([]byte("# Managed by pmm-managed. DO NOT EDIT.\n---\n"), b...)

		err = ioutil.WriteFile(alertmanagerConfigPath, b, 0o644)
		if err != nil {
			svc.l.Errorf("Failed to write alertmanager config %s: %s.", alertmanagerConfigPath, err)
			return
		}
	}
	svc.l.Infof("%s created", alertmanagerConfigPath)
}

// SendAlerts sends given alerts. It is the caller's responsibility
// to call this method every now and then.
func (svc *Service) SendAlerts(ctx context.Context, alerts ammodels.PostableAlerts) {
	if len(alerts) == 0 {
		svc.l.Debug("0 alerts to send, exiting.")
		return
	}

	svc.l.Debugf("Sending %d alerts...", len(alerts))
	_, err := amclient.Default.Alert.PostAlerts(&alert.PostAlertsParams{
		Alerts:  alerts,
		Context: ctx,
	})
	if err != nil {
		svc.l.Error(err)
	}
}

// GetAlerts returns alerts available in alertmanager.
func (svc *Service) GetAlerts(ctx context.Context) ([]*ammodels.GettableAlert, error) {
	resp, err := amclient.Default.Alert.GetAlerts(&alert.GetAlertsParams{
		Context: ctx,
	})
	if err != nil {
		return nil, err
	}

	return resp.Payload, nil
}

// FindAlertByID searches alert by ID in alertmanager.
func (svc *Service) FindAlertByID(ctx context.Context, id string) (*ammodels.GettableAlert, error) {
	alerts, err := svc.GetAlerts(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get alerts form alertmanager")
	}

	for _, a := range alerts {
		if *a.Fingerprint == id {
			return a, nil
		}
	}

	return nil, status.Errorf(codes.NotFound, "Alert with id %s not found", id)
}

// Silence mutes alert with specified id.
func (svc *Service) Silence(ctx context.Context, id string) error {
	a, err := svc.FindAlertByID(ctx, id)
	if err != nil {
		return err
	}

	if len(a.Status.SilencedBy) != 0 {
		// already silenced
		return nil
	}

	matchers := make([]*ammodels.Matcher, 0, len(a.Labels))
	for label, value := range a.Labels {
		matchers = append(matchers,
			&ammodels.Matcher{
				IsRegex: pointer.ToBool(false),
				Name:    pointer.ToString(label),
				Value:   pointer.ToString(value),
			})
	}

	starts := strfmt.DateTime(time.Now())
	ends := strfmt.DateTime(time.Now().Add(100 * 365 * 24 * time.Hour)) // Mute for 100 years
	_, err = amclient.Default.Silence.PostSilences(&silence.PostSilencesParams{
		Silence: &ammodels.PostableSilence{
			Silence: ammodels.Silence{
				Comment:   pointer.ToString(""),
				CreatedBy: pointer.ToString("PMM"),
				StartsAt:  &starts,
				EndsAt:    &ends,
				Matchers:  matchers,
			},
		},
		Context: ctx,
	})

	return errors.Wrapf(err, "failed to silence alert with id: %s", id)
}

// Unsilence unmutes alert with specified id.
func (svc *Service) Unsilence(ctx context.Context, id string) error {
	a, err := svc.FindAlertByID(ctx, id)
	if err != nil {
		return err
	}

	for _, silenceID := range a.Status.SilencedBy {
		_, err = amclient.Default.Silence.DeleteSilence(&silence.DeleteSilenceParams{
			SilenceID: strfmt.UUID(silenceID),
			Context:   ctx,
		})

		if err != nil {
			return errors.Wrapf(err, "failed to delete silence with id %s for alert %s", silenceID, id)
		}
	}

	return nil
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
