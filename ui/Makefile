dev:
	yarn dev

ci: setup lint test build

format: setup
	yarn format

lint: setup
	yarn lint

test: setup
	yarn test

setup:
	yarn install

build: setup
	yarn build

release: build

clean:
	yarn clean

grafana-be-build:
	docker exec -u root $(PMM_SERVER_CONTAINER) bash -c 'cd $(GRAFANA_SRC_DIR) && go run build.go build-backend'

grafana-be-swap:
	docker exec -u root $(PMM_SERVER_CONTAINER) bash -c 'cp $(GRAFANA_SRC_DIR)/bin/linux-amd64/grafana /usr/sbin/grafana && supervisorctl restart grafana'

grafana-be-refresh: grafana-be-build grafana-be-swap