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

// Package alertmanager contains business logic of working with Alertmanager.
package alertmanager

import (
	"context"
	"crypto/sha256"
	_ "embed" // for email templates
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/AlekSi/pointer"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/percona/promconfig"
	"github.com/percona/promconfig/alertmanager"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"gopkg.in/reform.v1"
	"gopkg.in/yaml.v3"

	"github.com/percona/pmm/api/alertmanager/amclient"
	"github.com/percona/pmm/api/alertmanager/amclient/alert"
	"github.com/percona/pmm/api/alertmanager/amclient/silence"
	"github.com/percona/pmm/api/alertmanager/ammodels"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
	"github.com/percona/pmm/managed/utils/dir"
	"github.com/percona/pmm/utils/pdeathsig"
)

const (
	updateBatchDelay           = time.Second
	configurationUpdateTimeout = 3 * time.Second

	alertmanagerDir     = "/srv/alertmanager"
	alertmanagerCertDir = "/srv/alertmanager/cert"
	alertmanagerDataDir = "/srv/alertmanager/data"
	dirPerm             = os.FileMode(0o775)

	alertmanagerConfigPath     = "/etc/alertmanager.yml"
	alertmanagerBaseConfigPath = "/srv/alertmanager/alertmanager.base.yml"

	receiverNameSeparator = " + "

	// CheckFilter represents AlertManager filter for Checks/Advisor results.
	CheckFilter = "stt_check=1"
	// IAFilter represents AlertManager filter for Integrated Alerts.
	IAFilter = "ia=1"
)

var notificationLabels = []string{
	"node_name", "node_id", "service_name", "service_id", "service_type", "rule_id",
	"alertgroup", "template_name", "severity", "agent_id", "agent_type", "job",
}

//go:embed email_template.html
var emailTemplate string

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
		client:   &http.Client{}, // TODO instrument with utils/irt; see vmalert package https://jira.percona.com/browse/PMM-7229
		l:        logrus.WithField("component", "alertmanager"),
		reloadCh: make(chan struct{}, 1),
	}
}

// GenerateBaseConfigs generates alertmanager.base.yml if it is absent,
// and then writes basic alertmanager.yml if it is absent or empty.
// It is needed because Alertmanager was added to PMM
// with invalid configuration file (it will fail with "no route provided in config" error).
func (svc *Service) GenerateBaseConfigs() {
	for _, dirPath := range []string{alertmanagerDir, alertmanagerDataDir, alertmanagerCertDir} {
		if err := dir.CreateDataDir(dirPath, "pmm", "pmm", dirPerm); err != nil {
			svc.l.Error(err)
		}
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
		err = os.WriteFile(alertmanagerBaseConfigPath, []byte(defaultBase), 0o644) //nolint:gosec
		if err != nil {
			svc.l.Errorf("Failed to write %s: %s", alertmanagerBaseConfigPath, err)
		}
	}

	// Don't call updateConfiguration() there as Alertmanager is likely to be in the crash loop at the moment.
	// Instead, write alertmanager.yml directly. main.go will request configuration update.
	stat, err := os.Stat(alertmanagerConfigPath)
	if err != nil || int(stat.Size()) <= len("---\n") { // https://github.com/percona/pmm-server/blob/main/alertmanager.yml
		svc.l.Infof("Creating %s", alertmanagerConfigPath)
		err = os.WriteFile(alertmanagerConfigPath, []byte(defaultBase), 0o644) //nolint:gosec
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
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, nil)
	if err != nil {
		return errors.WithStack(err)
	}
	resp, err := svc.client.Do(req)
	if err != nil {
		return errors.WithStack(err)
	}
	defer resp.Body.Close() //nolint:gosec,errcheck,nolintlint

	b, err := io.ReadAll(resp.Body)
	svc.l.Debugf("Alertmanager reload: %s", b)
	if err != nil {
		return errors.WithStack(err)
	}

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("expected 200, got %d", resp.StatusCode)
	}
	return nil
}

// loadBaseConfig returns parsed base configuration file, or empty configuration on error.
func (svc *Service) loadBaseConfig() *alertmanager.Config {
	buf, err := os.ReadFile(alertmanagerBaseConfigPath)
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
	f, err := os.CreateTemp("", "pmm-managed-config-alertmanager-")
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
	oldCfg, err := os.ReadFile(alertmanagerConfigPath)
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
			if err = os.WriteFile(alertmanagerConfigPath, oldCfg, fi.Mode()); err != nil {
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
	if err = os.WriteFile(alertmanagerConfigPath, b, fi.Mode()); err != nil {
		return errors.WithStack(err)
	}
	if err = svc.reload(ctx); err != nil {
		return err
	}
	svc.l.Infof("Configuration reloaded.")
	restore = false

	return nil
}

// convertTLSConfig converts model TLSConfig to promconfig TLSConfig.
// Resulting promconfig field
//   - CAFile is set to corresponding model field if CAFileContent is not specified, `sha256(id).ca` otherwise.
//   - CertFile is set to corresponding model field if CertFileContent is not specified, `sha256(id).crt` otherwise.
//   - KeyFile is set to corresponding model field if KeyFileContent is not specified, `sha256(id).key` otherwise.
func convertTLSConfig(id string, tls *models.TLSConfig) promconfig.TLSConfig {
	hashedIDBytes := sha256.Sum256([]byte(id))
	hashedID := hex.EncodeToString(hashedIDBytes[:])

	caFile := tls.CAFile
	if tls.CAFileContent != "" {
		caFile = path.Join(alertmanagerCertDir, fmt.Sprintf("%s.ca", hashedID))
	}
	certFile := tls.CertFile
	if tls.CertFileContent != "" {
		certFile = path.Join(alertmanagerCertDir, fmt.Sprintf("%s.crt", hashedID))
	}
	keyFile := tls.KeyFile
	if tls.KeyFileContent != "" {
		keyFile = path.Join(alertmanagerCertDir, fmt.Sprintf("%s.key", hashedID))
	}
	return promconfig.TLSConfig{
		CAFile:             caFile,
		CertFile:           certFile,
		KeyFile:            keyFile,
		ServerName:         tls.ServerName,
		InsecureSkipVerify: tls.InsecureSkipVerify,
	}
}

func cleanupTLSConfigFiles() error {
	des, err := os.ReadDir(alertmanagerCertDir)
	if err != nil {
		return errors.Wrap(err, "failed to list alertmanager certificates directory")
	}
	for _, de := range des {
		if de.IsDir() {
			continue
		}

		if err := os.Remove(path.Join(alertmanagerCertDir, de.Name())); err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

func tlsConfig(c *models.Channel) *models.TLSConfig {
	if c.WebHookConfig != nil &&
		c.WebHookConfig.HTTPConfig != nil &&
		c.WebHookConfig.HTTPConfig.TLSConfig != nil {
		return c.WebHookConfig.HTTPConfig.TLSConfig
	}

	return nil
}

// recreateTLSConfigFiles cleanups old tls config files and creates new ones for each channel using the content
// from CAFileContent, CertFileContent, KeyFileContent if it is set.
func recreateTLSConfigFiles(chanMap map[string]*models.Channel) error {
	fi, err := os.Stat(alertmanagerCertDir)
	if err != nil {
		return errors.WithStack(err)
	}

	if err := cleanupTLSConfigFiles(); err != nil {
		return errors.WithStack(err)
	}

	for _, c := range chanMap {
		tlsConfig := tlsConfig(c)
		if tlsConfig == nil {
			continue
		}

		convertedTLSConfig := convertTLSConfig(c.ID, tlsConfig)
		fileContentMap := map[string]string{
			convertedTLSConfig.CAFile:   tlsConfig.CAFileContent,
			convertedTLSConfig.CertFile: tlsConfig.CertFileContent,
			convertedTLSConfig.KeyFile:  tlsConfig.KeyFileContent,
		}
		for filePath, content := range fileContentMap {
			if filePath == "" || content == "" {
				continue
			}

			if err := os.WriteFile(filePath, []byte(content), fi.Mode()); err != nil {
				return errors.WithStack(err)
			}
		}
	}

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
		return errors.Errorf("failed to fetch items from database: %v", e)
	}

	chanMap := make(map[string]*models.Channel, len(channels))
	for _, ch := range channels {
		chanMap[ch.ID] = ch
	}
	if err := recreateTLSConfigFiles(chanMap); err != nil {
		return err
	}

	if cfg.Global == nil {
		cfg.Global = &alertmanager.GlobalConfig{}
	}

	findReceiverIdx := func(name string) int {
		for i, r := range cfg.Receivers {
			if r.Name == name {
				return i
			}
		}
		return -1
	}

	// make sure that "empty" receiver is there
	if findReceiverIdx("empty") == -1 {
		cfg.Receivers = append(cfg.Receivers, &alertmanager.Receiver{
			Name: "empty",
		})
	}

	disabledReceiver := &alertmanager.Receiver{
		Name: "disabled",
	}
	// Override if there is any user defined receiver `disabled`, needs to be empty
	if disabledReceiverIdx := findReceiverIdx("disabled"); disabledReceiverIdx != -1 {
		cfg.Receivers[disabledReceiverIdx] = disabledReceiver
	} else {
		cfg.Receivers = append(cfg.Receivers, disabledReceiver)
	}

	// set default route if absent
	if cfg.Route == nil {
		cfg.Route = &alertmanager.Route{
			Receiver: "empty",
		}
	}

	if settings.Alerting.EmailAlertingSettings != nil {
		svc.l.Warn("Setting global email config, any user defined changes to the base config might be overwritten.")

		cfg.Global.SMTPFrom = settings.Alerting.EmailAlertingSettings.From
		cfg.Global.SMTPHello = settings.Alerting.EmailAlertingSettings.Hello
		cfg.Global.SMTPSmarthost = settings.Alerting.EmailAlertingSettings.Smarthost
		cfg.Global.SMTPAuthIdentity = settings.Alerting.EmailAlertingSettings.Identity
		cfg.Global.SMTPAuthUsername = settings.Alerting.EmailAlertingSettings.Username
		cfg.Global.SMTPAuthPassword = settings.Alerting.EmailAlertingSettings.Password
		cfg.Global.SMTPAuthSecret = settings.Alerting.EmailAlertingSettings.Secret
		cfg.Global.SMTPRequireTLS = settings.Alerting.EmailAlertingSettings.RequireTLS
	}

	if settings.Alerting.SlackAlertingSettings != nil {
		svc.l.Warn("Setting global Slack config, any user defined changes to the base config might be overwritten.")

		cfg.Global.SlackAPIURL = settings.Alerting.SlackAlertingSettings.URL
	}

	recvSet := make(map[string]models.ChannelIDs) // stores unique combinations of channel IDs
	for _, r := range rules {
		// skip rules with 0 notification channels
		if len(r.ChannelIDs) == 0 {
			continue
		}

		route := &alertmanager.Route{
			Match: map[string]string{
				"rule_id": r.ID,
			},
			MatchRE: make(map[string]string),
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
		enabledChannels := make(models.ChannelIDs, 0, len(r.ChannelIDs))
		for _, chID := range r.ChannelIDs {
			if channel, ok := chanMap[chID]; ok {
				if !channel.Disabled {
					enabledChannels = append(enabledChannels, chID)
				}
			}
		}
		// make sure same slice with different order are not considered unique.
		sort.Strings(enabledChannels)
		recv := strings.Join(enabledChannels, receiverNameSeparator)
		if len(enabledChannels) == 0 {
			recv = "disabled"
		} else {
			recvSet[recv] = enabledChannels
		}
		route.Receiver = recv

		cfg.Route.Routes = append(cfg.Route.Routes, route)
	}

	receivers, err := svc.generateReceivers(chanMap, recvSet)
	if err != nil {
		return err
	}

	cfg.Receivers = append(cfg.Receivers, receivers...)
	return nil
}

func formatSlackText(labels ...string) string {
	const listEntryFormat = "{{ if .Labels.%[1]s }}     • *%[1]s:* `{{ .Labels.%[1]s }}`\n{{ end }}"

	text := "{{ range .Alerts -}}\n" +
		"*Alert:* {{ if .Labels.severity }}`{{ .Labels.severity | toUpper }}`{{ end }} {{ .Annotations.summary }}\n" +
		"*Description:* {{ .Annotations.description }}\n" +
		"*Details:*\n"
	for _, l := range labels {
		text += fmt.Sprintf(listEntryFormat, l)
	}

	text += "\n\n{{ end }}"

	return text
}

func formatPagerDutyFiringDetails(labels ...string) string {
	const listEntryFormat = "{{ if .Labels.%[1]s }}  - %[1]s: {{ .Labels.%[1]s }}\n{{ end }}"

	text := "{{ range .Alerts -}}\n" +
		"Alert: {{ if .Labels.severity }}[{{ .Labels.severity | toUpper }}]{{ end }} {{ .Annotations.summary }}\n" +
		"Description: {{ .Annotations.description }}\n" +
		"Details:\n"
	for _, l := range labels {
		text += fmt.Sprintf(listEntryFormat, l)
	}

	text += "\n\n{{ end }}"

	return text
}

// generateReceivers takes the channel map and a unique set of rule combinations and generates a slice of receivers.
func (svc *Service) generateReceivers(chanMap map[string]*models.Channel, recvSet map[string]models.ChannelIDs) ([]*alertmanager.Receiver, error) {
	receivers := make([]*alertmanager.Receiver, 0, len(recvSet))

	for name, channelIDs := range recvSet {
		recv := &alertmanager.Receiver{
			Name: name,
		}

		for _, ch := range channelIDs {
			channel, ok := chanMap[ch]
			if !ok {
				svc.l.Warnf("Missing channel %s, skip it.", ch)
				continue
			}
			switch channel.Type {
			case models.Email:
				for _, to := range channel.EmailConfig.To {
					recv.EmailConfigs = append(recv.EmailConfigs, &alertmanager.EmailConfig{
						NotifierConfig: alertmanager.NotifierConfig{
							SendResolved: channel.EmailConfig.SendResolved,
						},
						To:   to,
						HTML: emailTemplate,
						Headers: map[string]string{
							"Subject": `[{{ .Status | toUpper }}{{ if eq .Status "firing" }}:{{ .Alerts.Firing | len }}{{ end }}]`,
						},
					})
				}

			case models.PagerDuty:
				pdConfig := &alertmanager.PagerdutyConfig{
					NotifierConfig: alertmanager.NotifierConfig{
						SendResolved: channel.PagerDutyConfig.SendResolved,
					},
					Description: `[{{ .Status | toUpper }}{{ if eq .Status "firing" }}:{{ .Alerts.Firing | len }}{{ end }}]` +
						"{{ range .Alerts -}}{{ if .Labels.severity }}[{{ .Labels.severity | toUpper }}]{{ end }} {{ .Annotations.summary }}{{ end }}",
					Details: map[string]string{
						"firing": formatPagerDutyFiringDetails(notificationLabels...),
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
					Title:   `[{{ .Status | toUpper }}{{ if eq .Status "firing" }}:{{ .Alerts.Firing | len }}{{ end }}]`,
					Text:    formatSlackText(notificationLabels...),
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
						webhookConfig.HTTPConfig.TLSConfig = convertTLSConfig(channel.ID,
							channel.WebHookConfig.HTTPConfig.TLSConfig)
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
func (svc *Service) GetAlerts(ctx context.Context, fp *services.FilterParams) ([]*ammodels.GettableAlert, error) {
	alertParams := alert.NewGetAlertsParams()
	alertParams.Context = ctx

	if fp != nil {
		if fp.IsCheck {
			alertParams.Filter = append(alertParams.Filter, CheckFilter)
		}
		if fp.IsIA {
			alertParams.Filter = append(alertParams.Filter, IAFilter)
		}
		if fp.ServiceID != "" {
			alertParams.Filter = append(alertParams.Filter, fmt.Sprintf("service_id=\"%s\"", fp.ServiceID))
		}
		if fp.AlertID != "" {
			alertParams.Filter = append(alertParams.Filter, fmt.Sprintf("alert_id=\"%s\"", fp.AlertID))
		}
	}

	svc.l.Debugf("%+v", alertParams)
	resp, err := amclient.Default.Alert.GetAlerts(alertParams)
	if err != nil {
		return nil, err
	}

	return resp.Payload, nil
}

// FindAlertsByID searches alerts by IDs in alertmanager.
func (svc *Service) FindAlertsByID(ctx context.Context, params *services.FilterParams, ids []string) ([]*ammodels.GettableAlert, error) {
	alerts, err := svc.GetAlerts(ctx, params)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get alerts from alertmanager")
	}

	l := len(ids)
	m := make(map[string]struct{}, l)
	for _, id := range ids {
		m[id] = struct{}{}
	}

	res := make([]*ammodels.GettableAlert, 0, l)
	for _, a := range alerts {
		if _, ok := m[*a.Fingerprint]; ok {
			res = append(res, a)
		}
	}

	return res, nil
}

// SilenceAlerts silences a group of provided alerts.
func (svc *Service) SilenceAlerts(ctx context.Context, alerts []*ammodels.GettableAlert) error {
	var err error
	for _, a := range alerts {
		if len(a.Status.SilencedBy) != 0 {
			// Skip already silenced alerts
			continue
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
		if err != nil {
			return errors.Wrapf(err, "failed to silence alert with id: %s", *a.Fingerprint)
		}
	}

	return nil
}

// UnsilenceAlerts unmutes the provided alerts.
func (svc *Service) UnsilenceAlerts(ctx context.Context, alerts []*ammodels.GettableAlert) error {
	var err error
	for _, a := range alerts {
		for _, silenceID := range a.Status.SilencedBy {
			_, err = amclient.Default.Silence.DeleteSilence(&silence.DeleteSilenceParams{
				SilenceID: strfmt.UUID(silenceID),
				Context:   ctx,
			})
			if err != nil {
				return errors.Wrapf(err, "failed to delete silence with id %s for alert %s", silenceID, *a.Fingerprint)
			}
		}
	}

	return nil
}

// IsReady verifies that Alertmanager works.
func (svc *Service) IsReady(ctx context.Context) error {
	u := "http://127.0.0.1:9093/alertmanager/-/ready"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return errors.WithStack(err)
	}
	resp, err := svc.client.Do(req)
	if err != nil {
		return errors.WithStack(err)
	}
	defer resp.Body.Close() //nolint:gosec,errcheck,nolintlint

	b, err := io.ReadAll(resp.Body)
	svc.l.Debugf("Alertmanager ready: %s", b)
	if err != nil {
		return errors.WithStack(err)
	}
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("expected 200, got %d", resp.StatusCode)
	}

	return nil
}

// configure default client; we use it mainly because we can't remove it from generated code
//
//nolint:gochecknoinits
func init() {
	amclient.Default.SetTransport(httptransport.New("127.0.0.1:9093", "/alertmanager/api/v2", []string{"http"}))
}
