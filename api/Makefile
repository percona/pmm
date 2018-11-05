help:        ## Display this help message.
	@echo "Please use \`make <target>\` where <target> is one of:"
	@grep '^[a-zA-Z]' $(MAKEFILE_LIST) | \
	    awk -F ':.*?## ' 'NF==2 {printf "  %-26s%s\n", $$1, $$2}'

cert:        ## Generate TLS certificate for nginx for local testing with mkcert.
	cd nginx && mkcert pmm-api pmm-api.test pmm-api.localhost 127.0.0.1

init:        ## Install prototool and fill vendor/.
	# https://github.com/uber/prototool#installation
	curl -L https://github.com/uber/prototool/releases/download/v1.3.0/prototool-$(shell uname -s)-$(shell uname -m) -o ./prototool
	chmod +x ./prototool

	dep ensure -v

gen: clean   ## Generate files.
	go install -v ./vendor/github.com/golang/protobuf/protoc-gen-go \
					./vendor/github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway \
					./vendor/github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger \
					./vendor/github.com/go-swagger/go-swagger/cmd/swagger

	./prototool all

	# no public API for pmm-agent
	rm -f agent/*.swagger.json

	swagger mixin inventory/inventory.json inventory/*.swagger.json --output=inventory.swagger.json
	swagger validate inventory.swagger.json
	rm -f inventory/*.swagger.json

	mkdir json
	swagger generate client --spec=inventory.swagger.json --target=json \
		--additional-initialism=pmm \
		--additional-initialism=rds
	go install -v ./...

clean:       ## Remove generated files.
	find . -name '*.pb.go' -not -path './vendor/*' -delete
	find . -name '*.pb.gw.go' -not -path './vendor/*' -delete
	find . -name '*.swagger.json' -not -path './vendor/*' -delete

	rm -fr json

serve:       ## Serve API documentation with nginx.
	# https://pmm-api.test:8443/
	nginx -p . -c nginx/nginx.conf
