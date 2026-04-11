PROJECT_NAME  := pmm-managed
BUILD_SOURCE  := ./cmd/...
TEST_FLAGS    ?= -timeout=180s
TEST_PARALLEL := -p 1

include ../go.mk

IS_ARM        := $(filter arm64,$(shell uname -m))
PMM_RACE_FLAG ?= $(if $(IS_ARM),,-race)

PMM_TEST_FILES ?= ./...

gen: clean                      ## Generate files
	go generate ./...

clean:                          ## Remove generated files
	find . -name *_reform.go -delete

pi-validate:                    ## Validate Percona Intelligence advisors and checks
	go run ./cmd/pi-validator/main.go advisors --advisors.dir ./data/advisors/ --checks.dir ./data/checks/

release:                        ## Build pmm-managed release binaries
	env CGO_ENABLED=0 go build -v $(LD_FLAGS) -o $(PMM_RELEASE_PATH)/ ./cmd/...

release-encryption-rotation:    ## Build PMM encryption rotation tool
	env CGO_ENABLED=0 go build -v $(LD_FLAGS) -o $(PMM_RELEASE_PATH)/ ./cmd/pmm-encryption-rotation/...

release-starlark:               ## Build startlark binary
	env CGO_ENABLED=0 go build -v $(LD_FLAGS) -o $(PMM_RELEASE_PATH)/ ./cmd/pmm-managed-starlark/...
	$(PMM_RELEASE_PATH)/pmm-managed-starlark --version

release-dev:										## Build pmm-managed binaries for development
	go build $(PMM_RACE_FLAG) -gcflags="all=-N -l" -v $(LD_FLAGS) -o $(PMM_RELEASE_PATH)/ ./cmd/...

test:                           ## Run tests
	go test $(TEST_FLAGS) -p 1 -race $(PMM_TEST_FILES)

test-cover:                     ## Run tests and collect per-package coverage information
	go test $(TEST_FLAGS) -p 1 -race -coverprofile=cover.out -covermode=atomic -coverpkg=$(PMM_TEST_FILES) $(PMM_TEST_FILES)

fuzz-grafana:                   ## Run fuzzer for services/grafana package
	# go get -u github.com/dvyukov/go-fuzz/go-fuzz github.com/dvyukov/go-fuzz/go-fuzz-build
	mkdir -p managed/services/grafana/fuzzdata/corpus
	cd managed/services/grafana && go-fuzz-build
	cd managed/services/grafana && go-fuzz -workdir=fuzzdata
