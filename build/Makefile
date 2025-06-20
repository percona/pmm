export PACKER_CACHE_DIR := .cache
export PACKER_VERSION := 1.11.2
# check available versions at https://vagrantcloud.com/api/v2/vagrant/bento/oraclelinux-9
export BOX_VERSION := 202502.21.0
export PMM_SERVER_IMAGE ?= docker.io/perconalab/pmm-server:3-dev-latest
export WATCHTOWER_IMAGE ?= docker.io/perconalab/watchtower:dev-latest

.DEFAULT_GOAL := help

## ----------------- HELP ------------------
help: ## Show this help message
	@echo 'PMM Build System'
	@echo '================'
	@echo 'Build PMM Server images and Jenkins agents for various cloud platforms.'
	@echo ''
	@echo 'Usage: make [TARGET]'
	@echo ''
	@echo 'Available targets:'
	@echo ''
	@echo 'PMM Server Images:'
	@grep -E '^pmm-[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-30s %s\n", $$1, $$2}'
	@echo ''
	@echo 'Jenkins Agent Images:'
	@grep -E '^jenkins-agents-[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-30s %s\n", $$1, $$2}'
	@echo ''
	@echo 'Packer Tools:'
	@grep -E '^(deps|fetch|packer-init):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-30s %s\n", $$1, $$2}'
	@echo ''
	@echo 'Other:'
	@grep -E '^check:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-30s %s\n", $$1, $$2}'
	@echo ''
	@echo 'Environment Variables:'
	@echo '  DO_API_TOKEN                   DigitalOcean API token (required for DigitalOcean builds)'
	@echo '  HCLOUD_TOKEN                   Hetzner Cloud API token (required for Hetzner builds)'
	@echo '  HCLOUD_LOCATION                Hetzner location (default: fsn1, options: fsn1, nbg1, hel1, ash, hil)'
	@echo '  PMM_SERVER_IMAGE               PMM Server Docker image (default: docker.io/perconalab/pmm-server:3-dev-latest)'
	@echo '  WATCHTOWER_IMAGE               Watchtower Docker image (default: docker.io/perconalab/watchtower:dev-latest)'

## ----------------- PACKER ------------------
fetch: ## Download required files for PMM Server OVF builds
	mkdir -p ${PACKER_CACHE_DIR}/box
	test -f ${PACKER_CACHE_DIR}/id_rsa_vagrant \
		|| curl -L https://raw.githubusercontent.com/hashicorp/vagrant/master/keys/vagrant -o ${PACKER_CACHE_DIR}/id_rsa_vagrant
	chmod 600 ${PACKER_CACHE_DIR}/id_rsa_vagrant

	# Add the box using Vagrant
	test -f ${PACKER_CACHE_DIR}/box/box.ovf \
		|| VAGRANT_HOME=${PACKER_CACHE_DIR}/box vagrant box add bento/oraclelinux-9 --box-version ${BOX_VERSION} --provider virtualbox

	test -f ${PACKER_CACHE_DIR}/box/box.ovf \
		|| cp -rp ${PACKER_CACHE_DIR}/box/boxes/bento-VAGRANTSLASH-oraclelinux-9/${BOX_VERSION}/amd64/virtualbox/* ${PACKER_CACHE_DIR}/box

deps: ## Install Packer binary
	mkdir -p ${PACKER_CACHE_DIR} ~/bin || :
	curl -fL https://releases.hashicorp.com/packer/${PACKER_VERSION}/packer_${PACKER_VERSION}_linux_amd64.zip -o ${PACKER_CACHE_DIR}/packer.zip
	unzip -o ${PACKER_CACHE_DIR}/packer.zip -d ~/bin

pmm-ovf: fetch ## Build PMM Server OVF for VirtualBox
	/usr/bin/packer build \
		-var 'pmm_server_image_name=${PMM_SERVER_IMAGE}' \
		-var 'watchtower_image_name=${WATCHTOWER_IMAGE}' \
		-only virtualbox-ovf -color=false packer/pmm.json | tee build.log

pmm-digitalocean: ## Build PMM Server image for DigitalOcean
	packer build -only digitalocean -var 'single_disk=true' packer/pmm.json

pmm-azure: ## Build PMM Server image for Azure
	packer build -only azure-arm packer/pmm.json

pmm-ami: ## Build PMM Server AMI for AWS
	docker run --rm -v ${HOME}/.aws:/root/.aws -v `pwd`:/build -w /build \hashicorp/packer:${PACKER_VERSION} \
		build -var 'pmm_server_image_name=${PMM_SERVER_IMAGE}' \
		-var 'watchtower_image_name=${WATCHTOWER_IMAGE}' \
		-only amazon-ebs -color=false packer/pmm.json | tee build.log

packer-init: ## Initialize Packer plugins
	cd packer && packer init aws.pkr.hcl
	cd packer && packer init do.pkr.hcl
	cd packer && packer init hetzner.pkr.hcl

jenkins-agents-aws: ## Build Jenkins agents for AWS (x86_64 and ARM64)
	cd packer && packer build aws.pkr.hcl

jenkins-agents-digitalocean: ## Build Jenkins agent for DigitalOcean
	@if [ -z "${DO_API_TOKEN}" ]; then \
		echo "Error: DO_API_TOKEN environment variable is not set"; \
		echo "Please run: export DO_API_TOKEN='your-token-here'"; \
		exit 1; \
	fi
	cd packer && packer build -var="do_api_token=${DO_API_TOKEN}" -color=false do.pkr.hcl

jenkins-agents-hetzner: ## Build Jenkins agents for Hetzner Cloud (x86_64 and ARM64)
	@if [ -z "${HCLOUD_TOKEN}" ]; then \
		echo "Error: HCLOUD_TOKEN environment variable is not set"; \
		echo "Please run: export HCLOUD_TOKEN='your-token-here'"; \
		exit 1; \
	fi
	cd packer && packer build -var="hcloud_token=${HCLOUD_TOKEN}" $(if $(HCLOUD_LOCATION),-var="location=$(HCLOUD_LOCATION)") hetzner.pkr.hcl

jenkins-agents-hetzner-amd64: ## Build Jenkins agent for Hetzner Cloud (x86_64 only)
	@if [ -z "${HCLOUD_TOKEN}" ]; then \
		echo "Error: HCLOUD_TOKEN environment variable is not set"; \
		echo "Please run: export HCLOUD_TOKEN='your-token-here'"; \
		exit 1; \
	fi
	cd packer && packer build -var="hcloud_token=${HCLOUD_TOKEN}" $(if $(HCLOUD_LOCATION),-var="location=$(HCLOUD_LOCATION)") -only=jenkins-farm.hcloud.jenkins-agent hetzner.pkr.hcl

jenkins-agents-hetzner-arm64: ## Build Jenkins agent for Hetzner Cloud (ARM64 only)
	@if [ -z "${HCLOUD_TOKEN}" ]; then \
		echo "Error: HCLOUD_TOKEN environment variable is not set"; \
		echo "Please run: export HCLOUD_TOKEN='your-token-here'"; \
		exit 1; \
	fi
	cd packer && packer build -var="hcloud_token=${HCLOUD_TOKEN}" $(if $(HCLOUD_LOCATION),-var="location=$(HCLOUD_LOCATION)") -only=jenkins-farm.hcloud.jenkins-agent-arm hetzner.pkr.hcl

jenkins-agents-all: jenkins-agents-aws jenkins-agents-digitalocean jenkins-agents-hetzner ## Build all Jenkins agents
## ----------------- PACKER ------------------

check: ## Run syntax checks on Ansible playbooks
	echo "TODO: Since update.yml has been deprecated, see if other playbooks need to be linted"
	# ansible-playbook --syntax-check ansible/pmm-docker/main.yml
	# ansible-playbook --check ansible/pmm-docker/main.yml
	# ansible-lint ansible/pmm-docker/main.yml
