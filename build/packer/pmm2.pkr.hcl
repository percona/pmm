packer {
  required_plugins {
    amazon = {
      version = "=1.0.8"
      source  = "github.com/hashicorp/amazon"
    }
  }
}

variable "pmm_server_image_name" {
  type = string
  default = "perconalab/pmm-server:dev-latest"
}

variable "single_disk" {
  type = string
  default = "false"
}

variable "pmm2_server_repo" {
  type = string
  default = "testing"
}

variable "pmm_client_repos" {
  type = string
  default = "original testing"
}

variable "pmm_client_repo_name" {
  type = string
  default = "percona-testing-x86_64"
}

source "virtualbox-ovf" "image" {
  export_opts          = ["--ovf10", "--manifest", "--vsys", "0", "--product", "Percona Monitoring and Management", "--producturl", "https://www.percona.com/software/database-tools/percona-monitoring-and-management", "--vendor", "Percona", "--vendorurl", "https://www.percona.com", "--version", "${formatdate("YYYY-MM-DD", timestamp())}", "--description", "Percona Monitoring and Management (PMM) is an open-source platform for managing and monitoring MySQL and MongoDB performance"]
  format               = "ovf"
  guest_additions_mode = "disable"
  headless             = true
  output_directory     = "pmm2-virtualbox-ovf"
  shutdown_command     = "rm -rf ~/.ssh/authorized_keys; cat /dev/zero > zero.fill; sync; sleep 1; sync; rm -f zero.fill; sudo shutdown -P now"
  source_path          = ".cache/2004.01/box.ovf"
  ssh_private_key_file = ".cache/id_rsa_vagrant"
  ssh_pty              = true
  ssh_username         = "vagrant"
  vboxmanage           = [["modifyvm", "{{ .Name }}", "--memory", "4096"], ["modifyvm", "{{ .Name }}", "--audio", "none"], ["createhd", "--format", "VMDK", "--filename", "/tmp/{{ .Name }}-disk2.vmdk", "--variant", "STREAM", "--size", "409600"], ["storagectl", "{{ .Name }}", "--name", "SCSI Controller", "--add", "scsi", "--controller", "LSILogic"], ["storageattach", "{{ .Name }}", "--storagectl", "SCSI Controller", "--port", "1", "--type", "hdd", "--medium", "/tmp/{{ .Name }}-disk2.vmdk"]]
  vm_name              = "PMM2-Server-${formatdate("YYYY-MM-DD", timestamp())}"
}

source "amazon-ebs" "image" {
  ami_name          = "PMM2 Server [${formatdate("YYYY-MM-DD hhmm", timestamp())}]"
  instance_type     = "c4.xlarge"
  ena_support       = "true"
  region            = "us-east-1"
  subnet_id         = "subnet-ee06e8e1"
  security_group_id = "sg-688c2b1c"
  ssh_username      = "ec2-user"

  launch_block_device_mappings {
    delete_on_termination = true
    device_name           = "/dev/xvda"
    volume_size           = 10
    volume_type           = "gp3"
  }

  launch_block_device_mappings {
    delete_on_termination = false
    device_name           = "/dev/xvdb"
    volume_size           = 100
    volume_type           = "gp3"
  }

  source_ami_filter {
    filters = {
      name                = "*amzn2-ami-hvm-*"
      root-device-type    = "ebs"
      virtualization-type = "hvm"
      architecture        = "x86_64"
    }
    most_recent = true
    owners      = ["amazon"]
  }
  tags = {
    iit-billing-tag = "pmm-worker"
  }
  run_tags = {
    iit-billing-tag = "pmm-ami"
  }
  run_volume_tags = {
    iit-billing-tag = "pmm-ami"
  }
}

build {
  name = "pmm2"
  sources = [
    "source.amazon-ebs.image",
    "source.virtualbox-ovf.image"
  ]
  provisioner "ansible" {
    extra_arguments = [
        "-v",
        "-b",
        "--become-user",
        "root",
        "--extra-vars",
        "pmm_server_image_name=${var.pmm_server_image_name}"
    ]
    playbook_file = "./packer/ansible/pmm2.yml"
  }
}
