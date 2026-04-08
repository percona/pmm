all: build

init:           ## Installs development tools
	cd tools && go generate -x -tags=tools

build:
	go install -v ./...
	go test -c -v ./alerting
	go test -c -v ./backup
	go test -c -v ./inventory
	go test -c -v ./management
	go test -c -v -tags settings ./server
	go test -c -v ./user

run:
	go test -count=1 -p 1 -v ./... 2>&1 | tee pmm-api-tests-output.txt
	go test -count=1 -p 1 -v -tags settings ./server/... 2>&1 | tee -a pmm-api-tests-output.txt
	cat pmm-api-tests-output.txt | bin/go-junit-report > pmm-api-tests-junit-report.xml

run-dev:
	go test -count=1 -p 1 -v ./...
	go test -count=1 -p 1 -v -tags settings ./server/...

run-race:
	go test -count=1 -p 1 -v -race ./... 2>&1 | tee pmm-api-tests-output.txt
	go test -count=1 -p 1 -v -race -tags settings ./server/... 2>&1 | tee -a pmm-api-tests-output.txt
	cat pmm-api-tests-output.txt | bin/go-junit-report > pmm-api-tests-junit-report.xml

clean:
	rm -f ./pmm-api-tests-output.txt
	rm -f ./pmm-api-tests-junit-report.xml
