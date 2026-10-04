# pmm Jenkins agent AMIs (Docker Farm labels agent-amd64, agent-arm64, cli, docker,
# and the pmm-staging VMs). Baked by the agent-ami-factory GitHub Actions workflow
# over AWS Session Manager (no inbound SSH, no static AWS keys, OIDC role), then
# boot-tested by smoke/smoke.pkr.hcl and promoted by tag. Manual bake in an
# emergency: see README.md.
#
# Tag contract (iit-billing-tag is the selector every consumer filters on):
#   builder and smoke instances   pmm-agent-ami-factory      never selected by a consumer
#   freshly baked candidate        pmm-worker-3-candidate     never selected by a consumer
#   promoted live pair             pmm-worker-3               what the Docker Farm cloud and
#                                                             the staging pipelines select
#   demoted predecessor            pmm-worker-3-previous      rollback target
#   env=test uses pmm-worker-3-test-candidate / pmm-worker-3-test / pmm-worker-3-test-previous
# The AMI inherits run_tags at CreateImage and gets the candidate tags right after, so at
# no point does an untested image carry the live value.

packer {
  required_plugins {
    amazon = {
      version = "1.8.1" # exact pin, supply chain
      source  = "github.com/hashicorp/amazon"
    }
  }
}

variable "region" {
  type    = string
  default = "us-east-2"
}

variable "env" {
  type        = string
  default     = "prod"
  description = "prod = candidates the promote step may make live. test = isolated tag family no consumer selects."
  validation {
    condition     = contains(["prod", "test"], var.env)
    error_message = "Value of env must be prod or test."
  }
}

variable "factory_run" {
  type        = string
  default     = "manual"
  description = "Run identity stamped on instances, volumes, AMIs and snapshots (the workflow passes github.run_id). The cleanup step terminates instances by this tag."
}

variable "ansible_core_version" {
  type        = string
  default     = "2.20.9"
  description = "ansible-core installed on the builder to run the playbook locally. The workflow passes its own pin so the check job and the bake agree."
}

variable "builder_instance_profile" {
  type        = string
  default     = "pmm-agent-ami-builder-ssm"
  description = "Instance profile with AmazonSSMManagedInstanceCore so Session Manager can reach the builder (percona-cd-platform terraform)."
}

variable "builder_security_group_name" {
  type        = string
  default     = "pmm-agent-ami-factory-builder"
  description = "Pre-created egress-only security group in the pmm VPC. Supplying it disables Packer's temporary SG, so the OIDC role needs no SG-create rights."
}

locals {
  ts            = formatdate("YYYYMMDD-hhmmss", timestamp())
  name_prefix   = var.env == "test" ? "TEST-" : ""
  builder_tag   = "pmm-agent-ami-factory"
  candidate_tag = var.env == "test" ? "pmm-worker-3-test-candidate" : "pmm-worker-3-candidate"

  # The AlmaLinux base image ships no SSM agent, and session_manager needs the
  # agent to connect in the first place, so user_data installs it at boot.
  # Regional rpm first, global fallback. The playbook installs it again into
  # the baked image so the smoke test and the agents also register.
  ssm_bootstrap = <<-EOT
    #!/bin/bash
    case "$(uname -m)" in aarch64) a=arm64 ;; *) a=amd64 ;; esac
    dnf install -y https://s3.${var.region}.amazonaws.com/amazon-ssm-${var.region}/latest/linux_$a/amazon-ssm-agent.rpm \
      || dnf install -y https://s3.amazonaws.com/ec2-downloads-windows/SSMAgent/latest/linux_$a/amazon-ssm-agent.rpm
    systemctl enable --now amazon-ssm-agent
  EOT

  run_tags = {
    Name            = "packer-pmm-agent-ami-factory"
    iit-billing-tag = local.builder_tag
    factory_run     = var.factory_run
    factory_env     = var.env
  }
}

source "amazon-ebs" "amd64-agent" {
  region        = var.region
  ami_name      = "${local.name_prefix}Jenkins Agent AL amd64 ${local.ts}"
  instance_type = "c6i.xlarge" # fixed performance, a bake drains burst credits

  source_ami_filter {
    filters = {
      name                = "AlmaLinux OS 9*"
      root-device-type    = "ebs"
      virtualization-type = "hvm"
      architecture        = "x86_64"
    }
    most_recent = true
    owners      = ["764336703387"]
  }

  # Session Manager tunnel, no inbound SG rule, no public key on the image.
  communicator              = "ssh"
  ssh_username              = "ec2-user"
  ssh_interface             = "session_manager"
  ssh_timeout               = "12m"
  ssh_clear_authorized_keys = true
  iam_instance_profile      = var.builder_instance_profile
  skip_profile_validation   = true # the OIDC role has PassRole only, not iam:GetInstanceProfile
  user_data                 = local.ssm_bootstrap

  # Candidates age out on their own if never promoted. The promote step clears
  # this on the live pair and sets a shorter date on demoted predecessors.
  deprecate_at = timeadd(timestamp(), "1440h")

  launch_block_device_mappings {
    device_name           = "/dev/sda1"
    volume_size           = 60
    volume_type           = "gp3"
    delete_on_termination = true
  }

  vpc_filter {
    filters = {
      "tag:Name" : "jenkins-pmm-amzn2"
    }
  }
  # Any of the pmm VPC's subnets (B and C, two availability zones), so one
  # zone short of c6i or c7g capacity does not fail the bake.
  subnet_filter {
    filters = {
      "tag:Name" : "jenkins-pmm-amzn2-*"
    }
    random = true
  }
  security_group_filter {
    filters = {
      "group-name" = var.builder_security_group_name
    }
  }

  run_tags        = local.run_tags
  run_volume_tags = local.run_tags

  tags = {
    Name            = "${local.name_prefix}Jenkins Agent AL amd64 ${local.ts}"
    arch            = "x86_64"
    role            = "candidate"
    factory_env     = var.env
    factory_run     = var.factory_run
    base_ami        = "{{ .SourceAMI }}"
    base_name       = "{{ .SourceAMIName }}"
    iit-billing-tag = local.candidate_tag
  }
  snapshot_tags = {
    Name            = "${local.name_prefix}Jenkins Agent AL amd64 ${local.ts}"
    role            = "candidate"
    factory_env     = var.env
    factory_run     = var.factory_run
    iit-billing-tag = local.candidate_tag
  }
}

source "amazon-ebs" "arm64-agent" {
  region        = var.region
  ami_name      = "${local.name_prefix}Jenkins Agent AL arm64 ${local.ts}"
  instance_type = "c7g.xlarge" # fixed performance, a bake drains burst credits

  source_ami_filter {
    filters = {
      name                = "AlmaLinux OS 9*"
      root-device-type    = "ebs"
      virtualization-type = "hvm"
      architecture        = "arm64"
    }
    most_recent = true
    owners      = ["764336703387"]
  }

  communicator              = "ssh"
  ssh_username              = "ec2-user"
  ssh_interface             = "session_manager"
  ssh_timeout               = "12m"
  ssh_clear_authorized_keys = true
  iam_instance_profile      = var.builder_instance_profile
  skip_profile_validation   = true
  user_data                 = local.ssm_bootstrap

  deprecate_at = timeadd(timestamp(), "1440h")

  launch_block_device_mappings {
    device_name           = "/dev/sda1"
    volume_size           = 60
    volume_type           = "gp3"
    delete_on_termination = true
  }

  vpc_filter {
    filters = {
      "tag:Name" : "jenkins-pmm-amzn2"
    }
  }
  # Any of the pmm VPC's subnets (B and C, two availability zones), so one
  # zone short of c6i or c7g capacity does not fail the bake.
  subnet_filter {
    filters = {
      "tag:Name" : "jenkins-pmm-amzn2-*"
    }
    random = true
  }
  security_group_filter {
    filters = {
      "group-name" = var.builder_security_group_name
    }
  }

  run_tags        = local.run_tags
  run_volume_tags = local.run_tags

  tags = {
    Name            = "${local.name_prefix}Jenkins Agent AL arm64 ${local.ts}"
    arch            = "arm64"
    role            = "candidate"
    factory_env     = var.env
    factory_run     = var.factory_run
    base_ami        = "{{ .SourceAMI }}"
    base_name       = "{{ .SourceAMIName }}"
    iit-billing-tag = local.candidate_tag
  }
  snapshot_tags = {
    Name            = "${local.name_prefix}Jenkins Agent AL arm64 ${local.ts}"
    role            = "candidate"
    factory_env     = var.env
    factory_run     = var.factory_run
    iit-billing-tag = local.candidate_tag
  }
}

build {
  name = "jenkins-farm"
  sources = [
    "source.amazon-ebs.amd64-agent",
    "source.amazon-ebs.arm64-agent",
  ]

  # The SSM agent rpm starts the agent from its post-install scriptlet, so the
  # session opens while the user_data dnf transaction still holds the rpm lock
  # and the playbook's first dnf task cannot import the repository key. Wait
  # for cloud-init, which returns once the user_data script has finished.
  provisioner "shell" {
    timeout = "10m"
    inline = [
      # cloud-init exits 2 for a finished boot with recoverable errors ("degraded done"), so the status text is what counts, not the exit code.
      "status=$(sudo cloud-init status --wait 2>/dev/null || true); case \"$${status}\" in *done*|*disabled*) ;; *) echo \"cloud-init: $${status}\"; sudo cloud-init status --long; exit 1 ;; esac",
    ]
  }

  # The playbook runs ON the builder, not on the runner. Ansible over Packer's
  # SSH proxy adapter tunnels a second SSH connection through the Session
  # Manager port forward, and that connection loses a task's completion and
  # never recovers: the playbook goes silent, twenty minutes of silence trip the
  # Session Manager idle timeout, the tunnel is torn down and the build hangs.
  # A file upload plus a shell provisioner ride Packer's own communicator, which
  # streams output continuously and reconnects on its own, and each task runs
  # without a network round trip.
  provisioner "file" {
    source      = "./ansible"
    destination = "/tmp"
  }

  provisioner "shell" {
    timeout        = "60m"
    inline_shebang = "/bin/bash"
    environment_vars = [
      "ANSIBLE_CORE_VERSION=${var.ansible_core_version}",
      "ANSIBLE_NOCOLOR=True",
      "ANSIBLE_COLLECTIONS_PATH=/tmp/factory-ansible/collections",
      "PYTHONUNBUFFERED=1",
    ]
    inline = [
      "set -euo pipefail",
      "test -f /tmp/ansible/agent-aws.yml",
      # The controller needs a newer python than the AlmaLinux 9 default, the
      # modules keep running under the system python. The builder IS the image,
      # so the controller lives in a throwaway venv under /tmp and its
      # collections next to it: nothing of it may shadow the AppStream
      # ansible-core the playbook installs for the jobs.
      "sudo dnf install -y -q python3.12",
      "python3.12 -m venv /tmp/factory-ansible",
      "/tmp/factory-ansible/bin/pip install --quiet \"ansible-core==$${ANSIBLE_CORE_VERSION}\"",
      # No pipe here: `head` closing the pipe early makes ansible-playbook die
      # of a broken pipe, and under pipefail that fails the build.
      "/tmp/factory-ansible/bin/ansible-playbook --version > /tmp/ansible-version.txt",
      "grep -F \"core $${ANSIBLE_CORE_VERSION}\" /tmp/ansible-version.txt",
      "cd /tmp/ansible && /tmp/factory-ansible/bin/ansible-galaxy collection install -r requirements.yml",
      # The playbook targets `hosts: default`, and the modules must keep running
      # under the system python: it owns the dnf, selinux and libselinux
      # bindings, and installing a newer python for the controller would
      # otherwise win interpreter discovery. An inventory variable leaves the
      # docker task's own interpreter override in force.
      "echo 'default ansible_connection=local ansible_python_interpreter=/usr/bin/python3' > /tmp/ansible/local-inventory",
      "cd /tmp/ansible && /tmp/factory-ansible/bin/ansible-playbook -i local-inventory agent-aws.yml",
      # Leave no factory residue and no builder identity in the image: the
      # controller venv and python go, cloud-init forgets this instance so the
      # first boot of every agent re-runs it (fresh SSH host keys, launch key),
      # and an empty machine-id makes systemd mint a new one per agent.
      "cd / && sudo rm -rf /tmp/factory-ansible /tmp/ansible /tmp/ansible-version.txt ~/.ansible/tmp",
      "sudo dnf remove -y -q python3.12",
      "sudo cloud-init clean --logs",
      "sudo truncate -s 0 /etc/machine-id",
    ]
  }

  post-processor "manifest" {
    output     = "manifest.json"
    strip_path = true
  }
}
