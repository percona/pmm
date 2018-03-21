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
	"time"
)

type Log struct {
	Path       string
	Journalctl string
}

type File struct {
	Name string
	Data []byte
	Err  error
}

var (
	DefaultLogs = []Log{
		// Managed by supervisord
		{"/var/log/mysql.log", ""},
		{"/var/log/consul.log", "consul"},
		{"/var/log/cron.log", ""},
		{"/var/log/qan-api.log", "percona-qan-api"},
		{"/var/log/prometheus.log", "prometheus"},
		{"/var/log/createdb.log", ""},
		{"/var/log/orchestrator.log", "orchestrator"},
		{"/var/log/dashboard-upgrade.log", ""},
		{"/var/log/node_exporter.log", "node_exporter"},
		{"/var/log/pmm-manage.log", "pmm-manage"},
		{"/var/log/pmm-managed.log", ""},
		{"/var/log/supervisor/supervisord.log", ""},
		// Grafana and Nginx
		{"/var/log/grafana/grafana.log", ""},
		{"/var/log/nginx.log", "nginx"},
		{"/var/log/nginx/access.log", ""},
		{"/var/log/nginx/error.log", ""},
		// AMI/OVF
		{"/var/log/mysqld.log", "mysqld"},
		{"/var/log/nginx/access.log", ""},
		{"/var/log/nginx/error.log", ""},
		{"", "crond"},
	}
)

// Logs is responsible for interactions with logs.
type Logs struct {
	logs []Log
	n    int
}

// New creates a new Logs service.
func New(logs []Log, n int) *Logs {
	return &Logs{
		logs: logs,
		n:    n,
	}
}

// Zip creates .zip archive with all logs.
func (l *Logs) Zip(w io.Writer) error {
	// Create a new zip archive.
	zw := zip.NewWriter(w)

	// Add logs to the archive.
	for _, log := range l.logs {
		name, content, err := l.TailN(log, l.n)
		if err != nil {
			content = append(content, '\n')
			content = append(content, []byte(err.Error())...)
		}
		f, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, err = f.Write(content)
		if err != nil {
			return err
		}
	}

	// Make sure to check the error on Close.
	return zw.Close()
}

// Files returns list of logs and their content.
func (l *Logs) Files() []File {
	files := make([]File, 0, len(l.logs))

	for _, log := range l.logs {
		file := File{}
		file.Name, file.Data, file.Err = l.TailN(log, l.n)
		files = append(files, file)
	}

	return files
}

// TailN reads last n lines from defined Log configuration
func (l *Logs) TailN(log Log, n int) (name string, data []byte, err error) {
	if _, err := exec.LookPath("journalctl"); err == nil && log.Journalctl != "" {
		data, err := JournalctlN(log.Journalctl, n)
		return log.Journalctl, data, err
	} else if log.Path != "" {
		data, err := TailN(log.Path, n)
		return filepath.Base(log.Path), data, err
	}

	return "", nil, fmt.Errorf("unable to get log: %v", log)
}

// JournalctlN reads last n lines from systemd unit u.
func JournalctlN(u string, n int) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "journalctl", "-n", strconv.Itoa(n), "-u", u)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	b, err := cmd.Output()
	if err != nil {
		return b, fmt.Errorf("JournalctlN(%s, %d): %s: %s", u, n, err, stderr.String())
	}
	return b, nil
}

// TailN reads last n lines from log file at given path.
func TailN(path string, n int) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "/usr/bin/tail", "-n", strconv.Itoa(n), path)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	b, err := cmd.Output()
	if err != nil {
		return b, fmt.Errorf("TailN(%s, %d): %s: %s", path, n, err, stderr.String())
	}
	return b, nil
}
