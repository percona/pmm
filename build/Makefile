export PACKER_CACHE_DIR := .cache
export PACKER_VERSION := 1.8.2
export BOX_VERSION := 202407.23.0
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
			|| curl -fL https://pmm-build-cache.s3.us-east-2.amazonaws.com/VBOXES/oracle9-202407.23.0.box -o ${PACKER_CACHE_DIR}/box/oracle9.ova

	test -f ${PACKER_CACHE_DIR}/box/box.ovf \
			|| tar -C ${PACKER_CACHE_DIR}/box -xvf ${PACKER_CACHE_DIR}/box/oracle9.ova

	# # Add the box using Vagrant
	# test -f ${PACKER_CACHE_DIR}/box/box.ovf \
	# 	|| VAGRANT_HOME=${PACKER_CACHE_DIR}/box vagrant box add bento/oraclelinux-9 --box-version ${BOX_VERSION} --provider virtualbox

	test -f ${PACKER_CACHE_DIR}/box/box.ovf \
		|| cp -rp ${PACKER_CACHE_DIR}/box/boxes/bento-VAGRANTSLASH-oraclelinux-9/${BOX_VERSION}/amd64/virtualbox/* ${PACKER_CACHE_DIR}/box

deps:
	mkdir -p ${PACKER_CACHE_DIR} ~/bin || :
	curl https://releases.hashicorp.com/packer/${PACKER_VERSION}/packer_${PACKER_VERSION}_linux_amd64.zip -o ${PACKER_CACHE_DIR}/packer.zip
	unzip -o ${PACKER_CACHE_DIR}/packer.zip -d ~/bin

pmm2-ovf: fetch
	packer build -only virtualbox-ovf packer/pmm2.json

pmm2-ovf-el9-rc: fetch-el9
	sed -i 's|become_method: su|become_method: sudo|g' update/tasks/roles/postgres/tasks/main.yml && \
	/usr/bin/packer build \
			-var 'pmm_client_repos=pmm2-client testing' \
			-var 'pmm_client_repo_name=pmm2-client-testing-x86_64' \
			-var 'pmm2_server_repo=testing' \
			-only virtualbox-ovf -color=false packer/pmm2.el9.json \
					| tee build.log

pmm2-ovf-el9-dev-latest: fetch-el9
	sed -i 's|become_method: su|become_method: sudo|g' update/tasks/roles/postgres/tasks/main.yml && \
	/usr/bin/packer build \
			-var 'pmm_client_repos=pmm2-client experimental' \
			-var 'pmm_client_repo_name=pmm2-client-experimental-x86_64' \
			-var 'pmm2_server_repo=experimental' \
			-only virtualbox-ovf -color=false packer/pmm2.el9.json \
					| tee build.log

pmm2-digitalocean:
	packer build -only digitalocean -var 'single_disk=true' packer/pmm2.json

pmm2-azure:
	packer build -only azure-arm packer/pmm2.json

pmm2-ami:
	docker run --rm -v ${HOME}/.aws:/root/.aws -v `pwd`:/build -w /build hashicorp/packer:${PACKER_VERSION} \
			build -var 'pmm_client_repos=pmm2-client experimental' \
						-var 'pmm_client_repo_name=pmm2-client-experimental-x86_64' \
						-var 'pmm2_server_repo=experimental' \
						-only amazon-ebs -color=false \
				  	packer/pmm2.json

pmm2-ami-rc:
	docker run --rm -v ${HOME}/.aws:/root/.aws -v `pwd`:/build -w /build hashicorp/packer:${PACKER_VERSION} \
			build -var 'pmm_client_repos=pmm2-client testing' \
				  	-var 'pmm_client_repo_name=pmm2-client-testing-x86_64' \
				  	-var 'pmm2_server_repo=testing' \
				  	-only amazon-ebs '-color=false' \
				  	packer/pmm2.json

pmm2-ami-el9:
	mkdir -p update && \
	cp -r ../update/ansible/playbook/* update/ && \
		sed -i 's|become_method: su|become_method: sudo|g' update/tasks/roles/postgres/tasks/main.yml && \
		docker run --rm -v ${HOME}/.aws:/root/.aws -v `pwd`:/build -w /build hashicorp/packer:${PACKER_VERSION} \
		build -var 'pmm_client_repos=pmm2-client experimental' \
					-var 'pmm_client_repo_name=pmm2-client-experimental-x86_64' \
					-var 'pmm2_server_repo=experimental' \
					-only amazon-ebs -color=false \
					packer/pmm2.el9.json

pmm2-ami-el9-rc:
	mkdir -p update && \
			cp -r ../update/ansible/playbook/* update/ && \
			sed -i 's|become_method: su|become_method: sudo|g' update/tasks/roles/postgres/tasks/main.yml && \
			docker run --rm -v ${HOME}/.aws:/root/.aws -v `pwd`:/build -w /build hashicorp/packer:${PACKER_VERSION} \
			build -var 'pmm_client_repos=pmm2-client testing' \
				  	-var 'pmm_client_repo_name=pmm2-client-testing-x86_64' \
				  	-var 'pmm2_server_repo=testing' \
				  	-only amazon-ebs '-color=false' \
				  	packer/pmm2.el9.json

## ----------------- PACKER ------------------
