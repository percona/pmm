#!/bin/bash
# No-AWS gate for the pmm agent AMI factory: fmt-check, init, validate both
# templates for every env and arch, and syntax-check the playbook with the
# runner's pinned collections. Run from build/packer. The workflow check job
# and a local run execute THIS file, so the two gates cannot drift apart.
set -euo pipefail

packer fmt -check -diff aws.pkr.hcl
packer fmt -check -diff smoke/smoke.pkr.hcl

packer init aws.pkr.hcl
packer init smoke/smoke.pkr.hcl

for env_name in prod test; do
  echo "validate aws.pkr.hcl env=${env_name}"
  packer validate -var "env=${env_name}" aws.pkr.hcl
  for only in jenkins-farm.amazon-ebs.amd64-agent jenkins-farm.amazon-ebs.arm64-agent; do
    packer validate -only="${only}" -var "env=${env_name}" aws.pkr.hcl
  done
done

for arch in x86_64 arm64; do
  echo "validate smoke arch=${arch}"
  packer validate -var "arch=${arch}" smoke/smoke.pkr.hcl
done

ansible-galaxy collection install -r ansible/requirements.yml >/dev/null
ansible-playbook --syntax-check -i default, ansible/agent-aws.yml
