# Docker

The [PMM Client Docker image](https://hub.docker.com/r/percona/pmm-client/tags/) is a convenient way to run PMM Client as a preconfigured [Docker](https://docs.docker.com/get-docker/) container.

```plantuml source="_resources/diagrams/Setting-Up_Client_Docker.puml"
```

1. Pull the PMM Client docker image.

	```sh
    docker pull \
	percona/pmm-client:2
	```

2. Use the image as a template to create a persistent data store that preserves local data when the image is updated.

	```sh
    docker create \
	--volume /srv \
	--name pmm-client-data \
	percona/pmm-client:2 /bin/true
	```

3. Run the container to start [PMM Agent](../../details/commands/pmm-agent.md) in setup mode. Set `X.X.X.X` to the IP address of your PMM Server. (Do not use the `docker --detach` option as PMM agent only logs to the console.)

	```sh
	PMM_SERVER=X.X.X.X:443
    docker run \
	--rm \
	--name pmm-client \
    -e PMM_AGENT_SERVER_ADDRESS=${PMM_SERVER} \
    -e PMM_AGENT_SERVER_USERNAME=admin \
    -e PMM_AGENT_SERVER_PASSWORD=admin \
    -e PMM_AGENT_SERVER_INSECURE_TLS=1 \
    -e PMM_AGENT_SETUP=1 \
    -e PMM_AGENT_CONFIG_FILE=pmm-agent.yml \
    --volumes-from pmm-client-data \
	percona/pmm-client:2
	```

4. Check status.

	```sh
	docker exec	pmm-client \
	pmm-admin status
	```

	In the PMM user interface you will also see an increase in the number of monitored nodes.

You can now add services with [`pmm-admin`](../../details/commands/pmm-admin.md) by prefixing commands with `docker exec pmm-client`.

!!! alert alert-success "Tips"
    - Adjust host firewall and routing rules to allow Docker communications. ([Read more in the FAQ.](../../faq.md#how-do-i-troubleshoot-communication-issues-between-pmm-client-and-pmm-server))
	- For help:
		```sh
		docker run --rm percona/pmm-client:2 --help
		```
