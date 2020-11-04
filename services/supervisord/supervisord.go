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

// Package supervisord provides facilities for working with Supervisord.
package supervisord

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"math"
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
	"github.com/percona/pmm/utils/pdeathsig"
	"github.com/percona/pmm/version"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm-managed/models"
)

// Service is responsible for interactions with Supervisord via supervisorctl.
type Service struct {
	configDir         string
	supervisorctlPath string
	l                 *logrus.Entry
	pmmUpdateCheck    *PMMUpdateChecker

	eventsM    sync.Mutex
	subs       map[chan *event]sub
	lastEvents map[string]eventType

	pmmUpdatePerformLogM sync.Mutex
	supervisordConfigsM  sync.Mutex

	vmParams *models.VictoriaMetricsParams
}

type sub struct {
	program    string
	eventTypes []eventType
}

// values from supervisord configuration
const (
	dashboardUpgradeProgram = "dashboard-upgrade"
	pmmUpdatePerformProgram = "pmm-update-perform"
	pmmUpdatePerformLog     = "/srv/logs/pmm-update-perform.log"
)

// New creates new service.
func New(configDir string, pmmUpdateCheck *PMMUpdateChecker, vmParams *models.VictoriaMetricsParams) *Service {
	path, _ := exec.LookPath("supervisorctl")
	return &Service{
		configDir:         configDir,
		supervisorctlPath: path,
		l:                 logrus.WithField("component", "supervisord"),
		pmmUpdateCheck:    pmmUpdateCheck,
		subs:              make(map[chan *event]sub),
		lastEvents:        make(map[string]eventType),
		vmParams:          vmParams,
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
	if err != nil && os.IsNotExist(err) {
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
	case os.IsNotExist(err):
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

// UpdateRunning returns true if dashboard-upgrade or pmm-update-perform is not done yet.
func (s *Service) UpdateRunning() bool {
	return s.programRunning(dashboardUpgradeProgram) || s.programRunning(pmmUpdatePerformProgram)
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
	defer f.Close() //nolint:errcheck

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
	// See https://github.com/Supervisor/supervisor/issues/1264 for explanation
	// why we do reread + stop/remove/add.

	if _, err := s.supervisorctl("reread"); err != nil {
		s.l.Warn(err)
	}
	if _, err := s.supervisorctl("stop", name); err != nil {
		s.l.Warn(err)
	}
	if _, err := s.supervisorctl("remove", name); err != nil {
		s.l.Warn(err)
	}

	_, err := s.supervisorctl("add", name)
	return err
}

// marshalConfig marshals supervisord program configuration.
func (s *Service) marshalConfig(tmpl *template.Template, settings *models.Settings) ([]byte, error) {
	retentionMonths := int(math.Ceil(settings.DataRetention.Hours() / 24 / 30))
	if retentionMonths <= 0 {
		retentionMonths = 1
	}
	templateParams := map[string]interface{}{
		"DataRetentionHours":  int(settings.DataRetention.Hours()),
		"DataRetentionDays":   int(settings.DataRetention.Hours() / 24),
		"DataRetentionMonths": retentionMonths,
		"VMAlertFlags":        s.vmParams.VMAlertFlags,
		"VMDBCacheDisable":    !settings.VictoriaMetrics.CacheEnabled,
		"PerconaTestDbaas":    settings.DBaaS.Enabled,
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

// saveConfigAndReload saves given supervisord program configuration to file and reloads it.
// If configuration can't be reloaded for some reason, old file is restored, and configuration is reloaded again.
// Returns true if configuration was changed.
func (s *Service) saveConfigAndReload(name string, cfg []byte) (bool, error) {
	// read existing content
	path := filepath.Join(s.configDir, name+".ini")
	oldCfg, err := ioutil.ReadFile(path) //nolint:gosec
	if os.IsNotExist(err) {
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
			if err = ioutil.WriteFile(path, oldCfg, 0644); err != nil {
				s.l.Errorf("Failed to restore: %s.", err)
			}
			if err = s.reload(name); err != nil {
				s.l.Errorf("Failed to restore/reload: %s.", err)
			}
		}
	}()

	// write and reload
	if err = ioutil.WriteFile(path, cfg, 0644); err != nil {
		return false, errors.WithStack(err)
	}
	if err = s.reload(name); err != nil {
		return false, err
	}
	s.l.Infof("%s configuration reloaded.", name)
	restore = false
	return true, nil
}

// UpdateConfiguration updates Prometheus, Alertmanager, and qan-api2 configurations, restarting them if needed.
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

		b, e := s.marshalConfig(tmpl, settings)
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

// TODO Switch from /srv/alertmanager/alertmanager.base.yml to /etc/alertmanager.yml
// once we start generating it. See alertmanager service.

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
[program:victoriametrics]
priority = 7
command =
	/usr/sbin/victoriametrics
		--promscrape.config=/etc/victoriametrics-promscrape.yml
		--retentionPeriod={{ .DataRetentionMonths }}
		--storageDataPath=/srv/victoriametrics/data
		--httpListenAddr=127.0.0.1:9090
		--search.disableCache={{.VMDBCacheDisable}}
		--prometheusDataPath=/srv/prometheus/data
		--http.pathPrefix=/prometheus
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
{{end}}

{{define "vmalert"}}
[program:vmalert]
priority = 7
command =
	/usr/sbin/vmalert
        --notifier.url="{{ .AlertmanagerURL }}"
        --notifier.basicAuth.password='{{ .AlertManagerPassword }}'
        --notifier.basicAuth.username="{{ .AlertManagerUser}}"
        --external.url=http://localhost:9090/prometheus
        --datasource.url=http://127.0.0.1:9090/prometheus
        --remoteRead.url=http://127.0.0.1:9090/prometheus
        --remoteWrite.url=http://127.0.0.1:9090/prometheus
        --rule=/srv/prometheus/rules/*.yml
        --httpListenAddr=127.0.0.1:8880
{{- range $index, $param := .VMAlertFlags}}
        {{$param}}
{{- end}}
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

{{define "alertmanager"}}
[program:alertmanager]
priority = 8
command =
	/usr/sbin/alertmanager
		--config.file=/srv/alertmanager/alertmanager.base.yml
		--storage.path=/srv/alertmanager/data
		--data.retention={{ .DataRetentionHours }}h
		--web.external-url=http://localhost:9093/alertmanager/
		--web.listen-address=127.0.0.1:9093
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
`))
