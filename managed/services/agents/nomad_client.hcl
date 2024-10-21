log_level = "{{ .LogLevel }}"

disable_update_check = true
data_dir = "{{ .DataDir }}" # it shall be persistent
region = "global"
datacenter = "PMM Deployment"
name = "PMM Agent {{ .NodeName }}"

ui {
  enabled = false
}

addresses {
  http = "127.0.0.1"
  rpc = "127.0.0.1"
}

advertise {
  # 127.0.0.1 is not applicable here
  http = "{{ .NodeAddress }}" # filled by PMM Server
  rpc = "{{ .NodeAddress }}"  # filled by PMM Server
}

client {
  enabled = true
  cpu_total_compute = 1000

  servers = ["{{ .PMMServerAddress }}"] # filled by PMM Server

  # disable Docker plugin
  options = {
    "driver.denylist" = "docker,qemu,java,exec"
    "driver.allowlist" = "raw_exec"
  }

  # optional lables set to Nomad Client, may be the same as for PMM Agent.
  meta {
    node_id = "{{ .NodeID }}"
    node_name = "{{ .NodeName }}"
    pmm-agent = "1"
  }
}

server {
  enabled = false
}

tls {
  http = true
  rpc  = true
  ca_file   = "{{ .CaFile }}" # filled by PMM Agent
  cert_file = "{{ .CertFile }}" # filled by PMM Agent
  key_file  = "{{ .KeyFile }}" # filled by PMM Agent

  verify_server_hostname = true
}

# Enabled plugins
plugin "raw_exec" {
  config {
      enabled = true
  }
}