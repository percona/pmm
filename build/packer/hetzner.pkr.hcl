packer {
  required_plugins {
    hcloud = {
      version = ">= 1.6.1"
      source  = "github.com/hetznercloud/hcloud"
    }
    ansible = {
      version = "~> 1"
      source  = "github.com/hashicorp/ansible"
    }
  }
}

variable "hcloud_token" {
  type        = string
  description = "Hetzner Cloud API Token"
  sensitive   = true
}

variable "ssh_key_name" {
  type        = string
  description = "Name of the SSH key in Hetzner Cloud (optional - leave empty to use temporary key)"
  default     = ""
}

variable "volume_size" {
  type        = number
  description = "Size of the server volume in GB"
  default     = 60
}

locals {
  timestamp  = formatdate("YYYYMMDD-HHmmss", timestamp())
  uuid_short = substr(uuidv4(), 0, 8)
}

source "hcloud" "jenkins-agent" {
  token         = var.hcloud_token
  image         = "rocky-9" # Using Rocky Linux 9 as Oracle Linux not available on Hetzner
  location      = "fsn1"    # Falkenstein, Germany - or use "hel1" (Helsinki), "nbg1" (Nuremberg)
  server_type   = "ccx23"   # 4 dedicated vCPUs, 16GB RAM - Intel-based (matches AWS t3.xlarge)
  ssh_username  = "root"
  snapshot_name = "Docker Agent v3 Hetzner"
  snapshot_labels = {
    type                    = "jenkins-agent"
    arch                    = "x86_64"
    iit-billing-tag         = "pmm-worker"
    "jenkins.io/cloud-name" = "pmm-htz"
  }
  server_name = "packer-pmm-x86-${local.uuid_short}"
  ssh_keys    = var.ssh_key_name == "" ? [] : [var.ssh_key_name]
  server_labels = {
    iit-billing-tag = "pmm-worker"
  }
}

source "hcloud" "jenkins-agent-arm" {
  token         = var.hcloud_token
  image         = "rocky-9" # Using Rocky Linux 9 as Oracle Linux not available on Hetzner
  location      = "fsn1"
  server_type   = "cax31" # 8 vCPUs ARM, 16GB RAM - Best ARM option (AWS t4g.xlarge has 4 vCPUs)
  ssh_username  = "root"
  snapshot_name = "Docker Agent ARM v3 Hetzner"
  snapshot_labels = {
    type                    = "jenkins-agent"
    arch                    = "arm64"
    iit-billing-tag         = "pmm-worker"
    "jenkins.io/cloud-name" = "pmm-htz"
  }
  server_name = "packer-pmm-arm-${local.uuid_short}"
  ssh_keys    = var.ssh_key_name == "" ? [] : [var.ssh_key_name]
  server_labels = {
    iit-billing-tag = "pmm-worker"
  }
}

build {
  name = "jenkins-farm"
  sources = [
    "source.hcloud.jenkins-agent",
    "source.hcloud.jenkins-agent-arm"
  ]

  provisioner "ansible" {
    use_proxy        = false
    user             = "root"
    ansible_env_vars = ["ANSIBLE_NOCOLOR=True"]
    extra_arguments = [
      "--ssh-extra-args", "-o HostKeyAlgorithms=+ssh-rsa -o StrictHostKeyChecking=no -o ForwardAgent=yes -o UserKnownHostsFile=/dev/null", "-vvv"
    ]
    playbook_file = "./ansible/agent-hetzner.yml"
  }
}
