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
	"sort"
	"strconv"
	"strings"
	"time"

	servicelib "github.com/percona/kardianos-service"
	"github.com/pkg/errors"
	"gopkg.in/reform.v1"
	"gopkg.in/yaml.v2"

	"github.com/percona/pmm-managed/models"
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

// overridden in tests
var logsRootDir = "/var/log/"
var logsDataValumeContainerDir = "/srv/logs/"

var defaultLogs = []Log{
	{logsDataValumeContainerDir + "consul.log", "consul", nil},
	{logsDataValumeContainerDir + "createdb.log", "", nil},
	{logsDataValumeContainerDir + "cron.log", "crond", nil},
	{logsDataValumeContainerDir + "dashboard-upgrade.log", "", nil},
	{logsRootDir + "grafana/grafana.log", "", nil},
	{logsRootDir + "mysql.log", "", nil},
	{logsRootDir + "mysqld.log", "mysqld", nil},
	{logsDataValumeContainerDir + "nginx.log", "nginx", nil},
	{logsRootDir + "nginx/access.log", "", nil},
	{logsRootDir + "nginx/error.log", "", nil},
	{logsDataValumeContainerDir + "node_exporter.log", "node_exporter", nil},
	{logsRootDir + "orchestrator.log", "orchestrator", nil},
	{logsDataValumeContainerDir + "pmm-manage.log", "pmm-manage", nil},
	{logsDataValumeContainerDir + "pmm-managed.log", "pmm-managed", nil},
	{logsDataValumeContainerDir + "prometheus1.log", "prometheus1", nil},
	{logsDataValumeContainerDir + "prometheus.log", "prometheus", nil},
	{logsDataValumeContainerDir + "qan-api.log", "percona-qan-api", nil},
	{logsRootDir + "supervisor/supervisord.log", "", nil},

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
	{"consul_nodes.json", "", []string{"consul"}},
	{"qan-api_instances.json", "", []string{"http", "http://localhost/qan-api/instances"}},
	{"managed_RDS-Aurora.json", "", []string{"rds"}},
	{"pmm-version.txt", "", []string{"pmmVersion", ""}},
}

// Logs is responsible for interactions with logs.
type Logs struct {
	pmmVersion string
	consul     *consul.Client
	db         *reform.DB
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
func New(pmmVersion string, consul *consul.Client, db *reform.DB, rds *rds.Service, logs []Log) *Logs {
	if logs == nil {
		logs = defaultLogs
	}

	l := &Logs{
		pmmVersion: pmmVersion,
		consul:     consul,
		db:         db,
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

// Files returns list of logs and their content.
func (l *Logs) Files(ctx context.Context) []File {
	files := make([]File, 0, len(l.logs))

	for _, log := range l.logs {
		var f File
		f.Name, f.Data, f.Err = l.readLog(ctx, &log)
		files = append(files, f)
	}

	var agents []reform.Struct
	err := l.db.InTransaction(func(tx *reform.TX) error {
		var node models.Node
		err := tx.FindOneTo(&node, "type", models.PMMServerNodeType)
		if err != nil {
			return errors.Wrap(err, "failed to get PMM Server node")
		}

		agents, err = tx.FindAllFrom(models.AgentTable, "runs_on_node_id", node.ID)
		return errors.Wrap(err, "failed to get agents running in PMM Server")
	})
	if err != nil {
		logger.Get(ctx).WithField("component", "logs").Errorf("Failed to get RDS agents: %s.", err)
	}

	// collect names in set, not in slice,
	// in case a single agent does several jobs and is returned several times
	names := make(map[string]struct{}, len(agents))
	for _, a := range agents {
		agent := a.(*models.Agent)
		name := models.NameForSupervisor(agent.Type, *agent.ListenPort)
		names[name] = struct{}{}
	}

	for name := range names {
		log := Log{
			FilePath: logsRootDir + name + ".log",
			UnitName: name,
		}
		var f File
		f.Name, f.Data, f.Err = l.readLog(ctx, &log)
		files = append(files, f)
	}

	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	return files
}

// Zip creates .zip archive with all logs.
func (l *Logs) Zip(ctx context.Context, w io.Writer) error {
	zw := zip.NewWriter(w)
	now := time.Now().UTC()
	for _, file := range l.Files(ctx) {
		if file.Name == "" {
			continue
		}

		if file.Err != nil {
			logger.Get(ctx).WithField("component", "logs").Error(file.Err)

			// do not let a single error break the whole archive
			if len(file.Data) > 0 {
				file.Data = append(file.Data, "\n\n"...)
			}
			file.Data = append(file.Data, []byte(file.Err.Error())...)
		}

		f, err := zw.CreateHeader(&zip.FileHeader{
			Name:     file.Name,
			Method:   zip.Deflate,
			Modified: now,
		})
		if err != nil {
			return errors.Wrap(err, "failed to create zip file header")
		}
		if _, err = f.Write(file.Data); err != nil {
			return errors.Wrap(err, "failed to write zip file data")
		}
	}
	return errors.Wrap(zw.Close(), "failed to close zip file")
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
