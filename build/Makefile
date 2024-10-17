export PACKER_CACHE_DIR := .cache
export PACKER_VERSION := 1.9.4
export PMM_SERVER_IMAGE ?= docker.io/perconalab/pmm-server:3-dev-latest

## ----------------- PACKER ------------------
fetch:
	mkdir -p ${PACKER_CACHE_DIR}/box || :
	test -f ${PACKER_CACHE_DIR}/id_rsa_vagrant \
	    || curl -L https://raw.githubusercontent.com/hashicorp/vagrant/master/keys/vagrant \
		-o ${PACKER_CACHE_DIR}/id_rsa_vagrant
	chmod 600 ${PACKER_CACHE_DIR}/id_rsa_vagrant
	test -f ${PACKER_CACHE_DIR}/box/oracle9.ova \
		|| curl -fL https://pmm-build-cache.s3.us-east-2.amazonaws.com/VBOXES/oracle9-202401.31.0.box -o ${PACKER_CACHE_DIR}/box/oracle9.ova

	# NOTE: image from vagrant registry is twice as large
	test -f ${PACKER_CACHE_DIR}/box/box.ovf \
		|| tar -C ${PACKER_CACHE_DIR}/box -xvf ${PACKER_CACHE_DIR}/box/oracle9.ova

deps:
	mkdir -p ${PACKER_CACHE_DIR} ~/bin || :
	curl -fL https://releases.hashicorp.com/packer/${PACKER_VERSION}/packer_${PACKER_VERSION}_linux_amd64.zip -o ${PACKER_CACHE_DIR}/packer.zip
	unzip -o ${PACKER_CACHE_DIR}/packer.zip -d ~/bin

pmm-ovf: fetch
	/usr/bin/packer build -var 'pmm_server_image_name=${PMM_SERVER_IMAGE}'  -only virtualbox-ovf -color=false packer/pmm.json | tee build.log

pmm-digitalocean:
	packer build -only digitalocean -var 'single_disk=true' packer/pmm.json

pmm-azure:
	packer build -only azure-arm packer/pmm.json

pmm-ami:
	docker run --rm -v ${HOME}/.aws:/root/.aws -v `pwd`:/build -w /build \hashicorp/packer:${PACKER_VERSION} \
		build -var 'pmm_server_image_name=${PMM_SERVER_IMAGE}' -only amazon-ebs -color=false packer/pmm.json | tee build.log
## ----------------- PACKER ------------------

check:
	echo "TODO: Since update.yml has been deprecated, see if other playbooks need to be linted"
	# ansible-playbook --syntax-check ansible/pmm-docker/update.yml
	# ansible-playbook --check ansible/pmm-docker/update.yml
	# ansible-lint ansible/pmm-docker/update.yml
