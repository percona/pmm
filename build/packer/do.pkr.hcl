packer {
  required_plugins {
    digitalocean = {
      version = "=1.0.4"
      source  = "github.com/digitalocean/digitalocean"
    }
  }
}

source "digitalocean" "pmm-ovf" {
  droplet_name  = "pmm-ovf-agent-builder"
  image         = "centos-7-x64"
  region        = "ams3"
  size          = "s-4vcpu-8gb-intel"
  ssh_username  = "root"
  snapshot_name = "pmm-ovf-agent"
}

build {
  name    = "jenkins-farm"
  sources = ["source.digitalocean.pmm-ovf"]

  provisioner "ansible" {
    use_proxy       = false  # otherwise it fails to connect ansible to the host
    extra_arguments = ["-v"] # -vvv for more verbose output
    max_retries     = 1
    playbook_file   = "./ansible/agent.yml"
  }
}
