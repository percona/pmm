#!/bin/bash -e

# Add grafana repository
cat <<EOF > /etc/yum.repos.d/grafana.repo
[grafana]
name=grafana
baseurl=https://rpm.grafana.com
repo_gpgcheck=1
enabled=0
gpgcheck=1
gpgkey=https://rpm.grafana.com/gpg.key
sslverify=1
sslcacert=/etc/pki/tls/certs/ca-bundle.crt
EOF

# Install loki and promtail, which also creates dedicated users
dnf install -y --disablerepo="*" --enablerepo=grafana loki promtail

# Add promtail and loki users to pmm user group
usermod -a -G pmm promtail
usermod -a -G pmm loki

mkdir -p /srv/loki
chown pmm:pmm /srv/loki

cat <<EOF > /etc/supervisord.d/loki.ini
[program:loki]
priority = 20
command =
        /usr/bin/loki
                -config.file /etc/loki/config.yml
user = pmm
autorestart = true
autostart = true
startretries = 1000
startsecs = 3
stopsignal = TERM
stopwaitsecs = 10
stdout_logfile = /srv/logs/loki.log
stdout_logfile_maxbytes = 10MB
stdout_logfile_backups = 3
redirect_stderr = true


[program:promtail]
priority = 21
command =
        /usr/bin/promtail
                -config.file /srv/loki/promtail.yml
user = pmm
autorestart = true
autostart = true
startretries = 1000
startsecs = 3
stopsignal = TERM
stopwaitsecs = 10
stdout_logfile = /srv/logs/promtail.log
stdout_logfile_maxbytes = 10MB
stdout_logfile_backups = 3
redirect_stderr = true
EOF

cat <<EOF > /etc/loki/config.yml
auth_enabled: false

server:
  http_listen_port: 3100
  grpc_listen_port: 9096
  log_level: debug
  grpc_server_max_concurrent_streams: 1000

common:
  instance_addr: 127.0.0.1
  path_prefix: /srv/loki
  storage:
    filesystem:
      chunks_directory: /srv/loki/chunks
      rules_directory: /srv/loki/rules
  replication_factor: 1
  ring:
    kvstore:
      store: inmemory

ingester_rf1:
  enabled: false

query_range:
  results_cache:
    cache:
      embedded_cache:
        enabled: true
        max_size_mb: 100

schema_config:
  configs:
    - from: 2020-10-24
      store: tsdb
      object_store: filesystem
      schema: v13
      index:
        prefix: index_
        period: 24h

pattern_ingester:
  enabled: true
  metric_aggregation:
    enabled: true
    loki_address: 127.0.0.1:3100

ruler:
  alertmanager_url: http://127.0.0.1:9093

frontend:
  encoding: protobuf

analytics:
  reporting_enabled: false
EOF

cat <<EOF > /srv/loki/promtail.yml
# Important: too much scraping during init process can overload the system.
# https://github.com/grafana/loki/issues/11398

server:
  http_listen_port: 9080
  grpc_listen_port: 0

positions:
  filename: /tmp/positions.yaml

clients:
- url: http://127.0.0.1:3100/loki/api/v1/push

scrape_configs:
- job_name: nginx
  static_configs:
  - targets:
      - 127.0.0.1
    labels:
      job: nginx
      __path__: /srv/logs/nginx.log

- job_name: grafana
  static_configs:
  - targets:
      - 127.0.0.1
    labels:
      job: grafana
      __path__: /srv/logs/grafana.log

- job_name: pmm-agent
  static_configs:
  - targets:
      - 127.0.0.1
    labels:
      job: pmm-agent
      __path__: /srv/logs/pmm-agent.log
      node_name: pmm-server

- job_name: pmm-managed
  static_configs:
  - targets:
      - 127.0.0.1
    labels:
      job: pmm-managed
      __path__: /srv/logs/pmm-managed.log

- job_name: qan
  static_configs:
  - targets:
      - 127.0.0.1
    labels:
      job: qan
      __path__: /srv/logs/qani-api2.log

- job_name: victoriametrics
  static_configs:
  - targets:
      - 127.0.0.1
    labels:
      job: victoriametrcis
      __path__: /srv/logs/victoriametrics.log

- job_name: clickhouse
  static_configs:
  - targets:
      - 127.0.0.1
    labels:
      job: clickhouse
      __path__: /srv/logs/clickhouse-server.log

- job_name: supervisor
  static_configs:
  - targets:
      - 127.0.0.1
    labels:
      job: supervisor
      __path__: /srv/logs/supervisord.log
EOF

cat <<EOF > /usr/share/grafana/conf/provisioning/datasources/loki.yml
apiVersion: 1
datasources:
  - name: Loki
    type: loki
    uid: loki
    access: proxy
    url: http://127.0.0.1:3100
EOF

# Change ownership of all files we added
chown pmm:pmm /etc/supervisord.d/loki.ini
chown pmm:pmm /etc/loki/config.yml
chown pmm:pmm /srv/loki/promtail.yml
chown pmm:pmm /usr/share/grafana/conf/provisioning/datasources/loki.yml
