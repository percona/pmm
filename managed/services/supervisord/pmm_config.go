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
	"bytes"
	"os"
	"text/template"

	"github.com/pkg/errors"
)

// SavePMMConfig renders and saves pmm config.
func SavePMMConfig(params map[string]any) error {
	cfg, err := marshalConfig(params)
	if err != nil {
		return err
	}
	if err := saveConfig(pmmConfig, cfg); err != nil {
		return errors.Wrapf(err, "failed to save pmm config")
	}
	return nil
}

func marshalConfig(params map[string]any) ([]byte, error) {
	var buf bytes.Buffer
	if err := pmmTemplate.Execute(&buf, params); err != nil {
		return nil, errors.Wrapf(err, "failed to render pmm template")
	}
	return buf.Bytes(), nil
}

// saveConfig saves config in default directory.
func saveConfig(path string, cfg []byte) (err error) {
	// read existing content
	oldCfg, err := os.ReadFile(path) //nolint:gosec
	if os.IsNotExist(err) {
		err = nil
	}
	if err != nil {
		return errors.WithStack(err)
	}

	// compare with new config
	if bytes.Equal(cfg, oldCfg) {
		// nothing to change
		return nil
	}

	// restore old content
	defer func() {
		if err == nil {
			return
		}
		if resErr := os.WriteFile(path, oldCfg, 0o644); resErr != nil { //nolint:gosec
			err = errors.Wrap(err, errors.Wrap(resErr, "failed to restore config").Error())
		}
	}()

	if err = os.WriteFile(path, cfg, 0o644); err != nil { //nolint:gosec
		err = errors.Wrap(err, "failed to write new config")
	}
	return
}

var pmmTemplate = template.Must(template.New("").Option("missingkey=error").Parse(`[unix_http_server]
chmod = 0700
username = dummy
password = dummy

[supervisorctl]
username = dummy
password = dummy

[program:pmm-init]
command = /usr/bin/ansible-playbook /opt/ansible/pmm-docker/init.yml
directory = /
autorestart = unexpected
priority=-1
exitcodes = 0
autostart = true
startretries = 3
startsecs = 1
stopsignal = TERM
stopwaitsecs = 300
stdout_logfile = /srv/logs/pmm-init.log
stdout_logfile_maxbytes = 20MB
stdout_logfile_backups = 3
redirect_stderr = true
environment = ANSIBLE_CONFIG="/opt/ansible/ansible.cfg"
{{- if not .DisableInternalDB }}

[program:postgresql]
priority = 1
command =
    /usr/pgsql-14/bin/postgres
        -D /srv/postgres14
        -c shared_preload_libraries=pg_stat_statements
        -c pg_stat_statements.max=10000
        -c pg_stat_statements.track=all
        -c pg_stat_statements.save=off
        -c logging_collector=off
autorestart = true
autostart = true
startretries = 10
startsecs = 1
stopsignal = INT  ; Fast Shutdown mode
stopwaitsecs = 300
; postgresql.conf contains settings to log to stdout,
; so we delegate logfile management to supervisord
stdout_logfile = /srv/logs/postgresql14.log
stdout_logfile_maxbytes = 30MB
stdout_logfile_backups = 2
redirect_stderr = true
{{- end }}
{{- if not .DisableInternalClickhouse }}

[program:clickhouse]
priority = 2
command = /usr/bin/clickhouse-server --config-file=/etc/clickhouse-server/config.xml
autorestart = true
autostart = true
startretries = 10
startsecs = 1
stopsignal = TERM
stopwaitsecs = 300
; config.xml contains settings to log to stdout (console),
; so we delegate logfile managemenet to supervisord
stdout_logfile = /srv/logs/clickhouse-server.log
stdout_logfile_maxbytes = 50MB
stdout_logfile_backups = 2
redirect_stderr = true
{{- end }}

[program:nginx]
priority = 4
command = nginx
autorestart = true
autostart = true
startretries = 10
startsecs = 1
stopsignal = TERM
stopwaitsecs = 10
stdout_logfile = /srv/logs/nginx.log
stdout_logfile_maxbytes = 50MB
stdout_logfile_backups = 2
redirect_stderr = true

[program:pmm-managed]
priority = 14
command =
    /usr/sbin/pmm-managed
        --victoriametrics-config=/etc/victoriametrics-promscrape.yml
        --supervisord-config-dir=/etc/supervisord.d
autorestart = true
autostart = true
startretries = 1000
startsecs = 1
stopsignal = TERM
stopwaitsecs = 300
stdout_logfile = /srv/logs/pmm-managed.log
stdout_logfile_maxbytes = 50MB
stdout_logfile_backups = 2
redirect_stderr = true

[program:pmm-agent]
priority = 15
command = /usr/sbin/pmm-agent --config-file=/usr/local/percona/pmm/config/pmm-agent.yaml --paths-tempdir=/srv/pmm-agent/tmp --paths-nomad-data-dir=/srv/nomad/data
autorestart = true
autostart = false
startretries = 1000
startsecs = 1
stopsignal = TERM
stopwaitsecs = 10
stdout_logfile = /srv/logs/pmm-agent.log
stdout_logfile_maxbytes = 50MB
stdout_logfile_backups = 2
redirect_stderr = true
`))
