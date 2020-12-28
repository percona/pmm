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
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/AlekSi/pointer"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/percona/pmm/api/alertmanager/amclient"
	"github.com/percona/pmm/api/alertmanager/amclient/alert"
	"github.com/percona/pmm/api/alertmanager/amclient/silence"
	"github.com/percona/pmm/api/alertmanager/ammodels"
	"github.com/percona/pmm/utils/pdeathsig"
	"github.com/percona/promconfig"
	"github.com/percona/promconfig/alertmanager"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/yaml.v3"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/dir"
)

const (
	updateBatchDelay           = time.Second
	configurationUpdateTimeout = 3 * time.Second

	alertmanagerDir     = "/srv/alertmanager"
	alertmanagerDataDir = "/srv/alertmanager/data"
	dirPerm             = os.FileMode(0o775)

	alertmanagerConfigPath     = "/etc/alertmanager.yml"
	alertmanagerBaseConfigPath = "/srv/alertmanager/alertmanager.base.yml"

	receiverNameSeparator = " + "
)

// Service is responsible for interactions with Alertmanager.
type Service struct {
	db     *reform.DB
	client *http.Client

	l        *logrus.Entry
	reloadCh chan struct{}
}

// New creates new service.
func New(db *reform.DB) *Service {
	return &Service{
		db:       db,
		client:   new(http.Client), // TODO instrument with utils/irt; see vmalert package https://jira.percona.com/browse/PMM-7229
		l:        logrus.WithField("component", "alertmanager"),
		reloadCh: make(chan struct{}, 1),
	}
}

// GenerateBaseConfigs generates alertmanager.base.yml if it is absent,
// and then writes basic alertmanager.yml if it is absent or empty.
// It is needed because Alertmanager was added to PMM
// with invalid configuration file (it will fail with "no route provided in config" error).
func (svc *Service) GenerateBaseConfigs() {
	if err := dir.CreateDataDir(alertmanagerDir, "pmm", "pmm", dirPerm); err != nil {
		svc.l.Error(err)
	}
	if err := dir.CreateDataDir(alertmanagerDataDir, "pmm", "pmm", dirPerm); err != nil {
		svc.l.Error(err)
	}

	defaultBase := strings.TrimSpace(`
---
# You can edit this file; changes will be preserved.

route:
    receiver: empty
    routes: []

receivers:
    - name: empty
	`) + "\n"

	_, err := os.Stat(alertmanagerBaseConfigPath)
	svc.l.Debugf("%s status: %v", alertmanagerBaseConfigPath, err)
	if os.IsNotExist(err) {
		svc.l.Infof("Creating %s", alertmanagerBaseConfigPath)
		err = ioutil.WriteFile(alertmanagerBaseConfigPath, []byte(defaultBase), 0o644) //nolint:gosec
		if err != nil {
			svc.l.Errorf("Failed to write %s: %s", alertmanagerBaseConfigPath, err)
		}
	}

	// Don't call updateConfiguration() there as Alertmanager is likely to be in the crash loop at the moment.
	// Instead, write alertmanager.yml directly. main.go will request configuration update.
	stat, err := os.Stat(alertmanagerConfigPath)
	if err != nil || int(stat.Size()) <= len("---\n") { // https://github.com/percona/pmm-server/blob/PMM-2.0/alertmanager.yml
		svc.l.Infof("Creating %s", alertmanagerConfigPath)
		err = ioutil.WriteFile(alertmanagerConfigPath, []byte(defaultBase), 0o644) //nolint:gosec
		if err != nil {
			svc.l.Errorf("Failed to write %s: %s", alertmanagerConfigPath, err)
		}
	}
}

// Run runs Alertmanager configuration update loop until ctx is canceled.
func (svc *Service) Run(ctx context.Context) {
	// If you change this and related methods,
	// please do similar changes in victoriametrics and vmalert packages.

	svc.l.Info("Starting...")
	defer svc.l.Info("Done.")

	// reloadCh, configuration update loop, and RequestConfigurationUpdate method ensure that configuration
	// is reloaded when requested, but several requests are batched together to avoid too often reloads.
	// That allows the caller to just call RequestConfigurationUpdate when it seems fit.
	if cap(svc.reloadCh) != 1 {
		panic("reloadCh should have capacity 1")
	}

	for {
		select {
		case <-ctx.Done():
			return

		case <-svc.reloadCh:
			// batch several update requests together by delaying the first one
			sleepCtx, sleepCancel := context.WithTimeout(ctx, updateBatchDelay)
			<-sleepCtx.Done()
			sleepCancel()

			if ctx.Err() != nil {
				return
			}

			nCtx, cancel := context.WithTimeout(ctx, configurationUpdateTimeout)
			if err := svc.updateConfiguration(nCtx); err != nil {
				svc.l.Errorf("Failed to update configuration, will retry: %+v.", err)
				svc.RequestConfigurationUpdate()
			}
			cancel()
		}
	}
}

// RequestConfigurationUpdate requests Alertmanager configuration update.
func (svc *Service) RequestConfigurationUpdate() {
	select {
	case svc.reloadCh <- struct{}{}:
	default:
	}
}

// updateConfiguration updates Alertmanager configuration.
func (svc *Service) updateConfiguration(ctx context.Context) error {
	start := time.Now()
	defer func() {
		if dur := time.Since(start); dur > time.Second {
			svc.l.Warnf("updateConfiguration took %s.", dur)
		}
	}()

	base := svc.loadBaseConfig()
	b, err := svc.marshalConfig(base)
	if err != nil {
		return err
	}

	return svc.configAndReload(ctx, b)
}

// reload asks Alertmanager to reload configuration.
func (svc *Service) reload(ctx context.Context) error {
	u := "http://127.0.0.1:9093/alertmanager/-/reload"
	req, err := http.NewRequestWithContext(ctx, "POST", u, nil)
	if err != nil {
		return errors.WithStack(err)
	}
	resp, err := svc.client.Do(req)
	if err != nil {
		return errors.WithStack(err)
	}
	defer resp.Body.Close() //nolint:errcheck

	b, err := ioutil.ReadAll(resp.Body)
	svc.l.Debugf("Alertmanager reload: %s", b)
	if err != nil {
		return errors.WithStack(err)
	}

	if resp.StatusCode != 200 {
		return errors.Errorf("expected 200, got %d", resp.StatusCode)
	}
	return nil
}

// loadBaseConfig returns parsed base configuration file, or empty configuration on error.
func (svc *Service) loadBaseConfig() *alertmanager.Config {
	buf, err := ioutil.ReadFile(alertmanagerBaseConfigPath)
	if err != nil {
		if !os.IsNotExist(err) {
			svc.l.Errorf("Failed to load base Alertmanager config %s: %s", alertmanagerBaseConfigPath, err)
		}

		return &alertmanager.Config{}
	}

	var cfg alertmanager.Config
	if err := yaml.Unmarshal(buf, &cfg); err != nil {
		svc.l.Errorf("Failed to parse base Alertmanager config %s: %s.", alertmanagerBaseConfigPath, err)

		return &alertmanager.Config{}
	}

	return &cfg
}

// marshalConfig marshals Alertmanager configuration.
func (svc *Service) marshalConfig(base *alertmanager.Config) ([]byte, error) {
	cfg := base
	if err := svc.populateConfig(cfg); err != nil {
		return nil, err
	}

	b, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "can't marshal Alertmanager configuration file")
	}

	b = append([]byte("# Managed by pmm-managed. DO NOT EDIT.\n---\n"), b...)

	return b, nil
}

// validateConfig validates given configuration with `amtool check-config`.
func (svc *Service) validateConfig(ctx context.Context, cfg []byte) error {
	f, err := ioutil.TempFile("", "pmm-managed-config-alertmanager-")
	if err != nil {
		return errors.WithStack(err)
	}
	if _, err = f.Write(cfg); err != nil {
		return errors.WithStack(err)
	}
	defer func() {
		_ = f.Close()
		_ = os.Remove(f.Name())
	}()

	args := []string{"check-config", "--output=json", f.Name()}
	cmd := exec.CommandContext(ctx, "amtool", args...) //nolint:gosec
	pdeathsig.Set(cmd, unix.SIGKILL)

	b, err := cmd.CombinedOutput()
	if err != nil {
		svc.l.Errorf("%s", b)
		return errors.Wrap(err, string(b))
	}
	svc.l.Debugf("%s", b)

	return nil
}

// configAndReload saves given Alertmanager configuration to file and reloads Alertmanager.
// If configuration can't be reloaded for some reason, old file is restored, and configuration is reloaded again.
func (svc *Service) configAndReload(ctx context.Context, b []byte) error {
	oldCfg, err := ioutil.ReadFile(alertmanagerConfigPath)
	if err != nil {
		return errors.WithStack(err)
	}

	fi, err := os.Stat(alertmanagerConfigPath)
	if err != nil {
		return errors.WithStack(err)
	}

	// restore old content and reload in case of error
	var restore bool
	defer func() {
		if restore {
			if err = ioutil.WriteFile(alertmanagerConfigPath, oldCfg, fi.Mode()); err != nil {
				svc.l.Error(err)
			}
			if err = svc.reload(ctx); err != nil {
				svc.l.Error(err)
			}
		}
	}()

	if err = svc.validateConfig(ctx, b); err != nil {
		return err
	}

	restore = true
	if err = ioutil.WriteFile(alertmanagerConfigPath, b, fi.Mode()); err != nil {
		return errors.WithStack(err)
	}
	if err = svc.reload(ctx); err != nil {
		return err
	}
	svc.l.Infof("Configuration reloaded.")
	restore = false

	return nil
}

// populateConfig adds configuration from the database to cfg.
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

	// make sure that "empty" receiver is there
	var emptyFound bool
	for _, r := range cfg.Receivers {
		if r.Name == "empty" {
			emptyFound = true
			break
		}
	}
	if !emptyFound {
		cfg.Receivers = append(cfg.Receivers, &alertmanager.Receiver{
			Name: "empty",
		})
	}

	// set default route if absent
	if cfg.Route == nil {
		cfg.Route = &alertmanager.Route{
			Receiver: "empty",
		}
	}

	if settings.IntegratedAlerting.EmailAlertingSettings != nil {
		svc.l.Warn("Setting global email config, any user defined changes to the base config might be overwritten.")

		cfg.Global.SMTPFrom = settings.IntegratedAlerting.EmailAlertingSettings.From
		cfg.Global.SMTPHello = settings.IntegratedAlerting.EmailAlertingSettings.Hello
		cfg.Global.SMTPSmarthost = settings.IntegratedAlerting.EmailAlertingSettings.Smarthost
		cfg.Global.SMTPAuthIdentity = settings.IntegratedAlerting.EmailAlertingSettings.Identity
		cfg.Global.SMTPAuthUsername = settings.IntegratedAlerting.EmailAlertingSettings.Username
		cfg.Global.SMTPAuthPassword = settings.IntegratedAlerting.EmailAlertingSettings.Password
		cfg.Global.SMTPAuthSecret = settings.IntegratedAlerting.EmailAlertingSettings.Secret
	}

	if settings.IntegratedAlerting.SlackAlertingSettings != nil {
		svc.l.Warn("Setting global Slack config, any user defined changes to the base config might be overwritten.")

		cfg.Global.SlackAPIURL = settings.IntegratedAlerting.SlackAlertingSettings.URL
	}

	chanMap := make(map[string]*models.Channel, len(channels))
	for _, ch := range channels {
		chanMap[ch.ID] = ch
	}

	recvSet := make(map[string]models.ChannelIDs) // stores unique combinations of channel IDs
	for _, r := range rules {

		// FIXME We should handle disabled channels. https://jira.percona.com/browse/PMM-7231

		// FIXME we should use filters there, not custom labels

		route := &alertmanager.Route{
			Match: map[string]string{
				"rule_id": r.ID,
			},
			MatchRE: map[string]string{},
		}

		for _, f := range r.Filters {
			switch f.Type {
			case models.Equal:
				route.Match[f.Key] = f.Val
			case models.Regex:
				route.MatchRE[f.Key] = f.Val
			default:
				svc.l.Warnf("Unhandled filter: %+v", f)
			}
		}

		// make sure same slice with different order are not considered unique.
		sort.Strings(r.ChannelIDs)
		recv := strings.Join(r.ChannelIDs, receiverNameSeparator)
		recvSet[recv] = r.ChannelIDs
		route.Receiver = recv

		cfg.Route.Routes = append(cfg.Route.Routes, route)
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
				if channel.PagerDutyConfig.RoutingKey != "" {
					pdConfig.RoutingKey = channel.PagerDutyConfig.RoutingKey
				}
				if channel.PagerDutyConfig.ServiceKey != "" {
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
				return nil, errors.Errorf("invalid channel type: %q", channel.Type)
			}
		}

		receivers = append(receivers, recv)
	}

	sort.Slice(receivers, func(i, j int) bool { return receivers[i].Name < receivers[j].Name })
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
	u := "http://127.0.0.1:9093/alertmanager/-/ready"
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return errors.WithStack(err)
	}
	resp, err := svc.client.Do(req)
	if err != nil {
		return errors.WithStack(err)
	}
	defer resp.Body.Close() //nolint:errcheck

	b, err := ioutil.ReadAll(resp.Body)
	svc.l.Debugf("Alertmanager ready: %s", b)
	if err != nil {
		return errors.WithStack(err)
	}
	if resp.StatusCode != 200 {
		return errors.Errorf("expected 200, got %d", resp.StatusCode)
	}

	return nil
}

// configure default client; we use it mainly because we can't remove it from generated code
//nolint:gochecknoinits
func init() {
	amclient.Default.SetTransport(httptransport.New("127.0.0.1:9093", "/alertmanager/api/v2", []string{"http"}))
}
