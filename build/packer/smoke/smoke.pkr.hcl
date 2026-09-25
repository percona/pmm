# Fresh-boot smoke test for a pmm agent AMI candidate. Boots the candidate with
# skip_create_ami (no new image), connects over Session Manager, and asserts what
# the Jenkins EC2 plugin and the staging pipelines rely on: Java 21 as the
# only JVM (remoting floor since Jenkins 2.555.1, Java 17 removed), a working
# Docker daemon usable by ec2-user, the SSM agent registered, cloud-init
# finished. This template only validates. The workflow keeps a failed candidate
# tagged role=smoke-failed for a look (the janitor removes it a week later) and
# promotes a passing one in a separate, retried step.
#
#   packer init smoke.pkr.hcl && packer build -var candidate_ami=ami-... -var arch=arm64 smoke.pkr.hcl

packer {
  required_plugins {
    amazon = {
      version = "1.8.1" # exact pin, matches aws.pkr.hcl
      source  = "github.com/hashicorp/amazon"
    }
  }
}

variable "candidate_ami" {
  type    = string
  default = "ami-00000000000000000" # placeholder so `packer validate` works without a bake
}

variable "arch" {
  type = string # x86_64 | arm64
  validation {
    condition     = contains(["x86_64", "arm64"], var.arch)
    error_message = "Value of arch must be x86_64 or arm64."
  }
}

variable "region" {
  type    = string
  default = "us-east-2"
}

variable "env" {
  type    = string
  default = "prod"
}

variable "factory_run" {
  type    = string
  default = "manual"
}

variable "builder_instance_profile" {
  type    = string
  default = "pmm-agent-ami-builder-ssm"
}

variable "builder_security_group_name" {
  type    = string
  default = "pmm-agent-ami-factory-builder"
}

locals {
  # t4g.small is the smallest production shape (the cli label), so the arm64
  # smoke proves the image boots and starts a JVM there. x86_64 uses t3.large.
  instance_type = var.arch == "arm64" ? "t4g.small" : "t3.large"
  uname_arch    = var.arch == "arm64" ? "aarch64" : "x86_64"
  run_tags = {
    Name            = "pmm-agent-ami-smoke"
    iit-billing-tag = "pmm-agent-ami-factory"
    factory_run     = var.factory_run
    factory_env     = var.env
  }
}

source "amazon-ebs" "smoke" {
  region        = var.region
  instance_type = local.instance_type
  source_ami    = var.candidate_ami
  # ami_name is required by the builder even though nothing is created.
  ami_name                = "pmm-agent-smoke-noop-${var.arch}"
  skip_create_ami         = true
  communicator            = "ssh"
  ssh_username            = "ec2-user"
  ssh_interface           = "session_manager"
  ssh_timeout             = "10m"
  iam_instance_profile    = var.builder_instance_profile
  skip_profile_validation = true

  vpc_filter {
    filters = {
      "tag:Name" : "jenkins-pmm-amzn2"
    }
  }
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
}

build {
  name    = "pmm-agent-smoke"
  sources = ["source.amazon-ebs.smoke"]

  # Runs as ec2-user, the account the Jenkins launchers log in as. HCL
  # interpolates ${var..} and ${local..}, shell $(...) passes through.
  provisioner "shell" {
    inline_shebang = "/bin/bash"
    inline = [
      "set -euo pipefail",
      ". /etc/os-release && [[ \"$${ID}\" == almalinux ]] || { echo \"wrong distro ID $${ID}\"; exit 1; }",
      "[[ \"$(uname -m)\" == '${local.uname_arch}' ]] || { echo \"wrong arch $(uname -m)\"; exit 1; }",
      # cloud-init exits 2 for a finished boot with recoverable errors ("degraded done"), so the status text is what counts, not the exit code.
      "status=$(sudo cloud-init status --wait 2>/dev/null || true); case \"$${status}\" in *done*|*disabled*) ;; *) echo \"cloud-init: $${status}\"; sudo cloud-init status --long; exit 1 ;; esac",
      "test -s /etc/machine-id || { echo 'machine-id was not generated on first boot'; exit 1; }",
      "java -version 2>&1 | grep -q 'version \"21' || { echo 'default java is not 21'; java -version; exit 1; }",
      "readlink -f /etc/alternatives/java | grep -q 'java-21' || { echo 'alternatives java is not 21'; readlink -f /etc/alternatives/java; exit 1; }",
      "other_jvms=$(rpm -qa 'java-*-openjdk*' | grep -v '^java-21-' || true); [[ -z \"$${other_jvms}\" ]] || { echo \"JVMs other than 21 installed: $${other_jvms}\"; exit 1; }",
      "[[ \"$(command -v ansible-playbook)\" == /usr/bin/ansible-playbook ]] || { echo \"ansible-playbook resolves to $(command -v ansible-playbook), expected the AppStream one\"; exit 1; }",
      "java -XX:MaxRAMPercentage=25 -XX:+PrintFlagsFinal -version 2>/dev/null | grep -q MaxHeapSize || { echo 'JVM did not start with the agent heap policy'; exit 1; }",
      "rpm -q amazon-ssm-agent >/dev/null || { echo 'ssm-agent not baked'; exit 1; }",
      "systemctl is-active --quiet amazon-ssm-agent || { echo 'ssm-agent not active'; exit 1; }",
      "systemctl is-enabled --quiet amazon-ssm-agent || { echo 'ssm-agent not enabled'; exit 1; }",
      "id -nG ec2-user | grep -qw docker || { echo 'ec2-user not in docker group'; exit 1; }",
      "systemctl is-active --quiet docker || { echo 'docker daemon not active'; exit 1; }",
      "docker info >/dev/null 2>&1 || { echo 'docker daemon not reachable as ec2-user'; exit 1; }",
      "docker run --pull=never --rm oraclelinux:9 true || { echo 'cannot run a pre-pulled container'; exit 1; }",
      "systemctl is-active --quiet sshd || { echo 'sshd not active (the Jenkins launchers connect over SSH)'; exit 1; }",
      "ss -ltn | grep -q ':22 ' || { echo 'nothing listening on :22'; exit 1; }",
      "test -s /home/ec2-user/.ssh/authorized_keys || { echo 'cloud-init did not inject the launch key for ec2-user'; exit 1; }",
      "case \"$(systemctl show -p UnitFileState --value dnf-makecache.timer)\" in disabled|masked) ;; *) echo 'dnf-makecache.timer is enabled (boot lock contention)'; exit 1 ;; esac",
      "for tool in kubectl helm eksctl doctl yq aws node npm docker-compose bats git svn jq wget unzip envsubst bc chromium-browser; do command -v \"$${tool}\" >/dev/null || { echo \"missing tool: $${tool}\"; exit 1; }; done",
      "docker buildx version >/dev/null || { echo 'docker buildx plugin missing'; exit 1; }",
      "getent hosts repo.ci.percona.com | grep -q '^10.30.6.9 ' || { echo 'repo.ci.percona.com is not pinned in /etc/hosts'; exit 1; }",
      "echo \"smoke ok: $(java -version 2>&1 | head -1), docker $(docker --version)\"",
    ]
  }
}
