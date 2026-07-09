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
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

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
func (s *Service) Run(ctx context.Context) { //nolint:gocognit
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

		err = cmd.Start()
		if err != nil {
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
					if slices.Contains(sub.eventTypes, e.Type) {
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

		err = scanner.Err()
		if err != nil {
			s.l.Errorf("Scanner: %s", err)
		}

		err = cmd.Wait()
		if err != nil {
			s.l.Errorf("%s: wait failed: %s", cmdLine, err)
		}
	}
}

// UpdateConfiguration updates VictoriaMetrics, Grafana and qan-api2 configurations, restarting them if needed.
func (s *Service) UpdateConfiguration(settings *models.Settings) error {
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
		if tmpl.Name() == "" {
			continue
		}

		if tmpl.Name() == "victoriametrics" && s.vmParams.ExternalVM() {
			e := os.Remove(filepath.Join(s.configDir, tmpl.Name()+".ini"))
			if e != nil && !errors.Is(e, fs.ErrNotExist) {
				s.l.Warnf("Failed to remove %s config for external VM: %s.", tmpl.Name(), e)
			}
			continue
		}

		if tmpl.Name() == "nomad-server" && !settings.IsNomadEnabled() {
			e := os.Remove(filepath.Join(s.configDir, tmpl.Name()+".ini"))
			if e != nil && !errors.Is(e, fs.ErrNotExist) {
				s.l.Warnf("Failed to remove %s config when disabled: %s.", tmpl.Name(), e)
			}
			continue
		}

		b, e := s.marshalConfig(tmpl, settings)
		if e != nil {
			s.l.Errorf("Failed to marshal config: %s.", e)
			err = e
			continue
		}
		_, e = s.saveConfigAndReload(tmpl.Name(), b)
		if e != nil {
			s.l.Errorf("Failed to save/reload: %s.", e)
			err = e
			continue
		}
	}
	return err
}

// StartSupervisedService starts given service.
func (s *Service) StartSupervisedService(serviceName string) error {
	return s.supervisorctl("start", serviceName)
}

// StopSupervisedService stops given service.
func (s *Service) StopSupervisedService(serviceName string) error {
	return s.supervisorctl("stop", serviceName)
}

var templates = template.Must(template.New("").Option("missingkey=error").Parse(`

{{define "victoriametrics"}}
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
autorestart = true
autostart = {{ not .ExternalVM }}
startretries = 10
startsecs = 1
stopsignal = INT
stopwaitsecs = 300
stdout_logfile = /srv/logs/victoriametrics.log
stdout_logfile_maxbytes = 10MB
stdout_logfile_backups = 3
redirect_stderr = true
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
    {{- if .HAEnabled}}
    GF_UNIFIED_ALERTING_HA_LISTEN_ADDRESS="0.0.0.0:{{ .GrafanaGossipPort }}",
    GF_UNIFIED_ALERTING_HA_ADVERTISE_ADDRESS="{{ .HAAdvertiseAddress }}:{{ .GrafanaGossipPort }}",
    GF_UNIFIED_ALERTING_HA_PEERS="{{ .HANodes }}"
    {{- end}}
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
autorestart = true
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
`))

func (s *Service) supervisorctl(args ...string) error {
	if s.supervisorctlPath == "" {
		return errors.New("supervisorctl not found")
	}

	cmd := exec.Command(s.supervisorctlPath, args...) //nolint:gosec,noctx
	cmdLine := strings.Join(cmd.Args, " ")
	s.l.Debugf("Running %q...", cmdLine)
	pdeathsig.Set(cmd, unix.SIGKILL)
	_, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("%s failed: %w", cmdLine, err)
	}
	return nil
}

// parseStatus parses `supervisorctl status <name>` output, returns true if <name> is running,
// false if definitely not, and nil if status can't be determined.
func parseStatus(status string) *bool {
	if f := strings.Fields(status); len(f) > 1 {
		switch status := f[1]; status {
		case "FATAL", "STOPPED": // will not be restarted
			return new(false)
		case "STARTING", "RUNNING", "BACKOFF", "STOPPING":
			return new(true)
		case "EXITED":
			// it might be restarted - we need to inspect last event
		default:
			// something else - we need to inspect last event
		}
	}
	return nil
}

// reload asks supervisord to reload configuration.
func (s *Service) reload(name string) error {
	err := s.supervisorctl("reread")
	if err != nil {
		s.l.Warn(err)
	}

	path := filepath.Join(s.configDir, name+".ini")
	_, err = os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		s.l.Warnf("Config file %s does not exist, skipping update", path)
		return nil
	}

	return s.supervisorctl("update", name)
}

// marshalConfig marshals supervisord program configuration.
func (s *Service) marshalConfig(tmpl *template.Template, settings *models.Settings) ([]byte, error) {
	clickhouseDatabase := envvars.GetEnv("PMM_CLICKHOUSE_DATABASE", defaultClickhouseDatabase)
	clickhouseAddr := envvars.GetEnv("PMM_CLICKHOUSE_ADDR", defaultClickhouseAddr)
	clickhouseAddrPair := strings.SplitN(clickhouseAddr, ":", 2) //nolint:mnd
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

	templateParams := map[string]any{
		"DataRetentionHours":           int(settings.DataRetention.Hours()),
		"DataRetentionDays":            int(settings.DataRetention.Hours() / 24), //nolint:mnd
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
		"NomadEnabled":                 settings.IsNomadEnabled(),
		"InterfaceToBind":              envvars.GetInterfaceToBind(),
		"ClickhouseAddr":               clickhouseAddr,
		"ClickhouseDatabase":           clickhouseDatabase,
		"ClickhouseHost":               clickhouseAddrPair[0],
		"ClickhousePort":               clickhouseAddrPair[1],
		"ClickhouseUser":               clickhouseUser,
		"ClickhousePassword":           clickhousePassword,
		"PMMServerHost":                "",
	}

	s.addPostgresParams(templateParams)
	s.addClusterParams(templateParams)

	if settings.PMMPublicAddress != "" {
		pmmPublicAddress := settings.PMMPublicAddress
		if !strings.HasPrefix(pmmPublicAddress, "https://") && !strings.HasPrefix(pmmPublicAddress, "http://") {
			pmmPublicAddress = "https://" + pmmPublicAddress
		}
		publicURL, err := url.Parse(pmmPublicAddress)
		if err != nil {
			return nil, fmt.Errorf("failed to parse PMM public address: %w", err)
		}
		templateParams["PMMServerHost"] = publicURL.Host
	}

	var buf bytes.Buffer
	err := tmpl.Execute(&buf, templateParams)
	if err != nil {
		return nil, fmt.Errorf("failed to render template %q: %w", tmpl.Name(), err)
	}
	b := append([]byte("; Managed by pmm-managed. DO NOT EDIT.\n"), buf.Bytes()...)
	return b, nil
}

// addPostgresParams adds pmm-server postgres database params to template config for grafana.
func (s *Service) addPostgresParams(templateParams map[string]any) {
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

func (s *Service) addClusterParams(templateParams map[string]any) {
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
		return false, err
	}

	// compare with new config
	if reflect.DeepEqual(cfg, oldCfg) {
		s.l.Infof("%s configuration not changed, doing nothing.", name)
		return false, nil
	}

	// restore old content and reload in case of error
	restore := oldCfg != nil
	defer func() {
		if restore {
			err = os.WriteFile(path, oldCfg, 0o664) //nolint:gosec,mnd
			if err != nil {
				s.l.Errorf("Failed to restore: %v.", err)
			}
			err = s.reload(name)
			if err != nil {
				s.l.Errorf("Failed to restore/reload: %s.", err)
			}
		}
	}()

	// write and reload
	err = os.WriteFile(path, cfg, 0o664) //nolint:gosec,mnd
	if err != nil {
		return false, err
	}

	err = s.reload(name)
	if err != nil {
		return false, err
	}
	s.l.Infof("%s configuration reloaded.", name)
	restore = false
	return true, nil
}
