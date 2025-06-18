packer {
  required_plugins {
    digitalocean = {
      version = "=1.4.0"
      source  = "github.com/digitalocean/digitalocean"
    }
    ansible = {
      version = "~> 1"
      source  = "github.com/hashicorp/ansible"
    }
  }
}

variable "do_api_token" {
  type        = string
  description = "DigitalOcean API Token"
  sensitive   = true
}

source "digitalocean" "pmm-ovf" {
  api_token     = var.do_api_token
  droplet_name  = "pmm-ovf-agent-builder"
  image         = "centos-stream-9-x64"
  region        = "ams3"
  size          = "s-4vcpu-8gb-intel"
  ssh_username  = "root"
  snapshot_name = "pmm-ovf-agent"
}

build {
  name    = "jenkins-farm"
  sources = ["source.digitalocean.pmm-ovf"]

  provisioner "ansible" {
    use_proxy        = false  # otherwise it fails to connect ansible to the host
    ansible_env_vars = ["ANSIBLE_NOCOLOR=True", "CLOUD_PROVIDER=do"]
    extra_arguments  = ["-v", "-e", "cloud_provider=do"] # -vvv for more verbose output
    max_retries      = 1
    playbook_file    = "./ansible/agent-do.yml"
  }
}
