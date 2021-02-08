---
written_by: Paul Jacobs
            Fábio Silva
reviewed_by: Paul Jacobs
             Someone else
reviewed_on: DATE
---

# Docker (Client)

Avoid installing the PMM Client package by using our [PMM Client Docker image](https://hub.docker.com/r/percona/pmm-client/tags/).

1. Install [Docker](https://docs.docker.com/get-docker/).

2. Pull the PMM Client docker image.

	    docker pull percona/pmm-client:2

3. Create a persistent data store (to preserve local data when pulling an updated image).

	    docker create -v /srv --name pmm-client-data percona/pmm-client:2 /bin/true

4. Run the container (starts PMM Agent in setup mode).

	    docker run --rm \
	    -e PMM_AGENT_SERVER_ADDRESS=pmm-server-IP-address:443 \
	    -e PMM_AGENT_SERVER_USERNAME=admin \
	    -e PMM_AGENT_SERVER_PASSWORD=admin \
	    -e PMM_AGENT_SERVER_INSECURE_TLS=1 \
	    -e PMM_AGENT_SETUP=1 \
	    -e PMM_AGENT_CONFIG_FILE=pmm-agent.yml \
	    --volumes-from pmm-client-data percona/pmm-client:2

## Connect to a Docker PMM Server by container name

You can connect to a Dockerized PMM Server by name instead of IP.

1. Put both containers on a non-default network.

	1. Create a network.

			docker network create <network-name>

	2. Connect a container to that network.

			docker network connect <network-name> <container>

2. Run the container with `PMM_AGENT_SERVER_ADDRESS` as container name instead of IP.

	    docker run --rm \
	    -e PMM_AGENT_SERVER_ADDRESS=your-pmm-server-container-name:443 \
	    -e PMM_AGENT_SERVER_USERNAME=admin \
	    -e PMM_AGENT_SERVER_PASSWORD=admin \
	    -e PMM_AGENT_SERVER_INSECURE_TLS=1 \
	    -e PMM_AGENT_SETUP=1 \
	    -e PMM_AGENT_CONFIG_FILE=pmm-agent.yml \
	    --volumes-from pmm-client-data percona/pmm-client:2

!!! alert alert-success "Tips"
    - Adjust host firewall and routing rules to allow Docker communications. ([Read more in the FAQ.](../../faq.md#how-do-i-troubleshoot-communication-issues-between-pmm-client-and-pmm-server))
	- To get help: `docker run --rm percona/pmm-client:2 --help`

!!! seealso "See also"
    [pmm-agent options and environment](../../details/commands/pmm-agent.md#options-and-environment)

<!--
TODO
- How to stop Docker image
- How to run 'pmm-admin add' and other client commands via Docker
-->
