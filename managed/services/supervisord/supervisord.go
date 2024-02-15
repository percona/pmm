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
	"io"
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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/envvars"
	"github.com/percona/pmm/utils/pdeathsig"
	"github.com/percona/pmm/version"
)

const (
	defaultClickhouseDatabase           = "pmm"
	defaultClickhouseAddr               = "127.0.0.1:9000"
	defaultClickhouseDataSourceAddr     = "127.0.0.1:8123"
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
	configDir          string
	supervisorctlPath  string
	gRPCMessageMaxSize uint32
	l                  *logrus.Entry
	pmmUpdateCheck     *PMMUpdateChecker

	eventsM    sync.Mutex
	subs       map[chan *event]sub
	lastEvents map[string]eventType

	pmmUpdatePerformLogM sync.Mutex
	supervisordConfigsM  sync.Mutex

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
	pmmUpdatePerformProgram = "pmm-update-perform"
	pmmUpdatePerformLog     = "/srv/logs/pmm-update-perform.log"
	pmmConfig               = "/etc/supervisord.d/pmm.ini"
)

// New creates new service.
func New(configDir string, pmmUpdateCheck *PMMUpdateChecker, params *models.Params, gRPCMessageMaxSize uint32) *Service {
	path, _ := exec.LookPath("supervisorctl")
	return &Service{
		configDir:          configDir,
		supervisorctlPath:  path,
		gRPCMessageMaxSize: gRPCMessageMaxSize,
		l:                  logrus.WithField("component", "supervisord"),
		pmmUpdateCheck:     pmmUpdateCheck,
		subs:               make(map[chan *event]sub),
		lastEvents:         make(map[string]eventType),
		vmParams:           params.VMParams,
		pgParams:           params.PGParams,
		haParams:           params.HAParams,
	}
}

// Run reads supervisord's log (maintail) and sends events to subscribers.
func (s *Service) Run(ctx context.Context) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		// pre-set installed packages info to cache it.
		s.pmmUpdateCheck.Installed(ctx)

		// Do not check for updates for the first 10 minutes.
		// That solves PMM Server building problems when we start pmm-managed.
		// TODO https://jira.percona.com/browse/PMM-4429
		sleepCtx, sleepCancel := context.WithTimeout(ctx, 10*time.Minute)
		<-sleepCtx.Done()
		sleepCancel()
		if ctx.Err() != nil {
			return
		}

		s.pmmUpdateCheck.run(ctx)
	}()
	defer wg.Wait()

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

// InstalledPMMVersion returns currently installed PMM version information.
func (s *Service) InstalledPMMVersion(ctx context.Context) *version.PackageInfo {
	return s.pmmUpdateCheck.Installed(ctx)
}

// LastCheckUpdatesResult returns last PMM update check result and last check time.
func (s *Service) LastCheckUpdatesResult(ctx context.Context) (*version.UpdateCheckResult, time.Time) {
	return s.pmmUpdateCheck.checkResult(ctx)
}

// ForceCheckUpdates forces check for PMM updates. Result can be obtained via LastCheckUpdatesResult.
func (s *Service) ForceCheckUpdates(ctx context.Context) error {
	return s.pmmUpdateCheck.check(ctx)
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

// StartUpdate starts pmm-update-perform supervisord program with some preparations.
// It returns initial log file offset.
func (s *Service) StartUpdate() (uint32, error) {
	if s.UpdateRunning() {
		return 0, status.Errorf(codes.FailedPrecondition, "Update is already running.")
	}

	// We need to remove and reopen log file for UpdateStatus API to be able to read it without it being rotated.
	// Additionally, SIGUSR2 is expected by our Ansible playbook.

	s.pmmUpdatePerformLogM.Lock()
	defer s.pmmUpdatePerformLogM.Unlock()

	// remove existing log file
	err := os.Remove(pmmUpdatePerformLog)
	if err != nil && errors.Is(err, fs.ErrNotExist) {
		err = nil
	}
	if err != nil {
		s.l.Warn(err)
	}

	// send SIGUSR2 to supervisord and wait for it to reopen log file
	ch := s.subscribe("supervisord", logReopen)
	b, err := s.supervisorctl("pid")
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil {
		return 0, errors.WithStack(err)
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return 0, errors.WithStack(err)
	}
	if err = p.Signal(unix.SIGUSR2); err != nil {
		s.l.Warnf("Failed to send SIGUSR2: %s", err)
	}
	s.l.Debug("Waiting for log reopen...")
	<-ch

	var offset uint32
	fi, err := os.Stat(pmmUpdatePerformLog)
	switch {
	case err == nil:
		if fi.Size() != 0 {
			s.l.Warnf("Unexpected log file size: %+v", fi)
		}
		offset = uint32(fi.Size())
	case errors.Is(err, fs.ErrNotExist):
		// that's expected as we remove this file above
	default:
		s.l.Warn(err)
	}

	_, err = s.supervisorctl("start", pmmUpdatePerformProgram)
	return offset, err
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

// UpdateRunning returns true if pmm-update-perform is not done yet.
func (s *Service) UpdateRunning() bool {
	return s.programRunning(pmmUpdatePerformProgram)
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

// UpdateLog returns some lines and a new offset from pmm-update-perform log starting from the given offset.
// It may return zero lines and the same offset. Caller is expected to handle this.
func (s *Service) UpdateLog(offset uint32) ([]string, uint32, error) {
	s.pmmUpdatePerformLogM.Lock()
	defer s.pmmUpdatePerformLogM.Unlock()

	f, err := os.Open(pmmUpdatePerformLog)
	if err != nil {
		return nil, 0, errors.WithStack(err)
	}
	defer f.Close() //nolint:errcheck,gosec,nolintlint

	if _, err = f.Seek(int64(offset), io.SeekStart); err != nil {
		return nil, 0, errors.WithStack(err)
	}

	lines := make([]string, 0, 10)
	reader := bufio.NewReader(f)
	newOffset := offset
	for {
		line, err := reader.ReadString('\n')
		if err == nil {
			newOffset += uint32(len(line))
			if newOffset-offset > s.gRPCMessageMaxSize {
				return lines, newOffset - uint32(len(line)), nil
			}
			lines = append(lines, strings.TrimSuffix(line, "\n"))
			continue
		}
		if err == io.EOF {
			err = nil
		}
		return lines, newOffset, errors.WithStack(err)
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

func getValueFromENV(envName string, defaultValue string) string {
	value, ok := os.LookupEnv(envName)
	if !ok {
		value = defaultValue
	}
	return value
}

// marshalConfig marshals supervisord program configuration.
func (s *Service) marshalConfig(tmpl *template.Template, settings *models.Settings, ssoDetails *models.PerconaSSODetails) ([]byte, error) {
	clickhouseDatabase := getValueFromENV("PERCONA_TEST_PMM_CLICKHOUSE_DATABASE", defaultClickhouseDatabase)
	clickhouseAddr := getValueFromENV("PERCONA_TEST_PMM_CLICKHOUSE_ADDR", defaultClickhouseAddr)
	clickhouseDataSourceAddr := getValueFromENV("PERCONA_TEST_PMM_CLICKHOUSE_DATASOURCE_ADDR", defaultClickhouseDataSourceAddr)
	clickhousePoolSize := getValueFromENV("PERCONA_TEST_PMM_CLICKHOUSE_POOL_SIZE", "")
	clickhouseBlockSize := getValueFromENV("PERCONA_TEST_PMM_CLICKHOUSE_BLOCK_SIZE", "")
	clickhouseAddrPair := strings.SplitN(clickhouseAddr, ":", 2)
	vmSearchDisableCache := getValueFromENV("VM_search_disableCache", strconv.FormatBool(!settings.VictoriaMetrics.CacheEnabled))
	vmSearchMaxQueryLen := getValueFromENV("VM_search_maxQueryLen", defaultVMSearchMaxQueryLen)
	vmSearchLatencyOffset := getValueFromENV("VM_search_latencyOffset", defaultVMSearchLatencyOffset)
	vmSearchMaxUniqueTimeseries := getValueFromENV("VM_search_maxUniqueTimeseries", defaultVMSearchMaxUniqueTimeseries)
	vmSearchMaxSamplesPerQuery := getValueFromENV("VM_search_maxSamplesPerQuery", defaultVMSearchMaxSamplesPerQuery)
	vmSearchMaxQueueDuration := getValueFromENV("VM_search_maxQueueDuration", defaultVMSearchMaxQueueDuration)
	vmSearchMaxQueryDuration := getValueFromENV("VM_search_maxQueryDuration", defaultVMSearchMaxQueryDuration)
	vmSearchLogSlowQueryDuration := getValueFromENV("VM_search_logSlowQueryDuration", defaultVMSearchLogSlowQueryDuration)
	vmPromscrapeStreamParse := getValueFromENV("VM_promscrape_streamParse", defaultVMPromscrapeStreamParse)

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
		"PerconaTestDbaas":             settings.DBaaS.Enabled,
		"InterfaceToBind":              envvars.GetInterfaceToBind(),
		"ClickhouseAddr":               clickhouseAddr,
		"ClickhouseDataSourceAddr":     clickhouseDataSourceAddr,
		"ClickhouseDatabase":           clickhouseDatabase,
		"ClickhousePoolSize":           clickhousePoolSize,
		"ClickhouseBlockSize":          clickhouseBlockSize,
		"ClickhouseHost":               clickhouseAddrPair[0],
		"ClickhousePort":               clickhouseAddrPair[1],
	}

	s.addPostgresParams(templateParams)
	s.addClusterParams(templateParams)

	templateParams["PMMServerHost"] = ""
	if settings.PMMPublicAddress != "" {
		pmmPublicAddress := settings.PMMPublicAddress
		if !strings.HasPrefix(pmmPublicAddress, "https://") && !strings.HasPrefix(pmmPublicAddress, "http://") {
			pmmPublicAddress = fmt.Sprintf("https://%s", pmmPublicAddress)
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

	if err := addAlertManagerParams(settings.AlertManagerURL, templateParams); err != nil {
		return nil, errors.Wrap(err, "cannot add AlertManagerParams to supervisor template")
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateParams); err != nil {
		return nil, errors.Wrapf(err, "failed to render template %q", tmpl.Name())
	}
	b := append([]byte("; Managed by pmm-managed. DO NOT EDIT.\n"), buf.Bytes()...)
	return b, nil
}

// addAlertManagerParams parses alertManagerURL
// and extracts url, username and password from it to templateParams.
func addAlertManagerParams(alertManagerURL string, templateParams map[string]interface{}) error {
	templateParams["AlertmanagerURL"] = "http://127.0.0.1:9093/alertmanager"
	templateParams["AlertManagerUser"] = ""
	templateParams["AlertManagerPassword"] = ""
	if alertManagerURL == "" {
		return nil
	}
	u, err := url.Parse(alertManagerURL)
	if err != nil {
		return errors.Wrap(err, "cannot parse AlertManagerURL")
	}
	if u.Opaque != "" || u.Host == "" {
		return errors.Errorf("AlertmanagerURL parsed incorrectly as %#v", u)
	}
	password, _ := u.User.Password()
	n := url.URL{
		Scheme:   u.Scheme,
		Host:     u.Host,
		Path:     u.Path,
		RawQuery: u.RawQuery,
		Fragment: u.Fragment,
	}
	templateParams["AlertManagerUser"] = fmt.Sprintf(",%s", u.User.Username())
	templateParams["AlertManagerPassword"] = fmt.Sprintf(",%s", strconv.Quote(password))
	templateParams["AlertmanagerURL"] = fmt.Sprintf("http://127.0.0.1:9093/alertmanager,%s", n.String())

	return nil
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
	//- GF_UNIFIED_ALERTING_HA_ADVERTISE_ADDRESS=172.20.0.5:9095
	//- GF_UNIFIED_ALERTING_HA_PEERS=pmm-server-active:9095,pmm-server-passive:9095
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

// UpdateConfiguration updates Prometheus, Alertmanager, Grafana and qan-api2 configurations, restarting them if needed.
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
{{define "dbaas-controller"}}
[program:dbaas-controller]
priority = 6
command = /usr/sbin/dbaas-controller
user = pmm
autorestart = {{ .PerconaTestDbaas }}
autostart = {{ .PerconaTestDbaas }}
startretries = 10
startsecs = 1
stopsignal = TERM
stopwaitsecs = 300
stdout_logfile = /srv/logs/dbaas-controller.log
stdout_logfile_maxbytes = 10MB
stdout_logfile_backups = 3
redirect_stderr = true
{{end}}

{{define "prometheus"}}
[program:prometheus]
command = /bin/echo Prometheus is substituted by VictoriaMetrics
user = pmm
autorestart = false
autostart = false
startretries = 10
startsecs = 1
stopsignal = TERM
stopwaitsecs = 300
stdout_logfile = /srv/logs/prometheus.log
stdout_logfile_maxbytes = 10MB
stdout_logfile_backups = 3
redirect_stderr = true
{{end}}

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
		--prometheusDataPath=/srv/prometheus/data
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
		--notifier.url="{{ .AlertmanagerURL }}"
		--notifier.basicAuth.password='{{ .AlertManagerPassword }}'
		--notifier.basicAuth.username="{{ .AlertManagerUser }}"
		--external.url={{ .VMURL }}
		--datasource.url={{ .VMURL }}
		--remoteRead.url={{ .VMURL }}
		--remoteRead.ignoreRestoreErrors=false
		--remoteWrite.url={{ .VMURL }}
		--rule=/srv/prometheus/rules/*.yml
		--rule=/etc/ia/rules/*.yml
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

{{define "alertmanager"}}
[program:alertmanager]
priority = 8
command =
	/usr/sbin/alertmanager
		--config.file=/etc/alertmanager.yml
		--storage.path=/srv/alertmanager/data
		--data.retention={{ .DataRetentionHours }}h
		--web.external-url=http://localhost:9093/alertmanager/
		--web.listen-address={{ .InterfaceToBind }}:9093
		--cluster.listen-address=""
user = pmm
autorestart = true
autostart = true
startretries = 1000
startsecs = 1
stopsignal = TERM
stopwaitsecs = 10
stdout_logfile = /srv/logs/alertmanager.log
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
	PERCONA_TEST_PMM_CLICKHOUSE_ADDR="{{ .ClickhouseAddr }}",
	PERCONA_TEST_PMM_CLICKHOUSE_DATABASE="{{ .ClickhouseDatabase }}",
{{ if .ClickhousePoolSize }}	PERCONA_TEST_PMM_CLICKHOUSE_POOL_SIZE={{ .ClickhousePoolSize }},{{- end}}
{{ if .ClickhouseBlockSize }}	PERCONA_TEST_PMM_CLICKHOUSE_BLOCK_SIZE={{ .ClickhouseBlockSize }}{{- end}}
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
    PERCONA_TEST_POSTGRES_ADDR="{{ .PostgresAddr }}",
    PERCONA_TEST_POSTGRES_DBNAME="{{ .PostgresDBName }}",
    PERCONA_TEST_POSTGRES_USERNAME="{{ .PostgresDBUsername }}",
    PERCONA_TEST_POSTGRES_DBPASSWORD="{{ .PostgresDBPassword }}",
    PERCONA_TEST_POSTGRES_SSL_MODE="{{ .PostgresSSLMode }}",
    PERCONA_TEST_POSTGRES_SSL_CA_PATH="{{ .PostgresSSLCAPath }}",
    PERCONA_TEST_POSTGRES_SSL_KEY_PATH="{{ .PostgresSSLKeyPath }}",
    PERCONA_TEST_POSTGRES_SSL_CERT_PATH="{{ .PostgresSSLCertPath }}",
    PERCONA_TEST_PMM_CLICKHOUSE_DATASOURCE_ADDR="{{ .ClickhouseDataSourceAddr }}",
    PERCONA_TEST_PMM_CLICKHOUSE_HOST="{{ .ClickhouseHost }}",
    PERCONA_TEST_PMM_CLICKHOUSE_PORT="{{ .ClickhousePort }}",
    {{- if .PerconaSSODetails}}
    GF_AUTH_SIGNOUT_REDIRECT_URL="https://{{ .IssuerDomain }}/login/signout?fromURI=https://{{ .PMMServerAddress }}/graph/login"
    {{- end}}
    {{- if .HAEnabled}}
    GF_UNIFIED_ALERTING_HA_LISTEN_ADDRESS="0.0.0.0:{{ .GrafanaGossipPort }}",
    GF_UNIFIED_ALERTING_HA_ADVERTISE_ADDRESS="{{ .HAAdvertiseAddress }}:{{ .GrafanaGossipPort }}",
    GF_UNIFIED_ALERTING_HA_PEERS="{{ .HANodes }}"
    {{- end}}
user = grafana
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
`))
