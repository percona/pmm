all: test

PACKAGES := $(shell go list ./... | grep -v vendor)

# installs tools to $GOPATH/bin which is expected to be in $PATH
init:
	go install -v ./vendor/gopkg.in/reform.v1/reform

	go install -v ./vendor/github.com/prometheus/prometheus/cmd/promtool

	go install -v ./vendor/github.com/golang/protobuf/protoc-gen-go
	go install -v ./vendor/github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
	go install -v ./vendor/github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger
	go install -v ./vendor/github.com/go-swagger/go-swagger/cmd/swagger

	go get -u github.com/AlekSi/gocoverutil

	go get -u gopkg.in/alecthomas/gometalinter.v1
	gometalinter.v1 --install

install:
	go install -v $(PACKAGES)
	go test -v -i $(PACKAGES)

install-race:
	go install -v -race $(PACKAGES)
	go test -v -race -i $(PACKAGES)

test: install
	go test -v $(PACKAGES)

test-race: install-race
	go test -v -race $(PACKAGES)

cover: install
	gocoverutil -ignore=github.com/percona/pmm-managed/api/... test -v $(PACKAGES)

check: install
	-gometalinter.v1 --tests --skip=api --deadline=180s ./...

run: install
	pmm-managed -prometheus-config=testdata/prometheus/prometheus.yml -swagger=rest

run-race: install-race
	pmm-managed -prometheus-config=testdata/prometheus/prometheus.yml -swagger=rest

gen:
	rm -f models/*_reform.go
	reform models/

	rm -fr api/*.pb.* api/swagger/*.json api/swagger/client api/swagger/models

	protoc -Iapi -Ivendor/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		api/*.proto --go_out=plugins=grpc:api
	protoc -Iapi -Ivendor/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		api/*.proto --grpc-gateway_out=logtostderr=true,request_context=true,allow_delete_body=true:api
	protoc -Iapi -Ivendor/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		api/*.proto --swagger_out=logtostderr=true,allow_delete_body=true:api/swagger

	swagger mixin api/swagger/*.swagger.json > api/swagger/swagger.json
	swagger validate api/swagger/swagger.json
	swagger generate client -f api/swagger/swagger.json -t api/swagger -A pmm-managed

	go install -v github.com/percona/pmm-managed/api github.com/percona/pmm-managed/api/swagger/client
