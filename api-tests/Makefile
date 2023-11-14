all: build

init:           ## Installs development tools
	cd tools && go generate -x -tags=tools

build:
	go install -v ./...
	go test -c -v ./inventory
	go test -c -v ./management
	go test -c -v ./server

run:

run-race:
	go test -count=1 -p 1 -v -race ./... 2>&1 | tee pmm3-api-tests-output.txt
	cat pmm3-api-tests-output.txt | bin/go-junit-report > pmm3-api-tests-junit-report.xml

clean:
	rm -f ./pmm3-api-tests-output.txt
	rm -f ./pmm3-api-tests-junit-report.xml
