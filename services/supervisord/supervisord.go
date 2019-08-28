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
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

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
	pmmUpdateCheck    *pmmUpdateChecker

	eventsM                   sync.Mutex
	subs                      map[chan *event]sub
	pmmUpdatePerformLastEvent eventType

	pmmUpdatePerformLogM sync.Mutex
	supervisordConfigsM  sync.Mutex
}

type sub struct {
	program    string
	eventTypes []eventType
}

// values from supervisord configuration
const (
	pmmUpdatePerformProgram = "pmm-update-perform"
	pmmUpdatePerformLog     = "/srv/logs/pmm-update-perform.log"
)

// New creates new service.
func New(configDir string) *Service {
	path, _ := exec.LookPath("supervisorctl")
	return &Service{
		configDir:                 configDir,
		supervisorctlPath:         path,
		l:                         logrus.WithField("component", "supervisord"),
		pmmUpdateCheck:            newPMMUpdateChecker(logrus.WithField("component", "supervisord/pmm-update-checker")),
		subs:                      make(map[chan *event]sub),
		pmmUpdatePerformLastEvent: unknown,
	}
}

// Run reads supervisord's log (maintail) and sends events to subscribers.
func (s *Service) Run(ctx context.Context) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

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

			var toDelete []chan *event
			for ch, sub := range s.subs {
				if e.Program == pmmUpdatePerformProgram {
					s.pmmUpdatePerformLastEvent = e.Type
				}

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
func (s *Service) InstalledPMMVersion() *version.PackageInfo {
	return s.pmmUpdateCheck.installed()
}

// LastCheckUpdatesResult returns last PMM update check result and last check time.
func (s *Service) LastCheckUpdatesResult() (*version.UpdateCheckResult, time.Time) {
	return s.pmmUpdateCheck.checkResult()
}

// ForceCheckUpdates forces check for PMM updates. Result can be obtained via LastCheckUpdatesResult.
func (s *Service) ForceCheckUpdates() error {
	return s.pmmUpdateCheck.check()
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
	s.l.Debug("Waiting for logreopen...")
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

// UpdateRunning returns true if pmm-update-perform supervisord program is running or being restarted,
// false if it is not running / failed.
func (s *Service) UpdateRunning() bool {
	// First check with status command is case we missed that event during maintail or pmm-managed restart.
	// See http://supervisord.org/subprocess.html#process-states
	b, err := s.supervisorctl("status", pmmUpdatePerformProgram)
	if err != nil {
		s.l.Warn(err)
	}
	s.l.Debugf("%s", b)
	if f := strings.Fields(string(b)); len(f) > 2 {
		switch status := f[1]; status {
		case "FATAL", "STOPPED": // will not be restarted
			return false
		case "STARTING", "RUNNING", "BACKOFF", "STOPPING":
			return true
		case "EXITED":
			// it might be restarted - we need to inspect last event
		default:
			s.l.Warnf("Unknown %s process status %q.", pmmUpdatePerformProgram, status)
			// inspect last event
		}
	}

	s.eventsM.Lock()
	lastEvent := s.pmmUpdatePerformLastEvent
	s.eventsM.Unlock()

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
		s.l.Warnf("Unhandled %s status (last event %q), assuming it is not running.", pmmUpdatePerformProgram, lastEvent)
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
	templateParams := map[string]interface{}{
		"DataRetentionDays": int(settings.DataRetention.Hours() / 24),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateParams); err != nil {
		return nil, errors.Wrapf(err, "failed to render template %q", tmpl.Name())
	}
	b := append([]byte("; Managed by pmm-managed. DO NOT EDIT.\n"), buf.Bytes()...)
	return b, nil
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

// UpdateConfiguration updates Prometheus and qan-api2 configurations, restarting them if needed.
func (s *Service) UpdateConfiguration(settings *models.Settings) error {
	if s.supervisorctlPath == "" {
		s.l.Errorf("supervisorctl not found, configuration updates are disabled.")
		return nil
	}

	s.supervisordConfigsM.Lock()
	defer s.supervisordConfigsM.Unlock()

	var err error
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

var templates = template.Must(template.New("").Option("missingkey=error").Parse(`
{{define "prometheus"}}
[program:prometheus]
priority = 7
command =
	/usr/sbin/prometheus
		--config.file=/etc/prometheus.yml
		--query.max-concurrency=30
		--storage.tsdb.path=/srv/prometheus/data
		--storage.tsdb.retention.time={{ .DataRetentionDays }}d
		--storage.tsdb.wal-compression
		--web.console.libraries=/usr/share/prometheus/console_libraries
		--web.console.templates=/usr/share/prometheus/consoles
		--web.enable-admin-api
		--web.enable-lifecycle
		--web.external-url=http://localhost:9090/prometheus/
		--web.listen-address=:9090
user = pmm
autorestart = true
autostart = true
startretries = 3
startsecs = 1
stopsignal = TERM
stopwaitsecs = 300
stdout_logfile = /srv/logs/prometheus.log
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
