# Packer templates to build the agents on AWS and DigitalOcean

### Building agents

- AWS: 
  - build all agents: `packer build aws.pkr.hcl`
  - build only amd64: `packer build -only=jenkins-farm.amazon-ebs.amd-agent aws.pkr.hcl`
  - build only arm64: `packer build -only=jenkins-farm.amazon-ebs.arm-agent aws.pkr.hcl`
- DigitalOcean:
  - buid DO agent: `packer build -color=false do.pkr.hcl`

### Turn on logging

Run: 
```
  PACKER_LOG_PATH="packer.log" PACKER_LOG=1 packer build aws.pkr.hcl
  PACKER_LOG_PATH="packer.log" PACKER_LOG=1 packer build do.pkr.hcl
```
