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

// Package supervisord provides facilities for working with Supervisord.
package supervisord

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/envvars"
	"github.com/percona/pmm/utils/pdeathsig"
)

const (
	defaultClickhouseDatabase           = "pmm"
	defaultClickhouseAddr               = "127.0.0.1:9000"
	defaultClickhouseUser               = "default"
	defaultClickhousePassword           = "clickhouse"
	defaultVMSearchMaxQueryLen          = "1MB"
	defaultVMSearchLatencyOffset        = "5s"
	defaultVMSearchMaxUniqueTimeseries  = "100000000"
	defaultVMSearchMaxSamplesPerQuery   = "1500000000"
	defaultVMSearchMaxQueueDuration     = "30s"
	defaultVMSearchMaxQueryDuration     = "90s"
	defaultVMSearchLogSlowQueryDuration = "30s"
	defaultVMPromscrapeStreamParse      = "true"
)

// Service is responsible for interactions with Supervisord via supervisorctl.
type Service struct {
	configDir         string
	supervisorctlPath string
	l                 *logrus.Entry

	eventsM    sync.Mutex
	subs       map[chan *event]sub
	lastEvents map[string]eventType

	supervisordConfigsM sync.Mutex

	vmParams *models.VictoriaMetricsParams
	pgParams *models.PGParams
	haParams *models.HAParams
}

type sub struct {
	program    string
	eventTypes []eventType
}

// values from supervisord configuration.
const (
	pmmConfig = "/etc/supervisord.d/pmm.ini"
)

// New creates new service.
func New(configDir string, params *models.Params) *Service {
	path, _ := exec.LookPath("supervisorctl")
	return &Service{
		configDir:         configDir,
		supervisorctlPath: path,
		l:                 logrus.WithField("component", "supervisord"),
		subs:              make(map[chan *event]sub),
		lastEvents:        make(map[string]eventType),
		vmParams:          params.VMParams,
		pgParams:          params.PGParams,
		haParams:          params.HAParams,
	}
}

// Run reads supervisord's log (maintail) and sends events to subscribers.
func (s *Service) Run(ctx context.Context) {
	if s.supervisorctlPath == "" {
		s.l.Errorf("supervisorctl not found, updates are disabled.")
		return
	}

	var lastEvent *event
	for ctx.Err() == nil {
		cmd := exec.CommandContext(ctx, s.supervisorctlPath, "maintail", "-f") //nolint:gosec
		cmdLine := strings.Join(cmd.Args, " ")
		pdeathsig.Set(cmd, unix.SIGKILL)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			s.l.Errorf("%s: StdoutPipe failed: %s", cmdLine, err)
			time.Sleep(time.Second)
			continue
		}

		if err := cmd.Start(); err != nil {
			s.l.Errorf("%s: Start failed: %s", cmdLine, err)
			time.Sleep(time.Second)
			continue
		}

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			e := parseEvent(scanner.Text())
			if e == nil {
				continue
			}
			s.l.Debugf("Got event: %+v", e)

			// skip old events (and events with exactly the same time as old events) if maintail was restarted
			if lastEvent != nil && !lastEvent.Time.Before(e.Time) {
				continue
			}
			lastEvent = e

			s.eventsM.Lock()

			s.lastEvents[e.Program] = e.Type

			var toDelete []chan *event
			for ch, sub := range s.subs {
				if e.Program == sub.program {
					var found bool
					for _, t := range sub.eventTypes {
						if e.Type == t {
							found = true
							break
						}
					}
					if found {
						ch <- e
						close(ch)
						toDelete = append(toDelete, ch)
					}
				}
			}

			for _, ch := range toDelete {
				delete(s.subs, ch)
			}

			s.eventsM.Unlock()
		}

		if err := scanner.Err(); err != nil {
			s.l.Errorf("Scanner: %s", err)
		}

		if err := cmd.Wait(); err != nil {
			s.l.Errorf("%s: wait failed: %s", cmdLine, err)
		}
	}
}

func (s *Service) subscribe(program string, eventTypes ...eventType) chan *event {
	ch := make(chan *event, 1)
	s.eventsM.Lock()
	s.subs[ch] = sub{
		program:    program,
		eventTypes: eventTypes,
	}
	s.eventsM.Unlock()
	return ch
}

func (s *Service) supervisorctl(args ...string) ([]byte, error) {
	if s.supervisorctlPath == "" {
		return nil, errors.New("supervisorctl not found")
	}

	cmd := exec.Command(s.supervisorctlPath, args...) //nolint:gosec
	cmdLine := strings.Join(cmd.Args, " ")
	s.l.Debugf("Running %q...", cmdLine)
	pdeathsig.Set(cmd, unix.SIGKILL)
	b, err := cmd.Output()
	return b, errors.Wrapf(err, "%s failed", cmdLine)
}

// parseStatus parses `supervisorctl status <name>` output, returns true if <name> is running,
// false if definitely not, and nil if status can't be determined.
func parseStatus(status string) *bool {
	if f := strings.Fields(status); len(f) > 1 {
		switch status := f[1]; status {
		case "FATAL", "STOPPED": // will not be restarted
			return pointer.ToBool(false)
		case "STARTING", "RUNNING", "BACKOFF", "STOPPING":
			return pointer.ToBool(true)
		case "EXITED":
			// it might be restarted - we need to inspect last event
		default:
			// something else - we need to inspect last event
		}
	}
	return nil
}

// UpdateRunning returns true if given supervisord program is running or being restarted,
// false if it is not running / failed.
func (s *Service) programRunning(program string) bool {
	// First check with status command is case we missed that event during maintail or pmm-managed restart.
	// See http://supervisord.org/subprocess.html#process-states
	b, err := s.supervisorctl("status", program)
	if err != nil {
		s.l.Warn(err)
	}
	s.l.Debugf("Status result for %q: %q", program, string(b))
	if status := parseStatus(string(b)); status != nil {
		s.l.Debugf("Status result for %q parsed: %v", program, *status)
		return *status
	}

	s.eventsM.Lock()
	lastEvent := s.lastEvents[program]
	s.eventsM.Unlock()

	s.l.Debugf("Status result for %q not parsed, inspecting last event %q.", program, lastEvent)
	switch lastEvent {
	case stopping, starting, running:
		return true
	case exitedUnexpected: // will be restarted
		return true
	case exitedExpected, fatal: // will not be restarted
		return false
	case stopped: // we don't know
		fallthrough
	default:
		s.l.Warnf("Unhandled status result for %q (last event %q), assuming it is not running.", program, lastEvent)
		return false
	}
}

// reload asks supervisord to reload configuration.
func (s *Service) reload(name string) error {
	if _, err := s.supervisorctl("reread"); err != nil {
		s.l.Warn(err)
	}
	_, err := s.supervisorctl("update", name)
	return err
}

// marshalConfig marshals supervisord program configuration.
func (s *Service) marshalConfig(tmpl *template.Template, settings *models.Settings, ssoDetails *models.PerconaSSODetails) ([]byte, error) {
	clickhouseDatabase := envvars.GetEnv("PMM_CLICKHOUSE_DATABASE", defaultClickhouseDatabase)
	clickhouseAddr := envvars.GetEnv("PMM_CLICKHOUSE_ADDR", defaultClickhouseAddr)
	clickhouseAddrPair := strings.SplitN(clickhouseAddr, ":", 2)
	clickhouseUser := envvars.GetEnv("PMM_CLICKHOUSE_USER", defaultClickhouseUser)
	clickhousePassword := envvars.GetEnv("PMM_CLICKHOUSE_PASSWORD", defaultClickhousePassword)
	vmSearchDisableCache := envvars.GetEnv("VM_search_disableCache", strconv.FormatBool(!settings.IsVictoriaMetricsCacheEnabled()))
	vmSearchMaxQueryLen := envvars.GetEnv("VM_search_maxQueryLen", defaultVMSearchMaxQueryLen)
	vmSearchLatencyOffset := envvars.GetEnv("VM_search_latencyOffset", defaultVMSearchLatencyOffset)
	vmSearchMaxUniqueTimeseries := envvars.GetEnv("VM_search_maxUniqueTimeseries", defaultVMSearchMaxUniqueTimeseries)
	vmSearchMaxSamplesPerQuery := envvars.GetEnv("VM_search_maxSamplesPerQuery", defaultVMSearchMaxSamplesPerQuery)
	vmSearchMaxQueueDuration := envvars.GetEnv("VM_search_maxQueueDuration", defaultVMSearchMaxQueueDuration)
	vmSearchMaxQueryDuration := envvars.GetEnv("VM_search_maxQueryDuration", defaultVMSearchMaxQueryDuration)
	vmSearchLogSlowQueryDuration := envvars.GetEnv("VM_search_logSlowQueryDuration", defaultVMSearchLogSlowQueryDuration)
	vmPromscrapeStreamParse := envvars.GetEnv("VM_promscrape_streamParse", defaultVMPromscrapeStreamParse)

	templateParams := map[string]interface{}{
		"DataRetentionHours":           int(settings.DataRetention.Hours()),
		"DataRetentionDays":            int(settings.DataRetention.Hours() / 24),
		"VMAlertFlags":                 s.vmParams.VMAlertFlags,
		"VMSearchDisableCache":         vmSearchDisableCache,
		"VMSearchMaxQueryLen":          vmSearchMaxQueryLen,
		"VMSearchLatencyOffset":        vmSearchLatencyOffset,
		"VMSearchMaxUniqueTimeseries":  vmSearchMaxUniqueTimeseries,
		"VMSearchMaxSamplesPerQuery":   vmSearchMaxSamplesPerQuery,
		"VMSearchMaxQueueDuration":     vmSearchMaxQueueDuration,
		"VMSearchMaxQueryDuration":     vmSearchMaxQueryDuration,
		"VMSearchLogSlowQueryDuration": vmSearchLogSlowQueryDuration,
		"VMPromscrapeStreamParse":      vmPromscrapeStreamParse,
		"VMURL":                        s.vmParams.URL(),
		"ExternalVM":                   s.vmParams.ExternalVM(),
		"InterfaceToBind":              envvars.GetInterfaceToBind(),
		"ClickhouseAddr":               clickhouseAddr,
		"ClickhouseDatabase":           clickhouseDatabase,
		"ClickhouseHost":               clickhouseAddrPair[0],
		"ClickhousePort":               clickhouseAddrPair[1],
		"ClickhouseUser":               clickhouseUser,
		"ClickhousePassword":           clickhousePassword,
	}

	s.addPostgresParams(templateParams)
	s.addClusterParams(templateParams)
	s.addAIChatParams(templateParams)

	templateParams["PMMServerHost"] = ""
	if settings.PMMPublicAddress != "" {
		pmmPublicAddress := settings.PMMPublicAddress
		if !strings.HasPrefix(pmmPublicAddress, "https://") && !strings.HasPrefix(pmmPublicAddress, "http://") {
			pmmPublicAddress = "https://" + pmmPublicAddress
		}
		publicURL, err := url.Parse(pmmPublicAddress)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse PMM public address.") //nolint:revive
		}
		templateParams["PMMServerHost"] = publicURL.Host
	}
	if ssoDetails != nil {
		u, err := url.Parse(ssoDetails.IssuerURL)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse host of IssuerURL")
		}
		templateParams["PerconaSSODetails"] = ssoDetails
		templateParams["PMMServerAddress"] = settings.PMMPublicAddress
		templateParams["PMMServerID"] = settings.PMMServerID
		templateParams["IssuerDomain"] = u.Host
	} else {
		templateParams["PerconaSSODetails"] = nil
	}

	if settings.IsNomadEnabled() {
		templateParams["NomadEnabled"] = "true"
	} else {
		templateParams["NomadEnabled"] = "false"
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateParams); err != nil {
		return nil, errors.Wrapf(err, "failed to render template %q", tmpl.Name())
	}
	b := append([]byte("; Managed by pmm-managed. DO NOT EDIT.\n"), buf.Bytes()...)
	return b, nil
}

// addPostgresParams adds pmm-server postgres database params to template config for grafana.
func (s *Service) addPostgresParams(templateParams map[string]interface{}) {
	if s.pgParams == nil {
		return
	}
	templateParams["PostgresAddr"] = s.pgParams.Addr
	templateParams["PostgresDBName"] = s.pgParams.DBName
	templateParams["PostgresDBUsername"] = s.pgParams.DBUsername
	templateParams["PostgresDBPassword"] = s.pgParams.DBPassword
	templateParams["PostgresSSLMode"] = s.pgParams.SSLMode
	templateParams["PostgresSSLCAPath"] = s.pgParams.SSLCAPath
	templateParams["PostgresSSLKeyPath"] = s.pgParams.SSLKeyPath
	templateParams["PostgresSSLCertPath"] = s.pgParams.SSLCertPath
}

func (s *Service) addClusterParams(templateParams map[string]interface{}) {
	templateParams["HAEnabled"] = s.haParams.Enabled
	if s.haParams.Enabled {
		templateParams["GrafanaGossipPort"] = s.haParams.GrafanaGossipPort
		templateParams["HAAdvertiseAddress"] = s.haParams.AdvertiseAddress
		nodes := make([]string, len(s.haParams.Nodes))
		for i, node := range s.haParams.Nodes {
			nodes[i] = fmt.Sprintf("%s:%d", node, s.haParams.GrafanaGossipPort)
		}
		templateParams["HANodes"] = strings.Join(nodes, ",")
	}
	// - GF_UNIFIED_ALERTING_HA_ADVERTISE_ADDRESS=172.20.0.5:9095
	// - GF_UNIFIED_ALERTING_HA_PEERS=pmm-server-active:9095,pmm-server-passive:9095
}

func (s *Service) addAIChatParams(templateParams map[string]interface{}) {
	// AI Chat configuration parameters with environment variable support
	templateParams["AIChatEnabled"] = envvars.GetEnv("PMM_AICHAT_ENABLED", "true")
	templateParams["AIChatPort"] = "3001"

	// Get provider and set appropriate default model
	provider := envvars.GetEnv("AICHAT_LLM_PROVIDER", "openai")
	templateParams["AIChatLLMProvider"] = provider

	// Set default model based on provider
	var defaultModel string
	switch provider {
	case "openai":
		defaultModel = "gpt-4o-mini"
	case "gemini":
		defaultModel = "gemini-2.5-flash"
	case "claude":
		defaultModel = "claude-3-5-haiku-20241022"
	case "ollama":
		defaultModel = "llama3.1:8b"
	case "mock":
		defaultModel = "mock-model"
	default:
		defaultModel = "gpt-4o-mini" // fallback to OpenAI
	}
	templateParams["AIChatLLMModel"] = envvars.GetEnv("AICHAT_LLM_MODEL", defaultModel)

	templateParams["AIChatAPIKey"] = envvars.GetEnv("AICHAT_API_KEY", "")
	templateParams["AIChatMCPServersFile"] = envvars.GetEnv("AICHAT_MCP_SERVERS_FILE", "/etc/aichat-backend/mcp-servers.json")
	templateParams["AIChatLogLevel"] = envvars.GetEnv("AICHAT_LOG_LEVEL", "info")
	templateParams["AIChatCORSOrigins"] = envvars.GetEnv("AICHAT_CORS_ORIGINS", "http://localhost:8080,http://localhost:8443")
}

// saveConfigAndReload saves given supervisord program configuration to file and reloads it.
// If configuration can't be reloaded for some reason, old file is restored, and configuration is reloaded again.
// Returns true if configuration was changed.
func (s *Service) saveConfigAndReload(name string, cfg []byte) (bool, error) {
	// read existing content
	path := filepath.Join(s.configDir, name+".ini")
	oldCfg, err := os.ReadFile(path) //nolint:gosec
	if errors.Is(err, fs.ErrNotExist) {
		err = nil
	}
	if err != nil {
		return false, errors.WithStack(err)
	}

	// compare with new config
	if reflect.DeepEqual(cfg, oldCfg) {
		s.l.Infof("%s configuration not changed, doing nothing.", name)
		return false, nil
	}

	// restore old content and reload in case of error
	restore := true
	defer func() {
		if restore {
			if err = os.WriteFile(path, oldCfg, 0o644); err != nil { //nolint:gosec
				s.l.Errorf("Failed to restore: %s.", err)
			}
			if err = s.reload(name); err != nil {
				s.l.Errorf("Failed to restore/reload: %s.", err)
			}
		}
	}()

	// write and reload
	if err = os.WriteFile(path, cfg, 0o644); err != nil { //nolint:gosec
		return false, errors.WithStack(err)
	}
	if err = s.reload(name); err != nil {
		return false, err
	}
	s.l.Infof("%s configuration reloaded.", name)
	restore = false
	return true, nil
}

// UpdateConfiguration updates VictoriaMetrics, Grafana and qan-api2 configurations, restarting them if needed.
func (s *Service) UpdateConfiguration(settings *models.Settings, ssoDetails *models.PerconaSSODetails) error {
	if s.supervisorctlPath == "" {
		s.l.Errorf("supervisorctl not found, configuration updates are disabled.")
		return nil
	}

	s.supervisordConfigsM.Lock()
	defer s.supervisordConfigsM.Unlock()

	var err error

	err = s.vmParams.UpdateParams()
	if err != nil {
		return err
	}

	for _, tmpl := range templates.Templates() {
		if tmpl.Name() == "" || (tmpl.Name() == "victoriametrics" && s.vmParams.ExternalVM()) {
			continue
		}

		// Skip aichat-backend if disabled
		if tmpl.Name() == "aichat-backend" && envvars.GetEnv("PMM_AICHAT_ENABLED", "true") != "true" {
			continue
		}

		b, e := s.marshalConfig(tmpl, settings, ssoDetails)
		if e != nil {
			s.l.Errorf("Failed to marshal config: %s.", e)
			err = e
			continue
		}
		if _, e = s.saveConfigAndReload(tmpl.Name(), b); e != nil {
			s.l.Errorf("Failed to save/reload: %s.", e)
			err = e
			continue
		}
	}
	return err
}

// RestartSupervisedService restarts given service.
func (s *Service) RestartSupervisedService(serviceName string) error {
	_, err := s.supervisorctl("restart", serviceName)
	return err
}

// StartSupervisedService starts given service.
func (s *Service) StartSupervisedService(serviceName string) error {
	_, err := s.supervisorctl("start", serviceName)
	return err
}

// StopSupervisedService stops given service.
func (s *Service) StopSupervisedService(serviceName string) error {
	_, err := s.supervisorctl("stop", serviceName)
	return err
}

//nolint:lll
var templates = template.Must(template.New("").Option("missingkey=error").Parse(`

{{define "victoriametrics"}}
{{- if not .ExternalVM }}
[program:victoriametrics]
priority = 7
command =
	/usr/sbin/victoriametrics
		--promscrape.config=/etc/victoriametrics-promscrape.yml
		--retentionPeriod={{ .DataRetentionDays }}d
		--storageDataPath=/srv/victoriametrics/data
		--httpListenAddr={{ .InterfaceToBind }}:9090
		--search.disableCache={{ .VMSearchDisableCache }}
		--search.maxQueryLen={{ .VMSearchMaxQueryLen }}
		--search.latencyOffset={{ .VMSearchLatencyOffset }}
		--search.maxUniqueTimeseries={{ .VMSearchMaxUniqueTimeseries }}
		--search.maxSamplesPerQuery={{ .VMSearchMaxSamplesPerQuery }}
		--search.maxQueueDuration={{ .VMSearchMaxQueueDuration }}
		--search.logSlowQueryDuration={{ .VMSearchLogSlowQueryDuration }}
		--search.maxQueryDuration={{ .VMSearchMaxQueryDuration }}
		--promscrape.streamParse={{ .VMPromscrapeStreamParse }}
		--http.pathPrefix=/prometheus
		--envflag.enable
		--envflag.prefix=VM_
user = pmm
autorestart = true
autostart = true
startretries = 10
startsecs = 1
stopsignal = INT
stopwaitsecs = 300
stdout_logfile = /srv/logs/victoriametrics.log
stdout_logfile_maxbytes = 10MB
stdout_logfile_backups = 3
redirect_stderr = true
{{end -}}
{{end}}

{{define "vmalert"}}
[program:vmalert]
priority = 7
command =
	/usr/sbin/vmalert
		--external.url={{ .VMURL }}
		--datasource.url={{ .VMURL }}
		--remoteRead.url={{ .VMURL }}
		--remoteWrite.url={{ .VMURL }}
		--rule=/srv/prometheus/rules/*.yml
		--httpListenAddr={{ .InterfaceToBind }}:8880
{{- range $index, $param := .VMAlertFlags }}
		{{ $param }}
{{- end }}
user = pmm
autorestart = true
autostart = true
startretries = 10
startsecs = 1
stopsignal = INT
stopwaitsecs = 300
stdout_logfile = /srv/logs/vmalert.log
stdout_logfile_maxbytes = 10MB
stdout_logfile_backups = 3
redirect_stderr = true
{{end}}

{{define "vmproxy"}}
[program:vmproxy]
priority = 9
command =
    /usr/sbin/vmproxy
      --target-url={{ .VMURL }}
      --listen-port=8430
      --listen-address={{ .InterfaceToBind }}
      --header-name=X-Proxy-Filter
user = pmm
autorestart = true
autostart = true
startretries = 10
startsecs = 1
stopsignal = INT
stopwaitsecs = 300
stdout_logfile = /srv/logs/vmproxy.log
stdout_logfile_maxbytes = 10MB
stdout_logfile_backups = 3
redirect_stderr = true
{{end}}

{{define "qan-api2"}}
[program:qan-api2]
priority = 13
command =
	/usr/sbin/percona-qan-api2
		--data-retention={{ .DataRetentionDays }}
environment =
	PMM_CLICKHOUSE_ADDR="{{ .ClickhouseAddr }}",
	PMM_CLICKHOUSE_DATABASE="{{ .ClickhouseDatabase }}",
	PMM_CLICKHOUSE_USER="{{ .ClickhouseUser }}",
	PMM_CLICKHOUSE_PASSWORD="{{ .ClickhousePassword }}",


user = pmm
autorestart = true
autostart = true
startretries = 1000
startsecs = 1
stopsignal = TERM
stopwaitsecs = 10
stdout_logfile = /srv/logs/qan-api2.log
stdout_logfile_maxbytes = 10MB
stdout_logfile_backups = 3
redirect_stderr = true
{{end}}

{{define "grafana"}}
[program:grafana]
priority = 3
command =
    /usr/sbin/grafana server
        --homepath=/usr/share/grafana
        --config=/etc/grafana/grafana.ini
        {{- if .PMMServerHost}}
        cfg:default.server.domain="{{ .PMMServerHost }}"
        {{- end}}
        {{- if .PerconaSSODetails}}
        cfg:default.auth.generic_oauth.enabled=true
        cfg:default.auth.generic_oauth.name="Percona Account"
        cfg:default.auth.generic_oauth.client_id="{{ .PerconaSSODetails.GrafanaClientID }}"
        cfg:default.auth.generic_oauth.scopes="openid profile email offline_access percona"
        cfg:default.auth.generic_oauth.auth_url="{{ .PerconaSSODetails.IssuerURL }}/authorize"
        cfg:default.auth.generic_oauth.token_url="{{ .PerconaSSODetails.IssuerURL }}/token"
        cfg:default.auth.generic_oauth.api_url="{{ .PerconaSSODetails.IssuerURL }}/userinfo"
        cfg:default.auth.generic_oauth.role_attribute_path="(contains(portal_admin_orgs[*], '{{ .PerconaSSODetails.OrganizationID }}') || contains(pmm_demo_ids[*], '{{ .PMMServerID }}')) && 'Admin' || 'Viewer'"
        cfg:default.auth.generic_oauth.use_pkce="true"
        cfg:default.auth.oauth_allow_insecure_email_lookup="true"
        {{- end}}
environment =
    PMM_POSTGRES_ADDR="{{ .PostgresAddr }}",
    PMM_POSTGRES_DBNAME="{{ .PostgresDBName }}",
    PMM_POSTGRES_USERNAME="{{ .PostgresDBUsername }}",
    PMM_POSTGRES_DBPASSWORD="{{ .PostgresDBPassword }}",
    PMM_POSTGRES_SSL_MODE="{{ .PostgresSSLMode }}",
    PMM_POSTGRES_SSL_CA_PATH="{{ .PostgresSSLCAPath }}",
    PMM_POSTGRES_SSL_KEY_PATH="{{ .PostgresSSLKeyPath }}",
    PMM_POSTGRES_SSL_CERT_PATH="{{ .PostgresSSLCertPath }}",
    PMM_CLICKHOUSE_HOST="{{ .ClickhouseHost }}",
    PMM_CLICKHOUSE_PORT="{{ .ClickhousePort }}",
    PMM_CLICKHOUSE_USER="{{ .ClickhouseUser }}",
    PMM_CLICKHOUSE_PASSWORD="{{ .ClickhousePassword }}",
    {{- if .PerconaSSODetails}}
    GF_AUTH_SIGNOUT_REDIRECT_URL="https://{{ .IssuerDomain }}/login/signout?fromURI=https://{{ .PMMServerAddress }}/graph/login"
    {{- end}}
    {{- if .HAEnabled}}
    GF_UNIFIED_ALERTING_HA_LISTEN_ADDRESS="0.0.0.0:{{ .GrafanaGossipPort }}",
    GF_UNIFIED_ALERTING_HA_ADVERTISE_ADDRESS="{{ .HAAdvertiseAddress }}:{{ .GrafanaGossipPort }}",
    GF_UNIFIED_ALERTING_HA_PEERS="{{ .HANodes }}"
    {{- end}}
user = pmm
directory = /usr/share/grafana
autorestart = true
autostart = true
startretries = 10
startsecs = 1
stopsignal = TERM
stopwaitsecs = 300
stdout_logfile = /srv/logs/grafana.log
stdout_logfile_maxbytes = 50MB
stdout_logfile_backups = 2
redirect_stderr = true
{{end}}

{{define "nomad-server"}}
[program:nomad-server]
priority = 5
command = /usr/local/percona/pmm/tools/nomad agent -config /srv/nomad/nomad-server-{{ .PMMServerHost }}.hcl
user = pmm
autorestart = {{ .NomadEnabled }}
autostart = {{ .NomadEnabled }}
startretries = 10
startsecs = 1
stopsignal = INT
stopwaitsecs = 300
stdout_logfile = /srv/logs/nomad-server.log
stdout_logfile_maxbytes = 10MB
stdout_logfile_backups = 3
redirect_stderr = true
{{end}}

{{define "aichat-backend"}}
{{- if eq .AIChatEnabled "true" }}
[program:aichat-backend]
priority = 16
command = /usr/sbin/aichat-backend --env-only
user = pmm
autorestart = true
autostart = true
startretries = 1000
startsecs = 1
stopsignal = TERM
stopwaitsecs = 300
stdout_logfile = /srv/logs/aichat-backend.log
stdout_logfile_maxbytes = 50MB
stdout_logfile_backups = 2
redirect_stderr = true
environment =
    PATH="/home/pmm/.local/bin:%(ENV_PATH)s",
    AICHAT_PORT="{{ .AIChatPort }}",
    AICHAT_LLM_PROVIDER="{{ .AIChatLLMProvider }}",
    AICHAT_LLM_MODEL="{{ .AIChatLLMModel }}",
    {{- if .AIChatAPIKey }}
    AICHAT_API_KEY="{{ .AIChatAPIKey }}",
    {{- end }}
    AICHAT_MCP_SERVERS_FILE="{{ .AIChatMCPServersFile }}",
    AICHAT_LOG_LEVEL="{{ .AIChatLogLevel }}",
    AICHAT_CORS_ORIGINS="{{ .AIChatCORSOrigins }}",
    AICHAT_DATABASE_URL="postgres://ai_chat_user:ai_chat_secure_password@127.0.0.1:5432/ai_chat?sslmode=disable",
    GIN_MODE="release"
{{end -}}
{{end}}
`))
