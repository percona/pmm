default: help

help:                                                ## Display this help message.
	@echo "Please use \`make <target>\` where <target> is one of:"
	@grep '^[a-zA-Z]' $(MAKEFILE_LIST) | \
	    awk -F ':.*?## ' 'NF==2 {printf "  %-26s%s\n", $$1, $$2}'

all: root-ssl server-ssl pmm-managed-ssl grafana-ssl ## Generates all required dev certs.
	rm -f ./certs/*.srl

root-ssl:                                            ## Generates root-ssl cert.
	openssl req -new -sha256 -nodes -newkey rsa:2048 \
		-config ./certs/root.cnf \
		-keyout ./certs/root.key \
		-out ./certs/root.csr
	openssl x509 -req -days 3653 -sha256 \
		-in ./certs/root.csr  \
		-extfile /etc/ssl/openssl.cnf -extensions v3_ca \
		-signkey ./certs/root.key \
		-out ./certs/root.crt

server-ssl:                                          ## Generates server-ssl cert.
	openssl req -new -sha256 -nodes -newkey rsa:2048 \
		-config ./certs/server.cnf \
		-keyout ./certs/server.key \
		-out /tmp/server.csr
	openssl x509 -req -days 3653 -sha256 \
		-extfile ./certs/server.cnf -extensions req_ext \
		-CA ./certs/root.crt -CAkey ./certs/root.key -CAcreateserial \
		-in /tmp/server.csr \
		-out ./certs/server.crt
	chmod 600 certs/server.crt
	chmod 600 certs/server.key

pmm-managed-ssl:                                    ## Generates pmm-managed-ssl cert.
	openssl req -new -sha256 -nodes -newkey rsa:2048 \
		-config ./certs/pmm-managed.cnf \
		-keyout ./certs/pmm-managed.key \
		-out /tmp/pmm-managed.csr
	openssl x509 -req -days 3653 -sha256 \
		-CA ./certs/root.crt -CAkey /tmp/root.key -CAcreateserial \
		-in /tmp/pmm-managed.csr \
		-out ./certs/pmm-managed.crt
	chmod 600 certs/pmm-managed.crt
	chmod 600 certs/pmm-managed.key

grafana-ssl:                                    ## Generates grafana-ssl cert.
	openssl req -new -sha256 -nodes -newkey rsa:2048 \
		-config ./certs/grafana.cnf \
		-keyout ./certs/grafana.key \
		-out /tmp/grafana.csr
	openssl x509 -req -days 3653 -sha256 \
		-CA ./certs/root.crt -CAkey /tmp/root.key -CAcreateserial \
		-in /tmp/grafana.csr \
		-out ./certs/grafana.crt
	chmod 600 certs/grafana.crt
	chmod 600 certs/grafana.key

grafana-ssl:                                        ## Generates grafana-ssl cert.
	openssl req -new -sha256 -nodes -newkey rsa:2048 \
		-config ./certs/grafana.cnf \
		-keyout ./certs/grafana.key \
		-out /tmp/grafana.csr
	openssl x509 -req -days 3653 -sha256 \
		-CA ./certs/root.crt -CAkey /tmp/root.key -CAcreateserial \
		-in /tmp/grafana.csr \
		-out ./certs/grafana.crt