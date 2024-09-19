default: run

PMM_TEST_FLAGS ?= -timeout=180s
PMM_TEST_FILES ?= ./...

run:
	go run main.go

release:
	go build

test:                           ## Run tests
	go test $(PMM_TEST_FLAGS) -p 1 -race $(PMM_TEST_FILES)

test-cover:                     ## Run tests and collect per-package coverage information
	go test $(PMM_TEST_FLAGS) -p 1 -race -coverprofile=cover.out -covermode=atomic -coverpkg=$(PMM_TEST_FILES) $(PMM_TEST_FILES)

ifeq ($(PROFILES),)
PROFILES := 'pmm'
endif

env-up: 							## Start devcontainer
	COMPOSE_PROFILES=$(PROFILES) \
	docker compose up -d --wait
