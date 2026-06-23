export PACKER_VERSION := 1.9.4
export PMM_SERVER_IMAGE ?= docker.io/perconalab/pmm-server:3-dev-latest
export WATCHTOWER_IMAGE ?= docker.io/perconalab/watchtower:dev-latest
# --- Go-related variables ----------------------------------------------------------------
GO_VERSION := $(shell grep '^go ' $(CURDIR)/../go.mod | awk '{print $$2}')

## ----------------- PACKER ------------------
pmm-ami:
	docker run --rm -v ${HOME}/.aws:/root/.aws -v `pwd`:/build -w /build \hashicorp/packer:${PACKER_VERSION} \
		build -var 'pmm_server_image_name=${PMM_SERVER_IMAGE}' \
		-var 'watchtower_image_name=${WATCHTOWER_IMAGE}' \
		-only amazon-ebs -color=false packer/pmm.json | tee build.log
## ----------------- PACKER ------------------

## RPM Build Images
rpmbuild-el9: ## Build rpmbuild Docker image for EL9 (Oracle Linux 9)
	cd docker/rpmbuild && docker build -t perconalab/rpmbuild:3 -f Dockerfile.el9 .

rpmbuild-hetzner-el9: ## Build rpmbuild Docker image for Hetzner (Rocky Linux 9)
	cd docker/rpmbuild && \
	docker build \
	-t perconalab/rpmbuild:3-hetzner \
	-f Dockerfile.hetzner-el9 \
	--build-arg GO_VERSION=$(GO_VERSION) \
	.

rpmbuild-all: rpmbuild-el9 rpmbuild-hetzner-el9 ## Build all rpmbuild Docker images

check:
	echo "TODO: Since update.yml has been deprecated, see if other playbooks need to be linted"
	# ansible-playbook --syntax-check ansible/pmm-docker/main.yml
	# ansible-playbook --check ansible/pmm-docker/main.yml
	# ansible-lint ansible/pmm-docker/main.yml
