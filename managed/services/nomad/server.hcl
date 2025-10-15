log_level = "INFO"

disable_update_check = true
data_dir = "/srv/nomad/"
bind_addr = "0.0.0.0" # the default
region = "global" # shall be the same as in cmd: nomad tls cert create ... -region <region>
datacenter = "PMM Deployment"
name = "PMM Server"

ports {
  # Bind HTTP interface to this port
  http = "4646"
  # Bind RPC interface to this port
  rpc  = "4647"
}

advertise {
  # Shall be reachable by Nomad CLI.
  # Do we need to access Nomad Server from outside PMM Server?
  # http = "127.0.0.1"

  # Shall be reachable by Nomad Client nodes.
  # PMM Server public address shall be defined.
  rpc = "{{ .Node.Address }}"
}

server {
  enabled          = true
  bootstrap_expect = 1
}

tls {
  # encrypt HTTP traffic to Nomad UI.
  http = true
  # encrypt Nomad Server <-> Nomad Client communication channel
  rpc  = true
  ca_file   = "/srv/nomad/certs/nomad-agent-ca.pem"
  cert_file = "/srv/nomad/certs/global-server-{{ .Node.Address }}.pem"
  key_file  = "/srv/nomad/certs/global-server-{{ .Node.Address }}-key.pem"

  verify_server_hostname = true
}

# Enabled plugins
plugin "raw_exec" {
  config {
    enabled = true
  }
}

telemetry {
  collection_interval = "10s"
  disable_hostname = true
  prometheus_metrics = true
  publish_allocation_metrics = true
  publish_node_metrics = true
}