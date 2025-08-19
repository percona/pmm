# Packer templates to build the agents on AWS, DigitalOcean, and Hetzner Cloud

### Building agents

- AWS: `packer build aws.pkr.hcl`
  - build only amd64: `packer build -only=jenkins-farm.amazon-ebs.agent aws.pkr.hcl`
  - build only arm64: `packer build -only=jenkins-farm.amazon-ebs.arm-agent aws.pkr.hcl`
- DigitalOcean: `packer build -color=false do.pkr.hcl`
- Hetzner Cloud: `packer build -var="hcloud_token=$HCLOUD_TOKEN" hetzner.pkr.hcl`
  - **Required**: Set `HCLOUD_TOKEN` environment variable before building: `export HCLOUD_TOKEN="your-token-here"`
  - build only amd64: `packer build -var="hcloud_token=$HCLOUD_TOKEN" -only=jenkins-farm.hcloud.jenkins-agent hetzner.pkr.hcl`
  - build only arm64: `packer build -var="hcloud_token=$HCLOUD_TOKEN" -only=jenkins-farm.hcloud.jenkins-agent-arm hetzner.pkr.hcl`

### Turn on logging

Run: 
```
  PACKER_LOG_PATH="packer.log" PACKER_LOG=1 packer build aws.pkr.hcl
  PACKER_LOG_PATH="packer.log" PACKER_LOG=1 packer build do.pkr.hcl
  PACKER_LOG_PATH="packer.log" PACKER_LOG=1 packer build -var="hcloud_token=$HCLOUD_TOKEN" hetzner.pkr.hcl
```
