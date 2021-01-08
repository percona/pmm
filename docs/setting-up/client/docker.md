# Docker

A PMM Client docker image is available from [percona/pmm-client](https://hub.docker.com/r/percona/pmm-client/tags/).

It runs with Docker 1.12.6 or later.

!!! alert alert-success "Tip"
    Make sure that the firewall and routing rules of the host do not constrain the Docker container. ([Read more in the FAQ.](../../faq.md#how-do-i-troubleshoot-communication-issues-between-pmm-client-and-pmm-server))

The Docker image is a collection of preinstalled software which lets you run a selected version of PMM Client.

The Docker image is not run directly.

You use it to create a Docker container for your PMM Client.

When launched, the Docker container gives access to the whole functionality of PMM Client.

## Running PMM Client as a Docker container

1. Pull the image

        docker pull percona/pmm-client:2

2. Create a persistent data store

        docker create -v /srv --name pmm-client-data percona/pmm-client:2 /bin/true

    !!! alert alert-info "Note"
        This container does not run, but exists only to make sure you retain all PMM data when upgrading to a newer image.

3. Run the container

        docker run --rm \
            -e PMM_AGENT_SERVER_ADDRESS=<your-pmm-server-IP-address>:443 \
            -e PMM_AGENT_SERVER_USERNAME=admin \
            -e PMM_AGENT_SERVER_PASSWORD=admin \
            -e PMM_AGENT_SERVER_INSECURE_TLS=1 \
            -e PMM_AGENT_SETUP=1 \
            -e PMM_AGENT_CONFIG_FILE=pmm-agent.yml \
            --volumes-from pmm-client-data percona/pmm-client:2
            
    **Connecting to a Docker PMM Server by container name**
    
    To connect to a Dockerized PMM Server by name instead of IP:

    1. Put both containers on a non-default network:
    
        - `docker network create <network-name>` to create a network,
        - `docker network connect <network-name> <container>` to connect a container to that network.
    
    2. Use `-e PMM_AGENT_SERVER_ADDRESS=<your-pmm-server-container-name>:443`.

!!! alert alert-success "Tip"
    To get help:

        docker run --rm percona/pmm-client:2 --help

!!! seealso "See also"
    - [pmm-agent options and environment](../../details/commands/pmm-agent.md#options-and-environment)
    - [Docker documentation](https://docs.docker.com)
