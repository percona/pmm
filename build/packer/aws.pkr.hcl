packer {
  required_plugins {
    amazon = {
      version = "=1.1.6"
      source  = "github.com/hashicorp/amazon"
    }
    ansible = {
      source  = "github.com/hashicorp/ansible"
      version = "~> 1"
    }    
  }
}

source "amazon-ebs" "agent" {
  ami_name              = "Docker Agent v3"
  instance_type         = "t3.xlarge"
  force_deregister      = true
  force_delete_snapshot = true
  region                = "us-east-2"
  source_ami_filter {
    filters = {
      name                = "OL9.3-*"
      root-device-type    = "ebs"
      virtualization-type = "hvm"
      architecture        = "x86_64"
    }
    most_recent = true
    owners      = ["131827586825"]
  }
  ssh_username = "ec2-user"
  tags = {
    Name            = "Jenkins Agent x86_64 v3"
    iit-billing-tag = "pmm-worker-3"
  }
  run_tags = {
    iit-billing-tag = "pmm-worker"
  }
  run_volume_tags = {
    iit-billing-tag = "pmm-worker"
  }
  launch_block_device_mappings {
    device_name = "/dev/sda1"
    volume_size = 50
    volume_type = "gp3"
    delete_on_termination = true
  }
  vpc_filter {
    filters = {
      "tag:Name" : "jenkins-pmm-amzn2"
    }
  }
  subnet_filter {
    filters = {
      "tag:Name" : "jenkins-pmm-amzn2-B"
    }
    random = true
  }
}

source "amazon-ebs" "arm-agent" {
  ami_name              = "Docker Agent ARM v3"
  instance_type         = "t4g.xlarge"
  force_deregister      = true
  force_delete_snapshot = true
  region                = "us-east-2"
  source_ami_filter {
    filters = {
      name                = "OL9.3-*"
      root-device-type    = "ebs"
      virtualization-type = "hvm"
      architecture        = "arm64"
    }
    most_recent = true
    owners      = ["131827586825"]
  }
  ssh_username = "ec2-user"
  tags = {
    Name            = "Jenkins Agent arm64 v3"
    iit-billing-tag = "pmm-worker-3"
  }
  run_tags = {
    iit-billing-tag = "pmm-worker",
  }
  run_volume_tags = {
    iit-billing-tag = "pmm-worker"
  }
  launch_block_device_mappings {
    device_name           = "/dev/sda1"
    volume_size           = 50
    volume_type           = "gp3"
    delete_on_termination = true
  }
  vpc_filter {
    filters = {
      "tag:Name" : "jenkins-pmm-amzn2"
    }
  }
  subnet_filter {
    filters = {
      "tag:Name" : "jenkins-pmm-amzn2-B"
    }
    random = true
  }
}

build {
  name = "jenkins-farm"
  sources = [
    "source.amazon-ebs.agent",
    "source.amazon-ebs.arm-agent"
  ]
  provisioner "ansible" {
    use_proxy              = false
    user                   = "ec2-user"
    ansible_env_vars       = ["ANSIBLE_NOCOLOR=True"]
    extra_arguments = [
      "--ssh-extra-args", "-o HostKeyAlgorithms=+ssh-rsa -o StrictHostKeyChecking=no -o ForwardAgent=yes -o UserKnownHostsFile=/dev/null", "-vvv"
    ]
    playbook_file          = "./ansible/agent-aws.yml"
  }
}
