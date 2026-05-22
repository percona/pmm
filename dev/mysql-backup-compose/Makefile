# PMM Server + MySQL backup local development environment.

COMPOSE_FILE ?= docker-compose.yml
CONTAINER ?= pmm-backup-server
WAIT_TIMEOUT ?= 300

.PHONY: help env-up env-up-rebuild env-down env-wait env-status env-logs env-register

help: ## Display this help message
	@echo "Please use \`make <target>\` where <target> is one of:"
	@grep '^[a-zA-Z]' $(MAKEFILE_LIST) | \
		awk -F ':.*?## ' 'NF==2 {printf "  %-20s%s\n", $$1, $$2}'

env-up: ## Build (if needed) and start PMM Server with MySQL + XtraBackup
	docker compose -f $(COMPOSE_FILE) up -d --build --wait --wait-timeout $(WAIT_TIMEOUT)
	@$(MAKE) env-wait
	@$(MAKE) env-status

env-up-rebuild: ## Rebuild the server image from scratch and start
	docker compose -f $(COMPOSE_FILE) build --no-cache
	docker compose -f $(COMPOSE_FILE) up -d --wait --wait-timeout $(WAIT_TIMEOUT)
	@$(MAKE) env-wait
	@$(MAKE) env-status

env-down: ## Stop containers and remove all data volumes
	docker compose -f $(COMPOSE_FILE) down -v --remove-orphans
	@for volume in $$(docker volume ls -q --filter label=com.docker.compose.project=pmm-mysql-backup); do \
		docker volume rm $$volume; \
	done

env-wait: ## Wait until mysql-backup is registered in PMM
	@echo "Waiting for mysql-backup service registration..."
	@for i in $$(seq 1 $(WAIT_TIMEOUT)); do \
		if docker exec $(CONTAINER) /usr/local/percona/pmm/bin/pmm-admin list \
			--server-url="https://$${PMM_ADMIN_USER:-admin}:$${PMM_ADMIN_PASSWORD:-admin}@127.0.0.1:8443/" \
			--server-insecure-tls 2>/dev/null | grep -q 'mysql-backup'; then \
			echo "mysql-backup is registered"; \
			exit 0; \
		fi; \
		sleep 1; \
	done; \
	echo "Timed out waiting for mysql-backup registration"; \
	$(MAKE) env-logs; \
	exit 1

env-status: ## Show mysqld, agent, and registered services
	@echo "=== supervisord programs ==="
	@docker exec $(CONTAINER) supervisorctl status mysqld-backup-dev mysql-backup-register pmm-agent 2>/dev/null || true
	@echo "=== mysqld ==="
	@docker exec $(CONTAINER) sh -c 'mysqladmin ping -h 127.0.0.1 -uroot -p"$${MYSQL_ROOT_PASSWORD:-secret}" --silent' 2>/dev/null \
		&& echo "mysqld: ready" || echo "mysqld: not ready"
	@echo "=== pmm-admin ==="
	@docker exec $(CONTAINER) /usr/local/percona/pmm/bin/pmm-admin status 2>/dev/null | head -15 || true
	@docker exec $(CONTAINER) /usr/local/percona/pmm/bin/pmm-admin list \
		--server-url="https://$${PMM_ADMIN_USER:-admin}:$${PMM_ADMIN_PASSWORD:-admin}@127.0.0.1:8443/" \
		--server-insecure-tls 2>/dev/null || true

env-logs: ## Tail registration and mysqld logs
	docker exec $(CONTAINER) sh -c 'tail -n 50 /srv/logs/mysql-backup-register.log /srv/logs/mysqld-backup-dev.log 2>/dev/null' || true

env-register: ## Re-run MySQL service registration manually
	docker exec $(CONTAINER) /usr/local/bin/register-mysql.sh
