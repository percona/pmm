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
	"net"
	"os"
	"sort"
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
	"github.com/percona/promconfig"
	"github.com/percona/promconfig/alertmanager"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/yaml.v3"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/dir"
)

const (
	alertmanagerDir     = "/srv/alertmanager"
	alertmanagerDataDir = "/srv/alertmanager/data"
	dirPerm             = os.FileMode(0o775)

	alertmanagerConfigPath     = "/etc/alertmanager.yml"
	alertmanagerBaseConfigPath = "/srv/alertmanager/alertmanager.base.yml"

	receiverNameSeparator = " + "
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

// RequestConfigurationUpdate requests Alertmanager configuration update.
func (svc *Service) RequestConfigurationUpdate() {
	// FIXME
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

		err = svc.populateConfig(&cfg)
		if err != nil {
			svc.l.Error(err)
			return
		}

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

func (svc *Service) populateConfig(cfg *alertmanager.Config) error {
	var settings *models.Settings
	var rules []*models.Rule
	var channels []*models.Channel
	e := svc.db.InTransaction(func(tx *reform.TX) error {
		var err error
		settings, err = models.GetSettings(tx.Querier)
		if err != nil {
			return err
		}

		rules, err = models.FindRules(tx.Querier)
		if err != nil {
			return err
		}

		channels, err = models.FindChannels(tx.Querier)
		if err != nil {
			return err
		}
		return nil
	})
	if e != nil {
		return errors.Errorf("Failed to fetch items from database: %s", e)
	}

	if cfg.Global == nil {
		cfg.Global = &alertmanager.GlobalConfig{}
	}

	svc.l.Warn("Setting global config, any user defined changes to the base config might be overwritten.")
	if settings.IntegratedAlerting.EmailAlertingSettings != nil {
		cfg.Global.SMTPFrom = settings.IntegratedAlerting.EmailAlertingSettings.From
		cfg.Global.SMTPHello = settings.IntegratedAlerting.EmailAlertingSettings.Hello
		cfg.Global.SMTPAuthIdentity = settings.IntegratedAlerting.EmailAlertingSettings.Identity
		cfg.Global.SMTPAuthUsername = settings.IntegratedAlerting.EmailAlertingSettings.Username
		cfg.Global.SMTPAuthPassword = settings.IntegratedAlerting.EmailAlertingSettings.Password
		cfg.Global.SMTPAuthSecret = settings.IntegratedAlerting.EmailAlertingSettings.Secret

		host, port, err := net.SplitHostPort(settings.IntegratedAlerting.EmailAlertingSettings.Smarthost)
		if err != nil {
			return errors.Errorf("Failed to set global email settings: %s", err)
		}
		cfg.Global.SMTPSmarthost.Host = host
		cfg.Global.SMTPSmarthost.Port = port
	}

	if settings.IntegratedAlerting.SlackAlertingSettings != nil {
		cfg.Global.SlackAPIURL = settings.IntegratedAlerting.SlackAlertingSettings.URL
	}

	chanMap := make(map[string]*models.Channel, len(channels))
	for _, ch := range channels {
		chanMap[ch.ID] = ch
	}

	recvSet := make(map[string]models.ChannelIDs) // stores unique combinations of channel IDs
	for _, r := range rules {
		match, _ := r.GetCustomLabels()
		match["rule_id"] = r.ID
		// make sure same slice with different order are not considered unique.
		sort.Strings(r.ChannelIDs)
		recv := strings.Join(r.ChannelIDs, receiverNameSeparator)
		recvSet[recv] = r.ChannelIDs
		cfg.Route.Routes = append(cfg.Route.Routes, &alertmanager.Route{
			Match:          match,
			Receiver:       recv,
			RepeatInterval: promconfig.Duration(r.For),
		})
	}

	receivers, err := generateReceivers(chanMap, recvSet)
	if err != nil {
		return err
	}

	cfg.Receivers = append(cfg.Receivers, receivers...)
	return nil
}

// generateReceivers takes the channel map and a unique set of rule combinations and generates a slice of receivers.
func generateReceivers(chanMap map[string]*models.Channel, recvSet map[string]models.ChannelIDs) ([]*alertmanager.Receiver, error) {
	receivers := make([]*alertmanager.Receiver, 0, len(recvSet))
	for name, channelIDs := range recvSet {
		recv := &alertmanager.Receiver{
			Name: name,
		}

		for _, ch := range channelIDs {
			channel := chanMap[ch]
			switch channel.Type {
			case models.Email:
				for _, to := range channel.EmailConfig.To {
					recv.EmailConfigs = append(recv.EmailConfigs, &alertmanager.EmailConfig{
						NotifierConfig: alertmanager.NotifierConfig{
							SendResolved: channel.EmailConfig.SendResolved,
						},
						To: to,
					})
				}
			case models.PagerDuty:
				pdConfig := &alertmanager.PagerdutyConfig{
					NotifierConfig: alertmanager.NotifierConfig{
						SendResolved: channel.PagerDutyConfig.SendResolved,
					},
				}
				if pdConfig.RoutingKey != "" {
					pdConfig.RoutingKey = channel.PagerDutyConfig.RoutingKey
				}
				if pdConfig.ServiceKey != "" {
					pdConfig.ServiceKey = channel.PagerDutyConfig.ServiceKey
				}
				recv.PagerdutyConfigs = append(recv.PagerdutyConfigs, pdConfig)
			case models.Slack:
				recv.SlackConfigs = append(recv.SlackConfigs, &alertmanager.SlackConfig{
					NotifierConfig: alertmanager.NotifierConfig{
						SendResolved: channel.SlackConfig.SendResolved,
					},
					Channel: channel.SlackConfig.Channel,
				})
			case models.WebHook:
				webhookConfig := &alertmanager.WebhookConfig{
					NotifierConfig: alertmanager.NotifierConfig{
						SendResolved: channel.WebHookConfig.SendResolved,
					},
					URL:       channel.WebHookConfig.URL,
					MaxAlerts: uint64(channel.WebHookConfig.MaxAlerts),
				}

				if channel.WebHookConfig.HTTPConfig != nil {
					webhookConfig.HTTPConfig = promconfig.HTTPClientConfig{
						BearerToken:     channel.WebHookConfig.HTTPConfig.BearerToken,
						BearerTokenFile: channel.WebHookConfig.HTTPConfig.BearerTokenFile,
						ProxyURL:        channel.WebHookConfig.HTTPConfig.ProxyURL,
					}
					if channel.WebHookConfig.HTTPConfig.BasicAuth != nil {
						webhookConfig.HTTPConfig.BasicAuth = &promconfig.BasicAuth{
							Username:     channel.WebHookConfig.HTTPConfig.BasicAuth.Username,
							Password:     channel.WebHookConfig.HTTPConfig.BasicAuth.Password,
							PasswordFile: channel.WebHookConfig.HTTPConfig.BasicAuth.PasswordFile,
						}
					}
					if channel.WebHookConfig.HTTPConfig.TLSConfig != nil {
						webhookConfig.HTTPConfig.TLSConfig = promconfig.TLSConfig{
							CAFile:             channel.WebHookConfig.HTTPConfig.TLSConfig.CaFile,
							CertFile:           channel.WebHookConfig.HTTPConfig.TLSConfig.CertFile,
							KeyFile:            channel.WebHookConfig.HTTPConfig.TLSConfig.KeyFile,
							ServerName:         channel.WebHookConfig.HTTPConfig.TLSConfig.ServerName,
							InsecureSkipVerify: channel.WebHookConfig.HTTPConfig.TLSConfig.InsecureSkipVerify,
						}
					}
				}
				recv.WebhookConfigs = append(recv.WebhookConfigs, webhookConfig)
			default:
				return nil, errors.Errorf("invalid channel type: %T", channel.Type)
			}
		}
		receivers = append(receivers, recv)
	}
	return receivers, nil
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
