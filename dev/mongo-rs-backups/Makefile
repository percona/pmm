PMM_SERVER_HOST?=pmm-server
PMM_NETWORK=?pmm_default

.PHONY: default, bootstrap, build, up, setup-pbm-permissions, run-pbm-agents, setup-pmm-agents, run-pbm-agents, kill-pbm-agents, kill-pmm-agents, add-mongos, down
default: help

help:                            ## Display this help message
	@echo "Please use \`make <target>\` where <target> is one of:"
	@grep '^[a-zA-Z]' $(MAKEFILE_LIST) | \
		awk -F ':.*?## ' 'NF==2 {printf "  %-26s%s\n", $$1, $$2}'

bootstrap:                       ## Full environment setup
	make build
	make up
	make setup-pbm-permissions
	make run-pbm-agents
	make setup-pmm-agents
	make run-pmm-agents
	make add-mongos

build:                           ## Docker image build
	docker build . --tag mongo-backups

up:                              ## Start required containers
	docker compose up -d --wait
	docker compose exec -T mongo1 mongo --nodb < rs-bootstrap.js

setup-pbm-permissions:           ## Setup PBM required permissions
	docker compose exec -T mongo1 mongo --host=mongodb://localhost:27017/?replicaSet=rs0 < permissions.js

run-pbm-agents: kill-pbm-agents  ## Run PBM agents
	docker compose exec -d mongo1 pbm-agent --mongodb-uri=mongodb://pbmuser:secretpwd@localhost:27017
	docker compose exec -d mongo2 pbm-agent --mongodb-uri=mongodb://pbmuser:secretpwd@localhost:27017
	docker compose exec -d mongo3 pbm-agent --mongodb-uri=mongodb://pbmuser:secretpwd@localhost:27017

setup-pmm-agents:                ## Setup PMM agents
	docker compose exec mongo1 pmm-agent setup --server-insecure-tls --server-address=$(PMM_SERVER_HOST) --server-username=admin --server-password=admin --force --config-file=/root/pmm-config
	docker compose exec mongo2 pmm-agent setup --server-insecure-tls --server-address=$(PMM_SERVER_HOST) --server-username=admin --server-password=admin --force --config-file=/root/pmm-config
	docker compose exec mongo3 pmm-agent setup --server-insecure-tls --server-address=$(PMM_SERVER_HOST) --server-username=admin --server-password=admin --force --config-file=/root/pmm-config

run-pmm-agents: kill-pmm-agents   ## Run PMM agents
	docker compose exec -d mongo1 pmm-agent --config-file=/root/pmm-config
	docker compose exec -d mongo2 pmm-agent --config-file=/root/pmm-config
	docker compose exec -d mongo3 pmm-agent --config-file=/root/pmm-config

kill-pbm-agents:                  ## Kill all PBM agents
	docker compose exec -d mongo1 killall -9 pbm-agent
	docker compose exec -d mongo2 killall -9 pbm-agent
	docker compose exec -d mongo3 killall -9 pbm-agent

kill-pmm-agents:                  ## Kill all PMM agents
	docker compose exec -d mongo1 killall -9 pmm-agent
	docker compose exec -d mongo2 killall -9 pmm-agent
	docker compose exec -d mongo3 killall -9 pmm-agent

add-mongos:                       ## Register MongoDB instances on PMM server
	docker compose exec mongo1 pmm-admin add mongodb --cluster=test-cluster --service-name=mongo1  --host=mongo1 --port=27017 --server-url=http://admin:admin@$(PMM_SERVER_HOST)/
	docker compose exec mongo2 pmm-admin add mongodb --cluster=test-cluster --service-name=mongo2  --host=mongo2 --port=27017 --server-url=http://admin:admin@$(PMM_SERVER_HOST)/
	docker compose exec mongo3 pmm-admin add mongodb --cluster=test-cluster --service-name=mongo3  --host=mongo3 --port=27017 --server-url=http://admin:admin@$(PMM_SERVER_HOST)/

down:                            ## Shutdown environment
	docker compose down
