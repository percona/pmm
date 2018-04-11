// pmm-managed
// Copyright (C) 2018 Percona LLC
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
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	servicelib "github.com/percona/kardianos-service"
	"github.com/sirupsen/logrus"
)

// File represents log file content.
type File struct {
	Name string
	Data []byte
	Err  error
}

type Log struct {
	FilePath string
	UnitName string
}

var DefaultLogs = []Log{
	{"/var/log/consul.log", "consul"},
	{"/var/log/createdb.log", ""},
	{"/var/log/cron.log", "crond"},
	{"/var/log/dashboard-upgrade.log", ""},
	{"/var/log/grafana/grafana.log", ""},
	{"/var/log/mysql.log", ""},
	{"/var/log/mysqld.log", "mysqld"},
	{"/var/log/nginx.log", "nginx"},
	{"/var/log/nginx/access.log", ""},
	{"/var/log/nginx/error.log", ""},
	{"/var/log/node_exporter.log", "node_exporter"},
	{"/var/log/orchestrator.log", "orchestrator"},
	{"/var/log/pmm-manage.log", "pmm-manage"},
	{"/var/log/pmm-managed.log", "pmm-managed"},
	{"/var/log/prometheus.log", "prometheus"},
	{"/var/log/qan-api.log", "percona-qan-api"},
	{"/var/log/supervisor/supervisord.log", ""},
}

// Logs is responsible for interactions with logs.
type Logs struct {
	n              int
	logs           []Log
	journalctlPath string
	l              *logrus.Entry
}

// New creates a new Logs service.
// n is a number of last lines of log to read.
func New(logs []Log, n int) *Logs {
	l := &Logs{
		n:    n,
		logs: logs,
		l:    logrus.WithField("component", "logs"),
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
			l.l.Error(err)

			// do not let a single error break the whole archive
			if len(content) > 0 {
				content = append(content, "\n\n"...)
			}
			content = append(content, []byte(err.Error())...)
		}

		f, err := zw.CreateHeader(&zip.FileHeader{
			Name:   name,
			Method: zip.Deflate,
			// Modified: now, TODO uncomment when we build with Go 1.10+
		})
		_ = now // TODO remove
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

// readLog reads last l.n lines from defined Log configuration.
func (l *Logs) readLog(ctx context.Context, log *Log) (name string, data []byte, err error) {
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

// journalctlN reads last l.n lines from systemd unit u using `journalctl` command.
func (l *Logs) journalctlN(ctx context.Context, u string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, l.journalctlPath, "-n", strconv.Itoa(l.n), "-u", u)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	b, err := cmd.Output()
	if err != nil {
		return b, fmt.Errorf("%s: %s: %s", strings.Join(cmd.Args, " "), err, stderr.String())
	}
	return b, nil
}

// tailN reads last l.n lines from log file at given path using `tail` command.
func (l *Logs) tailN(ctx context.Context, path string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "/usr/bin/tail", "-n", strconv.Itoa(l.n), path)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	b, err := cmd.Output()
	if err != nil {
		return b, fmt.Errorf("%s: %s: %s", strings.Join(cmd.Args, " "), err, stderr.String())
	}
	return b, nil
}
