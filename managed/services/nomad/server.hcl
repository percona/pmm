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
  rpc = "<PMM Server external address>"
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
  ca_file   = "/srv/nomad/certs/<PMM Server external address>-agent-ca.pem"
  cert_file = "/srv/nomad/certs/global-server-<PMM Server external address>.pem"
  key_file  = "/srv/nomad/certs/global-server-<PMM Server external address>-key.pem"

  verify_server_hostname = true
}

# Run local Nomad Agent on PMM Server
client {
  enabled = true

  cpu_total_compute = 1000
  servers = ["127.0.0.1:4647"]

  # disable Docker plugin
  options = {
    "driver.denylist" = "docker,qemu,java,exec"
    "driver.allowlist" = "raw_exec" # Only this task driver can be used in unpriviliged docker container
  }

  meta {
    pmm-server = "1"
  }
}

# Enabled plugins
plugin "raw_exec" {
  config {
    enabled = true
  }
}