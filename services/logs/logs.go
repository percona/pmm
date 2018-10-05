// pmm-managed
// Copyright (C) 2017 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package logs

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	servicelib "github.com/percona/kardianos-service"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/percona/pmm-managed/services/consul"
	"github.com/percona/pmm-managed/services/rds"
	"github.com/percona/pmm-managed/utils/logger"
)

// File represents log file content.
type File struct {
	Name string
	Data []byte
	Err  error
}

type Log struct {
	FilePath  string
	UnitName  string
	Extractor []string
}

const lastLines = 1000

var defaultLogs = []Log{
	{"/var/log/consul.log", "consul", nil},
	{"/var/log/createdb.log", "", nil},
	{"/var/log/cron.log", "crond", nil},
	{"/var/log/dashboard-upgrade.log", "", nil},
	{"/var/log/grafana/grafana.log", "", nil},
	{"/var/log/mysql.log", "", nil},
	{"/var/log/mysqld.log", "mysqld", nil},
	{"/var/log/nginx.log", "nginx", nil},
	{"/var/log/nginx/access.log", "", nil},
	{"/var/log/nginx/error.log", "", nil},
	{"/var/log/node_exporter.log", "node_exporter", nil},
	{"/var/log/orchestrator.log", "orchestrator", nil},
	{"/var/log/pmm-manage.log", "pmm-manage", nil},
	{"/var/log/pmm-managed.log", "pmm-managed", nil},
	{"/var/log/prometheus1.log", "prometheus1", nil},
	{"/var/log/prometheus.log", "prometheus", nil},
	{"/var/log/qan-api.log", "percona-qan-api", nil},
	{"/var/log/supervisor/supervisord.log", "", nil},

	// logs
	// TODO handle separately
	{"supervisorctl_status.log", "", []string{"exec", "supervisorctl status"}},
	{"systemctl_status.log", "", []string{"exec", "systemctl -l status"}},
	{"pt-summary.log", "", []string{"exec", "pt-summary"}},

	// configs
	// TODO handle separately
	{"/etc/prometheus.yml", "", []string{"cat", ""}},
	{"/etc/supervisord.d/pmm.ini", "", []string{"cat", ""}},
	{"/etc/nginx/conf.d/pmm.conf", "", []string{"cat", ""}},
	{"prometheus_targets.html", "", []string{"http", "http://localhost/prometheus/targets"}},
	{"consul_nodes.json", "", []string{"consul", "http://localhost/v1/internal/ui/nodes?dc=dc1"}},
	{"qan-api_instances.json", "", []string{"http", "http://localhost/qan-api/instances"}},
	{"managed_RDS-Aurora.json", "", []string{"rds", "http://localhost/managed/v0/rds"}},
	{"pmm-version.txt", "", []string{"pmmVersion", ""}},
}

// Logs is responsible for interactions with logs.
type Logs struct {
	pmmVersion string
	consul     *consul.Client
	rds        *rds.Service
	logs       []Log

	journalctlPath string
}

type manageConfig struct {
	Users []struct {
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"users"`
}

// getCredential fetchs PMM credential
func getCredential() (string, error) {
	var u string
	f, err := os.Open("/srv/update/pmm-manage.yml")
	if err != nil {
		return u, err
	}

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return u, err
	}

	var config manageConfig
	if err = yaml.Unmarshal(b, &config); err != nil {
		return u, err
	}

	if len(config.Users) > 0 && config.Users[0].Username != "" {
		u = strings.Join([]string{config.Users[0].Username, config.Users[0].Password}, ":")
	}

	err = f.Close()
	if err != nil {
		return u, err
	}
	return u, err
}

// New creates a new Logs service.
// n is a number of last lines of log to read.
func New(pmmVersion string, consul *consul.Client, rds *rds.Service, logs []Log) *Logs {
	if logs == nil {
		logs = defaultLogs
	}

	l := &Logs{
		pmmVersion: pmmVersion,
		consul:     consul,
		rds:        rds,
		logs:       logs,
	}

	// PMM Server Docker image contails journalctl, so we can't use exec.LookPath("journalctl") alone for detection.
	// TODO Probably, that check should be moved to supervisor service.
	//      Or the whole logs service should be merged with it.
	if servicelib.Platform() == "linux-systemd" {
		l.journalctlPath, _ = exec.LookPath("journalctl")
	}

	return l
}

// Zip creates .zip archive with all logs.
func (l *Logs) Zip(ctx context.Context, w io.Writer) error {
	zw := zip.NewWriter(w)
	now := time.Now().UTC()
	for _, log := range l.logs {
		name, content, err := l.readLog(ctx, &log)
		if name == "" {
			continue
		}

		if err != nil {
			logger.Get(ctx).WithField("component", "logs").Error(err)

			// do not let a single error break the whole archive
			if len(content) > 0 {
				content = append(content, "\n\n"...)
			}
			content = append(content, []byte(err.Error())...)
		}

		f, err := zw.CreateHeader(&zip.FileHeader{
			Name:     name,
			Method:   zip.Deflate,
			Modified: now,
		})
		if err != nil {
			return err
		}
		if _, err = f.Write(content); err != nil {
			return err
		}
	}

	// make sure to check the error on Close
	return zw.Close()
}

// Files returns list of logs and their content.
func (l *Logs) Files(ctx context.Context) []File {
	files := make([]File, len(l.logs))

	for i, log := range l.logs {
		var file File
		file.Name, file.Data, file.Err = l.readLog(ctx, &log)
		files[i] = file
	}

	return files
}

// readLog reads last lines from defined Log configuration.
func (l *Logs) readLog(ctx context.Context, log *Log) (name string, data []byte, err error) {
	if log.Extractor != nil {
		return l.readWithExtractor(ctx, log)
	}

	if log.UnitName != "" && l.journalctlPath != "" {
		name = log.UnitName
		data, err = l.journalctlN(ctx, log.UnitName)
		return
	}

	if log.FilePath != "" {
		name = filepath.Base(log.FilePath)
		data, err = l.tailN(ctx, log.FilePath)
		return
	}

	return
}

func (l *Logs) readWithExtractor(ctx context.Context, log *Log) (name string, data []byte, err error) {
	name = filepath.Base(log.FilePath)

	switch log.Extractor[0] {
	case "exec":
		data, err = l.collectExec(ctx, log.FilePath, log.Extractor[1])

	case "pmmVersion":
		data = []byte(l.pmmVersion)

	case "consul":
		data, err = l.getConsulNodes()

	case "rds":
		data, err = l.getRDSInstances(ctx)

	case "http":
		command := log.Extractor[1]
		s := strings.Split(command, "//")
		credential, err1 := getCredential()
		if len(s) > 1 && len(credential) > 1 {
			command = fmt.Sprintf("%s//%s@%s", s[0], credential, s[1])
		}
		data, err = l.readURL(command)
		if err1 != nil {
			err = fmt.Errorf("%v; %v", err1, err)
		}

	case "cat":
		data, err = ioutil.ReadFile(log.FilePath)

	default:
		panic("unhandled extractor")
	}

	return
}

// journalctlN reads last lines from systemd unit u using `journalctl` command.
func (l *Logs) journalctlN(ctx context.Context, u string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, l.journalctlPath, "-n", strconv.Itoa(lastLines), "-u", u)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	b, err := cmd.Output()
	if err != nil {
		return b, fmt.Errorf("%s: %s: %s", strings.Join(cmd.Args, " "), err, stderr.String())
	}
	return b, nil
}

// tailN reads last lines from log file at given path using `tail` command.
func (l *Logs) tailN(ctx context.Context, path string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "/usr/bin/tail", "-n", strconv.Itoa(lastLines), path)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	b, err := cmd.Output()
	if err != nil {
		return b, fmt.Errorf("%s: %s: %s", strings.Join(cmd.Args, " "), err, stderr.String())
	}
	return b, nil
}

// collectExec collects output from various commands
func (l *Logs) collectExec(ctx context.Context, path string, command string) ([]byte, error) {
	var cmd *exec.Cmd
	if filepath.Dir(path) != "." {
		cmd = exec.CommandContext(ctx, command, path)
	} else {
		command := strings.Split(command, " ")
		cmd = exec.CommandContext(ctx, command[0], command[1:]...)
	}
	var stderr bytes.Buffer
	cmd.Stderr = new(bytes.Buffer)
	b, err := cmd.Output()
	if err != nil {
		return b, fmt.Errorf("%s: %s: %s", strings.Join(cmd.Args, " "), err, stderr.String())
	}
	return b, nil
}

// readUrl reads content of a page
func (l *Logs) readURL(url string) ([]byte, error) {
	u, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer u.Body.Close()
	b, err := ioutil.ReadAll(u.Body)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// getConsulNodes gets list of nodes
func (l *Logs) getConsulNodes() ([]byte, error) {
	nodes, err := l.consul.GetNodes()
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(nodes, "", "  ")
}

// getRDSInstances gets list of monitored instances
func (l *Logs) getRDSInstances(ctx context.Context) ([]byte, error) {
	if l.rds == nil {
		return nil, errors.New("RDS service not initialized")
	}

	instances, err := l.rds.List(ctx)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(instances, "", " ")
}
