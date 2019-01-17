help:                 ## Display this help message.
	@echo "Please use \`make <target>\` where <target> is one of:"
	@grep '^[a-zA-Z]' $(MAKEFILE_LIST) | \
	    awk -F ':.*?## ' 'NF==2 {printf "  %-26s%s\n", $$1, $$2}'

cert:                 ## Generate TLS certificate for nginx for local testing with mkcert.
	cd nginx && mkcert pmm-api pmm-api.localhost 127.0.0.1

init:                 ## Install prototool and fill vendor/.
	# https://github.com/uber/prototool#installation
	curl -L https://github.com/uber/prototool/releases/download/v1.3.0/prototool-$(shell uname -s)-$(shell uname -m) -o ./prototool
	chmod +x ./prototool

	dep ensure -v

gen: clean            ## Generate files.
	go install -v ./vendor/github.com/golang/protobuf/protoc-gen-go \
					./vendor/github.com/mwitkow/go-proto-validators/protoc-gen-govalidators \
					./vendor/github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway \
					./vendor/github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger \
					./vendor/github.com/go-swagger/go-swagger/cmd/swagger

	./prototool all

	find . -name '*.swagger.json' -not -path './vendor/*' -print -exec swagger flatten --with-flatten=expand --with-flatten=remove-unused '{}' --output='{}' \;

	for API in agent_local inventory ; do \
		swagger mixin $$API/json/header.json $$API/*.swagger.json --output=$$API/json/$$API.json ; \
		swagger validate $$API/json/$$API.json ; \
		swagger generate client --spec=$$API/json/$$API.json --target=$$API/json \
			--additional-initialism=pmm \
			--additional-initialism=aws \
			--additional-initialism=rds ; \
	done

	make clean_swagger
	go install -v ./...

clean_swagger:
	find . -name '*.swagger.json' -not -path './vendor/*' -print -delete

clean: clean_swagger  ## Remove generated files.
	find . -name '*.pb.go' -not -path './vendor/*' -print -delete
	find . -name '*.pb.gw.go' -not -path './vendor/*' -print -delete

	for API in agent_local inventory ; do \
		rm -fr $$API/json/client $$API/json/$$API.json ; \
	done

serve:                ## Serve API documentation with nginx.
	# https://127.0.0.1:8443/
	nginx -p . -c nginx/nginx.conf
