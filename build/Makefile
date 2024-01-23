export PACKER_CACHE_DIR := .cache
export PACKER_VERSION := 1.8.2
export CENTOS_ISO := 2004.01

## ----------------- PACKER ------------------
fetch:
	mkdir -p ${PACKER_CACHE_DIR}/${CENTOS_ISO} || :
	test -f ${PACKER_CACHE_DIR}/id_rsa_vagrant \
	    || curl -L https://raw.githubusercontent.com/hashicorp/vagrant/master/keys/vagrant \
		-o ${PACKER_CACHE_DIR}/id_rsa_vagrant
	chmod 600 ${PACKER_CACHE_DIR}/id_rsa_vagrant
	test -f ${PACKER_CACHE_DIR}/${CENTOS_ISO}/CentOS7.ova \
		|| wget --progress=dot:giga https://app.vagrantup.com/centos/boxes/7/versions/${CENTOS_ISO}/providers/virtualbox.box \
		-O ${PACKER_CACHE_DIR}/${CENTOS_ISO}/CentOS7.ova
	test -f ${PACKER_CACHE_DIR}/${CENTOS_ISO}/box.ovf \
		|| tar -C ${PACKER_CACHE_DIR}/${CENTOS_ISO} -xvf ${PACKER_CACHE_DIR}/${CENTOS_ISO}/CentOS7.ova

fetch-el9:
	mkdir -p ${PACKER_CACHE_DIR}/box || :
	test -f ${PACKER_CACHE_DIR}/id_rsa_vagrant \
	    || curl -L https://raw.githubusercontent.com/hashicorp/vagrant/master/keys/vagrant \
		-o ${PACKER_CACHE_DIR}/id_rsa_vagrant
	chmod 600 ${PACKER_CACHE_DIR}/id_rsa_vagrant
	test -f ${PACKER_CACHE_DIR}/box/oracle9.ova \
		|| wget --progress=dot:giga  https://vagrantcloud.com/bento/boxes/oracle-9.0/versions/202207.20.0/providers/virtualbox.box -O ${PACKER_CACHE_DIR}/box/oracle9.ova

	# NOTE: image from vagrant registry is twice as large
	test -f ${PACKER_CACHE_DIR}/box/box.ovf \
		|| tar -C ${PACKER_CACHE_DIR}/box -xvf ${PACKER_CACHE_DIR}/box/oracle9.ova

deps:
	mkdir -p ${PACKER_CACHE_DIR} ~/bin || :
	curl https://releases.hashicorp.com/packer/${PACKER_VERSION}/packer_${PACKER_VERSION}_linux_amd64.zip -o ${PACKER_CACHE_DIR}/packer.zip
	unzip -o ${PACKER_CACHE_DIR}/packer.zip -d ~/bin

pmm-ovf: fetch
	packer build -only virtualbox-ovf packer/pmm.json

pmm-ovf-el9-rc: fetch-el9
	sed -i 's|become_method: su|become_method: sudo|g' update/tasks/roles/postgres/tasks/main.yml && \
	/usr/bin/packer build \
			-var 'pmm_client_repos=original testing' \
			-var 'pmm_client_repo_name=percona-testing-x86_64' \
			-var 'pmm_server_repo=testing' \
			-only virtualbox-ovf -color=false packer/pmm.el9.json \
					| tee build.log

pmm-ovf-el9-3-dev-latest: fetch-el9
	sed -i 's|become_method: su|become_method: sudo|g' update/tasks/roles/postgres/tasks/main.yml && \
	/usr/bin/packer build \
			-var 'pmm_client_repos=original experimental' \
			-var 'pmm_client_repo_name=percona-experimental-x86_64' \
			-var 'pmm_server_repo=experimental' \
			-only virtualbox-ovf -color=false packer/pmm.el9.json \
					| tee build.log

pmm-digitalocean:
	packer build -only digitalocean -var 'single_disk=true' packer/pmm.json

pmm-azure:
	packer build -only azure-arm packer/pmm.json

pmm-ami:
	docker run --rm -v ${HOME}/.aws:/root/.aws -v `pwd`:/build -w /build \hashicorp/packer:${PACKER_VERSION} \
	build -only amazon-ebs -color=false packer/pmm.json | tee build.log

## ----------------- PACKER ------------------

check:                          ## Run required checkers and linters
	ansible-playbook --syntax-check ansible/pmm-docker/update.yml
	ansible-playbook --check ansible/pmm-docker/update.yml
	ansible-lint ansible/pmm-docker/update.yml
