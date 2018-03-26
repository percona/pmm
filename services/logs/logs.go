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
)

type log struct {
	Path       string
	Journalctl string
}

type File struct {
	Name string
	Data []byte
	Err  error
}

var (
	defaultLogs = []log{
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
	n    int
	logs []log
}

// New creates a new Logs service.
func New(n int) *Logs {
	return &Logs{
		n:    n,
		logs: defaultLogs,
	}
}

// Zip creates .zip archive with all logs.
func (l *Logs) Zip(ctx context.Context, w io.Writer) error {
	// Create a new zip archive.
	zw := zip.NewWriter(w)

	// Add logs to the archive.
	for _, log := range l.logs {
		name, content, err := l.tailN(ctx, &log, l.n)
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
func (l *Logs) Files(ctx context.Context) []File {
	files := make([]File, 0, len(l.logs))

	for _, log := range l.logs {
		file := File{}
		file.Name, file.Data, file.Err = l.tailN(ctx, &log, l.n)
		files = append(files, file)
	}

	return files
}

// tailN reads last n lines from defined Log configuration
func (l *Logs) tailN(ctx context.Context, log *log, n int) (name string, data []byte, err error) {
	if _, err := exec.LookPath("journalctl"); err == nil && log.Journalctl != "" {
		data, err := journalctlN(ctx, log.Journalctl, n)
		return log.Journalctl, data, err
	} else if log.Path != "" {
		data, err := tailN(ctx, log.Path, n)
		return filepath.Base(log.Path), data, err
	}

	return "", nil, fmt.Errorf("unable to get log: %v", log)
}

// journalctlN reads last n lines from systemd unit u.
func journalctlN(ctx context.Context, u string, n int) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "journalctl", "-n", strconv.Itoa(n), "-u", u)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	b, err := cmd.Output()
	if err != nil {
		return b, fmt.Errorf("JournalctlN(%s, %d): %s: %s", u, n, err, stderr.String())
	}
	return b, nil
}

// tailN reads last n lines from log file at given path.
func tailN(ctx context.Context, path string, n int) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "/usr/bin/tail", "-n", strconv.Itoa(n), path)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	b, err := cmd.Output()
	if err != nil {
		return b, fmt.Errorf("TailN(%s, %d): %s: %s", path, n, err, stderr.String())
	}
	return b, nil
}
