export PACKER_CACHE_DIR := .cache
export PACKER_VERSION := 1.11.2
# check available versions at https://vagrantcloud.com/api/v2/vagrant/bento/oraclelinux-9
export BOX_VERSION := 202502.21.0
export PMM_SERVER_IMAGE ?= docker.io/perconalab/pmm-server:3-dev-latest
export WATCHTOWER_IMAGE ?= docker.io/perconalab/watchtower:dev-latest

## ----------------- PACKER ------------------
fetch:
	mkdir -p ${PACKER_CACHE_DIR}/box
	test -f ${PACKER_CACHE_DIR}/id_rsa_vagrant \
		|| curl -L https://raw.githubusercontent.com/hashicorp/vagrant/master/keys/vagrant -o ${PACKER_CACHE_DIR}/id_rsa_vagrant
	chmod 600 ${PACKER_CACHE_DIR}/id_rsa_vagrant

	# Add the box using Vagrant
	test -f ${PACKER_CACHE_DIR}/box/box.ovf \
		|| VAGRANT_HOME=${PACKER_CACHE_DIR}/box vagrant box add bento/oraclelinux-9 --box-version ${BOX_VERSION} --provider virtualbox

	test -f ${PACKER_CACHE_DIR}/box/box.ovf \
		|| cp -rp ${PACKER_CACHE_DIR}/box/boxes/bento-VAGRANTSLASH-oraclelinux-9/${BOX_VERSION}/amd64/virtualbox/* ${PACKER_CACHE_DIR}/box

deps:
	mkdir -p ${PACKER_CACHE_DIR} ~/bin || :
	curl -fL https://releases.hashicorp.com/packer/${PACKER_VERSION}/packer_${PACKER_VERSION}_linux_amd64.zip -o ${PACKER_CACHE_DIR}/packer.zip
	unzip -o ${PACKER_CACHE_DIR}/packer.zip -d ~/bin

pmm-ovf: fetch
	/usr/bin/packer build \
		-var 'pmm_server_image_name=${PMM_SERVER_IMAGE}' \
		-var 'watchtower_image_name=${WATCHTOWER_IMAGE}' \
		-only virtualbox-ovf -color=false packer/pmm.json | tee build.log

pmm-digitalocean:
	packer build -only digitalocean -var 'single_disk=true' packer/pmm.json

pmm-azure:
	packer build -only azure-arm packer/pmm.json

pmm-ami:
	docker run --rm -v ${HOME}/.aws:/root/.aws -v `pwd`:/build -w /build \hashicorp/packer:${PACKER_VERSION} \
		build -var 'pmm_server_image_name=${PMM_SERVER_IMAGE}' \
		-var 'watchtower_image_name=${WATCHTOWER_IMAGE}' \
		-only amazon-ebs -color=false packer/pmm.json | tee build.log

# Initialize Packer plugins
packer-init:
	cd packer && packer init aws.pkr.hcl
	cd packer && packer init do.pkr.hcl
	cd packer && packer init hetzner.pkr.hcl

# Jenkins Agent builds for various cloud providers
jenkins-agents-aws:
	cd packer && packer build aws.pkr.hcl

jenkins-agents-digitalocean:
	cd packer && packer build -color=false do.pkr.hcl

jenkins-agents-hetzner:
	@if [ -z "${HCLOUD_TOKEN}" ]; then \
		echo "Error: HCLOUD_TOKEN environment variable is not set"; \
		echo "Please run: export HCLOUD_TOKEN='your-token-here'"; \
		exit 1; \
	fi
	cd packer && packer build -var="hcloud_token=${HCLOUD_TOKEN}" hetzner.pkr.hcl

jenkins-agents-hetzner-amd64:
	@if [ -z "${HCLOUD_TOKEN}" ]; then \
		echo "Error: HCLOUD_TOKEN environment variable is not set"; \
		echo "Please run: export HCLOUD_TOKEN='your-token-here'"; \
		exit 1; \
	fi
	cd packer && packer build -var="hcloud_token=${HCLOUD_TOKEN}" -only=jenkins-farm.hcloud.jenkins-agent hetzner.pkr.hcl

jenkins-agents-hetzner-arm64:
	@if [ -z "${HCLOUD_TOKEN}" ]; then \
		echo "Error: HCLOUD_TOKEN environment variable is not set"; \
		echo "Please run: export HCLOUD_TOKEN='your-token-here'"; \
		exit 1; \
	fi
	cd packer && packer build -var="hcloud_token=${HCLOUD_TOKEN}" -only=jenkins-farm.hcloud.jenkins-agent-arm hetzner.pkr.hcl

# Build all Jenkins agents
jenkins-agents-all: jenkins-agents-aws jenkins-agents-digitalocean jenkins-agents-hetzner
## ----------------- PACKER ------------------

check:
	echo "TODO: Since update.yml has been deprecated, see if other playbooks need to be linted"
	# ansible-playbook --syntax-check ansible/pmm-docker/main.yml
	# ansible-playbook --check ansible/pmm-docker/main.yml
	# ansible-lint ansible/pmm-docker/main.yml
