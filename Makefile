# Devcontainer Makefile.

include Makefile.include

release-dev-managed:              ## Build pmm-managed
	make -C managed release-dev

release-dev-agent:                ## Build pmm-agent
	make -C agent release-dev

release-vmproxy:                  ## Build vmproxy
	make -C vmproxy release

release-qan:                      ## Build QAN
	make -C qan-api2 release

# used by host Makefile
_bash:
	/bin/bash

PMM_RELEASE_PATH ?= ./bin

run-managed: release-dev-managed    ## Replace pmm-managed from build, restart and tail logs
	supervisorctl stop pmm-managed
	cp $(PMM_RELEASE_PATH)/pmm-managed /usr/sbin/pmm-managed
	supervisorctl start pmm-managed &
	supervisorctl tail -f pmm-managed

run-managed-ci: release-dev-managed    ## Replace pmm-managed from build, restart (used in CI)
	supervisorctl stop pmm-managed
	cp $(PMM_RELEASE_PATH)/pmm-managed /usr/sbin/pmm-managed
	supervisorctl start pmm-managed

run-agent: release-dev-agent        ## Replace pmm-agent from build and restart
	supervisorctl stop pmm-agent
	cp $(PMM_RELEASE_PATH)/pmm-agent /usr/sbin/pmm-agent
	supervisorctl start pmm-agent

run-vmproxy: release-vmproxy
	supervisorctl stop vmproxy
	cp $(PMM_RELEASE_PATH)/vmproxy /usr/sbin/vmproxy
	supervisorctl start vmproxy

run-qan: release-qan
	supervisorctl stop qan-api2
	cp $(PMM_RELEASE_PATH)/qan-api2 /usr/sbin/percona-qan-api2
	echo -n > /srv/logs/qan-api2.log
	supervisorctl start qan-api2

run-all: run-agent run-managed       ## Run pmm-managed and pmm-agent

run:                                 ## Deprecated
	echo "Deprecated: please use run-all"

# TODO https://jira.percona.com/browse/PMM-3484, see maincover_test.go
# run-race-cover: install-race    ## Run pmm-managed with race detector and collect coverage information.
# 	go test -coverpkg="github.com/percona/pmm/managed/..." \
# 			-tags maincover \
# 			$(PMM_LD_FLAGS) \
# 			-race -c -o bin/pmm-managed.test
# 	bin/pmm-managed.test -test.coverprofile=cover.out -test.run=TestMainCover $(RUN_FLAGS)

psql:                           ## Open database for the pmm-managed instance in psql shell
	env PGPASSWORD=pmm-managed psql -U pmm-managed pmm-managed

psql-test:                      ## Open database used in unit tests in psql shell
	env psql -U postgres pmm-managed-dev

dlv/attach:                     ## Attach Delve to `pmm-managed`
	dlv --listen=:2345 --headless=true --api-version=2 --accept-multiclient attach $(shell pgrep pmm-managed)

refresh-swagger:                ## Refresh swagger files
	cp /root/go/src/github.com/percona/pmm/api/swagger/swagger.json /usr/share/pmm-managed/swagger/swagger.json
	cp /root/go/src/github.com/percona/pmm/api/swagger/swagger-dev.json /usr/share/pmm-managed/swagger/swagger-dev.json
