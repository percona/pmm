all: build

init:           ## Installs development tools
	cd tools && go generate -x -tags=tools

build:
	go install -v ./...
	go test -c -v ./alerting
	go test -c -v ./backup
	go test -c -v ./inventory
	go test -c -v ./management
	go test -c -v ./server

run:
	go test -count=1 -p 1 -v ./... 2>&1 | tee pmm-api-tests-output.txt
	cat pmm-api-tests-output.txt | bin/go-junit-report > pmm-api-tests-junit-report.xml

run-dev:
	go test -count=1 -p 1 -v ./...

run-race:
	go test -count=1 -p 1 -v -race ./... 2>&1 | tee pmm-api-tests-output.txt
	cat pmm-api-tests-output.txt | bin/go-junit-report > pmm-api-tests-junit-report.xml

clean:
	rm -f ./pmm-api-tests-output.txt
	rm -f ./pmm-api-tests-junit-report.xml
