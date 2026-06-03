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

package supervisord

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/managed/utils/envvars"
)

// otelcolConfig is the path to the OpenTelemetry Collector configuration file.
const otelcolConfig = "/etc/otelcol/config.yaml"

// SaveOtelcolConfig renders and saves the OpenTelemetry Collector configuration.
// The config does not carry a TTL: tables are owned by pmm-managed (managed/services/clickhouse, with
// create_schema=false), so the exporter's ttl option is a no-op; retention is enforced by
// ALTER TABLE ... MODIFY TTL in pmm-managed.
func SaveOtelcolConfig() error {
	cfg, err := marshalOtelcolConfig()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(otelcolConfig), 0o755); err != nil { //nolint:gosec,mnd
		return errors.Wrapf(err, "failed to create otelcol config directory")
	}
	if err := saveConfig(otelcolConfig, cfg); err != nil {
		return errors.Wrapf(err, "failed to save otelcol config")
	}
	logrus.Info("otelcol config.yaml has been updated.")
	return nil
}

func marshalOtelcolConfig() ([]byte, error) {
	clickhouseAddr := envvars.GetEnv("PMM_CLICKHOUSE_ADDR", defaultClickhouseAddr)
	clickhouseAddrPair := strings.SplitN(clickhouseAddr, ":", 2) //nolint:mnd
	if len(clickhouseAddrPair) != 2 {                            //nolint:mnd
		return nil, errors.Errorf("unexpected PMM_CLICKHOUSE_ADDR format: %q", clickhouseAddr)
	}

	params := map[string]any{
		"ClickhouseHost":     clickhouseAddrPair[0],
		"ClickhousePort":     clickhouseAddrPair[1],
		"ClickhouseDatabase": envvars.GetEnv("PMM_CLICKHOUSE_DATABASE", defaultClickhouseDatabase),
		"ClickhouseUser":     envvars.GetEnv("PMM_CLICKHOUSE_USER", defaultClickhouseUser),
		"ClickhousePassword": envvars.GetEnv("PMM_CLICKHOUSE_PASSWORD", defaultClickhousePassword),
	}

	var buf bytes.Buffer
	if err := otelcolTemplate.Execute(&buf, params); err != nil {
		return nil, errors.Wrapf(err, "failed to render otelcol template")
	}
	return buf.Bytes(), nil
}

// otelcolTemplate renders /etc/otelcol/config.yaml.
// The filelog receiver captures all server component logs from /srv/logs/*.log without touching any
// component; the OTLP receivers are loopback-only and fed by pmm-managed (client/DB logs). Both
// pipelines write the OTel schema into the existing ClickHouse via the clickhouseexporter, which does
// NOT create the schema (tables are owned by pmm-managed's managed/services/clickhouse migrations).
var otelcolTemplate = template.Must(template.New("otelcol").Option("missingkey=error").Parse(`receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 127.0.0.1:4317
      http:
        endpoint: 127.0.0.1:4318
  filelog:
    include:
      - /srv/logs/*.log
    exclude:
      - /srv/logs/otelcol.log
    start_at: end
    include_file_name: true
    include_file_path: false
    operators:
      - type: add
        field: resource["pmm.source"]
        value: server
      - type: regex_parser
        parse_from: attributes["log.file.name"]
        regex: '^(?P<service_name>[^.]+)'
      - type: move
        from: attributes.service_name
        to: resource["service.name"]
      - type: remove
        field: attributes["log.file.name"]

processors:
  memory_limiter:
    check_interval: 2s
    limit_percentage: 75
    spike_limit_percentage: 25
  batch:
    timeout: 5s
    send_batch_size: 1000

exporters:
  clickhouse:
    endpoint: "tcp://{{ .ClickhouseHost }}:{{ .ClickhousePort }}?dial_timeout=10s&compress=lz4"
    database: "{{ .ClickhouseDatabase }}"
    username: "{{ .ClickhouseUser }}"
    password: "{{ .ClickhousePassword }}"
    logs_table_name: logs
    traces_table_name: traces
    create_schema: false
    timeout: 10s
    sending_queue:
      queue_size: 1000
    retry_on_failure:
      enabled: true
      initial_interval: 5s
      max_interval: 30s

service:
  telemetry:
    logs:
      level: warn
  pipelines:
    logs:
      receivers: [otlp, filelog]
      processors: [memory_limiter, batch]
      exporters: [clickhouse]
    traces:
      receivers: [otlp]
      processors: [memory_limiter, batch]
      exporters: [clickhouse]
`))
